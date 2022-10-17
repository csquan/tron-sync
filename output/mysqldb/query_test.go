package mysqldb

import (
	"testing"

	"gotest.tools/assert"
)

func prepareDB(t *testing.T) (db *MysqlDB) {
	//var err error
	//db, err = NewMysqlDb("", 10)
	//assert.NilError(t, err, "create db error")
	//
	////use sqlite for unit test
	//engine, err := xorm.NewEngine("sqlite3", "../../data/ut.sqlite")
	//assert.NilError(t, err, "sqlite engine create failed")
	//engine.ShowSQL(true)
	//engine.Logger().SetLevel(core.LOG_DEBUG)
	//assert.NilError(t, err, "sqlite engine create failed")
	//db.engine = engine

	return dbtest
}

func TestGetBlockByRangeWithFullBlock(t *testing.T) {
	db := prepareDB(t)

	blks, err := db.getBlocksByRangeAndState(9305290, 9305291, DefaultFullBlockFilter, 0)
	assert.NilError(t, err, "query block error")
	assert.Equal(t, len(blks), 2, "blocks length not as expected")

	for _, blk := range blks {
		assert.Equal(t, len(blk.Txs), blk.TxCnt, "block txs count not as expected")

		logCount := 0
		maxIndex := 0
		for _, tx := range blk.Txs {
			logCount += len(tx.EventLogs)
			if len(tx.EventLogs) > 0 {
				maxIndex = int(tx.EventLogs[len(tx.EventLogs)-1].Index)
			}
		}

		assert.Equal(t, logCount, maxIndex+1, "block txs count not as expected")
	}
}

func TestGetBlockByRangeWithForErc20(t *testing.T) {

	db := prepareDB(t)

	blks, err := db.getBlocksByRangeAndState(9305290, 9305291, ERC20Filter, 0)
	assert.NilError(t, err, "query block error")
	assert.Equal(t, len(blks), 2, "blocks length not as expected")

	logCount := 0
	for _, blk := range blks {
		assert.Equal(t, len(blk.Txs), blk.TxCnt, "block txs count not as expected")

		for _, tx := range blk.Txs {
			logCount += len(tx.EventLogs)
		}
	}
	assert.Equal(t, logCount, 71, "block txs count not as expected")
}

func TestGetBlockByRangeWithForErc721(t *testing.T) {

	db := prepareDB(t)

	blks, err := db.getBlocksByRangeAndState(9305290, 9305300, ERC721Filter, 0)
	assert.NilError(t, err, "query block error")

	logCount := 0
	for _, blk := range blks {
		//assert.Equal(t, len(blk.Txs), blk.TxCnt, "block txs count not as expected")

		for _, tx := range blk.Txs {
			logCount += len(tx.EventLogs)
		}
	}
	assert.Equal(t, logCount, 17, "block txs count not as expected")
}

func TestGetCurrentBlock(t *testing.T) {
	b, err := dbtest.GetCurrentBlock()
	t.Logf("cur block:%v,err:%v", b, err)
	assert.NilError(t, err, "query error")
	assert.Equal(t, b.Number, 9305299, "query current block error")
}

func TestBlockExist(t *testing.T) {
	s := dbtest.engine.NewSession()
	ok, err := dbtest.isBlockExist(s, 9305290, 0)
	assert.NilError(t, err, "query error")
	assert.Assert(t, ok, "query exist block failed")
}
