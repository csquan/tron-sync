package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/chainmonitor/log"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/process"
	"github.com/sirupsen/logrus"

	"net/http"
	_ "net/http/pprof"
)

var (
	conffile string
	env      string
)

func init() {
	flag.StringVar(&conffile, "conf", "config.yaml", "conf file")
	flag.StringVar(&env, "env", "prod", "Deploy environment: [ prod | test ]. Default value: prod")
}

func main() {

	flag.Parse()
	fmt.Println(conffile)
	conf, err := config.LoadConf(conffile, env)
	if err != nil {
		fmt.Println(err.Error())
		panic(fmt.Errorf("load conf err %v", err))
	}
	if conf.ProfPort != 0 {
		go func() {
			err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", conf.ProfPort), nil)
			if err != nil {
				panic(fmt.Sprintf("start pprof server err:%v", err))
			}
		}()
	}
	err = log.Init(conf.AppName, conf.LogConf)
	if err != nil {
		log.Fatal(err)
	}

	outputConf := conf.OutPut

	logrus.Infof("init output sql_batch:%d", outputConf.SqlBatch)

	p, err := process.NewProcess(conf)
	if err != nil {
		logrus.Fatalf("new process err:%v", err)
	}

	leaseAlive()
	p.Run()
	removeFile()
}

var fName = `/tmp/huobi.lock`

func removeFile() {
	os.Remove(fName)
}

func leaseAlive() {
	f, err := os.OpenFile(fName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		panic(fmt.Sprintf("create alive file err:%v", err))
	}
	now := time.Now().Unix()
	fmt.Fprintf(f, "%d", now)

}
