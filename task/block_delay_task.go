package task

import (
	"container/list"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Shopify/sarama"
	"github.com/ethereum/go-ethereum/event"
	"github.com/sirupsen/logrus"
	"github.com/starslabhq/chainmonitor/config"
	"github.com/starslabhq/chainmonitor/db"
	"github.com/starslabhq/chainmonitor/mtypes"
	"github.com/starslabhq/chainmonitor/output/mysqldb"
	"github.com/starslabhq/chainmonitor/subscribe"
)

const BlockDelayTaskName = "block_send_delay"

type queue struct {
	cap int
	l   *list.List
}

func newQueue(cap int) *queue {
	return &queue{
		cap: cap,
		l:   list.New(),
	}
}

func (q *queue) isFull() bool {
	return q.l.Len() == q.cap
}

func (q *queue) isEmpty() bool {
	return q.l.Len() == 0
}

func (q *queue) front() *mtypes.Block {
	e := q.l.Front()
	if e == nil {
		return nil
	}
	return e.Value.(*mtypes.Block)
}

func (q *queue) deQueue() {
	if !q.isEmpty() {
		front := q.l.Front()
		q.l.Remove(front)
	}
}
func (q *queue) enQueue(b *mtypes.Block) {
	if !q.isFull() {
		q.l.PushBack(b)
	}
}
func (q *queue) rear() *mtypes.Block {
	e := q.l.Back()
	if e == nil {
		return nil
	}
	return e.Value.(*mtypes.Block)
}

func (q *queue) findElement(num uint64) *list.Element {
	for e := q.l.Front(); e != nil; e = e.Next() {
		v := e.Value.(*mtypes.Block)
		if v.Number == num {
			return e
		}
	}
	return nil
}

type DelayTask struct {
	topic      string
	name       string
	q          *queue
	sub        event.Subscription
	blockCh    chan *mtypes.Block
	stopCh     chan struct{}
	queryBatch int
	delay      uint64
	cur        uint64
	latest     uint64
	catchUP    chan struct{}
	db         db.IDB
	p          sarama.SyncProducer
	config     *config.BlockDelay
}

func NewDelayTask(name string, conf *config.BlockDelay, db db.IDB, p sarama.SyncProducer) (*DelayTask, error) {
	if conf.Topic == "" {
		return nil, fmt.Errorf("delay-send topic empty")
	}
	t := &DelayTask{
		topic:      conf.Topic,
		name:       name,
		delay:      uint64(conf.Delay),
		q:          newQueue(conf.Delay + 1), //cache 当前一个区块+delay个区块 rear.Number=front.Number+dealy
		blockCh:    make(chan *mtypes.Block, conf.BufferSize),
		stopCh:     make(chan struct{}),
		queryBatch: conf.BatchBlockCount,
		catchUP:    make(chan struct{}, 1),
		db:         db,
		p:          p,
		config:     conf,
	}
	task, err := t.db.GetTaskByName(t.name)
	if err != nil {
		return nil, err
	}
	t.cur = task.Number
	return t, nil
}

func (t *DelayTask) sendBlock(b *mtypes.Block) error {
	blockForSend := blockformat(b, t.config.TruncateLength)
	encoded, err := json.Marshal(blockForSend)
	if err != nil {
		return fmt.Errorf("delay-send json encode err:%v,b+%v", err, b)
	}
	msg := &sarama.ProducerMessage{
		Topic:     t.topic,
		Value:     sarama.ByteEncoder(encoded),
		Partition: 0,
	}
	partition, offset, err := t.p.SendMessage(msg)
	logrus.Infof("delay-send %d,latest:%d,partition:%d,offset:%d,err:%v",
		b.Number, t.latest, partition, offset, err)
	if err != nil {
		return fmt.Errorf("kafka send err: %v", err)
	}
	return nil
}

func (t *DelayTask) getBlocks(start, end uint64) ([]*mtypes.Block, error) {
	blocks, err := t.db.GetBlocksByRange(start, end, mysqldb.DefaultFullBlockFilter)
	return blocks, err
}

func (t *DelayTask) getBlockByNum(num uint64) (*mtypes.Block, error) {
	b, err := t.db.GetBlockByNum(num, mysqldb.DefaultFullBlockFilter)
	return b, err
}

