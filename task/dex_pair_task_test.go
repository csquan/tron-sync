package task

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/starslabhq/chainmonitor/mtypes"
)

func TestGetPairsToken(t *testing.T) {
	c, err := rpc.DialContext(context.Background(), "https://pub001.hg.network/rpc")
	if err != nil {
		t.Fail()
	}
	dt := &DexPairTask{
		BaseAsyncTask: &BaseAsyncTask{
			client: c,
		},
	}
	r := strings.NewReader(dexContract)
	abiDex, err := abi.JSON(r)
	if err != nil {
		t.Fatalf("abi load err:%v", err)
	}
	dt.abiDex = abiDex
	rname := strings.NewReader(nameAbi)

	abiName, err := abi.JSON(rname)
	if err != nil {
		t.Fatalf("abi name load err:%v", err)
	}
	dt.abiName = abiName
	pair0 := &mtypes.TokenPair{
		Addr: common.HexToAddress("0x615e6285c5944540fd8bd921c9c8c56739fd1e13"),
	}
	// pair1 := &mtypes.TokenPair{
	// 	Addr: common.HexToAddress("0x499b6e03749b4baf95f9e70eed5355b138ea6c31"),
	// }
	pairs := []*mtypes.TokenPair{
		pair0,
		// pair1,
	}
	err = dt.getPairsToken(pairs)
	if err != nil {
		t.Fatalf("get pairs token err:%v", err)
	}
	t.Logf("ret: addr:%s,t0:%s,t1:%s,name:%s", pair0.Addr.Hex(), pair0.Token0.Hex(), pair0.Token1.Hex(), pair0.PairName)
	// t.Logf("ret: addr:%s,t0:%s,t1:%s,name:%s", pair1.Addr.Hex(), pair1.Token0.Hex(), pair1.Token1.Hex(), pair1.PairName)
}
