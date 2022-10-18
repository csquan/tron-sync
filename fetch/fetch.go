package fetch

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/starslabhq/chainmonitor/db"

	"github.com/sirupsen/logrus"
	"github.com/starslabhq/chainmonitor/config"
	"github.com/starslabhq/chainmonitor/utils"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	log "github.com/sirupsen/logrus"
	"github.com/starslabhq/chainmonitor/mtypes"
	"github.com/starslabhq/chainmonitor/output/mysqldb"
)

type Fetcher struct {
	config *config.FetchConf

	Height uint64 //当前处理的blockNum

	interrupt   int32
	interruptCh chan struct{}

	c      *rpc.Client
	Client *ethclient.Client

	db db.IDB

	blockFeed *event.Feed
	startNum  uint64

	blockCache map[uint64]*mtypes.Block
	appName    string
}

func (f *Fetcher) SubBlock(ch chan *mtypes.Block) event.Subscription {
	return f.blockFeed.Subscribe(ch)
}

func (f *Fetcher) Stop() {
	if atomic.LoadInt32(&f.interrupt) == 0 {
		atomic.StoreInt32(&f.interrupt, 1)
		<-f.interruptCh
	}
}

func (f *Fetcher) isInterrupted() bool {
	return atomic.LoadInt32(&f.interrupt) == 1
}

func (f *Fetcher) getCurChainHeight() uint64 {
	curHeight := utils.HandleErrorWithRetryAndReturn(func() (interface{}, error) {
		return f.getBlockNumber()
	}, f.config.FetchRetryTimes, f.config.FetchRetryInterval)

	return curHeight.(uint64) - f.config.BlocksDelay
}

func (f *Fetcher) getLocalLatestBlock() *mysqldb.Block {
	blk := utils.HandleErrorWithRetryAndReturn(func() (interface{}, error) {
		return f.db.GetCurrentBlock()
	}, f.config.FetchRetryTimes, f.config.FetchRetryInterval)

	return blk.(*mysqldb.Block)
}

// [start,end)
func (f *Fetcher) Run(start, end uint64) {
	if start == 0 {
		curBlock := f.getLocalLatestBlock()
		if curBlock != nil {
			start = curBlock.Number + 1
		}
	}

	var continueSync bool
	if end == 0 {
		end = f.getCurChainHeight()
		continueSync = true
	}

	f.startNum = start
	f.Height = start

	firstLoop := true

	for {
		if f.isInterrupted() {
			f.interruptCh <- struct{}{}
			log.Info("fetch stop")
			return
		}

		for f.Height < end {
			if f.isInterrupted() {
				f.interruptCh <- struct{}{}
				return
			}

			var (
				latestNormalHeight uint64
				prevHash           string
				revertBlocks       []*mtypes.Block
			)
			//handle revert, if height long behind (20 blocks or more) from current height, revert won't happen
			if firstLoop || f.Height+f.config.BlocksSafe > end {
				firstLoop = false

				latestNormalHeight, prevHash, revertBlocks = f.getLatestNormalBlockInfoInDB(f.Height - 1)
				logrus.Debugf("get start height:%d", latestNormalHeight)
				if latestNormalHeight < (f.Height - 1) {
					sort.Slice(revertBlocks, func(i, j int) bool {
						return revertBlocks[i].Number > revertBlocks[j].Number
					})
					for _, blk := range revertBlocks {
						blk.State = mysqldb.Block_revert
						logrus.Infof("revert blck num:%d,hash:%s", blk.Number, blk.Hash)
						delete(f.blockCache, blk.Number)
						f.blockFeed.Send(blk)
					}
				}
				f.Height = latestNormalHeight + 1
			}

			heightEnd := f.Height + f.config.BlockBatch
			if heightEnd > end {
				heightEnd = end
			}
			bch, errch := f.GetBlocks(f.Height, heightEnd)
			blocks := make([]*mtypes.Block, 0)

			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				for b := range bch {
					blocks = append(blocks, b)
				}
				sort.Slice(blocks, func(i, j int) bool {
					return blocks[i].Number < blocks[j].Number
				})
			}()

			var errCnt int
			go func() {
				defer wg.Done()
				for err := range errch {
					if err != nil {
						errCnt++
						logrus.Errorf("getBlocks err:%v,s:%d,e:%d", err, f.Height, heightEnd)
					}
				}
				log.Debugf("wait err ok")
			}()
			wg.Wait()
			if errCnt != 0 {
				continue
			}
			st := time.Now()
			for _, b := range blocks {
				//make sure it's a chain
				if prevHash != "" && b.ParentHash != prevHash {
					break
				}

				prevHash = b.Hash

				delete(f.blockCache, b.Number-f.config.BlocksSafe*2)
				f.blockCache[b.Number] = b

				f.blockFeed.Send(b)
				f.Height = b.Number + 1
			}
			logrus.Infof("pub block cost:%v,bsize:%d", time.Since(st), len(blocks))
		}

		if !continueSync {
			log.Infof("fetch finished")
			f.interruptCh <- struct{}{}
			f.Stop()
			return
		}

		if !f.isInterrupted() {
			time.Sleep(f.config.BlockWaitTime)
			end = f.getCurChainHeight() + 1
			log.Infof("height now:%d,start:%d,delay:%d", end+f.config.BlocksDelay, f.Height, f.config.BlocksDelay)
		}
	}
}

