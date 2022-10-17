package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestGetBlock(t *testing.T) {
	cli, err := rpc.DialContext(context.Background(), "http://huobichain-dev-02.sinnet.huobiidc.com:5545")
	if err != nil {
		t.Fatalf("rpc client err:%v", err)
	}
	var ret json.RawMessage
	numberHex := hexutil.EncodeUint64(360806)
	t.Logf("number hex:%s", numberHex)
	// args := []interface{}{fmt.Sprintf("0x%x", 360806)}
	err = cli.CallContext(context.Background(), &ret, "eth_getBlockByNumber", numberHex, true)
	if err != nil {
		t.Fatalf("call rpc err:%v", err)
	}
	b, _ := ret.MarshalJSON()
	t.Logf("%s", b)
}

func TestGetReceipt(t *testing.T) {
	urlOnline := "https://http-mainnet-node.huobichain.com"
	cli, err := ethclient.DialContext(context.Background(), urlOnline)
	if err != nil {
		t.Fail()
	}
	hash := common.HexToHash("0x81969b02be6e0a3ad9ffd00b0ce987cb3d7f22dd6d8710f560995243f5a57243")
	receipt, err := cli.TransactionReceipt(context.Background(), hash)
	t.Logf("receipt:%v,err:%v", receipt, err)
}

func TestGetTransaction(t *testing.T) {
	urlOnline := "https://http-mainnet-node.huobichain.com"
	cli, err := ethclient.DialContext(context.Background(), urlOnline)
	if err != nil {
		t.Fail()
	}
	txhash := common.HexToHash("0x24b737df5c1ec87e91af5991d28d8200e604e4636787a37e55bbf293c568f4be")
	tx, _, err := cli.TransactionByHash(context.Background(), txhash)
	t.Logf("input:%v,err:%v", tx.Data(), err)

}

var content = `[
		{
			"constant": true,
			"inputs": [],
			"name": "name",
			"outputs": [
				{
					"name": "",
					"type": "string"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "_spender",
					"type": "address"
				},
				{
					"name": "_value",
					"type": "uint256"
				}
			],
			"name": "approve",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [],
			"name": "totalSupply",
			"outputs": [
				{
					"name": "",
					"type": "uint256"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "_from",
					"type": "address"
				},
				{
					"name": "_to",
					"type": "address"
				},
				{
					"name": "_value",
					"type": "uint256"
				}
			],
			"name": "transferFrom",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [],
			"name": "decimals",
			"outputs": [
				{
					"name": "",
					"type": "uint8"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [
				{
					"name": "_owner",
					"type": "address"
				}
			],
			"name": "balanceOf",
			"outputs": [
				{
					"name": "balance",
					"type": "uint256"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [],
			"name": "symbol",
			"outputs": [
				{
					"name": "",
					"type": "string"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "_to",
					"type": "address"
				},
				{
					"name": "_value",
					"type": "uint256"
				}
			],
			"name": "transfer",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"payable": false,
			"stateMutability": "nonpayable",
			"type": "function"
		},
		{
			"constant": true,
			"inputs": [
				{
					"name": "_owner",
					"type": "address"
				},
				{
					"name": "_spender",
					"type": "address"
				}
			],
			"name": "allowance",
			"outputs": [
				{
					"name": "",
					"type": "uint256"
				}
			],
			"payable": false,
			"stateMutability": "view",
			"type": "function"
		},
		{
			"payable": true,
			"stateMutability": "payable",
			"type": "fallback"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": true,
					"name": "owner",
					"type": "address"
				},
				{
					"indexed": true,
					"name": "spender",
					"type": "address"
				},
				{
					"indexed": false,
					"name": "value",
					"type": "uint256"
				}
			],
			"name": "Approval",
			"type": "event"
		},
		{
			"anonymous": false,
			"inputs": [
				{
					"indexed": true,
					"name": "from",
					"type": "address"
				},
				{
					"indexed": true,
					"name": "to",
					"type": "address"
				},
				{
					"indexed": false,
					"name": "value",
					"type": "uint256"
				}
			],
			"name": "Transfer",
			"type": "event"
		}
	]`

func TestContractCall(t *testing.T) {
	c, err := rpc.DialContext(context.Background(), "https://pub001.hg.network/rpc")
	if err != nil {
		t.Fail()
	}
	caller := ethclient.NewClient(c)

	r := strings.NewReader(content)
	erc20Balance, err := abi.JSON(r)
	if err != nil {
		t.Fatalf("generate abi err:%v", err)
	}
	contractAddrStr := `0x5545153CCFcA01fbd7Dd11C0b23ba694D9509A6F`
	contractAddr := common.HexToAddress(contractAddrStr)
	bc := bind.NewBoundContract(contractAddr, erc20Balance, caller, nil, nil)

	// balanceAddr := `0x8EBcc3d4546e240B4502DCf35f6D03a160dCd5eF`
	// var addr = common.HexToAddress(balanceAddr)
	// params := []interface{}{&addr}

	for i := 0; i < 1; i++ {
		var ret = make([]interface{}, 0)
		err2 := bc.Call(&bind.CallOpts{
			BlockNumber: nil,
			// From:        addr,
		}, &ret, "name")
		if err2 != nil {
			t.Fatalf("name err:%v", err2)
		}
		// if v, ok := ret[0].(*big.Int); ok {
		// 	t.Logf("balance :%s", v.String())
		// }
		t.Logf("name ret:%v", ret)
	}
}

