package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/starslabhq/chainmonitor/output/mysqldb"
	"gotest.tools/assert"
)

var (
	originDB *mysqldb.MysqlDB
	curDB    *mysqldb.MysqlDB
)

func init() {
	var oriDS = "root:123456sj@tcp(127.0.0.1:3306)/chain_bsc?charset=utf8mb4"
	var curDS = "root:123456sj@tcp(127.0.0.1:3306)/bsc_master?charset=utf8mb4"
	var err error
	originDB, err = mysqldb.NewMysqlDb(oriDS, 5)
	if err != nil {
		fmt.Printf("init db err:%v", err)
		os.Exit(1)
	}
	curDB, err = mysqldb.NewMysqlDb(curDS, 5)
	if err != nil {
		fmt.Printf("init db err:%v", err)
		os.Exit(1)
	}
}

func TestBlock(t *testing.T) {
	ori := getBlocks(originDB)
	cur := getBlocks(curDB)
	assert.DeepEqual(t, ori, cur)
}

func TestLog(t *testing.T) {
	ori := getLogs(originDB)
	cur := getLogs(curDB)
	assert.DeepEqual(t, ori, cur)
}

func TestTx(t *testing.T) {
	ori := getTxs(originDB)
	cur := getTxs(curDB)
	assert.DeepEqual(t, ori, cur)
}

func TestTxErc20(t *testing.T) {
	ori := getTxErc20(originDB)
	cur := getTxErc20(curDB)
	assert.DeepEqual(t, ori, cur)
}

func TestTxErc721(t *testing.T) {
	ori := getTxErc721(originDB)
	cur := getTxErc721(curDB)
	assert.DeepEqual(t, ori, cur)
}

func TestContract(t *testing.T) {
	ori := getContracts(originDB)
	cur := getContracts(curDB)
	assert.DeepEqual(t, ori, cur)
}

func TestBalance(t *testing.T) {
	ori := getBalance(originDB)
	cur := getBalance(curDB)
	assert.DeepEqual(t, ori, cur)
}

func TestBalanceErc20(t *testing.T) {
	ori := getBalanceErc20(originDB)
	cur := getBalanceErc20(curDB)
	assert.DeepEqual(t, ori, cur)
}

func getContracts(db *mysqldb.MysqlDB) []*mysqldb.Contract {
	var contract []*mysqldb.Contract
	err := db.GetEngine().Asc("addr").Find(&contract)
	if err != nil {
		exit(err)
	}
	return contract
}

func getBalanceErc20(db *mysqldb.MysqlDB) []*mysqldb.BalanceErc20 {
	var balanceErc20 []*mysqldb.BalanceErc20
	err := db.GetEngine().Asc("addr", "contract_addr").Find(&balanceErc20)
	if err != nil {
		exit(err)
	}
	for _, l := range balanceErc20 {
		l.Id = 0
		l.Balance = "0"
		l.Height = 0
	}
	return balanceErc20
}

func getBalance(db *mysqldb.MysqlDB) []*mysqldb.Balance {
	var balance []*mysqldb.Balance
	err := db.GetEngine().Asc("addr").Find(&balance)
	if err != nil {
		exit(err)
	}
	for _, l := range balance {
		l.Id = 0
		l.Balance = "0"
		l.Height = 0
	}
	return balance
}

func getLogs(db *mysqldb.MysqlDB) []*mysqldb.TxLog {
	var logs []*mysqldb.TxLog
	err := db.GetEngine().Asc("block_num", "log_index").Find(&logs)
	if err != nil {
		exit(err)
	}
	for _, l := range logs {
		l.Id = 0
	}
	return logs
}

func getBlocks(db *mysqldb.MysqlDB) []*mysqldb.Block {
	var blocks []*mysqldb.Block
	err := db.GetEngine().Asc("num").Find(&blocks)
	if err != nil {
		exit(err)
	}
	for _, b := range blocks {
		b.Id = 0
	}
	return blocks
}

func getTxs(db *mysqldb.MysqlDB) []*mysqldb.TxDB {
	var txs []*mysqldb.TxDB
	err := db.GetEngine().Asc("tx_hash").Find(&txs)
	if err != nil {
		exit(err)
	}
	for _, t := range txs {
		t.Id = 0
	}
	return txs
}

func getTxErc20(db *mysqldb.MysqlDB) []*mysqldb.TxErc20 {
	var txs []*mysqldb.TxErc20
	err := db.GetEngine().Asc("tx_hash", "log_index").Find(&txs)
	if err != nil {
		exit(err)
	}
	for _, t := range txs {
		t.Id = 0
	}
	return txs
}

func getTxErc721(db *mysqldb.MysqlDB) []*mysqldb.TxErc721 {
	var txs []*mysqldb.TxErc721
	err := db.GetEngine().Asc("tx_hash", "log_index").Find(&txs)
	if err != nil {
		exit(err)
	}
	for _, t := range txs {
		t.Id = 0
	}
	return txs
}

func exit(err error) {
	fmt.Printf("init db err:%v", err)
	os.Exit(1)
}