func NewFetcher(c *rpc.Client, db db.IDB, config *config.FetchConf, enableCache bool, appName string) (*Fetcher, error) {
	ethClient := ethclient.NewClient(c)

	f := &Fetcher{
		config:      config,
		Client:      ethClient,
		c:           c,
		db:          db,
		interruptCh: make(chan struct{}, 1),
		blockFeed:   &event.Feed{},
		blockCache:  make(map[uint64]*mtypes.Block),
		appName:     appName,
	}

	if enableCache {
		err := f.loadLocalBlocksToCache()
		return f, err
	}

	return f, nil
}

func (f *Fetcher) loadLocalBlocksToCache() error {
	endHeight, err := f.db.GetCurrentMainBlockNum()
	if err != nil {
		return err
	}

	if endHeight > 0 {
		var startHeight uint64
		if startHeight < endHeight-f.config.BlocksSafe*2 {
			startHeight = endHeight - f.config.BlocksSafe*2
		}

		blks, err := f.db.GetBlocksByRange(startHeight, endHeight, mysqldb.DefaultFullBlockFilter)
		if err != nil {
			return err
		}

		for _, blk := range blks {
			f.blockCache[blk.Number] = blk
		}
	}

	return nil
}

func (f *Fetcher) groupTxs(txs []*mtypes.TxJson, batch int) chan []*mtypes.TxJson {
	if batch == 0 {
		batch = 1
	}
	var end int
	size := len(txs)
	if size == 0 || batch == 0 {
		return nil
	}
	chSize := size/batch + 1
	var ret = make(chan []*mtypes.TxJson, chSize)
	go func() {
		for i := 0; i < size; {
			if i+batch < size {
				end = i + batch
			} else {
				end = size
			}
			txsPart := txs[i:end]
			ret <- txsPart
			i = end
		}
		close(ret)
	}()
	return ret
}

// 这里暂时不用以太坊标准的结构--底层返回的结构不全

