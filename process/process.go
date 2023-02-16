package process

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chainmonitor/db"
	"xorm.io/xorm"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/fetch"
	kafkalog "github.com/chainmonitor/log"
	"github.com/chainmonitor/output/mysqldb"
	"github.com/chainmonitor/task"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

type Process struct {
	config    *config.Config
	db        db.IDB
	monitorDb db.IDB

	client  *rpc.Client
	fetcher *fetch.Fetcher
}

func NewProcess(config *config.Config) (*Process, error) {
	fetchConf := config.Fetch

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// client, err := rpc.DialContext(ctx, fetchConf.RpcURL)
	client, err := rpc.DialHTTPWithClient(fetchConf.RpcURL, &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
		Timeout: fetchConf.FetchTimeout,
	})
	if err != nil {
		kafkalog.Fatal(fmt.Errorf("create eth client err:%v", err))
	}
	monitorOutput, err := mysqldb.NewMysqlDb(config.Monitor.DB, config.OutPut.SqlBatch)
	if err != nil {
		logrus.Fatalf("create monitor mysql output err:%v", err)
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

	if err := checkGenesisBlock(f, mysqlOutput.GetEngine()); err != nil {
		logrus.Fatalf("check genesisBlock err %v", err)
	}

	p := &Process{
		config:    config,
		db:        mysqlOutput,
		monitorDb: monitorOutput,
		client:    client,
		fetcher:   f,
	}

	return p, nil
}

func (p *Process) Run() {

	// blkFeed := p.fetcher.blockFeed

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	tasks, err := allTasks(p)
	if err != nil {
		logrus.Fatalf("c task err:%v", err)
	}

	for _, t := range tasks {
		go t.Start(p.fetcher)
	}
	logrus.Infof("fetch run start:%d,end:%d", p.config.Fetch.StartHeight, p.config.Fetch.EndHeight)
	go p.fetcher.Run(p.config.Fetch.StartHeight, p.config.Fetch.EndHeight)

	sig := <-sigCh
	logrus.Infof("terminate process sig:%d", sig)

	ctx, cancle := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancle()
	go func(ctx context.Context) {
		//stop fetcher first
		p.fetcher.Stop()
		for _, t := range tasks {
			t.Stop()
		}
		sigCh <- syscall.SIGINT
	}(ctx)

	select {
	case <-ctx.Done():
		logrus.Warnf("force terminate")
	case <-sigCh:
		logrus.Infof("all tasks terminated")
	}
}

func allTasks(p *Process) (tasks []task.ITask, err error) {
	baseData := task.NewBaseStorageTask(p.config, p.client, p.db, p.monitorDb)
	tasks = []task.ITask{baseData}
	//context := context.Background()

	//latestBlockNumberOnstart := utils.HandleErrorWithRetryAndReturn(func() (interface{}, error) {
	//	return p.fetcher.Client.BlockNumber(context)
	//}, p.config.Fetch.FetchRetryTimes, p.config.Fetch.FetchRetryInterval)

	for _, taskName := range p.config.Tasks {
		var t task.ITask
		switch taskName {
		case task.BalanceTaskName:
			t, err = task.NewBalanceTask(p.config, p.client, p.db, p.monitorDb)
		case task.Erc20TxTaskName:
			t, err = task.NewErc20TxTask(p.config, p.client, p.db, p.monitorDb)
		case task.Erc20BalanceTaskName:
			t, err = task.NewErc20BalanceTask(p.config, p.client, p.db, p.monitorDb)
		case task.Erc721TaskName:
			t, err = task.NewErc721Task(p.config, p.client, p.db, p.monitorDb)
		case task.DexPairTaskName:
			t, err = task.NewDexPairTask(p.config, p.client, p.db, p.monitorDb)
		case task.Erc1155TaskName:
			t, err = task.NewErc1155Task(p.config, p.client, p.db, p.monitorDb)
		case task.Erc1155BalanceTaskName:
			t, err = task.NewErc1155BalanceTask(p.config, p.client, p.db, p.monitorDb)
		case task.PushKafkaTaskName:
			producer, err0 := task.NewSyncProducer(p.config.Kafka)
			if err != nil {
				logrus.Error(err0)
				return nil, fmt.Errorf("new kafka produer err:%v", err0)
			}
			t, err = task.NewPushKafkaTask(p.config, p.client, p.db, producer, p.monitorDb)
		case task.BlockDelayTaskName:
			producer, err0 := task.NewSyncProducer(p.config.Kafka)
			if err != nil {
				logrus.Error(err)
				return nil, fmt.Errorf("new kafka produer err:%v", err0)
			}
			conf := p.config.BlockDelay
			t, err = task.NewDelayTask(task.BlockDelayTaskName, &conf, p.db, producer)
		default:
			err = fmt.Errorf("invalid task name taskName: %s,err:%v", taskName, err)
		}
		if err != nil {
			logrus.Error(err)
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return
}

func checkGenesisBlock(fetcher *fetch.Fetcher, db *xorm.Engine) error {

	b, err := fetcher.GetGenesisBlock()
	if err != nil {
		logrus.Warnf("get genesis block from remote failed: %v", err)
		os.Exit(0)
		return nil
	}

	blk := mysqldb.ConvertInBlock(b.ToBlock())

	var modelGenesisBlock mysqldb.Block
	exist, err := db.Where("num = 0").Get(&modelGenesisBlock)
	if err != nil {
		logrus.Errorf("get genesis block from database failed")
		os.Exit(0)
	}

	if !exist {
		modelGenesisBlock := &mysqldb.Block{
			Number:    blk.Number,
			BlockHash: blk.BlockHash,

			Difficulty:      blk.Difficulty,
			ExtraData:       blk.ExtraData,
			GasLimit:        blk.GasLimit,
			GasUsed:         blk.GasUsed,
			Miner:           blk.Miner,
			ParentHash:      blk.ParentHash,
			Size:            blk.Size,
			ReceiptsRoot:    blk.ReceiptsRoot,
			StateRoot:       blk.StateRoot,
			TxsCnt:          blk.TxsCnt,
			BlockTimestamp:  blk.BlockTimestamp,
			UnclesCnt:       blk.UnclesCnt,
			State:           uint8(blk.State),
			BaseFee:         blk.BaseFee,
			BurntFees:       blk.BurntFees,
			TotalDifficulty: blk.TotalDifficulty,
			Nonce:           blk.Nonce,
		}
		if _, err := db.Insert(modelGenesisBlock); err != nil {
			logrus.Errorf("write genesis block failed. %v", err)
			os.Exit(0)
		}

		logrus.Infof("write genesis block success.")

	} else {
		if blk.BlockHash != modelGenesisBlock.BlockHash {
			logrus.Errorf("genesis block not equal. %v %v", blk.BlockHash, modelGenesisBlock.BlockHash)
			os.Exit(0)
		}
	}

	return nil
}
