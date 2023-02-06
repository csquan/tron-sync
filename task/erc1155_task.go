package task

import (
	"fmt"
	"math/big"
	"strings"
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

var Erc1155SingleHash = `0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62`
var Erc1155BatchHash = `0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb`

type Erc1155Task struct {
	*BaseAsyncTask
	erc1155ABI abi.ABI
}

const Erc1155TaskName = "erc1155_basic"

func NewErc1155Task(config *config.Config, client *rpc.Client, db db.IDB, monitorDb db.IDB) (*Erc1155Task, error) {
	e := &Erc1155Task{}
	base, err := newBase(Erc1155TaskName, config, client, db, monitorDb, config.Erc1155Tx.BufferSize, e.handleBlock,
		e.fixHistoryData, e.revertBlock)
	if err != nil {
		return nil, err
	}
	e.BaseAsyncTask = base
	r := strings.NewReader(Erc1155ABI)
	e.erc1155ABI, err = abi.JSON(r)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Erc1155Task) handleBlock(blk *mtypes.Block) {
	logrus.Debugf("Erc1155Task recv block:%d", blk.Number)
	e.handleBlocks([]*mtypes.Block{blk})
}

// func (e *Erc1155Task) saveErc1155(txErc1155s []*mysqldb.TxErc1155, erc1155Tokens []*mysqldb.Erc1155Token, infos []*mtypes.Erc1155Info) error {
func (e *Erc1155Task) saveErc1155(txErc1155s []*mysqldb.TxErc1155, height uint64) error {
	s := e.db.GetSession()
	//begin transaction
	err := s.Begin()
	if err != nil {
		return fmt.Errorf("erc1155 db session beigin err:%v", err)
	}
	defer s.Close()
	err = e.db.SaveTxErc1155s(s, txErc1155s)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("erc1155 tx rollback err:%v", err1)
		}
		return fmt.Errorf("erc1155 tx insert err:%v", err)
	}
	err = e.db.UpdateAsyncTaskNumByName(s, e.name, height)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("erc1155 update task rollback err:%v", err1)
		}
		return fmt.Errorf("erc1155 update task err:%v", err)
	}
	//commit transaction
	err = s.Commit()
	if err != nil {
		if err1 := s.Rollback(); err1 != nil {
			logrus.Errorf("erc1155 rollback err:%v,bnum:%d", err1, height)
		}
		return fmt.Errorf("erc1155 db commit err:%v", err)
	}
	return nil
}

func isErc1155SingleTx(tlog *mtypes.EventLog) bool {
	return tlog.Topic0 == Erc1155SingleHash && tlog.Topic1 != "" && tlog.Topic2 != "" && tlog.Topic3 != ""
}

func isErc1155BatchTx(tlog *mtypes.EventLog) bool {
	return tlog.Topic0 == Erc1155BatchHash && tlog.Topic1 != "" && tlog.Topic2 != "" && tlog.Topic3 != ""
}

func (e *Erc1155Task) doSave(txErc1155s []*mysqldb.TxErc1155, height uint64) {
	utils.HandleErrorWithRetry(func() error {
		err := e.saveErc1155(txErc1155s, height)
		if err != nil {
			return err
		}
		e.curHeight = height
		return nil
	}, e.config.OutPut.RetryTimes, e.config.OutPut.RetryInterval)
}

func (e *Erc1155Task) handleBlocks(blks []*mtypes.Block) {
	var txErc1155s []*mysqldb.TxErc1155
	var blkCount int = 0
	for i := 0; i < len(blks); i++ {
		blk := blks[i]
		logrus.Debugf("blk_id:%d, len_tx:%d", blk.Number, len(blk.Txs))
		blkCount++
		for _, tx := range blk.Txs {
			txHash := tx.Hash
			for _, log := range tx.EventLogs {
				txErc1155s = append(txErc1155s, parseFromTransferSingle(e.erc1155ABI, blk, txHash, log)...)
				txErc1155s = append(txErc1155s, parseFromTransferBatch(e.erc1155ABI, blk, txHash, log)...)
			}
		}
		if blkCount >= e.config.Erc1155Tx.MaxBlockCount || len(txErc1155s) >= e.config.Erc1155Tx.MaxTxCount || i == len(blks)-1 {
			e.doSave(txErc1155s, blk.Number)
			blkCount = 0
			txErc1155s = txErc1155s[0:0]
		}
	}
}

func (e *Erc1155Task) fixHistoryData() {
	batch := e.config.Erc1155Tx.BatchBlockCount
	var (
		blks []*mtypes.Block
		blk  *mtypes.Block
		err  error
	)
	start := e.curHeight + 1
	end := start + uint64(batch) - 1
	if e.curHeight < e.latestHeight-uint64(e.bufferSize) {
		blks, err = e.db.GetBlocksByRange(start, end, mysqldb.ERC1155Filter)
	} else {
		blk, err = e.getBlkInfo(start, mysqldb.ERC1155Filter)
		if blk != nil {
			blks = []*mtypes.Block{blk}
		}
	}
	if len(blks) == 0 || err != nil {
		if err != nil {
			logrus.Errorf("erc1155 tx task query block info err:%v,start:%d", err, start)
		} else {
			logrus.Debugf("erc1155 tx handler. cur height:%v", e.curHeight)
			e.curHeight = end
		}
		time.Sleep(time.Second * 1)
		return
	}

	e.handleBlocks(blks)
}

