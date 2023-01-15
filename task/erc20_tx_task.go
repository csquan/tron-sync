package task

import (
	"encoding/json"
	"fmt"
	"github.com/chainmonitor/utils"
	"math/big"
	"strings"
	"time"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/db"
	"github.com/chainmonitor/kafka"
	"github.com/chainmonitor/mtypes"
	"github.com/chainmonitor/output/mysqldb"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
	"github.com/sirupsen/logrus"
)

var erc20Transfer = `0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef`

const erc20abi = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
const Erc20TxTaskName = "erc20_tx"

type Erc20TxTask struct {
	*BaseAsyncTask
	erc20ABI   abi.ABI
	erc20infos *lru.Cache
	kafka      *kafka.PushKafkaService
}

func NewErc20TxTask(config *config.Config, client *rpc.Client, db db.IDB) (*Erc20TxTask, error) {
	et := &Erc20TxTask{}
	base, err := newBase(Erc20TxTaskName, config, client, db, config.Balance.BufferSize,
		et.handleBlock, et.fixHistoryData, et.revertBlock)
	if err != nil {
		return nil, err
	}
	et.BaseAsyncTask = base
	et.erc20infos, err = lru.New(10000)
	if err != nil {
		return nil, err
	}
	r := strings.NewReader(erc20abi)
	et.erc20ABI, err = abi.JSON(r)
	if err != nil {
		return nil, err
	}
	p, err := kafka.NewSyncProducer(config.Kafka)
	if err != nil {
		return nil, err
	}
	et.kafka, err = kafka.NewPushKafkaService(config, p)
	if err != nil {
		return nil, err
	}
	et.kafka.Topic = et.config.Kafka.Topic
	return et, nil
}

func (et *Erc20TxTask) handleBlock(blk *mtypes.Block) {
	logrus.Debugf("recv block:%d", blk.Number)
	et.handleBlocks([]*mtypes.Block{blk})
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}

// get erc20 info
func (et *Erc20TxTask) getErc20Info(addr *common.Address, height uint64) (*mtypes.Erc20Info, error) {
	methods := []string{"name", "symbol", "decimals", "totalSupply"}
	elems := make([]rpc.BatchElem, 0)
	for _, method := range methods {
		input, _ := et.erc20ABI.Pack(method)
		var ret hexutil.Bytes
		msg := ethereum.CallMsg{
			To:   addr,
			Data: input,
		}
		elem := rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{toCallArg(msg), "latest"},
			Result: &ret,
		}
		elems = append(elems, elem)
	}
	var blockNumber hexutil.Uint64
	elems = append(elems, rpc.BatchElem{
		Method: "eth_blockNumber",
		Args:   []interface{}{},
		Result: &blockNumber,
	})
	err := et.client.BatchCall(elems)
	if err != nil {
		return nil, fmt.Errorf("batch call get erc20 info err:%v,addr:%s", err, addr.Hex())
	}
	if uint64(blockNumber) < height {
		return nil, fmt.Errorf("get erc20_info height too low")
	}
	info := &mtypes.Erc20Info{}
	for i, elem := range elems {
		if elem.Method == "eth_blockNumber" {
			continue
		}
		if elem.Error != nil {
			logrus.Infof("erc20 info elem err:%v,elem:%v,method:%s", elem.Error, elem, methods[i])
			continue
		}
		ret := elem.Result.(*hexutil.Bytes)
		if ret == nil || len(*ret) == 0 {
			logrus.Infof("erc20 info ret empty addr:%s,elem:%v,method:%s", addr.Hex(), elem, methods[i])
			continue
		}
		rets, err := et.erc20ABI.Unpack(methods[i], *ret)
		if err != nil {
			logrus.Infof("erc20 info unpack err:%v,addr:%s,method:%s,ret:%s", err, addr.Hex(), methods[i], *ret)
			continue
		}
		if len(rets) <= 0 {
			logrus.Infof("elem rets empty addr:%s", addr.Hex())
			continue
		}
		switch i {
		case 0:
			if name, ok := rets[0].(string); ok {
				info.Name = name
			} else {
				logrus.Infof("erc20 info name not string addr:%s", addr.Hex())
			}
		case 1:
			if symbol, ok := rets[0].(string); ok {
				info.Symbol = symbol
			} else {
				logrus.Infof("erc20 info symbol not string addr:%s", addr.Hex())
			}
		case 2:
			if decimals, ok := rets[0].(uint8); ok {
				info.Decimals = decimals
			} else {
				logrus.Infof("erc20 info decimals not uint8 addr:%s", addr.Hex())
			}
		case 3:
			if totoalSupply, ok := rets[0].(*big.Int); ok {
				info.TotoalSupply = totoalSupply.String()
			} else {
				logrus.Infof("erc20 info totoalSupply not *big.Int addr:%s", addr.Hex())
			}
		}
	}
	return info, nil
}

