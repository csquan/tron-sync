package mtypes

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	NormalBalance = iota
	Erc20Balance
	Erc721Balance
)

type Tx struct {
	TxType           uint8
	From             string
	To               string
	Hash             string
	Index            int
	Value            *big.Int
	Input            string
	Nonce            uint64
	GasPrice         *big.Int
	GasLimit         uint64
	GasUsed          uint64
	IsContract       bool
	IsContractCreate bool
	BlockTime        int64
	BlockNum         uint64
	BlockHash        string
	ExecStatus       uint64
	// Erc20Transfers       []Erc20Transfer
	EventLogs []*EventLog
	// Erc20Infos           []*Erc20Info
	BaseFee              *big.Int
	MaxFeePerGas         *big.Int //交易费上限
	MaxPriorityFeePerGas *big.Int //小费上限
	BurntFees            *big.Int //baseFee*gasused
}

type EventLog struct {
	TxHash   string
	TopicCnt int
	Topic0   string
	Topic1   string
	Topic2   string
	Topic3   string

	Data  string
	Index uint
	Addr  string
}

type Erc20Transfer struct {
	TxHash          string
	Addr            string //合约地址
	Sender          string
	Receiver        string
	Tokens          *big.Int
	LogIndex        int
	SenderBalance   *big.Int
	ReceiverBalance *big.Int
}

type Erc20Info struct {
	Addr               string
	Name               string
	Symbol             string
	Decimals           uint8
	TotoalSupply       string
	TotoalSupplyOrigin string
}

type Erc721Info struct {
	Addr         string
	Name         string
	Symbol       string
	TotoalSupply string
}

type Erc721Token struct {
	ContractAddr  string
	TokenId       *big.Int
	OwnerAddr     string
	TokenUri      string
	TokenMetadata string
	Height        uint64
}

type Block struct {
	Number           uint64   `json:"number"`
	Hash             string   `json:"hash"`
	Difficulty       *big.Int `json:"difficulty"`
	TotoalDifficulty *big.Int `json:"totalDifficulty"`
	Nonce            string   `json:"nonce"`
	ExtraData        string   `json:"extra_data"`
	GasLimit         uint64   `json:"gas_limit"`
	GasUsed          uint64   `json:"gas_used"`
	Miner            string   `json:"minner"`
	ParentHash       string   `json:"parent_hash"`
	ReceiptsRoot     string   `json:"receipts_root"`
	Sha3Uncles       string   `json:"shas_uncles"`
	Size             uint64   `json:"size"`
	StateRoot        string   `json:"state_root"`
	TxCnt            int      `json:"tx_cnt"`
	TimeStamp        uint64   `json:"timestamp"`
	UnclesCnt        int      `json:"uncles_cnt"`
	Txs              []*Tx    `json:"txs"`
	BaseFee          *big.Int
	BurntFees        *big.Int
	Balances         []*Balance
	Contract         []*Contract
	TxInternals      []*TxInternal
	State            int `json:"state"` //0:normal block 1:revert block
}

type Balance struct {
	Addr         common.Address
	ContractAddr common.Address //erc20地址
	Height       *big.Int       //高度
	Value        *big.Int       //数额
	ValueBytes   hexutil.Bytes
	ValueHexBig  hexutil.Big
	BalanceType  int
}

// TxInternal is the 'internal transaction' struct for the ehtereum node RPC
type TxForInternal struct {
	TransactionHash string           `json:"transactionHash"`
	BlockHash       string           `json:"blockHash,omitempty"`
	BlockNumber     uint64           `json:"blockNumber,omitempty"`
	Logs            []*TxInternalLog `json:"logs"`
}

// TxForInternalFantom is the 'internal transaction' struct form the Fantom chain.
type TxForInternalFantom struct {
	Action struct {
		CallType string `json:"callType"`
		From     string `json:"from"`
		To       string `json:"to"`
		Value    string `json:"value"`
		Gas      string `json:"gas"`
		Input    string `json:"input"`
	} `json:"action"`
	BlockHash   string `json:"blockHash"`
	BlockNumber int    `json:"blockNumber"`
	Result      struct {
		GasUsed string `json:"gasUsed"`
		Output  string `json:"output"`
	} `json:"result,omitempty"`
	Error               string   `json:"error,omitempty"`
	Subtraces           int      `json:"subtraces"`
	TraceAddress        []uint64 `json:"traceAddress"`
	TransactionHash     string   `json:"transactionHash"`
	TransactionPosition int      `json:"transactionPosition"`
	Type                string   `json:"type"`
}

// TxInternalLog is the 'log' struct for the ehtereum node RPC
type TxInternalLog struct {
	From string `json:"from"`
	To   string `json:"to,omitempty"`
	// Value   *hexutil.Big `json:"value,omitempty"`
	Value   *big.Int `json:"value,omitempty"`
	Success bool     `json:"success"`
	OPCode  string   `json:"opcode"`
	Depth   int      `json:"depth"`
	Gas     uint64   `json:"gas"`
	GasUsed uint64   `json:"gas_used"`
	//Input        []byte       `json:"input"`
	//Output       []byte       `json:"output,omitempty"`
	Input        string   `json:"input"`
	Output       string   `json:"output,omitempty"`
	TraceAddress []uint64 `json:"trace_address"`
}

// TxTenternal is the 'internal transaction log' data model for the database
type TxInternal struct {
	TxHash       string
	From         string
	To           string
	Value        *big.Int
	Success      bool
	OPCode       string
	Depth        int
	Gas          uint64
	GasUsed      uint64
	Input        string
	Output       string
	TraceAddress []uint64
}

// Contract is the contract data model for the database
type Contract struct {
	TxHash      string
	Addr        string
	CreatorAddr string
	ExecStatus  uint64
}

type BlockRevert struct {
	Num uint64
}

type TokenPair struct {
	Addr          common.Address `xorm:"addr"`
	PairName      string         `xorm:"pair_name"`
	PairNameBytes hexutil.Bytes
	Token0Name    string         `xorm:"token0_name"`
	Token1Name    string         `xorm:"token1_name"`
	Token0        common.Address `xorm:"token0"`
	Token0Bytes   hexutil.Bytes
	Token1        common.Address `xorm:"token1"`
	Token1Bytes   hexutil.Bytes
	Reserve0      string    `xorm:"reserve0"`
	Reserve1      string    `xorm:"reserve1"`
	Updated       time.Time `xorm:"updated"`
	BlockNum      int64     `xorm:"block_num"`
}
