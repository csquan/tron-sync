package task

import (
	"fmt"
	"time"

	"github.com/chainmonitor/output/mysqldb"
	"github.com/chainmonitor/subscribe"
	"github.com/chainmonitor/utils"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/db"
	"github.com/chainmonitor/mtypes"
	"github.com/ethereum/go-ethereum/event"
	"github.com/sirupsen/logrus"
)

type BaseStorageTask struct {
	config    *config.Config
	blockChan chan *mtypes.Block
	sub       event.Subscription

	curHeight uint64

	stopChan chan interface{}
	db       db.IDB
}

func NewBaseStorageTask(config *config.Config, db db.IDB) *BaseStorageTask {
	return &BaseStorageTask{
		config:    config,
		db:        db,
		stopChan:  make(chan interface{}),
		blockChan: make(chan *mtypes.Block, config.BaseStorage.BufferSize),
	}
}

func (b *BaseStorageTask) Start(s subscribe.Subscriber) {
	// b.sub = feed.Subscribe(b.blockChan)

	b.sub = s.SubBlock(b.blockChan)
	// b.revertSub = s.SubRevert(b.blockRevertCh)
	var err error
	//TODO
	if b.config.Fetch.StartHeight != 0 {
		b.curHeight = b.config.Fetch.StartHeight - 1
	} else {
		b.curHeight, err = b.db.GetCurrentMainBlockNum()
		if err != nil {
			logrus.Fatalf("can not get current block number from db: %v", err)
		}
	}

	blocks := make([]*mtypes.Block, 0)
	acc := []int{0, 0}
	timer := time.NewTimer(time.Millisecond * time.Duration(b.config.BaseStorage.MaxInterval))

	for {
		select {
		case <-b.stopChan:
			logrus.Info("base data storage task terminated")
			return

		case <-timer.C:
			if len(blocks) > 0 {
				logrus.Debugf("time to handle block. blk count:%v tx count:%v log count:%v", len(blocks), acc[0], acc[1])
				utils.HandleErrorWithRetry(func() error {
					return b.saveBlocks(blocks)
				}, b.config.OutPut.RetryTimes, b.config.OutPut.RetryInterval)

				blocks = make([]*mtypes.Block, 0)
				acc = []int{0, 0}
			}

			if !timer.Stop() && len(timer.C) > 0 {
				<-timer.C
			}
			timer.Reset(time.Millisecond * time.Duration(b.config.BaseStorage.MaxInterval))
		case blk := <-b.blockChan:
			if blk == nil {
				logrus.Warn("received nil block")
				continue
			}

			if blk.Number > b.curHeight+1 {
				logrus.Fatalf("unexpected block. db block num:%v, received block num:%v, hash:%v", b.curHeight, blk.Number, blk.Hash)
				continue
			}

			logrus.Debugf("base data cur height:%d", b.curHeight)

			switch blk.State {
			case mysqldb.Block_ok:
				if blk.Number != b.curHeight+1 {
					logrus.Fatalf("normal block unexpected height cur:%d,blk:%v", b.curHeight, blk)
				}
				blocks = append(blocks, blk)
				acc[0] += blk.TxCnt
				for _, t := range blk.Txs {
					acc[1] += len(t.EventLogs)
				}

				if len(blocks) > b.config.BaseStorage.MaxBlockCount || acc[0] > b.config.BaseStorage.MaxTxCount ||
					acc[1] > b.config.BaseStorage.MaxLogCount {
					//write data to db

					logrus.Debugf("batch handle block. blk count:%v tx count:%v log count:%v", len(blocks), acc[0], acc[1])
					utils.HandleErrorWithRetry(func() error {
						return b.saveBlocks(blocks)
					}, b.config.OutPut.RetryTimes, b.config.OutPut.RetryInterval)

					blocks = make([]*mtypes.Block, 0)
					acc = []int{0, 0}
				}
				b.curHeight = blk.Number

			case mysqldb.Block_revert:
				if len(blocks) != 0 {
					// still have normal blocks not saved
					utils.HandleErrorWithRetry(func() error {
						return b.saveBlocks(blocks)
					}, b.config.OutPut.RetryTimes, b.config.OutPut.RetryInterval)

					blocks = make([]*mtypes.Block, 0)
					acc = []int{0, 0}
				}
				if blk.Number != b.curHeight {
					logrus.Fatalf("revert block unexpected cur:%d,blk:%v", b.curHeight, blk)
				}
				utils.HandleErrorWithRetry(func() error {
					return b.db.UpdateBlockSate(blk.Number, mysqldb.Block_revert)
				}, b.config.OutPut.RetryTimes, b.config.OutPut.RetryInterval)
				b.curHeight = blk.Number - 1
				logrus.Warnf("base data do revert num:%d,hash:%s", blk.Number, blk.Hash)
			default:
				logrus.Fatalf("block state unexpected b:%v", blk)
			}

			if !timer.Stop() && len(timer.C) > 0 {
				<-timer.C
			}
			timer.Reset(time.Millisecond * time.Duration(b.config.BaseStorage.MaxInterval))
		}
	}
}

