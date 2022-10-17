package task

import (
	"fmt"
	"log"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
	"github.com/starslabhq/chainmonitor/config"
	"github.com/starslabhq/chainmonitor/db"
	"github.com/starslabhq/chainmonitor/mtypes"
	"github.com/starslabhq/chainmonitor/output/mysqldb"
	"github.com/starslabhq/chainmonitor/utils"
	"xorm.io/xorm"
)

const (
	syncEvent       = `Sync`
	dexContract     = `[{"anonymous":false,"inputs":[{"indexed":false,"internalType":"uint112","name":"reserve0","type":"uint112"},{"indexed":false,"internalType":"uint112","name":"reserve1","type":"uint112"}],"name":"Sync","type":"event"},{"constant":true,"inputs":[],"name":"token0","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"token1","outputs":[{"internalType":"address","name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"}]`
	nameAbi         = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"}]`
	DexPairTaskName = `dex_pair`
)

var (
	syncEventHash = `0x1c411e9a96e071241c2f21f7726b17ae89e3cab4c78be50e062b03a9fffbbad1`
)

type DexPairTask struct {
	*BaseAsyncTask
	abiDex   abi.ABI
	abiName  abi.ABI
	rpcCur   int
	rpcBatch int
	// dbGetBatch int
	caceSize int
	cache    *lru.Cache
}

func NewDexPairTask(config *config.Config, c *rpc.Client, db db.IDB) (*DexPairTask, error) {
	t := &DexPairTask{}
	taskConf := config.DexPair
	base, err := newBase(DexPairTaskName, config, c, db, taskConf.BufferSize, t.handleBlock, t.fixHistoryData, t.revertBlock)
	if err != nil {
		logrus.Fatalf("base err:%v", err)
		return nil, err
	}
	t.BaseAsyncTask = base
	r := strings.NewReader(dexContract)
	abiDex, err := abi.JSON(r)
	if err != nil {
		log.Fatalf("abi load err:%v", err)
	}
	t.abiDex = abiDex
	rname := strings.NewReader(nameAbi)

	abiName, err := abi.JSON(rname)
	if err != nil {
		log.Fatalf("abi name load err:%v", err)
	}
	t.abiName = abiName
	t.caceSize = taskConf.CacheSize
	t.rpcCur = taskConf.Concurrent
	t.rpcBatch = taskConf.RPCBatch
	t.cache, err = lru.New(t.caceSize)
	if err != nil {
		return nil, fmt.Errorf("create cache err: %v", err)
	}

	return t, nil
}

func isDexPair(txLog *mtypes.EventLog) bool {
	return txLog.Topic0 == syncEventHash && txLog.Topic1 == "" && txLog.Topic2 == "" && txLog.Topic3 == ""
}

func (t *DexPairTask) decoeReserve(tlog *mtypes.EventLog) (reserve0, reserve1 *big.Int, err error) {
	data, err := hexutil.Decode(tlog.Data)
	if err != nil {
		log.Fatalf("decode data err")
	}
	reserves, err := t.abiDex.Unpack(syncEvent, data)
	if err != nil {
		return nil, nil, fmt.Errorf("unpack sync data err:%v", err)
	}
	if len(reserves) != 2 {
		return nil, nil, fmt.Errorf("unpacked size not 2")
	}

	if v, ok := reserves[0].(*big.Int); ok {
		reserve0 = v
	} else {
		err = fmt.Errorf("reserve0 not *big.Int reserves:%v", reserves)
		return
	}
	if v, ok := reserves[1].(*big.Int); ok {
		reserve1 = v
	} else {
		err = fmt.Errorf("reserve1 not *big.Int reserves:%v", reserves)
	}

	return

}