func (t *DelayTask) getBlcok(num uint64) (*mtypes.Block, error) {
	// if t.q.isFull() {
	// 	return t.q.front(), nil
	// } else {
	// 	b, err := t.getBlockByNum(num)
	// 	return b, err
	// }
	if t.q.isFull() {
		logrus.Debugf("cache_full num:%d,front:%d,rear:%d", num, t.q.front().Number, t.q.rear().Number)
		return t.q.front(), nil
	}
	logrus.Debugf("cache_notfull num:%d,front:%d,rear:%d", num, t.q.front().Number, t.q.rear().Number)
	e := t.q.findElement(num)
	if e != nil {
		return e.Value.(*mtypes.Block), nil
	} else {
		b, err := t.getBlockByNum(num)
		return b, err
	}
}

// ---->front----->rear--->
func (t *DelayTask) addBlock(b *mtypes.Block) {
	if t.q.isEmpty() {
		t.q.enQueue(b)
		return
	}
	front := t.q.front()
	rear := t.q.rear()
	num := b.Number
	if num < front.Number {
		return
	} else if num > rear.Number {
		if num != rear.Number+1 {
			logrus.Fatalf("unexpected latest num latest:%d,num:%d", rear.Number, num)
		}
		if t.q.isFull() {
			t.q.deQueue()
		}
		t.q.enQueue(b)
	} else {
		e := t.q.findElement(b.Number)
		if e == nil {
			logrus.Fatalf("unexpectd bnum:%d", b.Number)
		}
		e.Value = b
	}
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	} else {
		return b
	}
}

func (t *DelayTask) Start(s subscribe.Subscriber) {
	t.sub = s.SubBlock(t.blockCh)
	for {
		curNum := t.cur + 1
		if t.latest > 0 {
			if !t.q.isEmpty() && curNum+t.delay < t.latest && len(t.catchUP) == 0 { //catching
				t.catchUP <- struct{}{}
			}
		}
		select {
		case <-t.stopCh:
			logrus.Infof("delay_task stop")
			return
		case block := <-t.blockCh:
			if block.State == 1 {
				continue
			}
			if block.Number < curNum {
				logrus.Warnf("overdelay blocks num:%d,cur:%d", block.Number, curNum-1)
				continue
			}

			t.addBlock(block)
			t.latest = t.q.rear().Number
			if curNum+t.delay == t.latest { //catched up
				b, err := t.getBlcok(curNum)
				if err != nil {
					logrus.Errorf("delay-send getBlock err:%v,num:%d", err, curNum)
				}
				if b != nil {
					err := t.sendBlock(b)
					if err == nil {
						err = t.db.UpdateAsyncTaskNumByName(t.db.GetEngine(), t.name, curNum)
						if err == nil {
							t.cur = curNum
						} else {
							logrus.Errorf("delay-send finish num:%d", t.cur)
						}
					} else {
						logrus.Errorf("delay-send err:%v,num:%d", err, b.Number)
					}
				} else {
					logrus.Warnf("get block nil num:%d", curNum)
				}
			}
		case <-t.catchUP:
			logrus.Infof("run catchup curNum:%d,latest:%d", curNum, t.latest)
			// for len(t.catchUP) > 0 {
			// 	<-t.catchUP
			// }

			end := min(curNum+uint64(t.queryBatch), t.latest-t.delay)
			if curNum > end {
				logrus.Warnf("catchup cur:%d,latest:%d,delay:%d", curNum, t.latest, t.delay)
				continue
			}
			blocks, err := t.getBlocks(curNum, end)
			sort.Slice(blocks, func(i, j int) bool {
				return blocks[i].Number < blocks[j].Number
			})
			if err != nil {
				logrus.Errorf("delay-send catchup err:%v,start:%d,end:%d", err, curNum, end)
				continue
			}
			for _, b := range blocks {
				err := t.sendBlock(b)
				if err == nil {
					err = t.db.UpdateAsyncTaskNumByName(t.db.GetEngine(), t.name, b.Number)
					if err == nil {
						t.cur = b.Number
					} else {
						logrus.Errorf("delay-send finish num:%d", b.Number)
					}
				} else {
					logrus.Errorf("delay-send err:%v,bnum:%d", err, b.Number)
					break
				}
			}
			if len(blocks) == 0 {
				err = t.db.UpdateAsyncTaskNumByName(t.db.GetEngine(), t.name, end)
				if err == nil {
					t.cur = end
				} else {
					logrus.Errorf("delay-send update cur err: %v,num:%d", err, end)
				}
			}
		}
	}
}

func (t *DelayTask) Stop() {
	close(t.stopCh)
}
