package task

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/starslabhq/chainmonitor/output/mysqldb"

	//"encoding/json"
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/starslabhq/chainmonitor/ktypes"
	"gotest.tools/assert"
)

var wg sync.WaitGroup
var kafka_url = "kafka2-test1.sinnet.huobiidc.com:9092"
var topic = "test_topic"
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
			if lastpushheight != 0 && lastpushheight+1 != blk.Number {
				logrus.Warnf("push kafka block height :%d error and last push height:%d", blk.Number, lastpushheight)
			}
			lastpushheight = blk.Number
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

func producerMsg(kafka_url string, topic string, send_value_chan chan string) {
	producer_config := sarama.NewConfig()
	producer_config.Producer.Return.Successes = true
	producer_config.Producer.Timeout = 5 * time.Second
	//none/gzip/snappy/lz4/ZStandard
	producer_config.Producer.Compression = 2

	producer_config.Producer.MaxMessageBytes = 5242880 //5M
	producer, err := sarama.NewSyncProducer(strings.Split(kafka_url, ","), producer_config)
	if err != nil {
		fmt.Println("NewSyncProducer error=", err)
	}

	//从数据库读取一个blk发送
	var start uint64 = 12846330
	mysqlOutput, err := mysqldb.NewMysqlDb(db_url, 500)
	if err != nil {
		logrus.Fatalf("create mysql output err:%v", err)
	}

	fmt.Printf("开始取出区块start:%d\n", start)
	blk, _ := mysqlOutput.GetBlocksByRange(start, start, mysqldb.DefaultFullBlockFilter)

	if len(blk) == 0 {
		fmt.Printf("从db中取出区块为空")
		return
	}

	kafkablk := blockformat(blk[0], 10)

	send_value, err := json.Marshal(kafkablk)
	if err != nil {
		fmt.Println("生成json字符串错误")
	}

	sendValue := string(send_value)

	//发送的消息,主题,key
	msg := sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(sendValue),
	}

	//sendSize := strings.Count(sendValue, "")
	length := msg.Value.Length()
	fmt.Printf("推送区块,长度为%d\n", length)
	PushKafka(producer, msg)
	send_value_chan <- sendValue
}

func TestKafkaBlk(t *testing.T) {
	blk_chan := make(chan string, 1)
	send_value_chan := make(chan string, 1)
	//生产
	go producerMsg(kafka_url, topic, send_value_chan)
	//消费
	go consumerKafka(kafka_url, topic, blk_chan)

	receiveValue := <-blk_chan
	sendValue := <-send_value_chan

	assert.Equal(t, receiveValue, sendValue, "block value send and receive by kafka not equal")
}
