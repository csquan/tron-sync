package task

import (
	"errors"
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
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

type Erc1155BalanceTask struct {
	*BaseAsyncTask
	erc1155ABI abi.ABI
	cache      *CacheFilter
	multiCall  *utils.MultiCall
}

const Erc1155BalanceTaskName = "erc1155_balance"

func NewErc1155BalanceTask(config *config.Config, client *rpc.Client, db db.IDB, monitorDb db.IDB) (*Erc1155BalanceTask, error) {
	b := &Erc1155BalanceTask{}
	base, err := newBase(Erc1155BalanceTaskName, config, client, db, monitorDb, config.Erc1155Balance.BufferSize,
		b.handleBlock, b.fixHistoryData, b.revertBlock)
	if err != nil {
		return nil, err
	}
	b.BaseAsyncTask = base
	r := strings.NewReader(Erc1155ABI)
	b.erc1155ABI, err = abi.JSON(r)
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
	} else {
		return nil, errors.New("multiCallContract not config")
	}
	return b, nil
}

func (b *Erc1155BalanceTask) handleBlock(block *mtypes.Block) {
	height := block.Number
	b.handleBlocks([]*mtypes.Block{block})
	err := b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, height)
	if err != nil {
		logrus.Errorf("update Erc1155BalanceTask height:%d err:%v", height, err)
	}
	b.curHeight = height
}

func (b *Erc1155BalanceTask) handleBlocks(blocks []*mtypes.Block) {
	var balances []*mysqldb.BalanceErc1155
	for _, block := range blocks {
		var txErc1155s []*mysqldb.TxErc1155
		for _, tx := range block.Txs {
			txHash := tx.Hash
			for _, log := range tx.EventLogs {
				txErc1155s = append(txErc1155s, parseFromTransferSingle(b.erc1155ABI, block, txHash, log)...)
				txErc1155s = append(txErc1155s, parseFromTransferBatch(b.erc1155ABI, block, txHash, log)...)
			}
		}
		for _, tx := range txErc1155s {
			if b.isNeedToUpdate(tx.Sender, tx.Addr, tx.TokenId, block.Number) && tx.Sender != "0x0000000000000000000000000000000000000000" {
				balances = append(balances, &mysqldb.BalanceErc1155{Addr: tx.Sender, ContractAddr: tx.Addr, TokenID: tx.TokenId, Height: block.Number})
			}
			if b.isNeedToUpdate(tx.Receiver, tx.Addr, tx.TokenId, block.Number) && tx.Receiver != "0x0000000000000000000000000000000000000000" {
				balances = append(balances, &mysqldb.BalanceErc1155{Addr: tx.Receiver, ContractAddr: tx.Addr, TokenID: tx.TokenId, Height: block.Number})
			}
		}
	}

	b.getErc1155BalanceAll(balances, b.config.Erc1155Balance.BatchRPC)
	utils.HandleErrorWithRetry(
		func() error {
			err := b.saveErc1155Balances(balances)
			return err
		}, b.config.OutPut.RetryTimes, b.config.OutPut.RetryInterval)
	for _, balance := range balances {
		b.updateCache(balance.Addr, balance.ContractAddr, balance.TokenID, balance.Height)
	}
}

func (b *Erc1155BalanceTask) updateCache(addr, contractAddr, tokenID string, h uint64) {
	key := addr + "|" + contractAddr + "|" + tokenID
	b.cache.Add(strings.ToLower(key), h)
}

func (b *Erc1155BalanceTask) fixHistoryData() {
	batch := b.config.Erc1155Balance.BatchBlockCount
	var (
		blks []*mtypes.Block
		blk  *mtypes.Block
		err  error
	)
	start := b.curHeight + 1
	end := start + uint64(batch) - 1
	if b.curHeight < b.latestHeight-uint64(b.bufferSize) {
		blks, err = b.db.GetBlocksByRange(start, end, mysqldb.ERC1155Filter)
	} else {
		blk, err = b.getBlkInfo(start, mysqldb.ERC1155Filter)
		if blk != nil {
			blks = []*mtypes.Block{blk}
		}
	}
	if len(blks) == 0 || err != nil {
		if err != nil {
			logrus.Errorf("erc1155 balance task query block info err:%v,start:%d", err, start)
		} else {
			logrus.Debugf("erc1155 balance handler. cur height:%v", b.curHeight)
			b.curHeight = end
		}
		time.Sleep(time.Second * 1)
		return
	}
	height := blks[len(blks)-1].Number
	b.handleBlocks(blks)
	err = b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, height)
	if err != nil {
		logrus.Errorf("update Erc1155BalanceTask height:%d err:%v", height, err)
	}
	b.curHeight = height
}

func (b *Erc1155BalanceTask) revertBlock(block *mtypes.Block) {
	b.cache.clean()
	b.handleBlock(block)
	err := b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, block.Number-1)
	if err != nil {
		logrus.Errorf("update Erc1155BalanceTask height:%d err:%v", block.Number-1, err)
	}
}