func isErc20Tx(tlog *mtypes.EventLog) bool {
	var topicOk bool
	if tlog.Topic0 != "" && tlog.Topic1 != "" && tlog.Topic2 != "" && tlog.Topic3 == "" {
		topicOk = true
	}
	return topicOk && tlog.Topic0 == erc20Transfer
}

func (et *Erc20TxTask) getOrigin(erc20Infos []*mtypes.Erc20Info, height uint64) {
	s := et.db.GetSession()
	defer s.Close()

	for _, info := range erc20Infos {
		if info.TotoalSupply != "" {
			//取出db中存储的原有totalsupply，以info.Addr查找Erc20_info中
			oldInfo, err := et.db.GetErc20info(info.Addr)
			if err != nil {
				fmt.Println("GetErc20info error" + err.Error())
			}
			if oldInfo == nil {
				fmt.Println("GetErc20info null")
			} else if oldInfo.TotoalSupply != "" {
				//这里应该先删除info.Addr这一行
				err := et.db.DeleteErc20InfoByAddr(et.db.GetSession(), info.Addr)
				if err != nil {
					fmt.Printf("DeleteErc20InfoByAddr error")
				}
				info.TotoalSupplyOrigin = oldInfo.TotoalSupply
			}
		}
	}
	err := s.Commit()
	if err != nil {
		if err1 := s.Rollback(); err1 != nil {
			fmt.Printf("erc20tx rollback err:%v,bnum:%d", err1, height)
		}
		fmt.Printf("erc20tx commit err:%v,bnum:%d", err, height)
	}
}

func (et *Erc20TxTask) doSave(txErc20s []*mysqldb.TxErc20, erc20Infos []*mtypes.Erc20Info, height uint64) {
	utils.HandleErrorWithRetry(func() error {
		st := time.Now()
		defer func() {
			logrus.Infof("erc20 db commit cost:%d", time.Since(st)/time.Millisecond)
		}()
		s := et.db.GetSession()
		err := s.Begin()
		if err != nil {
			return fmt.Errorf("tx erc20 session beigin err:%v", err)
		}
		defer s.Close()

		err = et.db.SaveTxErc20s(s, txErc20s)
		if err != nil {
			err1 := s.Rollback()
			if err1 != nil {
				return fmt.Errorf("tx erc20 rollback err:%v", err1)
			}
			return fmt.Errorf("tx erc20 insert err:%v", err)
		}

		params := mysqldb.ConvertInErc20Infos(erc20Infos)
		err = et.db.SaveErc20Infos(s, params)
		if err != nil {
			err1 := s.Rollback()
			if err1 != nil {
				return fmt.Errorf("erc20info rollback err:%v", err1)
			}
			return err
		}

		err = et.db.UpdateAsyncTaskNumByName(s, et.name, height)
		if err != nil {
			err1 := s.Rollback()
			if err1 != nil {
				return fmt.Errorf("update task rollback err:%v", err1)
			}
			return err
		}

		err = s.Commit()
		if err != nil {
			if err1 := s.Rollback(); err1 != nil {
				logrus.Errorf("erc20tx rollback err:%v,bnum:%d", err1, height)
			}
			return fmt.Errorf("erc20tx commit err:%v,bnum:%d", err, height)
		}

		et.curHeight = height

		return nil

	}, et.config.OutPut.RetryTimes, et.config.OutPut.RetryInterval)
}

