package mysqldb

import (
	"fmt"
	"math/big"
)

type BlockFilter struct {
	TxFilter         *TxFilter
	InternalTxFilter *InternalTxFilter
	LogFilter        *LogFilter
}

type TxFilter struct {
	From  *string
	To    *string
	Value *big.Int
}

type InternalTxFilter struct {
	From    *string
	To      *string
	Value   *big.Int
	Success *bool
}
type LogFilter struct {
	Contract *string
	Topic0   *string
	Topic1   *string
	Topic2   *string
	Topic3   *string
}

var DefaultFullBlockFilter = &BlockFilter{
	TxFilter:         &TxFilter{},
	InternalTxFilter: &InternalTxFilter{},
	LogFilter:        &LogFilter{},
}
var erc20Transfer = `0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef`
var ERC20Filter = &BlockFilter{
	TxFilter: &TxFilter{},
	LogFilter: &LogFilter{
		Topic0: &erc20Transfer,
		Topic1: &(AnyStringValue),
		Topic2: &(AnyStringValue),
		Topic3: &(VoidStringValue), //to filter out from erc721
	},
}

var ERC721Filter = &BlockFilter{
	LogFilter: &LogFilter{
		Topic0: &erc20Transfer,
		Topic1: &(AnyStringValue),
		Topic2: &(AnyStringValue),
		Topic3: &(AnyStringValue),
	},
}

var Erc1155SingleHash = `0xc3d58168c5ae7397731d063d5bbf3d657854427343f4c083240f7aacaa2d0f62`
var Erc1155BatchHash = `0x4a39dc06d4c0dbc64b70af90fd698a233a518aa5d07e595d983b8c0526c8f7fb`

var ERC1155Topic0 = fmt.Sprintf("%s,%s", Erc1155SingleHash, Erc1155BatchHash)

var ERC1155Filter = &BlockFilter{
	LogFilter: &LogFilter{
		Topic0: &ERC1155Topic0,
	},
}
