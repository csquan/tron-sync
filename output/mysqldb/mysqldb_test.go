package mysqldb

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"gotest.tools/assert"
)

var dbtest *MysqlDB

func init() {
	file, err := os.Open("../../sql/test_db.json")
	if err != nil {
		log.Printf("open test conf error: %v", err)
		return
	}
	defer file.Close()

	conf := make(map[string]string, 0)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&conf)
	if err != nil {
		log.Printf("load test conf error:%v", err)
		return
	}

	host := conf["db"]
	dbtest, err = NewMysqlDb(host, 5)
	if err != nil {
		log.Printf("open db error:%v", err)
		return
	}

	_, err = dbtest.engine.ImportFile("../../sql/dump.sql")
	if err != nil {
		log.Printf("import dump file error:%v", err)
	}
	dbtest.engine.ShowSQL(true)
}

func TestInsertBatch(t *testing.T) {
	bs := []*Balance{
		&Balance{
			Addr:    "1",
			Balance: "126",
		},
		&Balance{
			Addr:    "2",
			Balance: "125",
		},
		&Balance{
			Addr:    "3",
			Balance: "125",
		},
		&Balance{
			Addr:    "3",
			Balance: "125",
		},
		&Balance{
			Addr:    "5",
			Balance: "125",
		},
	}
	db := prepareDB(t)
	s := db.engine.NewSession()
	err := s.Begin()
	assert.NilError(t, err, "begin session error")

	orgCount, err := s.Count(new(Balance))
	assert.NilError(t, err, "query balance record count error")
	err = db.insertBalances(s, bs, 2)

	assert.NilError(t, err, "batch insert balance error")
	newCount, err := s.Count(new(Balance))
	assert.NilError(t, err, "query balance record count error")

	// 有一个重复的
	assert.Equal(t, orgCount+int64(len(bs)-1), newCount, "batch insert balance count error")

	err = s.Rollback()
	assert.NilError(t, err, "rollback error")

	revertCount, err := s.Count(new(Balance))
	assert.NilError(t, err, "rollback count error")

	assert.Equal(t, orgCount, revertCount, "rollback balance count error")
}

func TestInsertBatchErc20(t *testing.T) {
	bs := []*BalanceErc20{
		&BalanceErc20{
			Addr:         "1",
			Balance:      "126",
			ContractAddr: "contract",
		},
		&BalanceErc20{
			Addr:         "2",
			Balance:      "125",
			ContractAddr: "contract",
		},
		&BalanceErc20{
			Addr:         "3",
			Balance:      "125",
			ContractAddr: "contract",
		},
		&BalanceErc20{
			Addr:         "3",
			Balance:      "125",
			ContractAddr: "contract",
		},
		&BalanceErc20{
			Addr:         "5",
			Balance:      "125",
			ContractAddr: "contract",
		},
	}

	db := prepareDB(t)
	s := db.engine.NewSession()
	err := s.Begin()
	assert.NilError(t, err, "begin session error")

	orgCount, err := s.Count(new(Balance))
	assert.NilError(t, err, "query balance record count error")
	err = db.insertBalancesErc20(s, bs, 2)

	assert.NilError(t, err, "batch insert balance error")
	newCount, err := s.Count(new(Balance))
	assert.NilError(t, err, "query balance record count error")

	// 有一个重复的
	assert.Equal(t, orgCount+int64(len(bs)-1), newCount, "batch insert balance count error")

	err = s.Rollback()
	assert.NilError(t, err, "rollback error")

	revertCount, err := s.Count(new(Balance))
	assert.NilError(t, err, "rollback count error")

	assert.Equal(t, orgCount, revertCount, "rollback balance count error")
}

func TestInsertErc721Token(t *testing.T) {
	e := dbtest.GetEngine()
	err := dbtest.insertErc721Tokens(e, []*Erc721Token{
		&Erc721Token{
			ContractAddr:  "123",
			TokenId:       "1",
			OwnerAddr:     "owner",
			TokenUri:      "",
			TokenMetadata: "",
			Height:        11,
		},
	}, 1)
	if err != nil {
		t.Fatalf("insert erc721 err:%v", err)
	}
}
