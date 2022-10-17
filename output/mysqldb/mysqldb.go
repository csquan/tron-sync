package mysqldb

import (
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"github.com/starslabhq/chainmonitor/mtypes"
	"xorm.io/xorm"
	"xorm.io/xorm/log"
)

type MysqlDB struct {
	sqlBatch int
	engine   *xorm.Engine
}

func NewMysqlDb(datasource string, sqlBatch int) (*MysqlDB, error) {
	//"test:123@/test?charset=utf8"
	engine, err := xorm.NewEngine("mysql", datasource)
	if err != nil {
		return nil, err
	}
	engine.ShowSQL(false)
	engine.SetLogLevel(log.LOG_DEBUG)

	db := &MysqlDB{
		sqlBatch: sqlBatch,
		engine:   engine,
	}
	return db, nil
}

func (db *MysqlDB) insertBalancesErc20(s xorm.Interface, bs []*BalanceErc20, batch int) error {
	size := len(bs)
	for i := 0; i < size; {
		end := i + batch
		if end > size {
			end = size
		}
		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := `(?,?,?,?,?)`
			values = append(values, value)
		}

		v := strings.Join(values, ",")
		sql := `INSERT INTO balance_erc20(addr,contract_addr,balance,height,balance_origin) VALUES` + v + `ON DUPLICATE KEY UPDATE addr = VALUES(addr),contract_addr= VALUES(contract_addr),balance = VALUES(balance),height=VALUES(height),balance_origin=VALUES(balance_origin)`
		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			param = append(param, bs[j].Addr)
			param = append(param, bs[j].ContractAddr)
			if bs[j].Balance == "" {
				bs[j].Balance = "0"
			}
			param = append(param, bs[j].Balance)
			param = append(param, bs[j].Height)
			param = append(param, bs[j].BalanceOrigin)
		}
		// _, err = s.Exec(sql, param[0], param[1], param[2], param[3], param[4], param[5])
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert balances err:%v,params:%v", err, param)
		}
		i = end
	}

	return nil
}

func (db *MysqlDB) insertBalancesErc1155(s xorm.Interface, bs []*BalanceErc1155, batch int) error {
	size := len(bs)
	for i := 0; i < size; {
		end := i + batch
		if end > size {
			end = size
		}
		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := `(?,?,?,?,?,?)`
			values = append(values, value)
		}

		v := strings.Join(values, ",")
		sql := `INSERT INTO balance_erc1155(addr,contract_addr,token_id,balance,height,balance_origin) VALUES` + v + `ON DUPLICATE KEY UPDATE addr = VALUES(addr),contract_addr= VALUES(contract_addr),token_id= VALUES(token_id),balance = VALUES(balance),height=VALUES(height),balance_origin=VALUES(balance_origin)`
		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			param = append(param, bs[j].Addr)
			param = append(param, bs[j].ContractAddr)
			if bs[j].TokenID == "" {
				bs[j].TokenID = "0"
			}
			param = append(param, bs[j].TokenID)
			if bs[j].Balance == "" {
				bs[j].Balance = "0"
			}
			param = append(param, bs[j].Balance)
			param = append(param, bs[j].Height)
			param = append(param, bs[j].BalanceOrigin)
		}
		// _, err = s.Exec(sql, param[0], param[1], param[2], param[3], param[4], param[5])
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert balances err:%v,params:%v", err, param)
		}
		i = end
	}

	return nil
}

func (db *MysqlDB) insertBalances(s xorm.Interface, bs []*Balance, batch int) error {
	size := len(bs)
	for i := 0; i < size; {
		end := i + batch
		if end > size {
			end = size
		}

		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := `(?,?,?,?)`
			values = append(values, value)
		}

		v := strings.Join(values, ",")
		sql := `INSERT INTO balance(addr,balance,height,balance_origin) VALUES` + v + `ON DUPLICATE KEY UPDATE addr = VALUES(addr),balance = VALUES(balance),height=VALUES(height),balance_origin=VALUES(balance_origin)`
		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			param = append(param, bs[j].Addr)
			// param = append(param, bs[j].ContractAddr)
			if bs[j].Balance == "" {
				bs[j].Balance = "0"
			}
			param = append(param, bs[j].Balance)
			param = append(param, bs[j].Height)
			param = append(param, bs[j].BalanceOrigin)
			logrus.Tracef("db balance a:%s,b:%s,h:%d", bs[j].Addr, bs[j].Balance, bs[j].Height)
		}
		// _, err = s.Exec(sql, param[0], param[1], param[2], param[3], param[4], param[5])
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert balances err:%v,params:%v", err, param)
		}
		i = end
	}

	return nil
}

