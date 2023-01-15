package task

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/db"
	"github.com/chainmonitor/mtypes"
	"github.com/chainmonitor/output/mysqldb"
	"github.com/chainmonitor/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

type Erc20BalanceTask struct {
	*BaseAsyncTask
	erc20ABI  abi.ABI
	cache     *CacheFilter
	multiCall *utils.MultiCall
}

const Erc20BalanceTaskName = "erc20_balance"

func NewErc20BalanceTask(config *config.Config, client *rpc.Client, db db.IDB) (*Erc20BalanceTask, error) {
	b := &Erc20BalanceTask{}
	base, err := newBase(Erc20BalanceTaskName, config, client, db, config.Erc20Balance.BufferSize,
		b.handleBlock, b.fixHistoryData, b.revertBlock)
	if err != nil {
		return nil, err
	}
	b.BaseAsyncTask = base
	r := strings.NewReader(erc20abi)
	b.erc20ABI, err = abi.JSON(r)
	if err != nil {
		return nil, err
	}
	b.cache, err = b.newCache()
	if err != nil {
		return nil, err
	}
	if config.Fetch.MultiCallContract != "" {
		b.multiCall, err = utils.NewMultiCall(client, config.Fetch.MultiCallContract, config.Fetch)
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func (eb *Erc20BalanceTask) newCache() (*CacheFilter, error) {
	return NewCacheFilter(eb.config.Erc20Balance.CacheSize, 1024*1024*8*20, 20, func(key string) (interface{}, bool, error) {
		args := strings.Split(key, "|")

		arg0 := strings.ToLower(args[0])
		arg1 := strings.ToLower(args[1])

		res, err := eb.db.GetBalanceErc20(arg0, arg1)
		if err != nil {
			return nil, false, err
		}
		if res == nil {
			return nil, false, nil
		}
		return res.Height, true, nil
	})
}

func (eb *Erc20BalanceTask) handleBlock(blk *mtypes.Block) {
	st := time.Now()
	eb.handleBlocks([]*mtypes.Block{blk})
	logrus.Infof("erc20 balance cost handleBlock:%d", time.Since(st)/time.Millisecond)
	height := blk.Number
	err := eb.db.UpdateAsyncTaskNumByName(eb.db.GetEngine(), eb.name, height)
	if err != nil {
		logrus.Errorf("erc20 balance task update h:%d, err:%v", height, err)
	}
	eb.curHeight = height
}

func (eb *Erc20BalanceTask) getBalanceByMultiCall(bs []*mtypes.Balance) error {
	inputs := make([]utils.MultiCallInput, 0, len(bs))
	for _, b := range bs {
		input, err := eb.erc20ABI.Pack("balanceOf", b.Addr)
		if err != nil {
			return fmt.Errorf("panic erc20 balanceOf input err:%v", err)
		}

		inputs = append(inputs, utils.MultiCallInput{
			Target:   b.ContractAddr,
			CallData: input,
		})
	}
	st := time.Now()
	res := eb.multiCall.AggregateWithRetry(inputs, eb.latestHeight)
	logrus.Infof("erc20 balance cost AggregateWithRetry:%d,batch:%d", time.Since(st)/time.Millisecond, len(bs))
	if res.BlockNumber.Cmp(big.NewInt(int64(eb.latestHeight))) < 0 {
		//log error
		return fmt.Errorf("query erc20 balance error. latest height:%v, multicall height:%v", eb.latestHeight, res.BlockNumber.String())
	}

	for i, elem := range res.ReturnData {
		if elem.Err != nil {
			logrus.Warnf("get erc20 balance error. contract:%v, addr:%v, err:%v", bs[i].ContractAddr, bs[i].Addr, elem.Err)
			continue
		}

		bs[i].ValueBytes = elem.Output
		bs[i].Height = res.BlockNumber

		if len(bs[i].ValueBytes) == 0 {
			continue
		}
		rets, err := eb.erc20ABI.Unpack("balanceOf", bs[i].ValueBytes)
		if err != nil {
			logrus.Warnf("unpack erc20 balanceOf err:%v,contract:%v,address:%v", err, bs[i].ContractAddr, bs[i].Addr)
			continue
		}
		if len(rets) == 0 {
			logrus.Warnf("erc20 balanceOf ret size err:%v,contract:%v,address:%v", err, bs[i].ContractAddr, bs[i].Addr)
			continue
		}
		if v, ok := rets[0].(*big.Int); ok {
			bs[i].Value = v
		} else {
			logrus.Warnf("erc20 balanceOf ret not *big.Int:%v,contract:%v,address:%v", err, bs[i].ContractAddr, bs[i].Addr)
		}
	}

	return nil
}

func (eb *Erc20BalanceTask) getErc20BalancesBatch(bs []*mtypes.Balance, blockHeight uint64) error {
	elems := make([]rpc.BatchElem, 0, len(bs))
	for _, v := range bs {
		b := v
		input, err := eb.erc20ABI.Pack("balanceOf", b.Addr)
		if err != nil {
			return fmt.Errorf("panic erc20 balanceOf input err:%v", err)
		}
		arg := map[string]interface{}{
			"from": b.Addr,
			"to":   &b.ContractAddr,
			"data": hexutil.Bytes(input),
		}

		elem := rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{arg, "latest"},
			Result: &b.ValueBytes,
		}
		elems = append(elems, elem)
	}

	var chainHeight hexutil.Uint64
	elems = append(elems, rpc.BatchElem{
		Method: "eth_blockNumber",
		Args:   []interface{}{},
		Result: &chainHeight,
	})
	//
	////get height for current
	//if eb.latestHeight > eb.latestBlockNumOnstart {
	//	var chainHeight hexutil.Uint64
	//	err := eb.client.CallContext(context.Background(), &chainHeight, "eth_blockNumber")
	//	if err != nil {
	//		return err
	//	}
	//
	//	if uint64(chainHeight) < eb.latestHeight {
	//		return fmt.Errorf("latest height:%v got cur chain height:%v", eb.latestHeight, uint64(chainHeight))
	//	}
	//
	//	for _, e := range elems {
	//		e.Args[1] = chainHeight.String()
	//	}
	//}

	err := eb.client.BatchCallContext(context.Background(), elems)
	if err != nil {
		return fmt.Errorf("rpc erc20 balances err:%v", err)
	}
	for _, elem := range elems {
		if elem.Error != nil && !utils.HitNoMoreRetryErrors(elem.Error) {
			return fmt.Errorf("erc20 balances elem err:%v,elem:%v", elem.Error, elem)
		}
	}

	//check chain block number
	// if uint64(chainHeight) < eb.latestHeight {
	if uint64(chainHeight) < blockHeight {
		return fmt.Errorf("latest height:%v got cur chain height:%v", eb.latestHeight, uint64(chainHeight))
	}

	for _, b := range bs {
		b.Height = big.NewInt(int64(chainHeight))
		if len(b.ValueBytes) == 0 {
			continue
		}
		rets, err := eb.erc20ABI.Unpack("balanceOf", b.ValueBytes)
		if err != nil {
			logrus.Warnf("unpack erc20 balanceOf err:%v,contract:%v,address:%v", err, b.ContractAddr, b.Addr)
			continue
		}
		if len(rets) == 0 {
			logrus.Warnf("erc20 balanceOf ret size err:%v,contract:%v,address:%v", err, b.ContractAddr, b.Addr)
			continue
		}

		if v, ok := rets[0].(*big.Int); ok {
			b.Value = v
		} else {
			logrus.Warnf("erc20 balanceOf ret not *big.Int:%v,contract:%v,address:%v", err, b.ContractAddr, b.Addr)
		}
	}
	return nil
}

func (eb *Erc20BalanceTask) getErc20BalanceAll(bs []*mtypes.Balance, batch int, blockHeight uint64) {
	size := len(bs)
	//并发数 TODO
	var cur int = eb.config.Erc20Balance.Concurrent
	ch := make(chan struct{}, cur)
	var wg sync.WaitGroup
	for i := 0; i < size; {

		end := i + batch
		if end > size {
			end = size
		}
		wg.Add(1)
		ch <- struct{}{}
		// [s,e)
		go func(s, e int) {
			defer wg.Done()
			defer func() {
				<-ch
			}()

			utils.HandleErrorWithRetry(func() error {
				if eb.multiCall != nil {
					return eb.getBalanceByMultiCall(bs[s:e])
				}

				return eb.getErc20BalancesBatch(bs[s:e], blockHeight)
			}, eb.config.Fetch.FetchRetryTimes,
				eb.config.Fetch.FetchRetryInterval)
		}(i, end)
		i = end
	}
	wg.Wait()
}

func (eb *Erc20BalanceTask) saveErc20Balances(balances []*mtypes.Balance) error {
	s := eb.db.GetSession()
	err := s.Begin()
	if err != nil {
		return fmt.Errorf("erc20 balance session beigin err:%v", err)
	}
	defer s.Close()
	params := mysqldb.ConvertInErc20Balances(balances)
	err = eb.db.SaveErc20Balances(s, params)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("erc20 balances roolback err:%v", err1)
		}
		return fmt.Errorf("erc20 balances insert err:%v", err)
	}
	err = s.Commit()
	if err != nil {
		return fmt.Errorf("erc20 balances commit err:%v", err)
	}
	return nil
}