func (b *BaseStorageTask) Stop() {
	b.stopChan <- 1
}

// saveBlocks save blocks and related informations into DB
func (b *BaseStorageTask) saveBlocks(blocks []*mtypes.Block) error {
	start := time.Now()
	defer func() {
		logrus.Infof("block size:%d,number:%d,cost:%v", len(blocks), blocks[0].Number, time.Since(start))
	}()
	session := b.db.GetSession()
	defer session.Close()

	for _, block := range blocks {
		bexist, err := b.db.GetBlockByNumAndState(block.Number, block.State)
		if err != nil {
			return fmt.Errorf("get exist block err:%v,num:%d,state:%d", err, block.Number, block.State)
		}
		if bexist != nil {
			logrus.Warnf("block already commited num:%d,state:%d", block.Number, block.State)
			continue
		}
		err = session.Begin()
		if err != nil {
			return fmt.Errorf("session beigin err:%v,blk num:%d", err, blocks[0].Number)
		}
		var (
			blockdbs    []*mysqldb.Block
			txdbs       []*mysqldb.TxDB
			txLogs      []*mysqldb.TxLog
			txInternals []*mysqldb.TxInternal
			contracts   []*mysqldb.Contract
		)
		dbBlock := mysqldb.ConvertInBlock(block)
		blockdbs = append(blockdbs, dbBlock)

		for _, tx := range block.Txs {
			txdb := mysqldb.ConvertInTx(0, block, tx)
			txdbs = append(txdbs, txdb)

			//tx_log
			for _, tlog := range tx.EventLogs {
				txLogDB := mysqldb.ConvertInLog(0, block, tx, tlog)
				txLogs = append(txLogs, txLogDB)
			}
		}

		// tx_internal
		if block.TxInternals != nil {
			for _, v := range block.TxInternals {
				txInternal := mysqldb.ConvertInInternalTx(0, block, v)
				txInternals = append(txInternals, txInternal)
			}
		}
		// contract structure convert
		if block.Contract != nil {
			for _, v := range block.Contract {
				contract := mysqldb.ConvertInContract(0, block, v)
				contracts = append(contracts, contract)
			}
		}

		err = b.db.SaveBlocks(session, blockdbs)
		if err != nil {
			err2 := session.Rollback()
			if err2 != nil {
				logrus.Errorf("rollback session error:%v", err2)
			}
			return fmt.Errorf("db insert blocks err:%v", err)
		}

		// save internal transactions into database
		err = b.db.SaveInternalTxs(session, txInternals)
		if err != nil {
			err2 := session.Rollback()
			if err2 != nil {
				logrus.Errorf("rollback session error:%v", err2)
			}
			return fmt.Errorf("db insert tx_internals err:%v", err)
		}

		// save transactions into database
		err = b.db.SaveTxs(session, txdbs)
		if err != nil {
			err2 := session.Rollback()
			if err2 != nil {
				logrus.Errorf("rollback session error:%v", err2)
			}
			return fmt.Errorf("db insert txs err:%v", err)
		}

		// save logs into databases
		err = b.db.SaveLogs(session, txLogs)
		if err != nil {
			err2 := session.Rollback()
			if err2 != nil {
				logrus.Errorf("rollback session error:%v", err2)
			}
			return fmt.Errorf("db insert tx_logs err:%v", err)
		}

		// save contracts into database
		if err = b.db.SaveContracts(session, contracts); err != nil {
			if err2 := session.Rollback(); err != nil {
				logrus.Errorf("rollback session error:%v", err2)
			}
			return fmt.Errorf("db insert contract err:%v", err)
		}

		err = session.Commit()
		if err != nil {
			if err1 := session.Rollback(); err1 != nil {
				logrus.Errorf("block rollback err:%v,num:%d", err1, blocks[0].Number)
			}
			return fmt.Errorf("block commit err:%v,bnum:%d", err, blocks[0].Number)
		}

	}
	return nil
}
