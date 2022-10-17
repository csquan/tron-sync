package mysqldb

import (
	"fmt"
	"math/big"

	"github.com/starslabhq/chainmonitor/mtypes"
)

func convertOutBlocks(blocks []*Block) ([]*mtypes.Block, error) {
	outBlocks := make([]*mtypes.Block, 0, len(blocks))

	for _, block := range blocks {
		res := &mtypes.Block{
			Number:       block.Number,
			Hash:         block.BlockHash,
			Nonce:        block.Nonce,
			ExtraData:    block.ExtraData,
			GasLimit:     block.GasLimit,
			GasUsed:      block.GasUsed,
			Miner:        block.Miner,
			ParentHash:   block.ParentHash,
			ReceiptsRoot: block.ReceiptsRoot,
			Sha3Uncles:   "",
			Size:         block.Size,
			StateRoot:    block.StateRoot,
			TxCnt:        block.TxsCnt,
			TimeStamp:    block.BlockTimestamp,
			UnclesCnt:    block.UnclesCnt,
			TxInternals:  make([]*mtypes.TxInternal, 0),
			Txs:          make([]*mtypes.Tx, 0),
		}

		var ok bool
		res.Difficulty, ok = big.NewInt(0).SetString(block.Difficulty, 10)
		if !ok {
			return nil, fmt.Errorf("convert block difficulty failed:%v", block.Difficulty)
		}
		res.TotoalDifficulty, ok = big.NewInt(0).SetString(block.TotalDifficulty, 10)
		if !ok {
			return nil, fmt.Errorf("convert block total difficulty failed:%v", block.TotalDifficulty)
		}
		res.BaseFee, _ = big.NewInt(0).SetString(block.BaseFee, 10)
		res.BurntFees, _ = big.NewInt(0).SetString(block.BurntFees, 10)

		outBlocks = append(outBlocks, res)
	}

	return outBlocks, nil
}

func convertOutInternalTxs(txs []*TxInternal) ([]*mtypes.TxInternal, map[uint64][]*mtypes.TxInternal, error) {
	res := make([]*mtypes.TxInternal, 0, len(txs))
	res2 := make(map[uint64][]*mtypes.TxInternal, len(txs))
	var ok bool

	for _, tx := range txs {
		outTx := &mtypes.TxInternal{
			TxHash:  tx.TxHash,
			From:    tx.AddrFrom,
			To:      tx.AddrTo,
			Success: tx.Success,
			OPCode:  tx.OPCode,
			Depth:   tx.Depth,
			GasUsed: tx.GasUsed,
		}

		outTx.Value, ok = big.NewInt(0).SetString(tx.Value, 10)
		if !ok {
			return nil, nil, fmt.Errorf("convert inner tx value failed:%v", tx.Value)
		}

		res = append(res, outTx)
		if _, ok := res2[tx.BlockNum]; !ok {
			res2[tx.BlockNum] = make([]*mtypes.TxInternal, 0)
		}
		res2[tx.BlockNum] = append(res2[tx.BlockNum], outTx)
	}

	return res, res2, nil
}

func convertOutTxs(txs []*TxDB) ([]*mtypes.Tx, map[uint64][]*mtypes.Tx, error) {
	res := make([]*mtypes.Tx, 0, len(txs))
	res2 := make(map[uint64][]*mtypes.Tx, len(txs))
	var ok bool

	for _, tx := range txs {
		outTx := &mtypes.Tx{
			TxType:           tx.TxType,
			From:             tx.AddrFrom,
			To:               tx.AddrTo,
			Hash:             tx.Hash,
			Index:            tx.Index,
			Input:            tx.Input,
			Nonce:            tx.Nonce,
			GasLimit:         tx.GasLimit,
			GasUsed:          tx.GasUsed,
			IsContract:       tx.IsContract,
			IsContractCreate: tx.IsContractCreate,
			BlockTime:        tx.BlockTime,
			BlockNum:         tx.BlockNum,
			BlockHash:        tx.BlockHash,
			ExecStatus:       tx.ExecStatus,
			EventLogs:        make([]*mtypes.EventLog, 0),
		}

		outTx.Value, ok = big.NewInt(0).SetString(tx.Value, 10)
		if !ok {
			return nil, nil, fmt.Errorf("convert tx value failed:%v", tx.Value)
		}
		outTx.GasPrice, ok = big.NewInt(0).SetString(tx.GasPrice, 10)
		if !ok {
			return nil, nil, fmt.Errorf("convert tx gas price failed:%v", tx.GasPrice)
		}

		outTx.BaseFee, _ = big.NewInt(0).SetString(tx.BaseFee, 10)
		outTx.MaxFeePerGas, _ = big.NewInt(0).SetString(tx.MaxFeePerGas, 10)
		outTx.MaxPriorityFeePerGas, _ = big.NewInt(0).SetString(tx.MaxPriorityFeePerGas, 10)
		outTx.BurntFees, _ = big.NewInt(0).SetString(tx.BurntFees, 10)

		res = append(res, outTx)
		if _, ok := res2[tx.BlockNum]; !ok {
			res2[tx.BlockNum] = make([]*mtypes.Tx, 0)
		}
		res2[tx.BlockNum] = append(res2[tx.BlockNum], outTx)
	}

	return res, res2, nil
}

func convertOutLogs(logs []*TxLog) ([]*mtypes.EventLog, map[string][]*mtypes.EventLog, error) {
	res := make([]*mtypes.EventLog, 0, len(logs))
	res2 := make(map[string][]*mtypes.EventLog, len(logs))

	for _, log := range logs {
		outLog := &mtypes.EventLog{
			TxHash: log.Hash,
			Topic0: log.Topic0,
			Topic1: log.Topic1,
			Topic2: log.Topic2,
			Topic3: log.Topic3,
			Data:   log.Data,
			Index:  log.Index,
			Addr:   log.Addr,
		}
		res = append(res, outLog)
		res2[log.Hash] = append(res2[log.Hash], outLog)
	}

	return res, res2, nil
}