func (db *MysqlDB) insertTxs(s xorm.Interface, txs []*TxDB) error {
	txsCnt := len(txs)
	for i := 0; i < txsCnt; {
		end := i + db.sqlBatch
		if end > txsCnt {
			end = txsCnt
		}
		_, err := s.Table("tx").Insert(txs[i:end])
		if err != nil {
			return fmt.Errorf("err:%v,range %d:%d", err, i, end)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) insertErc20InfosV2(s xorm.Interface, infos []*Erc20Info, batch int) error {
	size := len(infos)
	for i := 0; i < size; {
		end := i + batch
		if end > size {
			end = size
		}
		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := `(?,?,?,?,?,?)`
			values = append(values, value)
		}
		v := strings.Join(values, ",")

		sql := `INSERT IGNORE INTO erc20_info(addr,name,symbol,decimals,total_supply,total_supply_origin) VALUES` + v
		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			param = append(param, infos[j].Addr)
			param = append(param, infos[j].Name)
			param = append(param, infos[j].Symbol)
			param = append(param, infos[j].Decimals)
			param = append(param, infos[j].TotoalSupply)
			param = append(param, infos[j].TotoalSupplyOrigin)
		}
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert erc20info err:%v,params:%v", err, param)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) insertErc721Infos(s xorm.Interface, infos []*Erc721Info, batch int) error {
	size := len(infos)
	for i := 0; i < size; {
		end := i + batch
		if end > size {
			end = size
		}
		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := `(?,?,?,?,?)`
			values = append(values, value)
		}
		v := strings.Join(values, ",")

		sql := `INSERT IGNORE INTO erc721_info(addr,name,symbol,total_supply,total_supply_origin) VALUES` + v
		//logrus.Infof("sql : %s", sql)
		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			param = append(param, infos[j].Addr)
			param = append(param, infos[j].Name)
			param = append(param, infos[j].Symbol)
			param = append(param, infos[j].TotoalSupply)
			param = append(param, infos[j].TotoalSupplyOrigin)
		}
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert erc721info err:%v,params:%v", err, param)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) insertErc721Tokens(s xorm.Interface, tokens []*Erc721Token, batch int) error {
	size := len(tokens)
	for i := 0; i < size; {
		end := i + batch
		if end > size {
			end = size
		}
		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := `(?,?,?,?)`
			values = append(values, value)
		}
		v := strings.Join(values, ",")

		sql := `INSERT INTO token_erc721(contract_addr,token_id,owner_addr,height) VALUES` + v + ` ON DUPLICATE KEY UPDATE owner_addr=VALUES(owner_addr),height=VALUES(height)`

		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			param = append(param, tokens[j].ContractAddr)
			param = append(param, tokens[j].TokenId)
			param = append(param, tokens[j].OwnerAddr)
			// param = append(param, tokens[j].TokenUri)
			// param = append(param, tokens[j].TokenMetadata)
			param = append(param, tokens[j].Height)
		}
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert erc721token err:%v,params:%v", err, param)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) insertTxErc20s(s xorm.Interface, txErc20s []*TxErc20) error {
	cnt := len(txErc20s)
	for i := 0; i < cnt; {
		end := i + db.sqlBatch
		if end > cnt {
			end = cnt
		}
		_, err := s.Table("tx_erc20").Insert(txErc20s[i:end])
		if err != nil {
			return fmt.Errorf("err:%v,range %d:%d", err, i, end)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) insertTxErc721s(s xorm.Interface, txErc721s []*TxErc721) error {
	cnt := len(txErc721s)
	for i := 0; i < cnt; {
		end := i + db.sqlBatch
		if end > cnt {
			end = cnt
		}
		_, err := s.Table("tx_erc721").Insert(txErc721s[i:end])
		if err != nil {
			return fmt.Errorf("insert tx_erc721 err:%v,range %d:%d", err, i, end)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) insertTxlogs(s xorm.Interface, txLogs []*TxLog) error {
	cnt := len(txLogs)
	for i := 0; i < cnt; {
		end := i + db.sqlBatch
		if end > cnt {
			end = cnt
		}
		_, err := s.Table("tx_log").Insert(txLogs[i:end])
		if err != nil {
			return fmt.Errorf("err:%v,range %d:%d", err, i, end)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) insertTxInternals(s xorm.Interface, txInternals []*TxInternal) error {
	cnt := len(txInternals)
	for i := 0; i < cnt; {
		end := i + db.sqlBatch
		if end > cnt {
			end = cnt
		}
		_, err := s.Table("tx_internal").Insert(txInternals[i:end])
		if err != nil {
			return fmt.Errorf("err:%v,range %d:%d", err, i, end)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) insertContractsV2(s xorm.Interface, contracts []*Contract) error {
	size := len(contracts)
	for i := 0; i < size; {
		end := i + db.sqlBatch
		if end > size {
			end = size
		}
		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := `(?,?,?,?,?,?)`
			values = append(values, value)
		}
		v := strings.Join(values, ",")
		sql := `INSERT IGNORE INTO contract(tx_hash,addr,creator_addr,exec_status,block_num,block_state) VALUES` + v
		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			param = append(param, contracts[j].TxHash)
			param = append(param, contracts[j].Addr)
			param = append(param, contracts[j].CreatorAddr)
			param = append(param, contracts[j].ExecStatus)
			param = append(param, contracts[j].BlockNum)
			param = append(param, contracts[j].BlockState)
		}
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert contract err:%v,params:%v", err, param)
		}

		i = end
	}
	return nil
}

func (db *MysqlDB) isBlockExist(s xorm.Interface, num uint64, state int) (bool, error) {
	ok, err := s.Table("block").Where("num = ? and state = ?", num, state).Exist()
	return ok, err
}

func (db *MysqlDB) InsertBlock(s xorm.Interface, block *mtypes.Block) (int64, error) {
	b := ConvertInBlock(block)
	_, err := s.InsertOne(b)
	return b.Id, err
}

func (db *MysqlDB) SetBlockOK(id int64) error {
	return db.updateBlockSate(id, Block_ok)
}
func (db *MysqlDB) SetBlockRevert(id int64) error {
	return db.updateBlockSate(id, Block_revert)
}

func (db *MysqlDB) updateBlockSate(id int64, state uint8) error {
	b := &Block{
		State: state,
	}
	_, err := db.engine.ID(id).Update(b)
	for err != nil {
		logrus.Infof("update block state err:%v id:%d,state:%d", err, id, state)
		time.Sleep(50 * time.Millisecond)
		_, err = db.engine.ID(id).Update(b)
	}
	return err
}

func (db *MysqlDB) UpdateBlockSate(bNum uint64, state int) error {
	s := db.engine.NewSession()
	defer s.Close()
	err := s.Begin()
	if err != nil {
		return err
	}
	_, err = s.Exec("update block set state = ? where num = ? and state = ?", state, bNum, Block_ok)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("update block state rollback err:%v,bNum:%d", err1, bNum)
		}
		return err
	}
	_, err = s.Exec("update tx set block_state = ? where block_num = ? and block_state = ?", state, bNum, Block_ok)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("update tx state rollback err:%v,bNum:%d", err1, bNum)
		}
		return err
	}
	_, err = s.Exec("update tx_log set block_state = ? where block_num = ? and block_state = ?", state, bNum, Block_ok)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("update tx_log state rollback err:%v,bNum:%d", err1, bNum)
		}

		return err
	}

	_, err = s.Exec("update tx_internal set block_state = ? where block_num = ? and block_state = ?", state, bNum, Block_ok)
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("update tx_internal state rollback err:%v,bNum:%d", err1, bNum)
		}
		return err
	}
	err = s.Commit()
	if err != nil {
		err1 := s.Rollback()
		if err1 != nil {
			return fmt.Errorf("commit fail rollback err:%v,bNum:%d", err1, bNum)
		}
		return fmt.Errorf("base data revert commit err:%v", err)
	}
	return nil
}