// balance是否需要更新
func (b *Erc1155BalanceTask) isNeedToUpdate(addr, contractAddr, tokenID string, h uint64) bool {
	key := addr + "|" + contractAddr + "|" + tokenID
	v, ok := b.cache.Get(strings.ToLower(key))
	if !ok {
		return true
	}
	updatedH, _ := v.(uint64)
	return h > updatedH
}

func (b *Erc1155BalanceTask) newCache() (*CacheFilter, error) {
	return NewCacheFilter(b.config.Erc1155Balance.CacheSize, 1024*1024*8*20, 20, func(key string) (interface{}, bool, error) {
		args := strings.Split(key, "|")

		addr := strings.ToLower(args[0])
		contractAddr := strings.ToLower(args[1])
		tokenID := strings.ToLower(args[2])

		res, err := b.db.GetErc1155Balance(addr, contractAddr, tokenID)
		if err != nil {
			return nil, false, err
		}
		if res == nil {
			return nil, false, nil
		}
		return res.Height, true, nil
	})
}

func (b *Erc1155BalanceTask) getErc1155BalanceAll(bs []*mysqldb.BalanceErc1155, batch int) {
	size := len(bs)
	var cur int = b.config.Erc1155Balance.Concurrent
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
				return b.getBalanceByMultiCall(bs[s:e])
			}, b.config.Fetch.FetchRetryTimes,
				b.config.Fetch.FetchRetryInterval)
		}(i, end)
		i = end
	}
	wg.Wait()
}

func (b *Erc1155BalanceTask) getBalanceByMultiCall(bs []*mysqldb.BalanceErc1155) error {
	inputs := make([]utils.MultiCallInput, 0, len(bs))
	for _, balance := range bs {
		tokenID, ok := new(big.Int).SetString(balance.TokenID, 10)
		if !ok {
			logrus.Warnf("erc1155 tokenID to big.int failed, tokenID:%s", tokenID)
			continue
		}
		input, err := b.erc1155ABI.Pack("balanceOf", common.HexToAddress(balance.Addr), tokenID)
		if err != nil {
			return fmt.Errorf("pack erc1155 balanceOf input err:%v", err)
		}
		inputs = append(inputs, utils.MultiCallInput{
			Target:   common.HexToAddress(balance.ContractAddr),
			CallData: input,
		})
	}
	st := time.Now()
	res := b.multiCall.AggregateWithRetry(inputs, b.latestHeight)
	logrus.Infof("erc1155 balance cost AggregateWithRetry:%d,batch:%d", time.Since(st)/time.Millisecond, len(bs))
	if res.BlockNumber.Cmp(big.NewInt(int64(b.latestHeight))) < 0 {
		return fmt.Errorf("query erc1155 balance error. latest height:%v, multicall height:%v", b.latestHeight, res.BlockNumber.String())
	}

	for i, elem := range res.ReturnData {
		if elem.Err != nil {
			logrus.Warnf("get erc1155 balance error. contract:%v, addr:%v, err:%v", bs[i].ContractAddr, bs[i].Addr, elem.Err)
			continue
		}

		if len(elem.Output) == 0 {
			continue
		}
		rets, err := b.erc1155ABI.Unpack("balanceOf", elem.Output)
		if err != nil {
			logrus.Warnf("unpack erc1155 balanceOf err:%v,contract:%v,address:%v", err, bs[i].ContractAddr, bs[i].Addr)
			continue
		}
		if len(rets) == 0 {
			logrus.Warnf("erc1155 balanceOf ret size err:%v,contract:%v,address:%v", err, bs[i].ContractAddr, bs[i].Addr)
			continue
		}
		if v, ok := rets[0].(*big.Int); ok {
			bs[i].Balance = v.String()
			if len(bs[i].Balance) > 65 {
				bs[i].BalanceOrigin = bs[i].Balance
				bs[i].Balance = bs[i].Balance[:65]
			}

		} else {
			logrus.Warnf("erc1155 balanceOf ret not *big.Int:%v,contract:%v,address:%v", err, bs[i].ContractAddr, bs[i].Addr)
		}
	}

	return nil
}

func (b *Erc1155BalanceTask) saveErc1155Balances(balances []*mysqldb.BalanceErc1155) error {
	s := b.db.GetSession()
	err := s.Begin()
	if err != nil {
		return fmt.Errorf("erc1155 balance session beigin err:%v", err)
	}
	defer s.Close()
	err = b.db.SaveErc1155Balances(s, balances)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("erc1155 balances roolback err:%v", err1)
		}
		return fmt.Errorf("erc1155 balances insert err:%v", err)
	}
	err = s.Commit()
	if err != nil {
		return fmt.Errorf("erc1155 balances commit err:%v", err)
	}
	return nil
}
