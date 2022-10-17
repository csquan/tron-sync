package mysqldb

import (
	"time"
)

const (
	Block_ok = iota
	Block_revert
	Block_all
)

type AsyncTask struct {
	Id        uint64 `xorm:"id"`
	Name      string `xorm:"name"`
	Number    uint64 `xorm:"num"`
	EndNumber uint64 `xorm:"end_num"`
}

type Block struct {
	Id              int64
	Number          uint64 `xorm:"num"`
	BlockHash       string `xorm:"block_hash"`
	Difficulty      string `xorm:"difficulty"`
	ExtraData       string `xorm:"text 'extra_data'"`
	GasLimit        uint64 `xorm:"gas_limit"`
	GasUsed         uint64 `xorm:"gas_used"`
	Miner           string `xorm:"miner"`
	ParentHash      string `xorm:"parent_hash"`
	Size            uint64 `xorm:"size"`
	ReceiptsRoot    string `xorm:"receipts_root"`
	StateRoot       string `xorm:"state_root"`
	TxsCnt          int    `xorm:"txs_cnt"`
	BlockTimestamp  uint64 `xorm:"block_timestamp"`
	UnclesCnt       int    `xorm:"uncles_cnt"`
	State           uint8  `xorm:"state"`
	BaseFee         string `xorm:"base_fee"`
	BurntFees       string `xorm:"burnt_fees"`
	TotalDifficulty string `xorm:"total_difficulty"`
	Nonce           string `xorm:"nonce"`
}

type TxDB struct {
	Id               uint64 `xorm:"id"`
	TxType           uint8  `xorm:"tx_type"`
	AddrTo           string `xorm:"addr_to"`
	AddrFrom         string `xorm:"addr_from"`
	Hash             string `xorm:"tx_hash"`
	Index            int    `xorm:"tx_index"`
	Value            string `xorm:"tx_value"`
	Input            string `xorm:"input"`
	Nonce            uint64 `xorm:"nonce"`
	GasPrice         string `xorm:"gas_price"`
	GasLimit         uint64 `xorm:"gas_limit"`
	GasUsed          uint64 `xorm:"gas_used"`
	IsContract       bool   `xorm:"is_contract"`
	IsContractCreate bool   `xorm:"is_contract_create"`
	BlockNum         uint64 `xorm:"block_num"`
	BlockHash        string `xorm:"block_hash"`
	BlockTime        int64  `xorm:"block_time"`
	ExecStatus       uint64 `xorm:"exec_status"`
	BlockState       uint8  `xorm:"block_state"`
	//eip1559
	BaseFee              string `xorm:"base_fee"`
	MaxFeePerGas         string `xorm:"max_fee_per_gas"`
	MaxPriorityFeePerGas string `xorm:"max_priority_fee_per_gas"`
	BurntFees            string `xorm:"burnt_fees"`
}

func (tx *TxDB) TableName() string {
	return "tx"
}

type TxLog struct {
	Id         uint64 `xorm:"id"`
	Hash       string `xorm:"tx_hash"`
	Addr       string `xorm:"addr"`
	AddrFrom   string `xorm:"addr_from"`
	AddrTo     string `xorm:"addr_to"`
	Topic0     string `xorm:"topic0"`
	Topic1     string `xorm:"topic1"`
	Topic2     string `xorm:"topic2"`
	Topic3     string `xorm:"topic3"`
	Data       string `xorm:"log_data"`
	Index      uint   `xorm:"log_index"`
	BlockState uint8  `xorm:"block_state"`
	BlockNum   uint64 `xorm:"block_num"`
	BlockTime  uint64 `xorm:"block_time"`
}

type TxErc20 struct {
	Id             uint64 `xorm:"id"`
	Hash           string `xorm:"tx_hash"`
	Addr           string `xorm:"addr"`
	Sender         string `xorm:"sender"`
	Receiver       string `xorm:"receiver"`
	TokenCnt       string `xorm:"token_cnt"`
	TokenCntOrigin string `xorm:"token_cnt_origin"`
	LogIndex       int    `xorm:"log_index"`
	BlockState     uint8  `xorm:"block_state"`
	BlockNum       uint64 `xorm:"block_num"`
	BlockTime      uint64 `xorm:"block_time"`
}

type TxErc721 struct {
	Id         uint64 `xorm:"id"`
	Hash       string `xorm:"tx_hash"`
	Addr       string `xorm:"addr"`
	Sender     string `xorm:"sender"`
	Receiver   string `xorm:"receiver"`
	TokenId    string `xorm:"token_id"`
	LogIndex   int    `xorm:"log_index"`
	BlockState uint8  `xorm:"block_state"`
	BlockNum   uint64 `xorm:"block_num"`
	BlockTime  uint64 `xorm:"block_time"`
}