// Log represents a contract log event. These events are generated by the LOG opcode and
// stored/indexed by the node.
type Logtmp struct {
	// Consensus fields:
	// address of the contract that generated the event
	Address common.Address `json:"address" gencodec:"required"`
	// list of topics provided by the contract.
	Topics []common.Hash `json:"topics" gencodec:"required"`
	// supplied by the contract, usually ABI-encoded
	Data []byte `json:"data" gencodec:"required"`

	// Derived fields. These fields are filled in by the node
	// but not secured by consensus.
	// block in which the transaction was included
	BlockNumber uint64 `json:"blockNumber" rlp:"-"`
	// hash of the transaction
	TxHash common.Hash `json:"transactionHash" gencodec:"required" rlp:"-"`
	// index of the transaction in the block
	TxIndex uint `json:"transactionIndex" rlp:"-"`
	// hash of the block in which the transaction was included
	BlockHash common.Hash `json:"blockHash" rlp:"-"`
	// index of the log in the block
	Index uint `json:"logIndex" rlp:"-"`

	// The Removed field is true if this log was reverted due to a chain reorganisation.
	// You must pay attention to this field if you receive logs through a filter query.
	Removed bool `json:"removed" rlp:"-"`
}

type Receiptmp struct {
	// Consensus fields: These fields are defined by the Yellow Paper
	Type              uint8  `json:"type,omitempty"`
	PostState         []byte `json:"root"`
	Status            uint64 `json:"status"`
	CumulativeGasUsed uint64 `json:"cumulativeGasUsed" gencodec:"required"`
	//Bloom             Bloom  `json:"logsBloom"         gencodec:"required"`
	Logs []*Logtmp `json:"logs"              gencodec:"required"`

	// Implementation fields: These fields are added by geth when processing a transaction.
	// They are stored in the chain database.
	TxHash          common.Hash    `json:"transactionHash" gencodec:"required"`
	ContractAddress common.Address `json:"contractAddress"`
	GasUsed         uint64         `json:"gasUsed" gencodec:"required"`

	// Inclusion information: These fields provide information about the inclusion of the
	// transaction corresponding to this receipt.
	BlockHash        common.Hash `json:"blockHash,omitempty"`
	BlockNumber      *big.Int    `json:"blockNumber,omitempty"`
	TransactionIndex uint        `json:"transactionIndex"`
}

func (f *Fetcher) createTxBatch(txs []*mtypes.TxJson, blockNum *big.Int, baseFee *big.Int) ([]*mtypes.Tx, []*mtypes.Contract, error) {
	retTxs := make([]*mtypes.Tx, 0)
	retContracts := make([]*mtypes.Contract, 0)

	elemsReceipts := make([]rpc.BatchElem, 0)
	receipts := make(map[string]*Receiptmp)

	for _, tx := range txs {
		//receipt := &types.Receipt{}
		receipt1 := &Receiptmp{}
		elemsReceipt := rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{tx.Hash},
			Result: receipt1,
		}
		receipts[tx.Hash] = receipt1
		elemsReceipts = append(elemsReceipts, elemsReceipt)
	}
	// call get receipts
	err := f.c.BatchCallContext(context.Background(), elemsReceipts)
	if err != nil {
		return nil, nil, fmt.Errorf("rpc getReceipts err:%v,args:%v", err, elemsReceipts)
	}
	for _, elem := range elemsReceipts {
		if elem.Error != nil {
			return nil, nil, fmt.Errorf("elem err:%v,args:%v,bum:%d", elem.Error, elem.Args, blockNum.Uint64())
		}
	}
	for _, tx := range txs {
		receipt := receipts[tx.Hash]
		var (
			toHex            string = tx.To
			isContractCreate bool
		)
		if toHex == "" || toHex == "0x" || receipt.ContractAddress.Hex() != `0x0000000000000000000000000000000000000000` {
			if receipt.Status == 1 {
				isContractCreate = true
				contract := &mtypes.Contract{
					TxHash:      tx.Hash,
					Addr:        strings.ToLower(receipt.ContractAddress.Hex()),
					CreatorAddr: strings.ToLower(tx.From),
					ExecStatus:  receipt.Status,
				}
				retContracts = append(retContracts, contract)
			}
		}
		var isContract bool
		if len(tx.Input) != 0 && tx.Input != "0x" { //是否调用合约
			isContract = true
		}
		txTmp := tx.ToTx()
		txTmp.GasUsed = receipt.GasUsed
		txTmp.IsContract = isContract
		txTmp.IsContractCreate = isContractCreate
		txTmp.ExecStatus = receipt.Status

		//eip1559
		if txTmp.TxType == types.DynamicFeeTxType {
			txTmp.BaseFee = baseFee
			temp := new(big.Int).SetUint64(txTmp.GasUsed)
			txTmp.BurntFees = temp.Mul(temp, baseFee)
		}

		eventLogs := make([]*mtypes.EventLog, 0)
		for _, item := range receipt.Logs {
			txlog := item

			elog := &mtypes.EventLog{}
			var topic0, topic1, topic2, topic3 string
			switch len(item.Topics) {
			case 1:
				topic0 = item.Topics[0].Hex()
			case 2:
				topic0 = item.Topics[0].Hex()
				topic1 = item.Topics[1].Hex()
			case 3:
				topic0 = item.Topics[0].Hex()
				topic1 = item.Topics[1].Hex()
				topic2 = item.Topics[2].Hex()
			case 4:
				topic0 = item.Topics[0].Hex()
				topic1 = item.Topics[1].Hex()
				topic2 = item.Topics[2].Hex()
				topic3 = item.Topics[3].Hex()
			}
			elog.Topic0 = topic0
			elog.Topic1 = topic1
			elog.Topic2 = topic2
			elog.Topic3 = topic3

			elog.Addr = strings.ToLower(txlog.Address.Hex())
			elog.Data = hexutil.Encode(txlog.Data) //TODO是否hash编码
			elog.Index = txlog.Index
			elog.TopicCnt = len(item.Topics)
			eventLogs = append(eventLogs, elog)
		}

		txTmp.EventLogs = eventLogs

		retTxs = append(retTxs, txTmp)
	}
	return retTxs, retContracts, nil
}

