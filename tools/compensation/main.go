package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/log"
	"github.com/sirupsen/logrus"
)

var (
	confFile    string
	startHeight uint64
	endHeight   uint64
)

func init() {
	flag.StringVar(&confFile, "conf", "config.yaml", "conf file")
	flag.Uint64Var(&startHeight, "start", 0, "start height(include)")
	flag.Uint64Var(&endHeight, "end", 0, "end height(include)")
}

func main() {
	flag.Parse()
	fmt.Println(fmt.Sprintf("config file:%v, start height:%v, end height:%v", confFile, startHeight, endHeight))
	fmt.Println(confFile)
	conf, err := config.LoadConf(confFile, "test")
	if err != nil {
		fmt.Println(err.Error())
	}

	err = log.Init(conf.AppName, conf.LogConf)
	if err != nil {
		log.Fatal(err)
	}

	outputConf := conf.OutPut

	logrus.Infof("init output sql_batch:%d", outputConf.SqlBatch)

	p, err := newProcess(conf)
	if err != nil {
		logrus.Fatalf("new process err:%v", err)
	}

	p.Run(startHeight, endHeight)
}
