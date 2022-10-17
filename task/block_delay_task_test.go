package task

import (
	"testing"

	"github.com/ethereum/go-ethereum/event"
	"github.com/starslabhq/chainmonitor/config"
	"github.com/starslabhq/chainmonitor/mtypes"
)

func TestQueue(t *testing.T) {
	q := newQueue(5)
	q.enQueue(&mtypes.Block{Number: 1})
	e := q.findElement(1)
	e.Value = &mtypes.Block{
		Number: 2,
	}
	for e := q.l.Front(); e != nil; e = e.Next() {
		t.Logf("num:%d", e.Value.(*mtypes.Block).Number)
	}
}

type testPub func(chan *mtypes.Block) event.Subscription

func (tp testPub) SubBlock(ch chan *mtypes.Block) event.Subscription {
	return tp(ch)
}

func TestDelaySend(t *testing.T) {
	producer, err := NewSyncProducer(config.Kafka{
		ProducerMax: 1000000000,
		Brokers:     []string{"localhost:9092"},
	})
	if err != nil {
		t.Fatalf("c producer err:%v", err)
	}
	conf := &config.BlockDelay{
		Topic:           "block_delay_2",
		Delay:           5,
		BufferSize:      1,
		BatchBlockCount: 5,
		TruncateLength:  10,
	}
	task, err := NewDelayTask("delaySend", conf, dbtest, producer)
	if err != nil {
		t.Fatalf("new delayTask err:%v", err)
	}
	task.cur = 9305290
	// task.cur = 9305300
	feed := &event.Feed{}
	pub := func(ch chan *mtypes.Block) event.Subscription {
		sub := feed.Subscribe(ch)
		return sub
	}
	go func() {
		var num uint64 = 9305300
		for i := 0; i < 10; i++ {
			feed.Send(&mtypes.Block{
				Number: num + uint64(i),
			})
		}
		for i := 5; i < 20; i++ {
			feed.Send(&mtypes.Block{
				Number: num + uint64(i),
				Miner:  "revert_test",
			})
		}
	}()
	task.Start(testPub(pub))
}

func TestChSize(t *testing.T) {
	ch := make(chan int, 1)
	t.Logf("%d", len(ch))
	ch <- 1
	t.Logf("writed %d", len(ch))
}
