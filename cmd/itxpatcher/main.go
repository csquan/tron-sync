package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/starslabhq/chainmonitor/config"
	"github.com/starslabhq/chainmonitor/fetch"
	"github.com/starslabhq/chainmonitor/mtypes"
	"github.com/starslabhq/chainmonitor/output/mysqldb"
)

var env string
var confName string
var chain string
var conf *viper.Viper

func init() {
	flag.StringVar(&env, "env", "test", "Deploy environment: [ prod | test ]")
	flag.StringVar(&confName, "conf", "config.yaml", "configure file path")
	flag.StringVar(&chain, "chain", "", "chain name")
}

// buildFatcher return a ethereum rpc client
func buildFetcher() (*fetch.Fetcher, error) {
	rpcURL := conf.GetString("url")
	var timeout time.Duration = 2 * time.Second
	client, err := rpc.DialHTTPWithClient(rpcURL, &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: timeout,
	})
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	f, _ := fetch.NewFetcher(client, nil, nil, false, conf.GetString("app_name"))
	return f, nil
}

func buildDBConn() (*mysqldb.MysqlDB, error) {
	dburl := conf.GetString("db")
	var batch int = 5
	dbconn, err := mysqldb.NewMysqlDb(dburl, batch)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}
	return dbconn, nil
}

// transTraceAddressToString transform traceAddress array to Etherscan style, like: "call_0_0_5_2"
func transTraceAddressToString(opcode string, traceAddress []uint64) string {
	var res = strings.ToLower(opcode)
	for _, addr := range traceAddress {
		res = fmt.Sprintf("%s_%d", res, addr)
	}
	return res
}

func fetchInternalTx(dbconn *mysqldb.MysqlDB, f *fetch.Fetcher, blockNum uint64) ([]*mysqldb.TxInternal, []*mysqldb.Contract) {
	internaltxs, contracts := f.GetTxInternalsByBlock(new(big.Int).SetUint64(blockNum))
	if len(internaltxs) == 0 && len(contracts) == 0 {
		return nil, nil
	}
	block, err := dbconn.GetBlockByNum(blockNum, mysqldb.DefaultFullBlockFilter)
	if err != nil {
		logrus.Error(err)
		block = &mtypes.Block{}
		block.Number = blockNum
		block.TimeStamp = 0
	}

	dbTxInternal := []*mysqldb.TxInternal{}
	dbContracts := []*mysqldb.Contract{}

	for _, internalTx := range internaltxs {
		var value string
		if internalTx.Value != nil {
			value = internalTx.Value.String()
		}
		dbitx := &mysqldb.TxInternal{
			TxHash:       internalTx.TxHash,
			AddrFrom:     strings.ToLower(internalTx.From),
			AddrTo:       strings.ToLower(internalTx.To),
			OPCode:       internalTx.OPCode,
			Value:        value,
			Success:      internalTx.Success,
			Depth:        internalTx.Depth,
			Gas:          internalTx.Gas,
			GasUsed:      internalTx.GasUsed,
			Input:        internalTx.Input,
			Output:       internalTx.Output,
			TraceAddress: transTraceAddressToString(internalTx.OPCode, internalTx.TraceAddress),
			BlockNum:     block.Number,
			BlockTime:    block.TimeStamp,
		}
		dbTxInternal = append(dbTxInternal, dbitx)
	}

	for _, contract := range contracts {
		c := &mysqldb.Contract{
			TxHash:      contract.TxHash,
			Addr:        strings.ToLower(contract.Addr),
			CreatorAddr: strings.ToLower(contract.CreatorAddr),
			ExecStatus:  contract.ExecStatus,
			BlockNum:    block.Number,
			BlockState:  0,
		}
		dbContracts = append(dbContracts, c)
	}

	return dbTxInternal, dbContracts
}

func saveInternalTxsAndContracts(db *mysqldb.MysqlDB, blockNum uint64, internals []*mysqldb.TxInternal, contracts []*mysqldb.Contract) error {
	// begin
	// delete old record use block number or block hash
	// save Internal txs
	// save contracts
	// commit
	return db.RenewInternalTransactions(blockNum, internals, contracts)
}

// fetchWorkerPool
type fetchWorkerPool struct {
	sync.WaitGroup
	input     chan uint64
	workerNum int
	db        *mysqldb.MysqlDB
	fetcher   *fetch.Fetcher
	save      *saveWorkerPool
}

func NewFetchWorkerPool(workNum int, save *saveWorkerPool) *fetchWorkerPool {
	pool := &fetchWorkerPool{
		WaitGroup: sync.WaitGroup{},
	}
	pool.Init(workNum, save)
	return pool
}