func (db *MysqlDB) insertTokenPairs(s xorm.Interface, pairs []*TokenPair) error {
	size := len(pairs)
	for i := 0; i < size; {
		end := i + db.sqlBatch
		if end > size {
			end = size
		}
		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := "(?,?,?,?,?,?,?)"
			values = append(values, value)
		}
		v := strings.Join(values, ",")
		sql := `INSERT INTO token_pair(addr,token0,token1,reserve0,reserve1,pair_name,block_num) VALUES` +
			v + `ON DUPLICATE KEY UPDATE addr = VALUES(addr),token0 = VALUES(token0),token1 = VALUES(token1),reserve0 = VALUES(reserve0),reserve1 = VALUES(reserve1),pair_name = VALUES(pair_name),block_num= VALUES(block_num)`
		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			pair := pairs[j]
			temp := []interface{}{
				pair.Addr,
				pair.Token0,
				pair.Token1,
				pair.Reserve0,
				pair.Reserve1,
				pair.PairName,
				pair.BlockNum,
			}
			param = append(param, temp...)
		}
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert token pairs err:%v,params:%v", err, param)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) updateTokenPairsReserve(s xorm.Interface, pairs []*TokenPair) error {
	size := len(pairs)
	for i := 0; i < size; {
		end := i + db.sqlBatch
		if end > size {
			end = size
		}
		var values []string
		cnt := end - i
		for j := 0; j < cnt; j++ {
			value := "(?,?,?,?)"
			values = append(values, value)
		}
		v := strings.Join(values, ",")
		sql := `INSERT INTO token_pair(addr,reserve0,reserve1,block_num) VALUES` +
			v + `ON DUPLICATE KEY UPDATE addr = VALUES(addr),reserve0 = VALUES(reserve0),reserve1 = VALUES(reserve1),block_num= VALUES(block_num)`
		var param = make([]interface{}, 0)
		param = append(param, sql)
		for j := i; j < end; j++ {
			pair := pairs[j]
			temp := []interface{}{
				pair.Addr,
				pair.Reserve0,
				pair.Reserve1,
				pair.BlockNum,
			}
			param = append(param, temp...)
		}
		_, err := s.Exec(param...)
		if err != nil {
			return fmt.Errorf("insert token pairs err:%v,params:%v", err, param)
		}
		i = end
	}
	return nil
}

