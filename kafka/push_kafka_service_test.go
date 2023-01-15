package kafka

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"sync"

	//"encoding/json"
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/chainmonitor/ktypes"
	"gotest.tools/assert"
)

var wg sync.WaitGroup
var kafka_url = "127.0.0.1:9092"
var topic = "test1"
var db_url = "root:12345678@tcp(huobichain-dev-01.sinnet.huobiidc.com:3306)/heco_data_test?charset=utf8mb4"

func consumerKafka(kafka_url string, topic string, blk_chan chan string) {

	consumer, err := sarama.NewConsumer([]string{kafka_url}, nil)
	if err != nil {
		fmt.Println("consumer connect err:", err)
		return
	}
	defer consumer.Close()

	partitions, err := consumer.Partitions(topic)
	if err != nil {
		fmt.Println("get partitions failed, err:", err)
		return
	}

	for _, p := range partitions {
		//sarama.OffsetNewest：从当前的偏移量开始消费，sarama.OffsetOldest：从最老的偏移量开始消费
		partitionConsumer, err := consumer.ConsumePartition(topic, p, sarama.OffsetNewest)
		if err != nil {
			fmt.Println("partitionConsumer err:", err)
			continue
		}
		wg.Add(1)
		for m := range partitionConsumer.Messages() {
			//fmt.Printf("key: %s, text: %s, offset: %d\n", string(m.Key), string(m.Value), m.Offset)
			blk_chan <- string(m.Value)
			blk := ktypes.Block{}
			_ = json.Unmarshal(m.Value, &blk)
			//fmt.Print(blk)
		}
		wg.Done()
	}
	wg.Wait()
}

func PushKafka(producer sarama.SyncProducer, msg sarama.ProducerMessage) {
	pid, offset, err := producer.SendMessage(&msg)
	if err != nil {
		logrus.Errorf("block-send err:%v", err)
	}
	logrus.Infof("send success pid:%v offset:%v\n", pid, offset)
}

type KafkaConsumer struct {
	Node  string
	Topic string
}

func TestKafkaBlk(t *testing.T) {
	blk_chan := make(chan string, 1)
	send_value_chan := make(chan string, 1)
	//消费
	consumerKafka(kafka_url, topic, blk_chan)

	receiveValue := <-blk_chan
	sendValue := <-send_value_chan

	assert.Equal(t, receiveValue, sendValue, "block value send and receive by kafka not equal")
}
