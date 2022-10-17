package task

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestGetErc20Info(t *testing.T) {
	r := strings.NewReader(erc20abi)
	erc20ABI, err := abi.JSON(r)
	if err != nil {
	}
	url := "https://mainnet.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161"
	url = "https://http-mainnet-node.huobichain.com"
	url = "https://bsc-dataseed.binance.org"
	client, err := rpc.DialHTTPWithClient(url, &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: 5 * time.Second,
	})
	task := &Erc20TxTask{
		erc20ABI: erc20ABI,
		BaseAsyncTask: &BaseAsyncTask{
			client: client,
		},
	}

	addr := common.HexToAddress("0x4803ac6b79f9582f69c4fa23c72cb76dd1e46d8C")

	info, err := task.getErc20Info(&addr, 15237387+100)
	if err != nil {
		t.Fatalf("getErc20 info err:%v", err)
	}
	b, _ := json.Marshal(info)
	t.Logf("erc20_info:%s,err:%v", b, err)

}
