package mysqldb

import (
	"fmt"
	"strings"

	"xorm.io/xorm"

	"github.com/chainmonitor/mtypes"
)

func (db *MysqlDB) GetSession() *xorm.Session {
	return db.engine.NewSession()
}

func (db *MysqlDB) GetEngine() *xorm.Engine {
	return db.engine
}

func (db *MysqlDB) GetCurrentBlock() (*Block, error) {
	block := &Block{}
	_, err := db.engine.SQL("select * from block where state = ? order by num desc limit 1", Block_ok).Get(block)
	return block, err
}

func (db *MysqlDB) GetBlockByNum(num uint64, filter *BlockFilter) (*mtypes.Block, error) {
	blks, err := db.GetBlocksByRange(num, num, filter)
	if err != nil {
		return nil, err
	}

	if len(blks) == 0 {
		return nil, nil
	}

	return blks[0], nil
}

func (db *MysqlDB) GetTaskByName(name string) (*AsyncTask, error) {
	task := &AsyncTask{}
	ok, err := db.engine.Where("name = ?", name).Get(task)
	if err != nil {
		return nil, err
	}
	if !ok {
		task = &AsyncTask{Name: name, Number: 0, EndNumber: 0}
		_, err := db.engine.InsertOne(task)
		if err != nil {
			return nil, err
		}
		return task, nil
	}

	return task, nil
}