// balance是否需要更新
func (eb *Erc20BalanceTask) isNeedToUpdate(addr, contracAddr string, h uint64) bool {
	key := addr + "|" + contracAddr
	v, ok := eb.cache.Get(key)
	if !ok {
		return true
	}
	updatedH, _ := v.(uint64)
	return h > updatedH
}
func (eb *Erc20BalanceTask) recordUpdatedHeight(addr, contractAddr string, h uint64) {
	key := addr + "|" + contractAddr
	eb.cache.Add(key, h)
}

func (eb *Erc20BalanceTask) handleBlocks(blks []*mtypes.Block) {
	balances := make([]*mtypes.Balance, 0)
	ch := make(chan *mtypes.Balance)
	size := len(blks)
	if size == 0 {
		return
	}
	blockHeight := blks[size-1].Number
	go func() {
		for _, blk := range blks {
			logrus.Debugf("erc20 balance task recv block %d", blk.Number)
			bNum := blk.Number
			for _, tx := range blk.Txs {
				for _, tlog := range tx.EventLogs {
					if !isErc20Tx(tlog) {
						continue
					}
					sender := common.HexToAddress(tlog.Topic1)
					receiver := common.HexToAddress(tlog.Topic2)
					contractAddr := common.HexToAddress(tlog.Addr)

					tokens := new(big.Int)
					data := hexutil.MustDecode(tlog.Data)
					tokens.SetBytes(data)
					if tokens.Uint64() != 0 {
						ch <- &mtypes.Balance{
							Addr:         sender,
							ContractAddr: contractAddr,
							Height:       big.NewInt(int64(bNum)),
						}
						ch <- &mtypes.Balance{
							Addr:         receiver,
							ContractAddr: contractAddr,
							Height:       big.NewInt(int64(bNum)),
						}
					}
				}
			}
		}
		close(ch)
	}()
	var wcnt int = 128
	var wg sync.WaitGroup
	wg.Add(wcnt)
	var l sync.Mutex
	for i := 0; i < 128; i++ {
		go func() {
			defer wg.Done()
			for b := range ch {
				addr := strings.ToLower(b.Addr.Hex())
				contract := strings.ToLower(b.ContractAddr.Hex())
				if eb.isNeedToUpdate(addr, contract, b.Height.Uint64()) {
					l.Lock()
					balances = append(balances, b)
					l.Unlock()
				}
			}
		}()
	}
	wg.Wait()

	eb.getErc20BalanceAll(balances, eb.config.Erc20Balance.BatchRPC, blockHeight)
	utils.HandleErrorWithRetry(
		func() error {
			err := eb.saveErc20Balances(balances)
			return err
		}, eb.config.OutPut.RetryTimes, eb.config.OutPut.RetryInterval)

	for _, b := range balances {
		eb.recordUpdatedHeight(strings.ToLower(b.Addr.Hex()), strings.ToLower(b.ContractAddr.Hex()), b.Height.Uint64())
	}
}

