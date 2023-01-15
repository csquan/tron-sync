package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/chainmonitor/mtypes"
	"github.com/chainmonitor/output/mysqldb"
	"github.com/ethereum/go-ethereum/core/types"
)

func compare(d1 *DataChain, d2 *DataDB, internal bool, chainName string) error {
	err := compareBlock(d1.Block, d2.Block, chainName)
	if err != nil {
		return err
	}
	err = comareTxs(d1.Txs, d2.Txs)
	if err != nil {
		return err
	}
	err = compareTxErc20(d1.TxErc20s, d2.TxErc20s)
	if err != nil {
		return err
	}
	if internal {
		err = compareTxInternal(d1.TxInternal, d2.TxInternals)
		if err != nil {
			return err
		}
	}
	err = compareTxLog(d1.TxLogs, d2.TxLogs)
	if err != nil {
		return err
	}
	if d1.Block.Number >= 7435782 {
		err = comareContract(d1.Contracs, d2.Contracts)
		if err != nil {
			return err
		}
	}
	return nil
}

func compareBlock(b1 *mtypes.Block, b2 *mysqldb.Block, chainName string) error {
	if b1 != nil && b2 == nil {
		return errors.New("blockdb not exist")
	}
	if b1.Number != b2.Number {
		return errors.New("bnum not equal")
	}
	if b1.Hash != b2.BlockHash {
		return fmt.Errorf("block hash not equal %s:%s", b1.Hash, b2.BlockHash)
	}
	if b1.ParentHash != b2.ParentHash || b1.Miner != b2.Miner ||
		b1.TxCnt != b2.TxsCnt || b1.GasLimit != b2.GasLimit ||
		b1.GasUsed != b2.GasUsed || b1.StateRoot != b2.StateRoot ||
		b1.ExtraData != b2.ExtraData || b1.ReceiptsRoot != b2.ReceiptsRoot ||
		b1.TimeStamp != b2.BlockTimestamp {

		if chainName == "eth" && b1.Number > 12_965_000 {
			if b1.BaseFee.String() != b2.BaseFee || b1.BurntFees.String() != b2.BurntFees {
				b1json, _ := json.Marshal(b1)
				b2json, _ := json.Marshal(b2)
				return fmt.Errorf("block not equal num:%d,%s\n%s", b1.Number, b1json, b2json)
			}
		}
		b1json, _ := json.Marshal(b1)
		b2json, _ := json.Marshal(b2)
		return fmt.Errorf("block not equal num:%d,%s\n%s", b1.Number, b1json, b2json)
	}
	return nil
}

func comareTxs(txs1 []*mtypes.Tx, txs2 []*mysqldb.TxDB) error {
	if len(txs1) != len(txs2) {
		return fmt.Errorf("txcnt not eqaul %d:%d", len(txs1), len(txs2))
	}
	sort.Slice(txs1, func(i, j int) bool {
		return txs1[i].Index < txs1[j].Index
	})
	sort.Slice(txs2, func(i, j int) bool {
		return txs2[i].Index < txs2[j].Index
	})
	size := len(txs1)
	for i := 0; i < size; i++ {
		tx1 := txs1[i]
		tx2 := txs2[i]
		if tx1.TxType != tx2.TxType || tx1.Hash != tx2.Hash || tx1.Value.String() != tx2.Value ||
			tx1.Index != tx2.Index || tx1.From != tx2.AddrFrom ||
			(tx1.To != "" && tx1.To != tx2.AddrTo) || tx1.Nonce != tx2.Nonce || tx1.Input != tx2.Input {

			if tx1.TxType == types.DynamicFeeTxType {
				if tx1.BaseFee.String() != tx2.BaseFee || tx1.MaxFeePerGas.String() != tx2.MaxFeePerGas ||
					tx1.MaxPriorityFeePerGas.String() != tx2.MaxPriorityFeePerGas ||
					tx1.BurntFees.String() != tx2.BurntFees {
					tx1json, _ := json.Marshal(tx1)
					tx2json, _ := json.Marshal(tx2)
					log.Printf("tx:%s\n%s", tx1json, tx2json)
					return fmt.Errorf("eip1559 tx not equal")
				}
			}
			tx1json, _ := json.Marshal(tx1)
			tx2json, _ := json.Marshal(tx2)
			log.Printf("tx:%s\n%s", tx1json, tx2json)
			return errors.New("tx not equal")
		}
	}
	return nil
}

func compareTxErc20(txs1M map[string][]mtypes.Erc20Transfer, txs2M map[string][]*mysqldb.TxErc20) error {
	for txhash, txs1 := range txs1M {
		txs2, ok := txs2M[txhash]
		if !ok {
			log.Fatalf("erc20 tx not found th:%s", txhash)
		}
		if len(txs1) != len(txs2) {
			j1, _ := json.Marshal(txs1)
			j2, _ := json.Marshal(txs2)
			return fmt.Errorf("erc20 tx cnt %s\n%s", j1, j2)
		}
		sort.Slice(txs1, func(i, j int) bool {
			return txs1[i].LogIndex < txs1[j].LogIndex
		})
		sort.Slice(txs2, func(i, j int) bool {
			return txs2[i].LogIndex < txs2[j].LogIndex
		})
		size := len(txs1)
		for i := 0; i < size; i++ {
			tx1 := txs1[i]
			tx2 := txs2[i]
			var tx2Token string
			if tx2.TokenCntOrigin != "" {
				tx2Token = tx2.TokenCntOrigin
			} else {
				tx2Token = tx2.TokenCnt
			}
			if tx1.TxHash != tx2.Hash || tx1.Addr != tx2.Addr || tx1.Tokens.String() != tx2Token ||
				tx1.Sender != tx2.Sender || tx1.Receiver != tx2.Receiver || tx1.LogIndex != tx2.LogIndex {
				j1, _ := json.Marshal(tx1)
				j2, _ := json.Marshal(tx2)
				return fmt.Errorf("erc 20 txs not eqaul %s\n%s", j1, j2)
			}
		}
	}
	return nil
}

