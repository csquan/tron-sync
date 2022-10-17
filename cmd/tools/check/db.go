package main

import (
	"log"

	"github.com/go-xorm/xorm"
	"github.com/starslabhq/chainmonitor/output/mysqldb"
)

type DataDB struct {
	Block       *mysqldb.Block
	Txs         []*mysqldb.TxDB
	TxErc20s    map[string][]*mysqldb.TxErc20
	TxInternals map[string][]*mysqldb.TxInternal
	TxLogs      map[string][]*mysqldb.TxLog
	Contracts   []*mysqldb.Contract
}

type DB struct {
	engine *xorm.Engine
}

func (db *DB) getBlock(num uint64) *mysqldb.Block {
	b := &mysqldb.Block{}
	_, err := db.engine.Table("block").Where("num = ? and state =?", num, 0).Get(b)
	if b.Id == 0 {
		log.Fatalf("get block empty num:%d", num)
	}
	if err != nil {
		log.Fatalf("get block err:%v,num:%d", err, num)
	}
	return b
}

func (db *DB) getTxs(blockID int64) []*mysqldb.TxDB {
	txs := make([]*mysqldb.TxDB, 0)
	err := db.engine.Table("tx").Where("block_id = ? and block_state = ?", blockID, 0).Find(&txs)
	if err != nil {
		log.Fatalf("get txs err:%v,bid:%d", err, blockID)
	}
	return txs
}

func (db *DB) getTxErc20s(blockID int64) map[string][]*mysqldb.TxErc20 {
	txs := make([]*mysqldb.TxErc20, 0)
	err := db.engine.Table("tx_erc20").Where("block_id = ? and block_state = ?", blockID, 0).Find(&txs)
	if err != nil {
		log.Fatalf("get txsErc20 err:%v,bid:%d", err, blockID)
	}
	log.Printf("erc20 size:%d", len(txs))
	ret := make(map[string][]*mysqldb.TxErc20)
	for _, tx := range txs {
		if v, ok := ret[tx.Hash]; ok {
			v = append(v, tx)
			ret[tx.Hash] = v
		} else {
			v = make([]*mysqldb.TxErc20, 0)
			v = append(v, tx)
			ret[tx.Hash] = v
		}
	}
	return ret
}

func (db *DB) getTxLogs(blockID int64) map[string][]*mysqldb.TxLog {
	txs := make([]*mysqldb.TxLog, 0)
	err := db.engine.Table("tx_log").Where("block_id = ? and block_state = ?", blockID, 0).Find(&txs)
	if err != nil {
		log.Fatalf("get txLogs err:%v,bid:%d", err, blockID)
	}
	ret := make(map[string][]*mysqldb.TxLog)
	for _, tx := range txs {
		if v, ok := ret[tx.Hash]; ok {
			v = append(v, tx)
			ret[tx.Hash] = v
		} else {
			v = make([]*mysqldb.TxLog, 0)
			v = append(v, tx)
			ret[tx.Hash] = v
		}
	}
	return ret
}

func (db *DB) getTxInternals(blockID int64) map[string][]*mysqldb.TxInternal {
	txs := make([]*mysqldb.TxInternal, 0)
	err := db.engine.Table("tx_internal").Where("block_id = ? and block_state = ?", blockID, 0).Find(&txs)
	if err != nil {
		log.Fatalf("get txInternals err:%v,bid:%d", err, blockID)
	}
	ret := make(map[string][]*mysqldb.TxInternal)
	for _, tx := range txs {
		if v, ok := ret[tx.TxHash]; ok {
			v = append(v, tx)
			ret[tx.TxHash] = v
		} else {
			v = make([]*mysqldb.TxInternal, 0)
			v = append(v, tx)
			ret[tx.TxHash] = v
		}
	}
	return ret
}

func (db *DB) getContracts(blockID int64) []*mysqldb.Contract {
	contracts := make([]*mysqldb.Contract, 0)
	err := db.engine.Table("contract").Where("block_id = ? and block_state = ?", blockID, 0).Find(&contracts)
	if err != nil {
		log.Fatalf("get contract err:%v,bid:%d", err, blockID)
	}
	return contracts
}

func (db *DB) GetData(num uint64) *DataDB {
	block := db.getBlock(num)
	txs := db.getTxs(block.Id)
	txEr20s := db.getTxErc20s(block.Id)
	txLogs := db.getTxLogs(block.Id)
	txInternals := db.getTxInternals(block.Id)
	contracs := db.getContracts(block.Id)
	ret := &DataDB{
		Block:       block,
		Txs:         txs,
		TxErc20s:    txEr20s,
		TxLogs:      txLogs,
		TxInternals: txInternals,
		Contracts:   contracs,
	}
	return ret
}
