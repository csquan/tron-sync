package main

import (
	"fmt"
	"net/http"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/fetch"
	kafkalog "github.com/chainmonitor/log"
	"github.com/chainmonitor/output/mysqldb"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

type Process struct {
	config *config.Config
	db     *mysqldb.MysqlDB

	client  *rpc.Client
	fetcher *fetch.Fetcher
}

func newProcess(config *config.Config) (*Process, error) {
	fetchConf := config.Fetch

	client, err := rpc.DialHTTPWithClient(fetchConf.RpcURL, &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: fetchConf.FetchTimeout,
	})
	if err != nil {
		kafkalog.Fatal(fmt.Errorf("create eth client err:%v", err))
	}
	mysqlOutput, err := mysqldb.NewMysqlDb(config.OutPut.DB, config.OutPut.SqlBatch)
	if err != nil {
		logrus.Fatalf("create mysql output err:%v", err)
	}

	logrus.Infof("init fetch block_worker:%d,tx_worker:%d,tx_batch:%d", fetchConf.BlockWorker, fetchConf.TxWorker, fetchConf.TxBatch)

	f, err := fetch.NewFetcher(client, mysqlOutput, &fetchConf, true, config.AppName)
	if err != nil {
		logrus.Fatalf("create fetcher failed:%v", err)
	}

	p := &Process{
		config:  config,
		db:      mysqlOutput,
		client:  client,
		fetcher: f,
	}

	return p, nil
}

func (p *Process) Run(startHeight, endHeight uint64) {

	t, err := NewErc20BalanceTask(p.config, p.client, p.db, startHeight-1, endHeight)
	if err != nil {
		logrus.Fatalf("create task error:%v", err)
		return
	}

	last := startHeight
	for t.curHeight < t.endHeight {
		if t.curHeight-last > 1000 {
			last = t.curHeight
			logrus.Infof("cur process height: %v ", t.curHeight)
		}
		t.fixHistoryData()
	}

}