func (eb *Erc20BalanceTask) fixHistoryData() {
	batch := eb.config.Erc20Balance.BatchBlockCount

	var blk *mtypes.Block
	var blks []*mtypes.Block
	var err error
	if eb.curHeight < eb.latestHeight-uint64(eb.bufferSize) {
		st := time.Now()
		blks, err = eb.db.GetBlocksByRange(eb.curHeight+1, eb.curHeight+1+uint64(batch), mysqldb.ERC20Filter)
		logrus.Debugf("block range cost:%d,task:%s,start:%d,batch:%d", time.Since(st)/time.Millisecond, eb.name, eb.curHeight+1, eb.curHeight+1+uint64(batch))
	} else {
		blk, err = eb.getBlkInfo(eb.curHeight+1, mysqldb.ERC20Filter)
		if blk != nil {
			blks = []*mtypes.Block{blk}
		}
	}

	if len(blks) == 0 || err != nil {
		if err != nil {
			logrus.Errorf("query block info error:%v", err)
		} else {
			logrus.Debugf("native balance handler. cur height:%v", eb.curHeight)
			lastHeight := eb.curHeight + 1 + uint64(batch)
			eb.curHeight = lastHeight
		}

		time.Sleep(time.Second * 1)
		return
	}
	eb.handleBlocks(blks)
	height := blks[len(blks)-1].Number
	err = eb.db.UpdateAsyncTaskNumByName(eb.db.GetEngine(), eb.name, height)
	if err != nil {
		logrus.Errorf("erc20 balance task update h:%d, err:%v", height, err)
	}
	eb.curHeight = height
}

func (eb *Erc20BalanceTask) revertBlock(blk *mtypes.Block) {
	eb.cache.clean()
	eb.handleBlock(blk)
	err := eb.db.UpdateAsyncTaskNumByName(eb.db.GetEngine(), eb.name, blk.Number-1)
	if err != nil {
		logrus.Errorf("update Erc20BalanceTask height:%d err:%v", blk.Number-1, err)
	}
}