// RenewInternalTransactions delete old internal tx and contracts, than insert the new records.
func (db *MysqlDB) RenewInternalTransactions(blockNum uint64, internals []*TxInternal, contracts []*Contract) error {
	s := db.engine.NewSession()
	defer s.Close()
	err := s.Begin()
	if err != nil {
		logrus.Error(err)
		return err
	}
	_, err = s.Table("tx_internal").Where("block_num = ?", blockNum).Delete(&TxInternal{})
	if err != nil {
		logrus.Error(err)
		if err := s.Rollback(); err != nil {
			logrus.Errorf("rollback error, block number: %v, err: %v", blockNum, err)
		}
		return err
	}
	_, err = s.Table("contract").Where("block_num = ?", blockNum).Delete(&Contract{})
	if err != nil {
		logrus.Error(err)
		if err := s.Rollback(); err != nil {
			logrus.Errorf("rollback error, block number: %v, err: %v", blockNum, err)
		}
		return err
	}
	if err := s.Commit(); err != nil {
		logrus.Error(err)
		if err := s.Rollback(); err != nil {
			logrus.Errorf("rollback error, block number: %v, err: %v", blockNum, err)
		}
		return err
	}

	err = s.Begin()
	if err != nil {
		logrus.Error(err)
		return err
	}
	if err := db.insertTxInternals(s, internals); err != nil {
		logrus.Error(err)
		if err := s.Rollback(); err != nil {
			logrus.Errorf("rollback error, block number: %v, err: %v", blockNum, err)
		}
		return err
	}
	if err := db.insertContractsV2(s, contracts); err != nil {
		logrus.Error(err)
		if err := s.Rollback(); err != nil {
			logrus.Errorf("rollback error, block number: %v, err: %v", blockNum, err)
		}
		return err
	}
	if err := s.Commit(); err != nil {
		logrus.Error(err)
		if err := s.Rollback(); err != nil {
			logrus.Errorf("rollback error, block number: %v, err: %v", blockNum, err)
		}
		return err
	}
	return nil
}