func (f *Fetcher) getBlockByNum(num uint64) (*mtypes.BlockJson, error) {
	bjson := &mtypes.BlockJson{}
	bum := new(big.Int).SetUint64(num)
	err := f.c.Call(bjson, "eth_getBlockByNumber", toBlockNumArg(bum), true)
	if err != nil {
		return nil, err
	}
	return bjson, nil
}

func (f *Fetcher) multiExecCreateTxBatch(txs []*mtypes.TxJson, workCnt int, blockNum *big.Int, baseFee *big.Int) ([]*mtypes.Tx, []*mtypes.Contract, []error) {
	if workCnt == 0 {
		workCnt = 1
	}
	txsGroupCh := f.groupTxs(txs, f.config.TxBatch)
	txsall := make([]*mtypes.Tx, 0)
	// balancesAll := make([]*mtypes.Balance, 0)
	contractsAll := make([]*mtypes.Contract, 0)
	var wg sync.WaitGroup
	wg.Add(workCnt)
	var l sync.Mutex
	var errs []error
	for i := 0; i < workCnt; i++ {
		go func() {
			defer wg.Done()
			for txGroup := range txsGroupCh {
				var (
					txs       []*mtypes.Tx
					contracts []*mtypes.Contract
					err       error
				)

				// txs, contracts, err = f.createTxBatch(txGroup, blockNum, baseFee)
				// var retryCnt int
				// for err != nil && retryCnt < f.config.FetchRetryTimes { //do retry
				// 	txs, contracts, err = f.createTxBatch(txGroup, blockNum, baseFee)
				// 	retryCnt++
				// }
				err = utils.HandleErrorWithRetryMaxTime(func() error {
					var err1 error
					txs, contracts, err1 = f.createTxBatch(txGroup, blockNum, baseFee)
					if err1 != nil {
						return err1
					}
					return nil
				}, f.config.ReceiptRetryTimes, f.config.FetchRetryInterval)
				l.Lock()
				if err != nil {
					errs = append(errs, fmt.Errorf("create tx batch err:%v,bum:%d", err, blockNum.Uint64()))
				} else {
					txsall = append(txsall, txs...)
					contractsAll = append(contractsAll, contracts...)
				}
				l.Unlock()
			}
		}()
	}
	wg.Wait()
	return txsall, contractsAll, errs
}

