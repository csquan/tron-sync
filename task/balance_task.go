package task

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/chainmonitor/output/mysqldb"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/db"
	"github.com/chainmonitor/mtypes"
	"github.com/chainmonitor/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

type BalanceTask struct {
	*BaseAsyncTask
	//latestBlockNumOnstart uint64
	//use cache to reduce repeat request
	balanceCache *CacheFilter
}

const BalanceTaskName = "native_balance"

func NewBalanceTask(config *config.Config, client *rpc.Client, db db.IDB) (*BalanceTask, error) {

	b := &BalanceTask{
		//latestBlockNumOnstart: latestBlockNumOnstart,
		// balanceCache:          cache,
	}

	base, err := newBase(BalanceTaskName, config, client, db, config.Balance.BufferSize,
		b.handleBlock, b.fixHistoryData, b.revertBlock)
	if err != nil {
		return nil, err
	}
	b.BaseAsyncTask = base

	cache, err := b.newCache(db)
	if err != nil {
		return nil, err
	}
	b.balanceCache = cache

	return b, nil
}

func (b *BalanceTask) newCache(db db.IDB) (c *CacheFilter, err error) {
	return NewCacheFilter(b.config.Balance.CacheSize, 1024*1024*8*20, 20, func(key string) (interface{}, bool, error) {
		res, err := db.GetBalance(key)
		if err != nil {
			return nil, false, err
		}

		if res == nil {
			return nil, false, nil
		}

		return big.NewInt(int64(res.Height)), true, nil

	})
}

func (b *BalanceTask) fixHistoryData() {
	batch := b.config.Balance.BatchBlockCount
	success := true
	filter := &mysqldb.BlockFilter{
		TxFilter: &mysqldb.TxFilter{},

		InternalTxFilter: &mysqldb.InternalTxFilter{
			Value:   big.NewInt(1),
			Success: &success,
		},
	}

	var blk *mtypes.Block
	var blks []*mtypes.Block
	var err error
	if b.curHeight < b.latestHeight-uint64(b.bufferSize) {
		blks, err = b.db.GetBlocksByRange(b.curHeight+1, b.curHeight+1+uint64(batch), filter)
	} else {
		blk, err = b.getBlkInfo(b.curHeight+1, filter)
		if blk != nil {
			blks = []*mtypes.Block{blk}
		}
	}

	if len(blks) == 0 || err != nil {
		if err != nil {
			logrus.Errorf("query block info error:%v", err)
		} else {
			logrus.Debugf("native balance handler. cur height:%v", b.curHeight)
			lastHeight := b.curHeight + 1 + uint64(batch)
			b.curHeight = lastHeight
		}

		time.Sleep(time.Second * 1)
		return
	}
	b.handleBlocks(blks)
	height := blks[len(blks)-1].Number
	err = b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, height)
	if err != nil {
		logrus.Errorf("update balance task height:%d err:%v", height, err)
	}
	b.curHeight = height
}

func (b *BalanceTask) handleBlock(blk *mtypes.Block) {
	b.handleBlocks([]*mtypes.Block{blk})
	height := blk.Number
	err := b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, height)
	if err != nil {
		logrus.Errorf("update balance task height:%d err:%v", height, err)
	}
	b.curHeight = height
}

func (b *BalanceTask) handleBlocks(blks []*mtypes.Block) {
	size := len(blks)
	if size == 0 {
		logrus.Warnf("blocks empty")
		return
	}
	curBlockNum := blks[size-1].Number
	addresses := make(map[string]interface{})
	balances := make([]*mtypes.Balance, 0, len(addresses))

	for _, blk := range blks {
		for _, tx := range blk.Txs {
			if v, ok := b.balanceCache.Get(tx.From); !ok || v.(*big.Int).Uint64() < blk.Number {
				addresses[tx.From] = nil
			}

			if v, ok := b.balanceCache.Get(tx.To); !ok || v.(*big.Int).Uint64() < blk.Number {
				addresses[tx.To] = nil
			}
		}

		for _, tx := range blk.TxInternals {
			if !tx.Success || tx.Value.Cmp(big.NewInt(0)) == 0 {
				continue
			}

			if v, ok := b.balanceCache.Get(tx.From); !ok || v.(*big.Int).Uint64() < blk.Number {
				addresses[tx.From] = nil
			}

			if v, ok := b.balanceCache.Get(tx.To); !ok || v.(*big.Int).Uint64() < blk.Number {
				addresses[tx.To] = nil
			}
		}
	}

	for addr := range addresses {
		balances = append(balances, &mtypes.Balance{
			Addr: common.HexToAddress(addr),
		})
	}
	//rpc query balance for addresses
	b.getBalances(balances, curBlockNum)

	for _, ba := range balances {
		b.balanceCache.Add(strings.ToLower(ba.Addr.Hex()), ba.Height)
	}

	dbBalances := mysqldb.ConvertInBalances(balances)

	utils.HandleErrorWithRetry(func() error {
		err := b.db.SaveBalances(dbBalances)
		if err != nil {
			return err
		}
		return nil
	}, b.config.OutPut.RetryTimes, b.config.OutPut.RetryInterval)
}

func (b *BalanceTask) getBalances(balances []*mtypes.Balance, num uint64) {
	start := 0
	batchSize := b.config.Fetch.FetchBatch

	queryChan := make(chan int, b.config.Balance.Concurrent)
	wg := &sync.WaitGroup{}

	for start < len(balances) {
		end := start + batchSize
		if end > len(balances) {
			end = len(balances)
		}
		batch := balances[start:end]
		start += batchSize

		wg.Add(1)
		queryChan <- 1

		go func(batch []*mtypes.Balance) {
			defer wg.Done()
			defer func() {
				<-queryChan
			}()

			elems := make([]rpc.BatchElem, 0, len(batch))
			for _, ba := range batch {
				elem := rpc.BatchElem{
					Method: "eth_getBalance",
					Args:   []interface{}{ba.Addr, "latest"},
					Result: &ba.ValueHexBig,
				}
				elems = append(elems, elem)
			}

			var blockNumber hexutil.Uint64
			heightReq := rpc.BatchElem{
				Method: "eth_blockNumber",
				Args:   []interface{}{},
				Result: &blockNumber,
			}
			elems = append(elems, heightReq)

			utils.HandleErrorWithRetry(func() error {
				ctx, cancel := context.WithTimeout(context.Background(), b.config.Fetch.FetchTimeout)
				defer cancel()

				err := b.client.BatchCallContext(ctx, elems)
				if err != nil {
					return err
				}

				for _, e := range elems {
					if e.Error != nil {
						return e.Error
					}
				}

				if uint64(blockNumber) < num {
					return fmt.Errorf("latest height:%v got cur chain height:%v", b.latestHeight, uint64(blockNumber))
				}

				for _, ba := range batch {
					ba.Height = big.NewInt(int64(blockNumber))
				}

				return nil
			},
				b.config.Fetch.FetchRetryTimes,
				b.config.Fetch.FetchRetryInterval)
		}(batch)
	}

	wg.Wait()
}

// Clean the cache first, do not update task offset
// There is a problem here. If the program is being
// rolled back and the program is restarted, the data
// that needs to be rolled back will be lost
func (b *BalanceTask) revertBlock(blk *mtypes.Block) {
	b.balanceCache.clean()
	b.handleBlock(blk)
	err := b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, blk.Number-1)
	if err != nil {
		logrus.Errorf("update balance task height:%d err:%v", blk.Number-1, err)
	}
}
