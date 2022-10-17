package ktypes

import (
	"math/big"
)

type EventLog struct {
	Topic0 string `json:"topic0"`
	Topic1 string `json:"topic1"`
	Topic2 string `json:"topic2"`
	Topic3 string `json:"topic3"`

	Data  string `json:"data"`
	Index uint   `json:"index"`
	Addr  string `json:"addr"`
}

type Tx struct {
	TxType   uint8  `json:"tx_type"`
	From     string `json:"from"`
	To       string `json:"to"`
	Hash     string `json:"hash"`
	Index    int    `json:"index"`
	Value    string `json:"value"`
	Input    string `json:"input"`
	Nonce    uint64 `json:"nonce"`
	GasPrice string `json:"gas_price"`
	GasLimit uint64 `json:"gas_limit"`
	GasUsed  uint64 `json:"gas_used"`
	// IsContract       bool     `json:""`
	// IsContractCreate bool   `json:""`
	BlockTime            int64       `json:"block_time"`
	BlockNum             uint64      `json:"block_num"`
	BlockHash            string      `json:"block_hash"`
	ExecStatus           uint64      `json:"exec_status"`
	EventLogs            []*EventLog `json:"tx_logs"`
	BaseFee              string      `json:"base_fee"`
	MaxFeePerGas         string      `json:"max_fee_per_gas"`          //交易费上限
	MaxPriorityFeePerGas string      `json:"max_priority_fee_per_gas"` //小费上限
	BurntFees            string      `json:"burnt_fees"`               //baseFee*gasused
}

type TxInternal struct {
	Hash         string   `json:"hash,omitempty"`
	From         string   `json:"from,omitempty"`
	To           string   `json:"to,omitempty"`
	Value        string   `json:"value,omitempty"`
	Success      bool     `json:"success,omitempty"`
	OPCode       string   `json:"opcode,omitempty"`
	Depth        int      `json:"depth,omitempty"`
	Gas          uint64   `json:"gas,omitempty"`
	GasUsed      uint64   `json:"gas_used,omitempty"`
	Input        string   `json:"input,omitempty"`
	Output       string   `json:"output,omitempty"`
	TraceAddress []uint64 `json:"trace_address,omitempty"`
}

type Block struct {
	Number       uint64        `json:"number"`
	Hash         string        `json:"hash"`
	Difficulty   *big.Int      `json:"difficulty"`
	Nonce        string        `json:"nonce"`
	ExtraData    string        `json:"extra_data"`
	GasLimit     uint64        `json:"gas_limit"`
	GasUsed      uint64        `json:"gas_used"`
	Miner        string        `json:"minner"`
	ParentHash   string        `json:"parent_hash"`
	ReceiptsRoot string        `json:"receipts_root"`
	StateRoot    string        `json:"state_root"`
	TxCnt        int           `json:"tx_cnt"`
	TimeStamp    uint64        `json:"timestamp"`
	UnclesCnt    int           `json:"uncles_cnt"`
	Size         uint64        `json:"size"`
	Txs          []*Tx         `json:"txs"`
	TxInternals  []*TxInternal `json:"tx_internals"`
	BaseFee      *big.Int      `json:"base_fee"`
	BurntFees    *big.Int      `json:"burnt_fees"`
	State        int           `json:"state"` //0:normal block 1:revert block
}