func (f *Fetcher) GetTxInternalsByBlockFantom(num *big.Int) ([]*mtypes.TxInternal, []*mtypes.Contract) {
	methodName := "trace_block"
	/*
		arg := map[string]interface{}{
			// "MinValue": 1,
		}
	*/
	ret := make([]*mtypes.TxForInternalFantom, 0)
	err := f.c.CallContext(context.Background(), &ret, methodName, toBlockNumArg(num))
	for err != nil {
		logrus.Errorf("get txinternals err:%v,bnum:%d", err, num.Uint64())
		time.Sleep(50 * time.Millisecond)
		err = f.c.CallContext(context.Background(), &ret, methodName, toBlockNumArg(num))
	}
	txInternals := make([]*mtypes.TxInternal, 0)
	contracts := make([]*mtypes.Contract, 0)
	for _, v := range ret {
		txhash := v.TransactionHash

		// Store the opcode ["CREATE"|"CREATE2"] logs into contract table
		if v.Action.CallType == "create" || v.Action.CallType == "create2" {
			// TODO: Fantom result exclude the transaction status, like tilog.Success
			if v.Error == "" {
				var status uint64 = 1
				if v.Action.To == `0x0000000000000000000000000000000000000000` {
					log.Fatalf("internal tx empty txhash:%s,from:%s,to:%s", txhash, v.Action.From, v.Action.To)
				}
				contracts = append(contracts, &mtypes.Contract{
					TxHash:      txhash,
					Addr:        strings.ToLower(v.Action.To),
					CreatorAddr: strings.ToLower(v.Action.From),
					ExecStatus:  status,
				})
			}
		}
		value := new(big.Int)
		if len(v.Action.Value) > 2 {
			var ok bool
			value, ok = value.SetString(v.Action.Value[2:], 16)
			if !ok {
				value = new(big.Int)
				logrus.Errorf("value set error: %s", v.Action.Value)
			}
		}
		var gasUsed uint64
		if len(v.Result.GasUsed) > 2 {
			gasUsed, err = strconv.ParseUint(v.Result.GasUsed[2:], 16, 64)
			if err != nil {
				logrus.Error(err)
			}
		}
		var gas uint64
		if len(v.Action.Gas) > 2 {
			gas, err = strconv.ParseUint(v.Action.Gas[2:], 16, 64)
			if err != nil {
				logrus.Error(err)
			}
		}
		ti := &mtypes.TxInternal{
			TxHash:       txhash,
			From:         strings.ToLower(v.Action.From),
			To:           strings.ToLower(v.Action.To),
			Value:        value,
			Success:      v.Error == "",
			OPCode:       strings.ToUpper(v.Action.CallType),
			Depth:        len(v.TraceAddress),
			GasUsed:      gasUsed,
			Gas:          gas,
			Input:        v.Action.Input,
			Output:       v.Result.Output,
			TraceAddress: v.TraceAddress,
		}
		txInternals = append(txInternals, ti)
	}
	// logrus.Debugf("get internals cost:%d,size:%d", time.Since(start)/time.Millisecond, len(ret))
	return txInternals, contracts
}

