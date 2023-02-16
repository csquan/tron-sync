package mysqldb

import (
	"fmt"

	"xorm.io/xorm"
)

func (db *MysqlDB) UpdateAsyncTaskNumByName(session xorm.Interface, name string, num uint64) error {
	_, err := session.Exec("update async_task set num = ? where name = ?", num, name)
	return err
}

func (db *MysqlDB) DeleteErc20InfoByAddr(session xorm.Interface, addr string) error {
	_, err := session.Exec("delete from erc20_info where addr = ?", addr)
	return err
}

func (db *MysqlDB) SaveBlock(session xorm.Interface, block *Block) (blkId int64, err error) {
	exist, err := db.isBlockExist(session, block.Number, Block_ok)
	if err != nil {
		err = fmt.Errorf("query db block exist err:%v, blk num:%v", err, block.Number)
		return
	}

	if exist {
		err = fmt.Errorf("block exist err:%v, blk num:%v", err, block.Number)
		return
	}

	blkId, err = session.InsertOne(block)
	if err != nil {
		return
	}

	return
}

func (db *MysqlDB) SaveBlocks(session xorm.Interface, blocks []*Block) error {
	cnt := len(blocks)
	for i := 0; i < cnt; {
		end := i + db.sqlBatch
		if end > cnt {
			end = cnt
		}
		_, err := session.Table("block").Insert(blocks[i:end])
		if err != nil {
			return fmt.Errorf("err:%v,range %d:%d", err, i, end)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) SaveInternalTxs(session xorm.Interface, txs []*TxInternal) error {
	err := db.insertTxInternals(session, txs)
	if err != nil {
		return fmt.Errorf("db insert internal txs err:%v", err)
	}

	return nil
}

func (db *MysqlDB) SaveTxs(session xorm.Interface, txs []*TxDB) error {
	err := db.insertTxs(session, txs)
	if err != nil {
		return fmt.Errorf("db insert txs err:%v", err)
	}

	return nil
}

func (db *MysqlDB) SaveLogs(session xorm.Interface, txLogs []*TxLog) error {
	err := db.insertTxlogs(session, txLogs)
	if err != nil {
		return fmt.Errorf("db insert tx_logs err:%v", err)
	}

	return nil
}

func (db *MysqlDB) SaveContracts(session xorm.Interface, contracts []*Contract) error {
	err := db.insertContractsV2(session, contracts)
	if err != nil {
		return fmt.Errorf("db insert contracts err:%v", err)
	}

	return nil
}

func (db *MysqlDB) SaveBalances(balances []*Balance) error {
	return db.insertBalances(db.engine, balances, db.sqlBatch)
}
func (db *MysqlDB) SaveRevertBalances(s xorm.Interface, balances []*Balance) error {
	return db.insertBalances(s, balances, db.sqlBatch)
}

func (db *MysqlDB) SaveTxErc20s(s xorm.Interface, txErc20s []*TxErc20) error {
	err := db.insertTxErc20s(s, txErc20s)
	if err != nil {
		return fmt.Errorf("db insert tx_erc20s err:%v", err)
	}
	return nil
}

func (db *MysqlDB) SaveTxErc721s(s xorm.Interface, txErc721s []*TxErc721) error {
	err := db.insertTxErc721s(s, txErc721s)
	if err != nil {
		return fmt.Errorf("db insert tx_erc721s err:%v", err)
	}
	return nil
}

func (db *MysqlDB) SaveTxErc1155s(s xorm.Interface, txErc1155s []*TxErc1155) error {
	cnt := len(txErc1155s)
	for i := 0; i < cnt; {
		end := i + db.sqlBatch
		if end > cnt {
			end = cnt
		}
		_, err := s.Table("tx_erc1155").Insert(txErc1155s[i:end])
		if err != nil {
			return fmt.Errorf("insert tx_erc1155 err:%v,range %d:%d", err, i, end)
		}
		i = end
	}
	return nil
}

func (db *MysqlDB) SaveErc20Infos(s xorm.Interface, infos []*Erc20Info) error {
	return db.insertErc20InfosV2(s, infos, db.sqlBatch)
}

func (db *MysqlDB) SaveErc721Infos(s xorm.Interface, infos []*Erc721Info) error {
	return db.insertErc721Infos(s, infos, db.sqlBatch)
}

func (db *MysqlDB) SaveErc721Tokens(s xorm.Interface, tokens []*Erc721Token) error {
	return db.insertErc721Tokens(s, tokens, db.sqlBatch)
}

func (db *MysqlDB) SaveErc20Balances(s xorm.Interface, bs []*BalanceErc20) error {
	return db.insertBalancesErc20(s, bs, db.sqlBatch)
}

func (db *MysqlDB) SaveErc1155Balances(s xorm.Interface, bs []*BalanceErc1155) error {
	return db.insertBalancesErc1155(s, bs, db.sqlBatch)
}
func (db *MysqlDB) SaveTokenPairs(s xorm.Interface, pairs []*TokenPair) error {
	return db.insertTokenPairs(s, pairs)
}

func (db *MysqlDB) UpdateTokenPairsReserve(s xorm.Interface, pairs []*TokenPair) error {
	return db.updateTokenPairsReserve(s, pairs)
}

func (db *MysqlDB) UpdateMonitorHash(done int, hash string, chain string) error {
	_, err := db.engine.Exec("update t_monitor_hash set f_push = ? where f_hash = ? and f_chain = ?", done, hash, chain)
	return err
}