func (db *MysqlDB) GetCurrentMainBlockNum() (uint64, error) {
	block := &Block{}
	ok, err := db.engine.SQL("select * from block where state = ? order by num desc limit 1", Block_ok).Get(block)

	if err != nil {
		return 0, err
	}

	if !ok {
		return 0, nil
	}

	return block.Number, nil
}
func (db *MysqlDB) GetBlocksByRange(start, end uint64, filter *BlockFilter) ([]*mtypes.Block, error) {
	return db.getBlocksByRangeAndState(start, end, filter, Block_ok)
}
func (db *MysqlDB) GetRevertBlocks(start, end uint64, filter *BlockFilter) ([]*mtypes.Block, error) {
	return db.getBlocksByRangeAndState(start, end, filter, Block_all)
}
func (db *MysqlDB) getBlocksByRangeAndState(start, end uint64, filter *BlockFilter, blockState int) ([]*mtypes.Block, error) {

	blocks := make([]*Block, 0)
	var err error
	if blockState == Block_all {
		err = db.engine.Where("num >= ? and num <= ?", start, end).Find(&blocks)
	} else {
		err = db.engine.Where("num >= ? and num <= ? and state = ?", start, end, blockState).Find(&blocks)
	}
	if err != nil {
		return nil, err
	}

	res, err := convertOutBlocks(blocks)
	if err != nil {
		return nil, err
	}

	if len(res) == 0 {
		return res, nil
	}

	outlogsMaps := make(map[string][]*mtypes.EventLog, 0)

	if filter.LogFilter != nil {
		logs := make([]*TxLog, 0)

		queries := []string{
			fmt.Sprintf("block_num >= %d", start),
			fmt.Sprintf("block_num <= %d", end),
		}
		if blockState != Block_all {
			queries = append(queries, fmt.Sprintf("block_state = %d", blockState))
		}
		queries, err = db.generateLogCondition(filter.LogFilter, queries)
		if err != nil {
			return nil, err
		}

		q := strings.Join(queries, " and ")

		err = db.engine.Where(q).OrderBy("block_num,log_index").Find(&logs)
		if err != nil {
			return nil, err
		}

		_, outlogsMaps, err = convertOutLogs(logs)
		if err != nil {
			return nil, err
		}
	}

	txs := make([]*TxDB, 0)
	if filter.LogFilter != nil && filter.TxFilter == nil {
		txHashes := make([]string, 0, len(outlogsMaps))
		for hash := range outlogsMaps {
			txHashes = append(txHashes, hash)
		}

		s := 0
		for s < len(txHashes) {
			e := s + 100
			if e > len(txHashes) {
				e = len(txHashes)
			}

			queries := []string{
				fmt.Sprintf("block_num >= %d", start),
				fmt.Sprintf("block_num <= %d", end),
			}
			if blockState != Block_all {
				queries = append(queries, fmt.Sprintf("block_state = %d", blockState))
			}

			queries = append(queries, fmt.Sprintf("tx_hash in ('%s')", strings.Join(txHashes[s:e], "','")))
			q := strings.Join(queries, " and ")

			sliceTxs := make([]*TxDB, 0)
			err = db.engine.Where(q).OrderBy("block_num,tx_index").Find(&sliceTxs)
			if err != nil {
				return nil, err
			}

			s = e
			txs = append(txs, sliceTxs...)
		}
	} else if filter != nil && filter.TxFilter != nil {
		queries := []string{
			fmt.Sprintf("block_num >= %d", start),
			fmt.Sprintf("block_num <= %d", end),
		}
		if blockState != Block_all {
			queries = append(queries, fmt.Sprintf("block_state = %d", blockState))
		}
		queries, err = db.generateTxCondition(filter.TxFilter, queries)
		if err != nil {
			return nil, err
		}

		q := strings.Join(queries, " and ")

		err = db.engine.Where(q).OrderBy("block_num,tx_index").Find(&txs)
		if err != nil {
			return nil, err
		}
	}

	//query related txs
	outTxs, outTxMaps, err := convertOutTxs(txs)
	if err != nil {
		return nil, err
	}

	for _, tx := range outTxs {
		if v, ok := outlogsMaps[tx.Hash]; ok {
			tx.EventLogs = v
		}
	}

	for _, blk := range res {
		if v, ok := outTxMaps[blk.Number]; ok {
			blk.Txs = v
		}
	}

	if filter != nil && filter.InternalTxFilter != nil {
		txs := make([]*TxInternal, 0)

		queries := []string{
			fmt.Sprintf("block_num >= %d", start),
			fmt.Sprintf("block_num <= %d", end),
		}
		if blockState != Block_all {
			queries = append(queries, fmt.Sprintf("block_state = %d", blockState))
		}
		queries, err = db.generateInternalTxCondition(filter.InternalTxFilter, queries)
		if err != nil {
			return nil, err
		}

		q := strings.Join(queries, " and ")

		err = db.engine.Where(q).Find(&txs)
		if err != nil {
			return nil, err
		}
		_, outInternalMaps, err := convertOutInternalTxs(txs)
		if err != nil {
			return nil, err
		}

		for _, blk := range res {
			if v, ok := outInternalMaps[blk.Number]; ok {
				blk.TxInternals = v
			}
		}
	}

	return res, nil
}

func (db *MysqlDB) generateInternalTxCondition(filter *InternalTxFilter, queries []string) ([]string, error) {
	if filter.From != nil {
		queries = append(queries, fmt.Sprintf(`from = '%s'`, *filter.From))
	}

	if filter.To != nil {
		queries = append(queries, fmt.Sprintf(`to = '%s'`, *filter.To))
	}

	if filter.Value != nil {
		queries = append(queries, fmt.Sprintf("value >= %s", filter.Value))
	}

	if filter.Success != nil {
		queries = append(queries, fmt.Sprintf("success = %v", *filter.Success))
	}

	return queries, nil
}

func (db *MysqlDB) generateTxCondition(filter *TxFilter, queries []string) ([]string, error) {
	if filter.From != nil {
		queries = append(queries, fmt.Sprintf(`from = '%s'`, *filter.From))
	}

	if filter.To != nil {
		queries = append(queries, fmt.Sprintf(`to = '%s'`, *filter.To))
	}

	if filter.Value != nil {
		queries = append(queries, fmt.Sprintf("value >= %s", filter.Value.String()))
	}

	return queries, nil
}

var AnyStringValue = "any string value"
var VoidStringValue = ""