func (e *Erc1155Task) doRevert(blk *mtypes.Block) error {
	s := e.db.GetSession()
	defer s.Close()
	//begin transaction
	err := s.Begin()
	if err != nil {
		logrus.Errorf("erc1155 revert session begin err:%v", err)
		return err
	}
	_, err = s.Exec("update tx_erc1155 set block_state = 1 where block_num = ?", blk.Number)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			logrus.Errorf("erc1155 revert rollback err:%v", err)
			return err1
		}
		logrus.Errorf("erc1155 revert err:%v", err)
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
			logrus.Errorf("erc1155_revert rollback err:%v,bnum:%d", err1, blk.Number)
			return err1
		}
		logrus.Errorf("erc1155 revert session commit err:%v", err)
		return err
	}
	return nil
}

func (e *Erc1155Task) revertBlock(blk *mtypes.Block) {
	bNum := blk.Number
	logrus.Debugf("tx_erc1155 task revert start bNum: %d", bNum)
	utils.HandleErrorWithRetry(func() error {
		return e.doRevert(blk)
	}, e.config.OutPut.RetryTimes, e.config.OutPut.RetryInterval)

}

func parseFromTransferSingle(erc1155ABI abi.ABI, blk *mtypes.Block, txHash string, log *mtypes.EventLog) (txErc1155s []*mysqldb.TxErc1155) {
	if isErc1155SingleTx(log) {
		operator := common.HexToAddress(log.Topic1)
		sender := common.HexToAddress(log.Topic2)
		receiver := common.HexToAddress(log.Topic3)

		var transferSingleData struct {
			Id    *big.Int
			Value *big.Int
		}
		// tokenAbi, err := abi.JSON(strings.NewReader(Erc1155ABI))
		// if err != nil {
		// 	logrus.Infof("erc1155 abi.JSON err: %v", err)
		// }
		err := erc1155ABI.UnpackIntoInterface(&transferSingleData, "TransferSingle", hexutil.MustDecode(log.Data))
		if err != nil {
			logrus.Infof("erc1155 single err: %v", err)
		}

		tokens := transferSingleData.Value
		tokenCnt := tokens.String()
		if len(tokenCnt) > 65 {
			tokenCnt = tokenCnt[:65]
		}

		txErc1155 := &mysqldb.TxErc1155{
			Hash:      txHash,
			Addr:      strings.ToLower(log.Addr),
			Operator:  strings.ToLower(operator.Hex()),
			Sender:    strings.ToLower(sender.Hex()),
			Receiver:  strings.ToLower(receiver.Hex()),
			TokenId:   transferSingleData.Id.String(),
			TokenCnt:  tokenCnt,
			LogIndex:  int(log.Index),
			BlockNum:  blk.Number,
			BlockTime: blk.TimeStamp,
		}
		txErc1155s = append(txErc1155s, txErc1155)
	}
	return
}

func parseFromTransferBatch(erc1155ABI abi.ABI, blk *mtypes.Block, txHash string, log *mtypes.EventLog) (txErc1155s []*mysqldb.TxErc1155) {
	if isErc1155BatchTx(log) {
		operator := common.HexToAddress(log.Topic1)
		sender := common.HexToAddress(log.Topic2)
		receiver := common.HexToAddress(log.Topic3)

		var transferBatchData struct {
			Ids    []*big.Int
			Values []*big.Int
		}
		// tokenAbi, err := abi.JSON(strings.NewReader(Erc1155ABI))
		// if err != nil {
		// 	logrus.Infof("erc1155 abi.JSON err: %v", err)
		// }
		err := erc1155ABI.UnpackIntoInterface(&transferBatchData, "TransferBatch", hexutil.MustDecode(log.Data))
		if err != nil {
			logrus.Infof("erc1155 abi.UnpackIntoInterface err: %v", err)
		}

		ids := transferBatchData.Ids
		values := transferBatchData.Values
		if len(values) != len(ids) {
			logrus.Warnf("erc1155 nonstandard tx_hash:%s,log_index:%d", txHash, log.Index)
			return
		}

		for index, id := range ids {
			tokenCnt := values[index].String()
			if len(tokenCnt) > 65 {
				tokenCnt = tokenCnt[:65]
			}
			txErc1155 := &mysqldb.TxErc1155{
				Hash:      txHash,
				Addr:      strings.ToLower(log.Addr),
				Operator:  strings.ToLower(operator.Hex()),
				Sender:    strings.ToLower(sender.Hex()),
				Receiver:  strings.ToLower(receiver.Hex()),
				TokenId:   id.String(),
				TokenCnt:  tokenCnt,
				LogIndex:  int(log.Index),
				BlockNum:  blk.Number,
				BlockTime: blk.TimeStamp,
			}
			txErc1155s = append(txErc1155s, txErc1155)
		}
	}
	return
}