func (wp *fetchWorkerPool) Init(workerNum int, save *saveWorkerPool) {
	var err error
	wp.workerNum = workerNum
	wp.input = make(chan uint64, 1)
	wp.db, err = buildDBConn()
	if err != nil {
		log.Fatal(err)
	}
	wp.fetcher, err = buildFetcher()
	if err != nil {
		log.Fatal(err)
	}
	wp.save = save
}

func (wp *fetchWorkerPool) work() {
	defer wp.Done()
	for i := range wp.input {
		// log.Printf("[begin] fetch block_num: %v", i)
		internals, contracts := fetchInternalTx(wp.db, wp.fetcher, i)
		if len(internals) == 0 && len(contracts) == 0 {
			continue
		}
		wp.save.Send([]interface{}{i, internals, contracts})
		// log.Printf("[ok] fetched block_num: %v", i)
	}
}

func (wp *fetchWorkerPool) Begin() {
	for i := 0; i < wp.workerNum; i++ {
		wp.Add(1)
		go wp.work()
	}
	wp.Wait()
}

func (wp *fetchWorkerPool) Send(blockNum uint64) {
	wp.input <- blockNum
}

func (wp *fetchWorkerPool) Stop() {
	close(wp.input)
}

// saveWorkerPool
type saveWorkerPool struct {
	sync.WaitGroup
	input     chan []interface{}
	workerNum int
	db        *mysqldb.MysqlDB
}

func NewSaveWorkerPool(workerNum int) *saveWorkerPool {
	save := &saveWorkerPool{
		WaitGroup: sync.WaitGroup{},
		input:     make(chan []interface{}, 2*workerNum+2),
	}
	save.Init(workerNum)
	return save
}

func (wp *saveWorkerPool) Init(workerNum int) {
	var err error
	wp.workerNum = workerNum
	wp.db, err = buildDBConn()
	if err != nil {
		logrus.Fatal(err)
	}
}

func (wp *saveWorkerPool) work() {
	defer wp.Done()
	for data := range wp.input {
		err := saveInternalTxsAndContracts(wp.db, data[0].(uint64), data[1].([]*mysqldb.TxInternal), data[2].([]*mysqldb.Contract))
		if err != nil {
			logrus.Error(err)
			time.Sleep(500 * time.Millisecond)
			go wp.Send(data)
		} else {
			logrus.Infof("[ok] save blocknum : %v, internaltx_num: %d, contract_num: %d",
				data[0].(uint64),
				len(data[1].([]*mysqldb.TxInternal)),
				len(data[2].([]*mysqldb.Contract)),
			)
		}
	}
}

func (wp *saveWorkerPool) Begin() {
	for i := 0; i < wp.workerNum; i++ {
		wp.Add(1)
		go wp.work()
	}
	wp.Wait()
}

func (wp *saveWorkerPool) Send(data []interface{}) {
	wp.input <- data
}

func (wp *saveWorkerPool) Stop() {
	close(wp.input)
}

func main() {

	flag.Parse()
	/*
		viper.AddConfigPath("./") //设置读取的文件路径
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		err := viper.ReadInConfig()
		if err != nil {
			log.Fatal(err)
		}
		url := viper.GetString("url")
		log.Println("rpc url", url)
		dburl := viper.GetString("db")
		log.Println("db url", dburl)
	*/
	conf = viper.New()
	conf.SetConfigType("yaml")
	if err := config.RemoteConfig(env, confName, conf); err != nil {
		log.Fatal(err)
	}
	conf = conf.Sub(chain)

	var i uint64
	var beginNum uint64 = conf.GetUint64("begin_num")
	var endNum uint64 = conf.GetUint64("end_num")
	fetchWorkerNum := conf.GetInt("fetch_worker_num")
	saveWorkerNum := conf.GetInt("save_worker_num")
	fmt.Println(conf.GetString("app_name"))
	fmt.Println(beginNum)
	fmt.Println(endNum)

	savePool := NewSaveWorkerPool(saveWorkerNum)
	fetchPool := NewFetchWorkerPool(fetchWorkerNum, savePool)

	var fName = `/tmp/huobi.lock`
	leaseAlive := func() {
		f, err := os.OpenFile(fName, os.O_RDWR|os.O_CREATE, os.ModePerm)
		if err != nil {
			panic(fmt.Sprintf("create alive file err:%v", err))
		}
		now := time.Now().Unix()
		fmt.Fprintf(f, "%d", now)

	}
	leaseAlive()

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		savePool.Begin()
	}()
	go func() {
		defer wg.Done()
		fetchPool.Begin()
		savePool.Stop()
	}()

	for i = beginNum; i <= endNum; i++ {
		log.Printf("send %v", i)
		fetchPool.Send(i)
	}

	fetchPool.Stop()

	for {
		log.Println("Done all the work")
		time.Sleep(60 * time.Second)
	}
	// wg.Wait()
}