// getTxInternalsByBlock fetch internal transactions(transactions which created by EVM) via geth rpc
func (f *Fetcher) GetTxInternalsByBlock(num *big.Int) ([]*mtypes.TxInternal, []*mtypes.Contract) {
	methodName := "debug_traceActionByBlockNumber"
	arg := map[string]interface{}{
		// "MinValue": 1,
	}
	ret := make([]*mtypes.TxForInternal, 0)
	err := f.c.CallContext(context.Background(), &ret, methodName, toBlockNumArg(num), arg)
	for err != nil {
		logrus.Errorf("get txinternals err:%v,bnum:%d", err, num.Uint64())
		time.Sleep(50 * time.Millisecond)
		err = f.c.CallContext(context.Background(), &ret, methodName, toBlockNumArg(num), arg)
	}
	txInternals := make([]*mtypes.TxInternal, 0)
	contracts := make([]*mtypes.Contract, 0)
	for _, v := range ret {
		txhash := v.TransactionHash

		for _, tilog := range v.Logs {
			// Store the opcode ["CREATE"|"CREATE2"] logs into contract table
			if tilog.OPCode == "CREATE" || tilog.OPCode == "CREATE2" {
				if tilog.Success {
					var status uint64 = 1
					if tilog.To == `0x0000000000000000000000000000000000000000` {
						log.Fatalf("internal tx empty txhash:%s,from:%s,to:%s", txhash, tilog.From, tilog.To)
					}
					contracts = append(contracts, &mtypes.Contract{
						TxHash:      txhash,
						Addr:        strings.ToLower(tilog.To),
						CreatorAddr: strings.ToLower(tilog.From),
						ExecStatus:  status,
					})
				}
			}
			if tilog.Value == nil {
				logrus.Debugf("internal tx value should not be nil, block num: %v, tx hash:%s, success:%v", num, txhash, tilog.Success)
				tilog.Value = new(big.Int)
			}
			ti := &mtypes.TxInternal{
				TxHash:       txhash,
				From:         strings.ToLower(tilog.From),
				To:           strings.ToLower(tilog.To),
				Value:        tilog.Value,
				Success:      tilog.Success,
				OPCode:       tilog.OPCode,
				Depth:        tilog.Depth,
				GasUsed:      tilog.GasUsed,
				Gas:          tilog.Gas,
				Input:        tilog.Input,
				Output:       tilog.Output,
				TraceAddress: tilog.TraceAddress,
			}
			txInternals = append(txInternals, ti)
		}
	}
	// logrus.Debugf("get internals cost:%d,size:%d", time.Since(start)/time.Millisecond, len(ret))
	return txInternals, contracts
}

func (f *Fetcher) createBlockInfo(b *mtypes.BlockJson) (*mtypes.Block, error) {
	block := b.ToBlock()

	if block.TxCnt == 0 {
		return block, nil
	}
	blockNum := new(big.Int).SetUint64(block.Number)
	txsall, contractsTx, errs := f.multiExecCreateTxBatch(b.Txs, f.config.TxWorker, blockNum, block.BaseFee)
	if len(errs) > 0 {
		return nil, fmt.Errorf("batch exec txinfo errs:%v", errs)
	}
	block.Txs = txsall
	// eip1559 set burnt fees
	if block.BaseFee != nil {
		burntFees := new(big.Int)
		burntFees = burntFees.Mul(new(big.Int).SetUint64(block.GasUsed), block.BaseFee)
		block.BurntFees = burntFees
	}
	block.Contract = contractsTx
	// block.Balances = bs
	//get txinternals for tx
	if f.config.IsDealInternal {
		var txInternalsAll []*mtypes.TxInternal
		var contractsInternal []*mtypes.Contract
		if f.appName == "sync-chain-ftm" {
			txInternalsAll, contractsInternal = f.GetTxInternalsByBlockFantom(blockNum)
		} else {
			txInternalsAll, contractsInternal = f.GetTxInternalsByBlock(blockNum)
		}

		block.TxInternals = txInternalsAll

		block.Contract = append(block.Contract, contractsInternal...)
	}

	return block, nil
}

func (f *Fetcher) getBlockNumber() (uint64, error) {
	return f.Client.BlockNumber(context.Background())
}

func (f *Fetcher) GetBlocks(startHeight, endHeight uint64) (<-chan *mtypes.Block, <-chan error) {
	bch := f.getBlocks(startHeight, endHeight, f.config.BlockWorker)
	infoCh, errCh := f.batchDealBlock(bch, f.config.BlockWorker)
	return infoCh, errCh
}

