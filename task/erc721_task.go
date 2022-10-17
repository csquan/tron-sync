package task

import (
	"fmt"
	"strings"
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
)

var erc721Transfer = erc20Transfer

const erc721abi = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`

type Erc721Task struct {
	*BaseAsyncTask
	erc721ABI   abi.ABI
	erc721Cache *lru.Cache
}

const Erc721TaskName = "erc721_basic"

func NewErc721Task(config *config.Config, client *rpc.Client, db db.IDB) (*Erc721Task, error) {
	e := &Erc721Task{}
	base, err := newBase(Erc721TaskName, config, client, db, config.Balance.BufferSize, e.handleBlock,
		e.fixHistoryData, e.revertBlock)
	if err != nil {
		return nil, err
	}
	e.BaseAsyncTask = base
	e.erc721Cache, err = lru.New(10000)
	if err != nil {
		return nil, err
	}
	r := strings.NewReader(erc721abi)
	e.erc721ABI, err = abi.JSON(r)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Erc721Task) handleBlock(blk *mtypes.Block) {
	logrus.Debugf("recv block:%d", blk.Number)
	e.handleBlocks([]*mtypes.Block{blk})
}

func toCallArg2(msg ethereum.CallMsg) interface{} {
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

func (e *Erc721Task) saveErc721(txErc721s []*mysqldb.TxErc721,
	erc721Tokens []*mysqldb.Erc721Token, infos []*mtypes.Erc721Info, height uint64) error {
	s := e.db.GetSession()
	//begin transaction
	err := s.Begin()
	if err != nil {
		return fmt.Errorf("erc721 db session beigin err:%v", err)
	}
	defer s.Close()
	//1, to save erc721 txs
	err = e.db.SaveTxErc721s(s, txErc721s)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("erc721 tx rollback err:%v", err1)
		}
		return fmt.Errorf("erc721 tx insert err:%v", err)
	}
	//2, to save erc721 tokens
	err = e.db.SaveErc721Tokens(s, erc721Tokens)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("erc721 token rollback err:%v", err1)
		}
		return fmt.Errorf("erc721 token insert err:%v", err)
	}
	//3, to save erc721 contract infos
	params := mysqldb.ConvertInErc721Infos(infos)
	err = e.db.SaveErc721Infos(s, params)
	if err != nil {
		err1 := s.Rollback()
		if err != nil {
			return fmt.Errorf("erc721 info rollback err:%v", err1)
		}
		return fmt.Errorf("erc721 Info insert err:%v", err)
	}
	//commit transaction
	err = s.Commit()
	if err != nil {
		if err1 := s.Rollback(); err1 != nil {
			logrus.Errorf("erc721 rollback err:%v,bnum:%d", err1, height)
		}
		return fmt.Errorf("erc721 db commit err:%v,bnum:%d", err, height)
	}
	return nil
}

// get erc721 info
func (e *Erc721Task) getErc721Info(addr *common.Address) (*mtypes.Erc721Info, error) {
	methods := []string{"name", "symbol"}
	elems := make([]rpc.BatchElem, 0)
	for _, method := range methods {
		input, _ := e.erc721ABI.Pack(method)
		var ret hexutil.Bytes
		msg := ethereum.CallMsg{
			To:   addr,
			Data: input,
		}
		elem := rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{toCallArg2(msg), "latest"},
			Result: &ret,
		}
		elems = append(elems, elem)
	}
	err := e.client.BatchCall(elems)
	if err != nil {
		return nil, fmt.Errorf("batch call get erc721 info err:%v,addr:%s", err, addr.Hex())
	}
	info := &mtypes.Erc721Info{}
	for i, elem := range elems {
		if elem.Error != nil {
			logrus.Warnf("erc721 info elem err:%v,elem:%v,method:%s", elem.Error, elem, methods[i])
			continue
		}
		ret := elem.Result.(*hexutil.Bytes)
		if ret == nil || len(*ret) == 0 {
			logrus.Warnf("erc721 info ret empty addr:%s,elem:%v,method:%s", addr.Hex(), elem, methods[i])
			continue
		}
		rets, err := e.erc721ABI.Unpack(methods[i], *ret)
		if err != nil {
			logrus.Warnf("erc721 info unpack err:%v,addr:%s,method:%s", err, addr.Hex(), methods[i])
			continue
		}
		if len(rets) <= 0 {
			logrus.Warnf("elem rets empty addr:%s", addr.Hex())
			continue
		}
		switch i {
		case 0:
			if name, ok := rets[0].(string); ok {
				info.Name = name
			} else {
				logrus.Warnf("erc721 info name not string addr:%s", addr.Hex())
			}
		case 1:
			if symbol, ok := rets[0].(string); ok {
				info.Symbol = symbol
			} else {
				logrus.Warnf("erc721 info symbol not string addr:%s", addr.Hex())
			}
		}
	}
	return info, nil
}

func isErc721Tx(tlog *mtypes.EventLog) bool {
	return tlog.Topic0 == erc721Transfer && tlog.Topic1 != "" && tlog.Topic2 != "" && tlog.Topic3 != ""
}

func (e *Erc721Task) doSave(txErc721s []*mysqldb.TxErc721, erc721Tokens []*mysqldb.Erc721Token, erc721Infos []*mtypes.Erc721Info, height uint64) {
	utils.HandleErrorWithRetry(func() error {
		return e.saveErc721(txErc721s, erc721Tokens, erc721Infos, height)
	}, e.config.OutPut.RetryTimes, e.config.OutPut.RetryInterval)

	utils.HandleErrorWithRetry(func() error {
		err := e.db.UpdateAsyncTaskNumByName(e.db.GetEngine(), e.name, height)
		if err != nil {
			return err
		}

		e.curHeight = height
		return nil
	}, e.config.OutPut.RetryTimes, e.config.OutPut.RetryInterval)
}

func parseTxErc721(blk *mtypes.Block) []*mysqldb.TxErc721 {
	var txErc721s []*mysqldb.TxErc721
	for _, tx := range blk.Txs {
		txhash := tx.Hash
		for _, tlog := range tx.EventLogs {
			if !isErc721Tx(tlog) {
				continue
			}
			sender := common.HexToAddress(tlog.Topic1)
			receiver := common.HexToAddress(tlog.Topic2)
			tokenid := common.HexToHash(tlog.Topic3).Big()

			addr := tlog.Addr
			txErc721 := &mysqldb.TxErc721{
				Hash:      txhash,
				Addr:      strings.ToLower(addr),
				Sender:    strings.ToLower(sender.Hex()),
				Receiver:  strings.ToLower(receiver.Hex()),
				TokenId:   tokenid.String(),
				LogIndex:  int(tlog.Index),
				BlockNum:  blk.Number,
				BlockTime: blk.TimeStamp,
			}
			txErc721s = append(txErc721s, txErc721)
		}
	}
	return txErc721s
}

func (e *Erc721Task) handleBlocks(blks []*mtypes.Block) {
	var txErc721s []*mysqldb.TxErc721
	var erc721Infos []*mtypes.Erc721Info
	var erc721Tokens []*mysqldb.Erc721Token
	var blkCount int = 0
	for i := 0; i < len(blks); i++ {
		blk := blks[i]
		logrus.Debugf("blk_id:%d, len_tx:%d", blk.Number, len(blk.Txs))
		blkCount++
		for _, tx := range blk.Txs {
			txhash := tx.Hash
			for _, tlog := range tx.EventLogs {
				if !isErc721Tx(tlog) {
					continue
				}
				sender := common.HexToAddress(tlog.Topic1)
				receiver := common.HexToAddress(tlog.Topic2)
				tokenid := common.HexToHash(tlog.Topic3).Big()

				addr := tlog.Addr
				txErc721 := &mysqldb.TxErc721{
					Hash:      txhash,
					Addr:      strings.ToLower(addr),
					Sender:    strings.ToLower(sender.Hex()),
					Receiver:  strings.ToLower(receiver.Hex()),
					TokenId:   tokenid.String(),
					LogIndex:  int(tlog.Index),
					BlockNum:  blk.Number,
					BlockTime: blk.TimeStamp,
				}
				txErc721s = append(txErc721s, txErc721)

				erc721Token := &mysqldb.Erc721Token{
					ContractAddr:  strings.ToLower(addr),
					TokenId:       tokenid.String(),
					OwnerAddr:     strings.ToLower(receiver.Hex()),
					TokenUri:      "",
					TokenMetadata: "",
					Height:        blk.Number,
				}
				erc721Tokens = append(erc721Tokens, erc721Token)

				if tlog.Addr != utils.ZeroAddress && !e.erc721Cache.Contains(addr) {
					ethAddr := common.HexToAddress(addr)
					erc721Info, err := e.getErc721Info(&ethAddr)
					if err != nil {
						logrus.Errorf("get erc721 info err:%v,addr:%s", err, addr)
					} else {
						erc721Info.Addr = strings.ToLower(addr)
						erc721Infos = append(erc721Infos, erc721Info)
						e.erc721Cache.Add(addr, struct{}{})
					}
					logrus.Debugf("erc721info:%v", erc721Info)
				}
			}
		}
		if blkCount >= e.config.Erc721Tx.MaxBlockCount || len(txErc721s) >= e.config.Erc721Tx.MaxTxCount || i == len(blks)-1 {
			logrus.Debugf("erc721 small trans blk from: %v to: %v, blkCount: %v  txCount: %v curheight:%v", i+1-blkCount, i+1, blkCount, len(txErc721s), blk.Number)
			e.doSave(txErc721s, erc721Tokens, erc721Infos, blk.Number)
			blkCount = 0
			txErc721s = txErc721s[0:0]
			erc721Tokens = erc721Tokens[0:0]
			erc721Infos = erc721Infos[0:0]
		}
	}
}

func (e *Erc721Task) fixHistoryData() {

	batch := e.config.Erc721Tx.BatchBlockCount

	var (
		blks []*mtypes.Block
		blk  *mtypes.Block
		err  error
	)
	start := e.curHeight + 1
	end := start + uint64(batch) - 1
	if e.curHeight < e.latestHeight-uint64(e.bufferSize) {
		blks, err = e.db.GetBlocksByRange(start, end, mysqldb.ERC721Filter)
	} else {
		blk, err = e.getBlkInfo(start, mysqldb.ERC721Filter)
		if blk != nil {
			blks = []*mtypes.Block{blk}
		}
	}
	if len(blks) == 0 || err != nil {
		if err != nil {
			logrus.Errorf("erc721 tx task query block info err:%v,start:%d", err, start)
		} else {
			logrus.Debugf("erc721 tx handler. cur height:%v", e.curHeight)
			lastHeight := end
			e.curHeight = lastHeight
		}
		time.Sleep(time.Second * 1)
		return
	}

	e.handleBlocks(blks)

}

func (e *Erc721Task) doRevert(blk *mtypes.Block) error {

	s := e.db.GetSession()
	defer s.Close()
	//begin transaction
	err := s.Begin()
	if err != nil {
		logrus.Errorf("erc721 revert session begin err:%v", err)
		return err
	}

	var txErc721s = parseTxErc721(blk)

	//To get contractAddr and tokenId
	var contractAddrs = make([]string, 0)
	var tokenIds = make([]string, 0)
	var senderMap = make(map[string]*string)
	for j, txErc721 := range txErc721s {
		//logrus.Infof("txErc721:",txErc721)
		var contractAddr string = txErc721.Addr
		var tokenId string = txErc721.TokenId
		_, ok := senderMap[contractAddr+tokenId]
		if !ok {
			senderMap[contractAddr+tokenId] = &(txErc721s[j].Sender)
			contractAddrs = append(contractAddrs, contractAddr)
			tokenIds = append(tokenIds, tokenId)
		}
		//else : reduplicated key!
	}
	//To update token_erc721 records by contractAddrs and tokenId
	for i, tokenId := range tokenIds {
		var contractAddr = contractAddrs[i]
		ownerAddr := senderMap[contractAddr+tokenId]
		//To update ownerAddr of token record
		_, err = s.Exec("update token_erc721 set owner_addr = ? where contract_addr = ? and token_id = ?", ownerAddr, contractAddr, tokenId)
		if err != nil {
			err1 := s.Rollback()
			if err1 != nil {
				logrus.Errorf("erc721 revert rollback err:%v", err)
				return err1
			}
			logrus.Errorf("erc721 revert err:%v", err)
			return err
		}
	}
	//At last delete tx_erc721 records
	_, err = s.Exec("update tx_erc721 set block_state = 1 where block_num = ?", blk.Number)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			logrus.Errorf("erc721 revert rollback err:%v", err)
			return err1
		}
		logrus.Errorf("erc721 revert err:%v", err)
		return err
	}
	err = e.db.UpdateAsyncTaskNumByName(s, e.name, blk.Number-1)
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
			logrus.Errorf("erc721tx_revert rollback err:%v,bnum:%d", err1, blk.Number)
			return err1
		}
		logrus.Errorf("erc721 revert session commit err:%v", err)
		return err
	}

	logrus.Infof("revert totally modify, tx erc721s records count : %d, token records count : %d", len(txErc721s), len(senderMap))
	return nil
}

func (e *Erc721Task) revertBlock(blk *mtypes.Block) {
	bNum := blk.Number
	logrus.Debugf("tx_erc721 task revert start bNum: %d", bNum)

	utils.HandleErrorWithRetry(func() error {
		err := e.doRevert(blk)
		if err != nil {
			return err
		}
		return nil
	}, e.config.OutPut.RetryTimes, e.config.OutPut.RetryInterval)

}