const Erc1155ABI = `[
    {
        "inputs": [
            {
                "internalType": "string",
                "name": "uri_",
                "type": "string"
            }
        ],
        "stateMutability": "nonpayable",
        "type": "constructor"
    },
    {
        "anonymous": false,
        "inputs": [
            {
                "indexed": true,
                "internalType": "address",
                "name": "account",
                "type": "address"
            },
            {
                "indexed": true,
                "internalType": "address",
                "name": "operator",
                "type": "address"
            },
            {
                "indexed": false,
                "internalType": "bool",
                "name": "approved",
                "type": "bool"
            }
        ],
        "name": "ApprovalForAll",
        "type": "event"
    },
    {
        "anonymous": false,
        "inputs": [
            {
                "indexed": true,
                "internalType": "address",
                "name": "operator",
                "type": "address"
            },
            {
                "indexed": true,
                "internalType": "address",
                "name": "from",
                "type": "address"
            },
            {
                "indexed": true,
                "internalType": "address",
                "name": "to",
                "type": "address"
            },
            {
                "indexed": false,
                "internalType": "uint256[]",
                "name": "ids",
                "type": "uint256[]"
            },
            {
                "indexed": false,
                "internalType": "uint256[]",
                "name": "values",
                "type": "uint256[]"
            }
        ],
        "name": "TransferBatch",
        "type": "event"
    },
    {
        "anonymous": false,
        "inputs": [
            {
                "indexed": true,
                "internalType": "address",
                "name": "operator",
                "type": "address"
            },
            {
                "indexed": true,
                "internalType": "address",
                "name": "from",
                "type": "address"
            },
            {
                "indexed": true,
                "internalType": "address",
                "name": "to",
                "type": "address"
            },
            {
                "indexed": false,
                "internalType": "uint256",
                "name": "id",
                "type": "uint256"
            },
            {
                "indexed": false,
                "internalType": "uint256",
                "name": "value",
                "type": "uint256"
            }
        ],
        "name": "TransferSingle",
        "type": "event"
    },
    {
        "anonymous": false,
        "inputs": [
            {
                "indexed": false,
                "internalType": "string",
                "name": "value",
                "type": "string"
            },
            {
                "indexed": true,
                "internalType": "uint256",
                "name": "id",
                "type": "uint256"
            }
        ],
        "name": "URI",
        "type": "event"
    },
    {
        "inputs": [
            {
                "internalType": "bytes4",
                "name": "interfaceId",
                "type": "bytes4"
            }
        ],
        "name": "supportsInterface",
        "outputs": [
            {
                "internalType": "bool",
                "name": "",
                "type": "bool"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "uint256",
                "name": "",
                "type": "uint256"
            }
        ],
        "name": "uri",
        "outputs": [
            {
                "internalType": "string",
                "name": "",
                "type": "string"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "address",
                "name": "account",
                "type": "address"
            },
            {
                "internalType": "uint256",
                "name": "id",
                "type": "uint256"
            }
        ],
        "name": "balanceOf",
        "outputs": [
            {
                "internalType": "uint256",
                "name": "",
                "type": "uint256"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "address[]",
                "name": "accounts",
                "type": "address[]"
            },
            {
                "internalType": "uint256[]",
                "name": "ids",
                "type": "uint256[]"
            }
        ],
        "name": "balanceOfBatch",
        "outputs": [
            {
                "internalType": "uint256[]",
                "name": "",
                "type": "uint256[]"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "address",
                "name": "operator",
                "type": "address"
            },
            {
                "internalType": "bool",
                "name": "approved",
                "type": "bool"
            }
        ],
        "name": "setApprovalForAll",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "address",
                "name": "account",
                "type": "address"
            },
            {
                "internalType": "address",
                "name": "operator",
                "type": "address"
            }
        ],
        "name": "isApprovedForAll",
        "outputs": [
            {
                "internalType": "bool",
                "name": "",
                "type": "bool"
            }
        ],
        "stateMutability": "view",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "address",
                "name": "from",
                "type": "address"
            },
            {
                "internalType": "address",
                "name": "to",
                "type": "address"
            },
            {
                "internalType": "uint256",
                "name": "id",
                "type": "uint256"
            },
            {
                "internalType": "uint256",
                "name": "amount",
                "type": "uint256"
            },
            {
                "internalType": "bytes",
                "name": "data",
                "type": "bytes"
            }
        ],
        "name": "safeTransferFrom",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
    },
    {
        "inputs": [
            {
                "internalType": "address",
                "name": "from",
                "type": "address"
            },
            {
                "internalType": "address",
                "name": "to",
                "type": "address"
            },
            {
                "internalType": "uint256[]",
                "name": "ids",
                "type": "uint256[]"
            },
            {
                "internalType": "uint256[]",
                "name": "amounts",
                "type": "uint256[]"
            },
            {
                "internalType": "bytes",
                "name": "data",
                "type": "bytes"
            }
        ],
        "name": "safeBatchTransferFrom",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
    }
]`