func (t *DexPairTask) isNeedToGetPairTokens(addr string) bool {
	_, ok := t.cache.Get(addr)
	if !ok {
		pair, err := t.db.GetTokenPairByAddr(addr)
		if err == nil && pair != nil {
			t.cache.Add(addr, struct{}{})
			ok = true
		}
		if err != nil {
			logrus.Fatalf("token pair query err:%v,addr:%s", err, addr)
		}
	}
	return !ok
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func (t *DexPairTask) getPairsToken(pairs []*mtypes.TokenPair) (err error) {
	start := time.Now()
	defer func() {
		logrus.Debugf("token pair rpc cost:%d,pairs:%d", time.Since(start)/time.Millisecond, len(pairs))
	}()
	var elems []rpc.BatchElem
	for _, pair := range pairs {
		//token0
		input0, err0 := t.abiDex.Pack("token0")
		if err0 != nil {
			logrus.Fatalf("pack token0 err:%v,addr:%v", err0, pair.Addr)
		}
		data0 := ethereum.CallMsg{
			To:   &pair.Addr,
			Data: input0,
		}
		elems = append(elems,
			rpc.BatchElem{
				Method: "eth_call",
				Args:   []interface{}{toCallArg(data0), toBlockNumArg(nil)},
				Result: &pair.Token0Bytes,
			})
		// token1
		input1, err1 := t.abiDex.Pack("token1")
		if err1 != nil {
			logrus.Fatalf("pack token1 err:%v,addr:%v", err1, pair.Addr)
		}
		data1 := ethereum.CallMsg{
			To:   &pair.Addr,
			Data: input1,
		}
		elems = append(elems,
			rpc.BatchElem{
				Method: "eth_call",
				Args:   []interface{}{toCallArg(data1), toBlockNumArg(nil)},
				Result: &pair.Token1Bytes,
			})
		// pair name
		input2, err2 := t.abiName.Pack("name")
		if err2 != nil {
			logrus.Fatalf("pack name err:%v,addr:%v", err, pair.Addr)
		}
		data2 := ethereum.CallMsg{
			To:   &pair.Addr,
			Data: input2,
		}
		elems = append(elems,
			rpc.BatchElem{
				Method: "eth_call",
				Args:   []interface{}{toCallArg(data2), toBlockNumArg(nil)},
				Result: &pair.PairNameBytes,
			})
	}
	err = t.client.BatchCall(elems)
	if err != nil {
		return fmt.Errorf("batch call elems err:%v", err)
	}
	for _, e := range elems {
		if e.Error != nil && !utils.HitNoMoreRetryErrors(e.Error) {
			return fmt.Errorf("get tokens err:%v,e:%v", e.Error, e)
		}
	}
	for _, pair := range pairs {
		if len(pair.Token0Bytes) != 0 {
			pair.Token0 = common.BytesToAddress(pair.Token0Bytes)
		}
		if len(pair.Token1Bytes) != 0 {
			pair.Token1 = common.BytesToAddress(pair.Token1Bytes)
		}
		if len(pair.PairNameBytes) != 0 {
			values, err := t.abiName.Unpack("name", pair.PairNameBytes)
			if err != nil {
				logrus.Errorf("token pair unpack err:%v,pair:%v", err, pair)
			}
			if len(values) == 0 {
				logrus.Errorf("token pair unpack ret size err data:%s", pair.PairNameBytes)
			}
			if name, ok := values[0].(string); ok {
				pair.PairName = name
			} else {
				logrus.Errorf("token pair unpack ret not str data:%s", pair.PairNameBytes)
			}
		}
	}
	return nil
}

func (t *DexPairTask) saveTokenPairs(s xorm.Interface, pairs []*mtypes.TokenPair) error {
	params := mysqldb.ConvertInTokenPairs(pairs)
	err := t.db.SaveTokenPairs(s, params)
	if err != nil {
		return fmt.Errorf("token pair save err:%v", err)
	}
	return nil
}

func (t *DexPairTask) updateTokenPairs(s xorm.Interface, pairs []*mtypes.TokenPair) error {
	params := mysqldb.ConvertInTokenPairs(pairs)
	err := t.db.UpdateTokenPairsReserve(s, params)
	return err
}

func (t *DexPairTask) handleBlock(blk *mtypes.Block) {
	st := time.Now()
	defer func() {
		logrus.Infof("token pair cost:%d", time.Since(st)/time.Millisecond)
	}()
	txLogs := make([]*mtypes.EventLog, 0)
	for _, tx := range blk.Txs {
		txLogs = append(txLogs, tx.EventLogs...)
	}
	sort.Slice(txLogs, func(i, j int) bool {
		return txLogs[i].Index < txLogs[j].Index
	})
	txLogCnt := len(txLogs)
	dealed := make(map[string]struct{})
	pairsToGetTokens := make([]*mtypes.TokenPair, 0)
	pairs := make([]*mtypes.TokenPair, 0)
	for i := txLogCnt - 1; i >= 0; i-- {

		if !isDexPair(txLogs[i]) {
			continue
		}
		addr := txLogs[i].Addr
		if _, ok := dealed[addr]; !ok {
			tlog := txLogs[i]
			r0, r1, err := t.decoeReserve(tlog)
			if err != nil {
				logrus.Errorf("decode reserve err:%v,logIndex:%d,blockNum:%d", err, tlog.Index, blk.Number)
				continue
			}
			pair := &mtypes.TokenPair{
				Addr: common.HexToAddress(addr),
			}
			pair.Reserve0 = r0.String()
			pair.Reserve1 = r1.String()
			pair.BlockNum = int64(blk.Number)
			if t.isNeedToGetPairTokens(addr) { //是否已经存在
				logrus.Infof("token pair need to get token:%s", addr)
				pairsToGetTokens = append(pairsToGetTokens, pair)
			} else {
				pairs = append(pairs, pair)
			}
			dealed[addr] = struct{}{}
		}
	}
	if t.rpcCur == 0 {
		logrus.Fatalf("token pair rpc cur not set")
	}
	if t.rpcBatch == 0 {
		logrus.Fatalf("token pair rpc batch not set")
	}
	ch := make(chan struct{}, t.rpcCur)
	pairCnt := len(pairsToGetTokens)
	if pairCnt > 0 {
		var wg sync.WaitGroup
		for i := 0; i < pairCnt; {
			end := i + t.rpcBatch
			if end > pairCnt {
				end = pairCnt
			}
			wg.Add(1)

			go func(s, e int) {
				ch <- struct{}{}
				defer func() {
					wg.Done()
					<-ch
				}()
				utils.HandleErrorWithRetry(func() error {
					return t.getPairsToken(pairsToGetTokens[s:e])
				}, t.config.Fetch.FetchRetryTimes, t.config.Fetch.FetchRetryInterval)
			}(i, end)
			i = end
		}
		wg.Wait()
	}
	//TODO
	utils.HandleErrorWithRetry(func() error {
		s := t.db.GetSession()
		defer s.Close()
		err := s.Begin()
		if err != nil {
			return fmt.Errorf("token pair sesssion begin err:%v", err)
		}

		err = t.saveTokenPairs(s, pairsToGetTokens)
		if err != nil {
			err1 := s.Rollback()
			if err1 != nil {
				return fmt.Errorf("token pair insert rollback err:%v", err1)
			}
			return fmt.Errorf("save token pairs err:%v", err)
		}

		err = t.updateTokenPairs(s, pairs)
		if err != nil {
			err1 := s.Rollback()
			if err1 != nil {
				return fmt.Errorf("token pair update err:%v", err)
			}
			return fmt.Errorf("token pair update err:%v", err)
		}

		err = t.db.UpdateAsyncTaskNumByName(s, t.name, blk.Number)
		if err != nil {
			err1 := s.Rollback()
			if err1 != nil {
				return fmt.Errorf("update task blk err:%v,tname:%s,bnum:%d", err, t.name, blk.Number)
			}
			return err
		}
		err = s.Commit()
		if err != nil {
			if err1 := s.Rollback(); err1 != nil {
				logrus.Errorf("dexpair rollback err:%v,bnum:%d", err1, blk.Number)
			}
			return fmt.Errorf("token pair commit err:%v,bnum:%d", err, blk.Number)
		}
		return nil
	}, t.config.OutPut.RetryTimes, t.config.OutPut.RetryInterval)
	t.curHeight = blk.Number
}
func (t *DexPairTask) handleBlocks(blks []*mtypes.Block) {
	for _, blk := range blks {
		t.handleBlock(blk)
	}
}

func (t *DexPairTask) fixHistoryData() {
	//TODO
	batch := 100
	filter := &mysqldb.BlockFilter{
		TxFilter: &mysqldb.TxFilter{},
		LogFilter: &mysqldb.LogFilter{
			Topic0: &syncEventHash,
		},
	}
	var blk *mtypes.Block
	var blks []*mtypes.Block
	var err error
	if t.curHeight < t.latestHeight-uint64(t.bufferSize) {
		blks, err = t.db.GetBlocksByRange(t.curHeight+1, t.curHeight+1+uint64(batch), filter)
		if err == nil && len(blks) == 0 {
			lastHeight := t.curHeight + 1 + uint64(batch)
			t.curHeight = lastHeight
			utils.HandleErrorWithRetry(func() error {
				return t.db.UpdateAsyncTaskNumByName(t.db.GetEngine(), t.name, t.curHeight)
			}, t.config.OutPut.RetryTimes, t.config.OutPut.RetryInterval)

			//TODO wait for retry
			time.Sleep(time.Second * 1)
			return
		}
	} else {
		blk, err = t.getBlkInfo(t.curHeight+1, filter)
		if err != nil {
			logrus.Errorf("token pair get blk info err:%v,h:%d", err, t.curHeight+1)
		}
		if blk != nil {
			blks = []*mtypes.Block{blk}
		} else {
			time.Sleep(time.Second * 1)
			return
		}
	}
	logrus.Infof("token pair history blks:%d,last:%d", len(blks), t.curHeight+1+uint64(batch))
	sort.Slice(blks, func(i, j int) bool {
		return blks[i].Number < blks[j].Number
	})
	t.handleBlocks(blks)
}

func (t *DexPairTask) revertBlock(blk *mtypes.Block) {
	t.curHeight = blk.Number
}

// func (t *DexPairTask) getPairsFromDB(addrs []string) []*mtypes.TokenPair {
// 	return nil
// }

// func (t *DexPairTask) revertBlock(blk *mtypes.Block) {
// 	if t.curHeight < blk.Number {
// 		return
// 	}
// 	filter := &mysqldb.BlockFilter{
// 		TxFilter: &mysqldb.TxFilter{},
// 		LogFilter: &mysqldb.LogFilter{
// 			Topic0: &syncEventHash,
// 		},
// 	}
// 	start := blk.Number
// 	var (
// 		blks []*mtypes.Block
// 		err  error
// 	)
// 	pairsToRevert := make([]*mtypes.TokenPair, 0)
// 	for start <= t.curHeight {
// 		end := start + uint64(t.dbGetBatch)
// 		utils.HandleErrorWithRetry(func() error {
// 			blks, err = t.db.GetRevertBlocks(start, end, filter)
// 			if err != nil {
// 				return err
// 			}
// 			return nil
// 		}, 1, 1)
// 		var addrsMap = make(map[string]struct{})
// 		for _, blk := range blks {
// 			for _, tx := range blk.Txs {
// 				for _, tlog := range tx.EventLogs {
// 					if !isDexPair(tlog) {
// 						continue
// 					}
// 					addrsMap[tlog.Addr] = struct{}{}
// 				}
// 			}
// 		}
// 		var addrs []string
// 		for addr := range addrsMap {
// 			addrs = append(addrs, addr)
// 		}
// 		pairs := t.getPairsFromDB(addrs)
// 		for _, pair := range pairs {
// 			if pair.BlockNum >= int64(blk.Number) {
// 				//使用block_num=-1标识回滚
// 				pair.BlockNum = -1
// 				pairsToRevert = append(pairsToRevert, pair)
// 			}
// 		}
// 		start = end + 1
// 	}
// 	utils.HandleErrorWithRetry(func() error {
// 		s := t.db.GetSession()
// 		defer s.Close()
// 		err := s.Begin()
// 		if err != nil {
// 			return fmt.Errorf("token pair sesssion begin err:%v", err)
// 		}
// 		err = t.saveTokenPairs(s, pairsToRevert)
// 		if err != nil {
// 			err1 := s.Rollback()
// 			if err1 != nil {
// 				return fmt.Errorf("token pair rollback err:%v", err1)
// 			}
// 			return err
// 		}
// 		err = t.db.UpdateAsyncTaskNumByName(s, t.name, blk.Number)
// 		if err != nil {
// 			err1 := s.Rollback()
// 			if err1 != nil {
// 				return fmt.Errorf("update task blk err:%v,tname:%s,bnum:%d", err, t.name, blk.Number)
// 			}
// 			return err
// 		}
// 		return nil
// 	}, 0, 0)
// 	t.curHeight = blk.Number
// }
