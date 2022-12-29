package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/go-xorm/xorm"
)

var (
	start, end uint64
	internal   bool
	chainName  string
)

func init() {
	flag.Uint64Var(&start, "s", 0, "start")
	flag.Uint64Var(&end, "e", 0, "end")
	flag.BoolVar(&internal, "i", false, "check internal")
	flag.StringVar(&chainName, "c", "", "chain")
}
func main() {
	flag.Parse()
	if start == 0 {
		start = 7300000
		// log.Fatalf("start not set")
	}
	if end == 0 {
		end = start + 1
	}
	// host := `http://huobichain-dev-08.sinnet.huobiidc.com:8545`
	// dburl := `root:12345678@tcp(huobichain-dev-01.sinnet.huobiidc.com:3306)/heco_data_test?charset=utf8mb4`
	// dburl = `heco:0WLkhaAtQB5ATL0r@tcp(bigdata-tidb-e604534a50b9d35d.elb.ap-northeast-1.amazonaws.com:4000)/chain?charset=utf8mb4`
	var host, dburl string
	switch chainName {
	case "heco":
		host = `http://172.26.21.36:8545`
		//heco
		dburl = `chain_heco_usrc:E8sPn42nYA_z9pMY@tcp(tidb-chain-data-proxy.huobiidc.com:3541)/chain_heco?charset=utf8mb4`
	case "eth":
		// eth
		host = `http://defi-node-eth-archive-14d7f26ead98a52d.elb.ap-northeast-1.amazonaws.com:8545`
		//eth
		dburl = `chain_eth_usrc:h8vW_ZkzTSNQhkkG@tcp(tidb-chain-data-proxy.huobiidc.com:3541)/chain_eth?charset=utf8mb4`
	case "bsc":
		//bsc
		host = `http://defi-node-bsc-archive-e45a0a972d57cac5.elb.ap-northeast-1.amazonaws.com:8545`
		//bsc
		dburl = `chain_bsc_usrc:v0bU_PRXkzaaY6Pt@tcp(tidb-chain-data-proxy.huobiidc.com:3541)/chain_bsc?charset=utf8mb4`
	case "poly":
		//poly
		host = `https://rpc-mainnet.maticvigil.com`
		//poly
		dburl = `chain_polygona_usrc:Z9bWBnYxMda_tnCs@tcp(tidb-chain-data-proxy.huobiidc.com:3541)/chain_polygona?charset=utf8mb4`
	default:
		log.Fatalf("chain not support")
	}

	ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
	c, err := rpc.DialContext(ctx, host)
	if err != nil {
		log.Fatalf("rpc client err:%v", err)
	}
	defer cancle()
	chain := &Chain{
		c:        c,
		Client:   ethclient.NewClient(c),
		internal: internal,
		name:     chainName,
	}

	engine, err := xorm.NewEngine("mysql", dburl)

	if err != nil {
		log.Fatalf("create mysqldb err:%v", err)
	}
	engine.ShowSQL(true)
	db := &DB{
		engine: engine,
	}
	for i := start; i < end; i++ {
		log.Printf("I:%d", i)
		start := time.Now()
		dchain := chain.GetData(i)
		dDB := db.GetData(i)
		err := compare(dchain, dDB, internal, chainName)
		if err != nil {
			log.Fatalf("check err:%v,bnum:%d", err, i)
		}
		log.Printf("finished bnum:%d,cost:%d", i, time.Since(start)/time.Millisecond)
	}

}