func (db *MysqlDB) generateLogCondition(filter *LogFilter, queries []string) ([]string, error) {
	if filter.Contract != nil {
		queries = append(queries, fmt.Sprintf(`contract = '%s'`, *filter.Contract))
	}

	if filter.Topic0 != nil {
		if args := strings.Split(*filter.Topic0, ","); len(args) == 2 {
			queries = append(queries, fmt.Sprintf("topic0 in ('%s', '%s')", args[0], args[1]))
		} else if filter.Topic0 == &AnyStringValue {
			queries = append(queries, "topic0 != ''")
		} else {
			queries = append(queries, fmt.Sprintf(`topic0 = '%s'`, *filter.Topic0))
		}
	}

	if filter.Topic1 != nil {
		if filter.Topic1 == &AnyStringValue {
			queries = append(queries, "topic1 != ''")
		} else {
			queries = append(queries, fmt.Sprintf(`topic1 = '%s'`, *filter.Topic1))
		}
	}

	if filter.Topic2 != nil {
		if filter.Topic2 == &AnyStringValue {
			queries = append(queries, "topic2 != ''")
		} else {
			queries = append(queries, fmt.Sprintf(`topic2 = '%s'`, *filter.Topic2))
		}
	}

	if filter.Topic3 != nil {
		if filter.Topic3 == &AnyStringValue {
			queries = append(queries, "topic3 != ''")
		} else {
			queries = append(queries, fmt.Sprintf(`topic3 = '%s'`, *filter.Topic3))
		}
	}

	return queries, nil
}

func (db *MysqlDB) GetBalance(address string) (balance *Balance, err error) {
	balance = &Balance{}
	ok, err := db.engine.Where("addr = ?", address).Get(balance)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}

	return balance, nil
}

// state=1 block 只有一条记录
func (db *MysqlDB) GetBlockByNumAndState(num uint64, state int) (*Block, error) {
	block := &Block{}
	ok, err := db.engine.Where("num = ? and state = ?", num, state).Limit(1).Get(block)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return block, err
}

// state=1 block 只有一条记录
func (db *MysqlDB) GetErc20info(addr string) (*Erc20Info, error) {
	info := &Erc20Info{}
	ok, err := db.engine.Where("addr = ?", addr).Limit(1).Get(info)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return info, err
}

func (db *MysqlDB) GetBalanceErc20(address string, contractAddr string) (balanceErc20 *BalanceErc20, err error) {
	balanceErc20 = &BalanceErc20{}
	ok, err := db.engine.Where("addr = ? and contract_addr = ?", address, contractAddr).Get(balanceErc20)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}
	return balanceErc20, nil
}

func (db *MysqlDB) GetTokenPairByAddr(addr string) (*TokenPair, error) {
	ret := &TokenPair{}
	ok, err := db.engine.Table("token_pair").Where("addr = ?", addr).Get(ret)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return ret, nil
}

func (db *MysqlDB) GetErc1155Balance(address string, contractAddr string, tokenID string) (balanceErc1155 *BalanceErc1155, err error) {
	balanceErc1155 = &BalanceErc1155{}
	ok, err := db.engine.Where("addr = ? and contract_addr = ? and token_id = ?", address, contractAddr, tokenID).Get(balanceErc1155)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}
	return balanceErc1155, nil
}

// 查询to地址，得到对应的UID
func (db *MysqlDB) GetMonitorUID(to string) (string, error) {
	monitor := &Monitor{}
	ok, err := db.engine.Table("t_monitor").Where("f_addr = ?", to).Limit(1).Get(monitor)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", nil
	}
	return monitor.Uid, nil
}

func (db *MysqlDB) GetOpenMonitorTx(chain string) ([]*TxMonitor, error) {
	txMonitors := make([]*TxMonitor, 0)
	var err error

	err = db.engine.Table("t_monitor_hash").Where("f_chain = ? and f_push != ?", chain, FOUNDRECEIPTANDPUSHSUCCESS).Find(&txMonitors)
	if err != nil {
		return nil, err
	}
	return txMonitors, nil
}