func compareTxInternal(txs1M map[string][]*mtypes.TxInternal, txs2M map[string][]*mysqldb.TxInternal) error {
	for txhash, txs1 := range txs1M {
		txs2, ok := txs2M[txhash]
		if !ok {
			if len(txs1) != 0 && (txs1[0].OPCode == "Create2" || txs1[0].OPCode == "Create") {
				// continue
			} else {
				j1, _ := json.Marshal(txs1M)
				j2, _ := json.Marshal(txs1M)
				log.Fatalf("internal tx not exist th:%s,%s\n%s", txhash, j1, j2)
			}
		}
		// var isCreate bool
		// for _, tx := range txs1 {
		// 	if tx.OPCode == "Create2" || tx.OPCode == "Create" {
		// 		isCreate = true
		// 	}
		// }
		// if isCreate {
		// continue
		// }
		if len(txs1) != len(txs2) {
			j1, _ := json.Marshal(txs1)
			j2, _ := json.Marshal(txs2)
			return fmt.Errorf("internal tx cnt not eqaul %s\n:%s", j1, j2)
		}
		sort.Slice(txs1, func(i, j int) bool {
			return txs1[i].TxHash < txs1[j].TxHash
		})
		sort.Slice(txs2, func(i, j int) bool {
			return txs2[i].TxHash < txs2[j].TxHash
		})
		size := len(txs1)
		for i := 0; i < size; i++ {
			tx1 := txs1[i]
			var tx2 *mysqldb.TxInternal
			for j := 0; j < len(txs2); j++ {
				tx2 = txs2[j]
				if tx1.TxHash == tx2.TxHash && tx1.From == tx2.AddrFrom && tx1.To == tx2.AddrTo && tx1.Value.String() == tx2.Value {
					break
				}
			}
			if tx2 == nil {
				tx1json, _ := json.Marshal(tx1)
				return fmt.Errorf("internal tx err not found tx:%s", tx1json)
			}
			if tx1.TxHash != tx2.TxHash || tx1.From != tx2.AddrFrom || tx1.To != tx2.AddrTo ||
				tx1.Value.String() != tx2.Value || tx1.OPCode != tx2.OPCode || tx1.Success != tx2.Success {
				tx1json, _ := json.Marshal(tx1)
				tx2json, _ := json.Marshal(tx2)
				return fmt.Errorf("internal tx not equal txhash:%s\n%s", tx1json, tx2json)
			}
		}
	}
	return nil
}

func compareTxLog(txLogs1M map[string][]mtypes.EventLog, txLogs2M map[string][]*mysqldb.TxLog) error {
	for txhash, txLogs1 := range txLogs1M {
		txLogs2, ok := txLogs2M[txhash]
		if !ok {
			log.Fatalf("txlog not exist txhash:%s", txhash)
		}
		if len(txLogs1) != len(txLogs2) {
			j1, _ := json.Marshal(txLogs1)
			j2, _ := json.Marshal(txLogs2)
			return fmt.Errorf("txlog cnt not equal %s\n%s", j1, j2)
		}
		sort.Slice(txLogs1, func(i, j int) bool {
			return txLogs1[i].Index < txLogs1[j].Index
		})
		sort.Slice(txLogs2, func(i, j int) bool {
			return txLogs2[i].Index < txLogs2[j].Index
		})
		size := len(txLogs1)
		for i := 0; i < size; i++ {
			log1 := txLogs1[i]
			log2 := txLogs2[i]
			if log1.Addr != (log2.Addr) || log1.Data != log2.Data || log1.Index != log2.Index ||
				log1.Topic0 != log2.Topic0 ||
				(log2.Topic1 != "" && log1.Topic1 != log2.Topic1) ||
				(log2.Topic2 != "" && log1.Topic2 != log2.Topic2) ||
				(log2.Topic3 != "" && log1.Topic3 != log2.Topic3) {
				log1Json, _ := json.Marshal(log1)
				log2Json, _ := json.Marshal(log2)
				return fmt.Errorf("txlog not equal %s\n%s", log1Json, log2Json)
			}
		}
	}
	return nil
}

func comareContract(cs1 []*mtypes.Contract, cs2 []*mysqldb.Contract) error {
	if len(cs1) != len(cs2) {
		j1, _ := json.Marshal(cs1)
		j2, _ := json.Marshal(cs2)
		return fmt.Errorf("contract size not equal %s\n%s", j1, j2)
	}

	sort.Slice(cs1, func(i, j int) bool {
		return cs1[i].TxHash < cs1[j].TxHash
	})
	sort.Slice(cs2, func(i, j int) bool {
		return cs2[i].TxHash < cs2[j].TxHash
	})
	size := len(cs1)
	for i := 0; i < size; i++ {
		cs1 := cs1[i]
		// cs2 := cs2[i]
		var contract2 *mysqldb.Contract
		for j := 0; j < len(cs2); j++ {
			if cs1.Addr == cs2[j].Addr {
				contract2 = cs2[j]
			}
		}
		if cs1.TxHash != contract2.TxHash || !strings.EqualFold(cs1.CreatorAddr, contract2.CreatorAddr) || cs1.ExecStatus != contract2.ExecStatus ||
			cs1.Addr != contract2.Addr {
			j1, _ := json.Marshal(cs1)
			j2, _ := json.Marshal(contract2)
			return fmt.Errorf("contract not equal %s\n%s", j1, j2)
		}

	}
	return nil
}
