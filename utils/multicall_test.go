package utils

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/chainmonitor/config"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestMultiCall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := rpc.DialContext(ctx, "https://polygon-rpc.com")
	if err != nil {
		t.Errorf("create eth client err:%v", err)
		return
	}

	conf := config.DefaultFetchConfig
	conf.FetchTimeout = time.Millisecond * time.Duration(conf.FetchTimeoutInt)

	mc, err := NewMultiCall(client, "0x11ce4B23bD875D7F5C6a31084f55fDe1e9A87507", conf)
	if err != nil {
		t.Errorf("create multicall err:%v", err)
		return
	}

	const erc20abi = `[{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	erc20ABI, err := abi.JSON(strings.NewReader(erc20abi))
	if err != nil {
		t.Errorf("abi create err:%v", err)
		return
	}

	inputStr := [][]string{
		{"0x7056a5da7d269b31eb2e54e5579e41ef283d7d2c", "0x3BA4c387f786bFEE076A58914F5Bd38d668B42c3"},
		{"0x3BA4c387f786bFEE076A58914F5Bd38d668B42c3", "0x7056a5da7d269b31eb2e54e5579e41ef283d7d2c"},
		{"0x561Bb4b1A206714b933415Bb04eF560f6444189A", "0x0000000000000000000000000000000000000000"},
	}

	inputs := make([]MultiCallInput, 0, len(inputStr))

	for _, arr := range inputStr {
		k, v := arr[0], arr[1]
		input, err := erc20ABI.Pack("balanceOf", common.HexToAddress(v))
		if err != nil {
			t.Errorf("panic erc20 balanceOf input err:%v", err)
			return
		}

		inputs = append(inputs, MultiCallInput{
			Target:   common.HexToAddress(k),
			CallData: input,
		})
	}

	res := mc.AggregateWithRetry(inputs, 10000000)

	t.Logf("aggregate ReturnData:%v", res)
	for _, v := range res.ReturnData {
		if v.Err != nil {
			t.Logf("aggregate error:%v", v.Err)
			continue
		}
		res, err := erc20ABI.Unpack("balanceOf", v.Output)
		if err != nil {
			t.Logf("unpack err:%v", err)
			continue
		}

		t.Logf("unpack result:%s", res)
	}

}