func (f *Fetcher) getBlocks(height, heightEnd uint64, workCnt int) <-chan *mtypes.BlockJson {
	if workCnt == 0 {
		workCnt = 1
	}
	ch := make(chan *mtypes.BlockJson)
	heighCh := make(chan uint64, heightEnd-height)
	//[）
	for i := height; i < heightEnd; i++ {
		heighCh <- i
	}
	close(heighCh)
	var wg sync.WaitGroup
	wg.Add(workCnt)
	for i := 0; i < workCnt; i++ {
		go func() {
			defer wg.Done()
			for height := range heighCh {

				b := utils.HandleErrorWithRetryAndReturn(func() (interface{}, error) {
					b, err := f.getBlockByNum(height)
					if err != nil {
						return b, err
					}

					if b == nil || b.Number == "" {
						return nil, fmt.Errorf("empty block for:%v", height)
					}

					return b, err
				}, f.config.FetchRetryTimes, f.config.FetchRetryInterval)

				ch <- b.(*mtypes.BlockJson)
			}
		}()
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	return ch
}

// batchDealBlock create block info
func (f *Fetcher) batchDealBlock(ch <-chan *mtypes.BlockJson, workCnt int) (<-chan *mtypes.Block, <-chan error) {
	bch := make(chan *mtypes.Block)
	var (
		wg     sync.WaitGroup
		errcCh = make(chan error)
	)
	wg.Add(workCnt)
	for i := 0; i < workCnt; i++ {
		go func() {
			defer wg.Done()
			for b := range ch {
				start := time.Now()
				blockInfo, err := f.createBlockInfo(b)
				logrus.Debugf("create block cost:%v,err:%v", time.Since(start), err)
				if err != nil {
					errcCh <- err
				} else {
					bch <- blockInfo
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(errcCh)
		close(bch)
	}()
	return bch, errcCh
}

func (f *Fetcher) getLatestNormalBlockInfoInDB(localHeight uint64) (uint64, string, []*mtypes.Block) {
	chainHeight := localHeight + 1

	var (
		localHash  string
		blockLocal *mtypes.Block
		blockChain *mtypes.BlockJson
		err        error
	)

	revertBlocks := make([]*mtypes.Block, 0)

	for {
		//current local block
		var ok bool
		blockLocal, ok = f.blockCache[localHeight]

		if ok {
			localHash = blockLocal.Hash
		}

		if !ok && f.Height != f.startNum {
			logrus.Fatalf("local block not found in local:%v", localHeight)
		}

		//next block
		utils.HandleErrorWithRetry(func() error {
			// Get Next Block
			blockChain, err = f.getBlockByNum(chainHeight)
			if err != nil {
				return err
			}
			if blockChain.Number == "" {
				return fmt.Errorf("b num empty h:%d", chainHeight)
			}
			return nil
		}, f.config.FetchRetryTimes, f.config.FetchRetryInterval)

		/*
			utils.HandleErrorWithRetryV2(f.config.FetchRetryTimes, f.config.FetchRetryInterval)(func() error {
				blockChain, err = f.getBlockByNum(chainHeight)
				if err != nil {
					return err
				}
				if blockChain.Number == "" {
					return fmt.Errorf("b num empty h:%d", chainHeight)
				}
				return nil
			})
		*/

		if blockLocal != nil &&
			blockChain.ParentHash != "0x0000000000000000000000000000000000000000000000000000000000000000" &&
			blockChain.ParentHash != "" &&
			blockLocal.Hash != blockChain.ParentHash {

			logrus.Warnf("revertBlock found hash:%s,parent:%s", blockLocal.Hash, blockChain.ParentHash)

			blktemp := *blockLocal
			revertBlocks = append(revertBlocks, &blktemp)

			localHeight--
			chainHeight--
		} else {
			return localHeight, localHash, revertBlocks
		}
	}
}

func (f *Fetcher) GetGenesisBlock() (*mtypes.BlockJson, error) {
	return f.getBlockByNum(0)
}