func (et *Erc20TxTask) handleBlocks(blks []*mtypes.Block) {
	var txErc20s []*mysqldb.TxErc20
	var erc20Infos []*mtypes.Erc20Info
	var txKakfkas []*mtypes.TxKakfa
	var blkCount int = 0

	for i := 0; i < len(blks); i++ {
		blk := blks[i]

		blkCount++
		for _, tx := range blk.Txs {
			txhash := tx.Hash
			for _, tlog := range tx.EventLogs {
				if !isErc20Tx(tlog) {
					continue
				}

				sender := common.HexToAddress(tlog.Topic1)
				receiver := common.HexToAddress(tlog.Topic2)
				tokens := new(big.Int)
				data := hexutil.MustDecode(tlog.Data)
				tokens.SetBytes(data)
				tokenCnt := tokens.String()
				var origin string
				if len(tokenCnt) > 65 {
					origin = tokenCnt
					tokenCnt = tokenCnt[:65]
				}
				addr := tlog.Addr
				txErc20 := &mysqldb.TxErc20{
					Hash:           txhash,
					Addr:           strings.ToLower(addr),
					Sender:         strings.ToLower(sender.Hex()),
					Receiver:       strings.ToLower(receiver.Hex()),
					TokenCnt:       tokenCnt,
					TokenCntOrigin: origin,
					LogIndex:       int(tlog.Index),
					BlockNum:       blk.Number,
					BlockTime:      blk.TimeStamp,
				}
				txKakfa := &mtypes.TxKakfa{
					From:      sender,
					To:        receiver,
					Amount:    tokenCnt,
					TokenAddr: addr,
				}
				txErc20s = append(txErc20s, txErc20)
				txKakfkas = append(txKakfkas, txKakfa)
				if tlog.Addr != utils.ZeroAddress && !et.erc20infos.Contains(addr) {
					// st := time.Now()
					ethAddr := common.HexToAddress(addr)
					erc20Info, err := et.getErc20Info(&ethAddr, blk.Number)
					if err != nil {
						logrus.Warnf("get erc20 info err:%v,addr:%s", err, addr)
					} else {
						erc20Info.Addr = strings.ToLower(addr)
						erc20Infos = append(erc20Infos, erc20Info)
						et.erc20infos.Add(addr, struct{}{})
					}
					// log.Printf("erc20info:%v", erc20Info)
					// logrus.Infof("erc20 info cost:%d", time.Since(st)/time.Millisecond)
				}
				//push erc20tx to kafka KakfaTx
				b, err := json.Marshal(txKakfkas)
				if err != nil {
					logrus.Warnf("Marshal txErc20s err:%v", err)
				}

				err = et.kafka.Pushkafka(b)
				if err != nil {
					logrus.Error(err)
				}
			}
		}
		if blkCount >= et.config.Erc20Tx.MaxBlockCount || len(txErc20s) >= et.config.Erc20Tx.MaxTxCount || i == len(blks)-1 {
			logrus.Debugf("erc20 small trans blk from: %v to: %v, blkCount: %v  txCount: %v curheight:%v", i+1-blkCount, i+1, blkCount, len(txErc20s), blk.Number)
			et.getOrigin(erc20Infos, blk.Number)
			et.doSave(txErc20s, erc20Infos, blk.Number)
			blkCount = 0
			txErc20s = txErc20s[0:0]
			erc20Infos = erc20Infos[0:0]
		}
	}
}

func (et *Erc20TxTask) fixHistoryData() {
	batch := et.config.Erc20Tx.BatchBlockCount

	var (
		blks []*mtypes.Block
		blk  *mtypes.Block
		err  error
	)
	start := et.curHeight + 1
	end := start + uint64(batch) - 1
	if et.curHeight < et.latestHeight-uint64(et.bufferSize) {
		blks, err = et.db.GetBlocksByRange(start, end, mysqldb.ERC20Filter)
	} else {
		blk, err = et.getBlkInfo(start, mysqldb.ERC20Filter)
		if blk != nil {
			blks = []*mtypes.Block{blk}
		}
	}
	if len(blks) == 0 || err != nil {
		if err != nil {
			logrus.Errorf("erc20 tx task query block info err:%v,start:%d", err, start)
		} else {
			logrus.Debugf("erc20 tx handler. cur height:%v", et.curHeight)
			lastHeight := end
			et.curHeight = lastHeight
		}
		time.Sleep(time.Second * 1)
		return
	}

	et.handleBlocks(blks)
}

func (et *Erc20TxTask) doRevert(bNum uint64) error {

	s := et.db.GetSession()
	defer s.Close()
	//begin transaction
	err := s.Begin()
	if err != nil {
		logrus.Errorf("tx erc20 revert session begin err:%v", err)
		return err
	}

	_, err = s.Exec("update tx_erc20 set block_state = 1 where block_num = ?", bNum)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			logrus.Errorf("tx erc20 revert rollback err:%v", err)
			return err1
		}
		logrus.Errorf("tx erc20 revert err:%v", err)
		return err
	}
	//TODO updae task number
	err = et.db.UpdateAsyncTaskNumByName(s, et.name, bNum-1)
	if err != nil {
		if err1 := s.Rollback(); err1 != nil {
			return err1
		}
		return err
	}
	//commit transaction
	err = s.Commit()
	if err != nil {
		if err1 := s.Rollback(); err1 != nil {
			logrus.Errorf("erc20tx_revert rollback err:%v,bnum:%d", err1, bNum)
			return err1
		}
		logrus.Errorf("erc20 revert session commit err:%v", err)
		return err
	}

	return nil
}

func (et *Erc20TxTask) revertBlock(blk *mtypes.Block) {
	bNum := blk.Number
	logrus.Debugf("tx_erc20 task revert start bNum: %d", bNum)

	utils.HandleErrorWithRetry(func() error {
		err := et.doRevert(bNum)
		if err != nil {
			return err
		}
		return nil
	}, et.config.OutPut.RetryTimes, et.config.OutPut.RetryInterval)

}
