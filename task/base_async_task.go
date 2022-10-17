package task

import (
	"container/list"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
	"github.com/starslabhq/chainmonitor/config"
	"github.com/starslabhq/chainmonitor/db"
	"github.com/starslabhq/chainmonitor/mtypes"
	"github.com/starslabhq/chainmonitor/output/mysqldb"
	"github.com/starslabhq/chainmonitor/subscribe"
)

type BaseAsyncTask struct {
	name   string
	task   *mysqldb.AsyncTask
	config *config.Config

	client *rpc.Client

	blockChan chan *mtypes.Block
	sub       event.Subscription
	//TODO sub
	bufferSize int
	cache      *list.List

	db db.IDB

	curHeight    uint64
	latestHeight uint64

	compensationChan chan int
	compensationFunc func()
	consumeFunc      func(*mtypes.Block)
	revertFunc       func(blk *mtypes.Block) //处理 number >=blk.Number的回滚
	stoped           int32
	stopChan         chan interface{}
}

func newBase(taskName string, config *config.Config, client *rpc.Client, db db.IDB,
	bufferSize int, consumeFunc func(*mtypes.Block), compensationFunc func(),
	revertFunc func(*mtypes.Block)) (b *BaseAsyncTask, err error) {

	b = &BaseAsyncTask{
		name:      taskName,
		config:    config,
		client:    client,
		db:        db,
		blockChan: make(chan *mtypes.Block, bufferSize),

		bufferSize: bufferSize,
		cache:      list.New(),

		compensationChan: make(chan int, 10),
		compensationFunc: compensationFunc,
		consumeFunc:      consumeFunc,
		revertFunc:       revertFunc,
		stopChan:         make(chan interface{}),
	}

	b.task, err = db.GetTaskByName(taskName)
	if err != nil {
		logrus.Warnf("get task err:%v", err)
		atomic.StoreInt32(&b.stoped, 1)
		return
	}
	logrus.Infof("task:%s ,task:%v,err:%v", taskName, b.task, err)
	if b.task != nil {
		//TODO b.curHeight=max(task.Number,db.MinNumber)
		b.curHeight = b.task.Number
	}

	return b, nil
}

func (b *BaseAsyncTask) Start(s subscribe.Subscriber) {
	if b.task == nil {
		atomic.StoreInt32(&b.stoped, 1)
		return
	}
	// loop read msg from upstream
	logrus.Debugf("task start name:%s", b.name)
	// b.sub = feed.Subscribe(b.blockChan)
	b.sub = s.SubBlock(b.blockChan)
	// b.revertSub = s.SubRevert(b.revertCh)

	for {
		if b.task.EndNumber != 0 && b.curHeight >= b.task.EndNumber {
			logrus.Infof("unscribe task:%s,cur:%d,end:%d", b.name, b.curHeight, b.task.EndNumber)
			b.sub.Unsubscribe()
			// b.revertSub.Unsubscribe()
			atomic.StoreInt32(&b.stoped, 1)
			return
		}
		// logrus.Infof("curheight:%d,latest:%d,stsize:%d", b.curHeight, b.latestHeight, len(b.stopChan))
		if b.curHeight < b.latestHeight {
			b.compensationChan <- 1
		}

		select {
		case <-b.stopChan:
			logrus.Infof("%v task terminated", b.name)
			atomic.StoreInt32(&b.stoped, 1)
			return
		case <-b.compensationChan:
			//clean up
			for len(b.compensationChan) > 0 {
				<-b.compensationChan
			}
			logrus.Debugf("run catchup name:%s", b.name)
			st := time.Now()
			//important!!! do not block too long time
			b.compensationFunc()
			logrus.Debugf("hisotry cost:%d,taskName:%s", time.Since(st)/time.Millisecond, b.name)
		case blk := <-b.blockChan:
			for {
				switch blk.State {
				case mysqldb.Block_ok:
					b.latestHeight = blk.Number      //latestHeight不再作为此处的判断依据，赋值只是为了给consumeFunc和compensationFunc函数内部使用。
					if b.curHeight+1 == blk.Number { //使用curHeight + 1进行判断，无论什么情况都不会发生重复写入。
						logrus.Debugf("do consume recv num:%d, task:%v", blk.Number, b.name)
						st := time.Now()
						b.consumeFunc(blk)
						logrus.Debugf("consumer cost:%d,taskName:%s,bnum:%d,txsize:%d", time.Since(st)/time.Millisecond, b.name, blk.Number, len(blk.Txs))
					} else if b.curHeight+1 < blk.Number {
						b.addBlockToCache(blk)
						//} else {
						//b.curHeight + 1 > blk.Number
						//发生此种情况有两种可能
						//1.回滚过程中重启
						//2.task的处理速度快于base，此时重启。
					}
				case mysqldb.Block_revert:
					if b.curHeight >= blk.Number {
						b.removeBlocksFromCache(blk.Number)
						b.revertFunc(blk)
						b.curHeight = blk.Number - 1
					}
					b.latestHeight = blk.Number - 1
				}
				if len(b.blockChan) == 0 {
					break
				}
				blk = <-b.blockChan
				logrus.Debugf("buf ch recv num:%d", blk.Number)
			}
		}
	}
}

func (b *BaseAsyncTask) Stop() {
	if atomic.LoadInt32(&b.stoped) != 1 {
		logrus.Debugf("task stop name: %s", b.name)
		b.stopChan <- 1
		logrus.Debugf("task stop finish name: %s", b.name)
	}
}

func (b *BaseAsyncTask) getBlkInfo(height uint64, filter *mysqldb.BlockFilter) (blk *mtypes.Block, err error) {
	blk, err = b.getBlkFromCache(height)
	if blk != nil || err != nil {
		return
	}

	blk, err = b.db.GetBlockByNum(height, filter)

	return
}

func (b *BaseAsyncTask) getBlkFromCache(height uint64) (blk *mtypes.Block, err error) {
	if b.cache.Len() == 0 {
		return nil, nil
	}

	if height >= b.cache.Front().Value.(*mtypes.Block).Number {
		cur := b.cache.Front()
		for cur != nil {
			next := cur.Next()
			b.cache.Remove(cur)

			if cur.Value.(*mtypes.Block).Number == height {
				return cur.Value.(*mtypes.Block), nil
			}
			cur = next
		}
	}

	return nil, nil
}

func (b *BaseAsyncTask) addBlockToCache(blk *mtypes.Block) {
	size := b.cache.Len()

	if size >= b.bufferSize {
		b.cache.Remove(b.cache.Front())
	}

	b.cache.PushBack(blk)
}

func (b *BaseAsyncTask) removeBlocksFromCache(revertHeight uint64) {
	for e := b.cache.Back(); e != nil; {
		block, _ := e.Value.(*mtypes.Block)
		if block.Number < revertHeight {
			break
		}

		b.cache.Remove(e)
		e = e.Prev()
	}
}