type TxErc1155 struct {
	Id         uint64 `xorm:"id"`
	Hash       string `xorm:"tx_hash"`
	Addr       string `xorm:"addr"`
	Operator   string `xorm:"operator"`
	Sender     string `xorm:"sender"`
	Receiver   string `xorm:"receiver"`
	TokenId    string `xorm:"token_id"`
	TokenCnt   string `xorm:"token_cnt"`
	LogIndex   int    `xorm:"log_index"`
	BlockState uint8  `xorm:"block_state"`
	BlockNum   uint64 `xorm:"block_num"`
	BlockTime  uint64 `xorm:"block_time"`
}

type Balance struct {
	Id            uint64 `xorm:"id"`
	Addr          string `xorm:"addr"`
	Balance       string `xorm:"balance"`
	Height        uint64 `xorm:"height"`
	BalanceOrigin string `xorm:"balance_origin"`
}

type BalanceErc20 struct {
	Id            uint64 `xorm:"id"`
	Addr          string `xorm:"addr"`
	ContractAddr  string `xorm:"contract_addr"`
	Balance       string `xorm:"balance"`
	Height        uint64 `xorm:"height"`
	BalanceOrigin string `xorm:"balance_origin"`
}

type BalanceErc721 struct {
	Id            uint64 `xorm:"id"`
	Addr          string `xorm:"addr"`
	ContractAddr  string `xorm:"contract_addr"`
	Balance       string `xorm:"balance"`
	Height        uint64 `xorm:"height"`
	BalanceOrigin string `xorm:"balance_origin"`
}

type BalanceErc1155 struct {
	Id            uint64 `xorm:"id"`
	Addr          string `xorm:"addr"`
	ContractAddr  string `xorm:"contract_addr"`
	TokenID       string `xorm:"token_id"`
	Balance       string `xorm:"balance"`
	Height        uint64 `xorm:"height"`
	BalanceOrigin string `xorm:"balance_origin"`
}

type Erc20Info struct {
	Id                 uint64 `xorm:"id"`
	Addr               string `xorm:"addr"`
	Name               string `xorm:"name"`
	Symbol             string `xorm:"symbol"`
	Decimals           uint8  `xorm:"decimals"`
	TotoalSupply       string `xorm:"total_supply"`
	TotoalSupplyOrigin string `xorm:"total_supply_origin"`
}

type Erc721Info struct {
	Id                 uint64 `xorm:"id"`
	Addr               string `xorm:"addr"`
	Name               string `xorm:"name"`
	Symbol             string `xorm:"symbol"`
	TotoalSupply       string `xorm:"total_supply"`
	TotoalSupplyOrigin string `xorm:"total_supply_origin"`
}

type Erc721Token struct {
	ContractAddr  string `xorm:"contract_addr"`
	TokenId       string `xorm:"token_id"`
	OwnerAddr     string `xorm:"owner_addr"`
	TokenUri      string `xorm:"token_uri"`
	TokenMetadata string `xorm:"token_metadata"`
	Height        uint64 `xorm:"height"`
}

type TxInternal struct {
	Id           uint64 `xorm:"id"`
	TxHash       string `xorm:"tx_hash"`
	AddrFrom     string `xorm:"addr_from"`
	AddrTo       string `xorm:"addr_to"`
	OPCode       string `xorm:"op_code"`
	Value        string `xorm:"value"`
	Success      bool   `xorm:"success"`
	Depth        int    `xorm:"depth"`
	Gas          uint64 `xorm:"gas"`
	GasUsed      uint64 `xorm:"gas_used"`
	Input        string `xorm:"input"`
	Output       string `xorm:"output"`
	TraceAddress string `xorm:"trace_address"`
	BlockState   uint8  `xorm:"block_state"`
	BlockNum     uint64 `xorm:"block_num"`
	BlockTime    uint64 `xorm:"block_time"`
}

type Contract struct {
	TxHash      string `xorm:"tx_hash"`
	Addr        string `xorm:"addr"`
	CreatorAddr string `xorm:"creator_addr"`
	ExecStatus  uint64 `xorm:"exec_status"`
	BlockNum    uint64 `xorm:"block_num"`
	BlockState  int    `xorm:"block_state"`
}

type TokenPair struct {
	Addr     string    `xorm:"addr"`
	PairName string    `xorm:"pair_name"`
	Token0   string    `xorm:"token0"`
	Token1   string    `xorm:"token1"`
	Reserve0 string    `xorm:"reserve0"`
	Reserve1 string    `xorm:"reserve1"`
	Updated  time.Time `xorm:"updated"`
	BlockNum int64     `xorm:"block_num"`
}
