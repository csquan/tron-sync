package kafka

import (
	"math/big"
	"time"

	"github.com/Shopify/sarama"
	"github.com/chainmonitor/config"
	"github.com/sirupsen/logrus"
)

type PushKafkaService struct {
	Producer sarama.SyncProducer
	Topic    string
}

func NewSyncProducer(config config.Kafka) (pro sarama.SyncProducer, e error) {
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Timeout = 5 * time.Second
	producerConfig.Producer.MaxMessageBytes = config.ProducerMax
	//none/gzip/snappy/lz4/ZStandard
	producerConfig.Producer.Compression = 2
	brokers := config.Brokers
	p, err := sarama.NewSyncProducer(brokers, producerConfig)
	if err != nil {
		logrus.Infof("NewSyncProducer error=%v", err)
	}
	return p, err
}

func NewPushKafkaService(config *config.Config, p sarama.SyncProducer) (*PushKafkaService, error) {
	b := &PushKafkaService{}

	b.Topic = config.PushBlk.Topic

	b.Producer = p

	return b, nil
}
func bigIntToStr(v *big.Int) string {
	if v == nil {
		return ""
	}
	return v.String()
}

func (b *PushKafkaService) Pushkafka(value []byte, topic string) error {
	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(value),
	}

	length := msg.Value.Length()
	logrus.Infof("before compension push block data size :%d", length)

	pid, offset, err := b.Producer.SendMessage(msg)
	if err != nil {
		logrus.Errorf("block-send err:%v", err)
	}
	logrus.Infof("send success pid:%v offset:%v\n", pid, offset)
	return err
}