func TestBatchCallErcBalance(t *testing.T) {
	contractAddrStr := `0xa71EdC38d189767582C38A3145b5873052c3e47a`
	contractAddr := common.HexToAddress(contractAddrStr)

	balanceAddr := `0xC3fD457A88Dd733eBF2A4f967399FAA6d55C9a38`
	var addr = common.HexToAddress(balanceAddr)

	r := strings.NewReader(content)
	erc20Balance, _ := abi.JSON(r)

	input, err := erc20Balance.Pack("balanceOf", addr)
	if err != nil {
		t.Fatalf("pack erc20 msg err:%v", err)
	}
	msg := ethereum.CallMsg{
		From: addr,
		To:   &contractAddr,
		Data: input,
	}

	var hex hexutil.Bytes

	elems := []rpc.BatchElem{
		{
			Method: "eth_call",
			Args:   []interface{}{toCallArg(msg), toBlockNumArg(big.NewInt(6877017))},
			Result: &hex,
		},
	}

	c, err := rpc.DialContext(context.Background(), "https://pub001.hg.network/rpc")
	if err != nil {
		t.Fail()
	}

	err = c.BatchCall(elems)
	if err != nil {
		t.Fatalf("batch call err:%v", err)
	}

	rets, err := erc20Balance.Unpack("balanceOf", hex)
	if err != nil {
		t.Fatalf("unpack ret err:%v", err)
	}
	if num, ok := rets[0].(*big.Int); ok {
		t.Logf("get balance:%s", num.String())
	}
}

func TestBloom(t *testing.T) {
	bf := bloom.NewWithEstimates(100000, 0.01)
	addr := common.HexToAddress("0xC3fD457A88Dd733eBF2A4f967399FAA6d55C9a38")
	bf.Add(addr.Bytes())

	t.Logf("exist:%v", bf.Test(addr.Bytes()))
}

func TestGetBlock2(t *testing.T) {
	// bjson := &mtypes.BlockJson{}
	var v interface{}
	bum := new(big.Int).SetUint64(100)
	c, err := rpc.DialContext(context.Background(), "https://matic-mainnet-full-rpc.bwarelabs.com")
	if err != nil {
		t.Fail()
	}
	// err = c.Call(&v, "eth_getBlockByNumber", toBlockNumArg(bum), true)
	err = c.Call(&v, "eth_getBlockByNumber", toBlockNumArg(bum), true)
	if err != nil {
		t.Fatalf("call getBlock %v", err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json encode err:%v", err)
	}
	t.Logf("block:%s", b)
}

// max poly block num 22393866
//  12.12 10 am block:22402525
func TestChOutput(t *testing.T) {
	ch := make(chan error, 3)
	ch <- errors.New("123")
	close(ch)
	var cnt int
	for e := range ch {
		if e != nil {
			cnt++
			t.Logf("---%v", e)
		}
	}
	t.Logf("%d", cnt)
}

func TestFetcher_getInternalTxByBlock(t *testing.T) {
	var rpcURL string = "http://172.23.18.4:8745"
	var timeout time.Duration = 2 * time.Second
	var blockNum uint64 = 52839
	client, err := rpc.DialHTTPWithClient(rpcURL, &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: timeout,
	})
	if err != nil {
		t.Error(err)
		return
	}
	f, _ := NewFetcher(client, nil, nil, false, "sync-test")
	bigIntBlockNum := new(big.Int).SetUint64(blockNum)
	txInternals, contracts := f.GetTxInternalsByBlock(bigIntBlockNum)
	for i := 0; i < len(txInternals); i++ {
		fmt.Printf("num %d", i)
		fmt.Println(txInternals[i])
	}
	t.Log(contracts)
}

func TestFetcher_getInternalTxByBlockFantom(t *testing.T) {
	var rpcURL string = "http://rpcapi-tracing.fantom.network"
	var timeout time.Duration = 2 * time.Second
	var blockNum uint64 = 39995486
	client, err := rpc.DialHTTPWithClient(rpcURL, &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: timeout,
	})
	if err != nil {
		t.Error(err)
		return
	}
	f, _ := NewFetcher(client, nil, nil, false, "sync-chain-ftm")
	bigIntBlockNum := new(big.Int).SetUint64(blockNum)
	txInternals, contracts := f.GetTxInternalsByBlockFantom(bigIntBlockNum)
	for i := 0; i < len(txInternals); i++ {
		fmt.Printf("num %d", i)
		fmt.Println()
		fmt.Println("value:", txInternals[i].Value)
		fmt.Println("opcode:", txInternals[i].OPCode)
		fmt.Println("input:", txInternals[i].Input)
		fmt.Println("output:", txInternals[i].Output)
		fmt.Println("gas:", txInternals[i].Gas)
		fmt.Println("gasUsed:", txInternals[i].GasUsed)
	}
	t.Log(contracts)
}
