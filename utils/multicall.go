package utils

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/starslabhq/chainmonitor/config"
)

const multiCallABI = `
[
    {
        "constant": false,
        "inputs": [
            {
                "components": [
                    {
                        "name": "target",
                        "type": "address"
                    },
                    {
                        "name": "callData",
                        "type": "bytes"
                    }
                ],
                "name": "calls",
                "type": "tuple[]"
            }
        ],
        "name": "aggregate",
        "outputs": [
            {
                "name": "blockNumber",
                "type": "uint256"
            },
            {
                "name": "MultiCallReturnData",
                "type": "bytes[]"
            }
        ],
        "payable": false,
        "stateMutability": "nonpayable",
        "type": "function"
    }
]
`

//MultiCall https://github.com/makerdao/multicall
/**
multicall address:
ETH : 0xeefba1e63905ef1d7acba5a8513c70307c1ce441
HECO: 0x2c55d51804cf5b436ba5af37bd7b8e5db70ebf29
BSC : 0x41263cba59eb80dc200f3e2544eda4ed6a90e76c
POLY: 0x11ce4b23bd875d7f5c6a31084f55fde1e9a87507
AVAX: 0xa00fb557aa68d2e98a830642dbbfa534e8512e5f
FTM : 0x27e724954c4a7e2c974d785e2ae30336145711a1
*/
type MultiCall struct {
	client  *rpc.Client
	address string
	abi     abi.ABI
	conf    config.FetchConf
}

func NewMultiCall(client *rpc.Client, address string, conf config.FetchConf) (*MultiCall, error) {
	call := &MultiCall{
		client:  client,
		address: address,
		conf:    conf,
	}

	var err error
	call.abi, err = abi.JSON(strings.NewReader(multiCallABI))

	return call, err
}

type MultiCallOut struct {
	Output []byte
	Err    error
}
type MultiCallReturnData struct {
	BlockNumber *big.Int
	ReturnData  []*MultiCallOut
}

type MultiCallInput struct {
	Target   common.Address
	CallData []byte
}

func (m *MultiCall) Aggregate(inputs []MultiCallInput, minHeight uint64) (res *MultiCallReturnData, err error) {
	data, err := m.abi.Pack("aggregate", inputs)
	if err != nil {
		return
	}

	args := map[string]interface{}{
		"to":   m.address,
		"data": hexutil.Bytes(data),
	}
	var result hexutil.Bytes

	elems := make([]rpc.BatchElem, 0)

	elems = append(elems, rpc.BatchElem{
		Method: "eth_call",
		Args:   []interface{}{args, "latest"},
		Result: &result,
	})

	var blockNumber hexutil.Uint64
	elems = append(elems, rpc.BatchElem{
		Method: "eth_blockNumber",
		Args:   []interface{}{},
		Result: &blockNumber,
	})

	err = HandleErrorWithRetryMaxTime(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), m.conf.FetchTimeout)
		defer cancel()

		err1 := m.client.BatchCallContext(ctx, elems)
		if err1 != nil {
			return err1
		}

		if uint64(blockNumber) < minHeight {
			return fmt.Errorf("latest height:%v got cur chain height:%v", minHeight, uint64(blockNumber))
		}

		return nil
	}, m.conf.FetchRetryTimes, m.conf.FetchRetryInterval)
	if err != nil {
		return
	}

	if elems[0].Error != nil {
		err = elems[0].Error
		return
	}

	resInterfaces, err := m.abi.Unpack("aggregate", result)
	if err != nil {
		return
	}

	res = &MultiCallReturnData{
		BlockNumber: resInterfaces[0].(*big.Int),
		ReturnData:  make([]*MultiCallOut, len(inputs)),
	}

	for i, data := range resInterfaces[1].([][]byte) {
		res.ReturnData[i] = &MultiCallOut{
			Output: data,
		}
	}

	return
}

func (m *MultiCall) AggregateWithRetry(inputs []MultiCallInput, minHeight uint64) (res *MultiCallReturnData) {
	// start := time.Now()
	// defer func() { logrus.Debugf("aggregate with retry cost:%v, input:%v", time.Now().Sub(start), len(inputs)) }()
	var err error
	res, err = m.Aggregate(inputs, minHeight)
	if err == nil || len(inputs) == 0 {
		return
	}

	if len(inputs) == 1 {
		res = &MultiCallReturnData{
			BlockNumber: big.NewInt(math.MaxInt64),
			ReturnData: []*MultiCallOut{{
				Err: err,
			}},
		}

		return
	}

	size := len(inputs)
	res1 := m.AggregateWithRetry(inputs[0:size/2], minHeight)
	res2 := m.AggregateWithRetry(inputs[size/2:], minHeight)

	res1.ReturnData = append(res1.ReturnData, res2.ReturnData...)
	if res2.BlockNumber.Cmp(res1.BlockNumber) < 0 {
		res1.BlockNumber = res2.BlockNumber
	}

	res = res1

	return
}
