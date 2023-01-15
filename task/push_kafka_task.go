package task

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/Shopify/sarama"
	"github.com/chainmonitor/output/mysqldb"

	"github.com/chainmonitor/config"
	"github.com/chainmonitor/db"
	"github.com/chainmonitor/ktypes"
	"github.com/chainmonitor/mtypes"
	"github.com/chainmonitor/utils"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

const PushKafkaTaskName = "push_kafka"

var lastpushheight uint64 //检测区块发送是否连续

type PushKafkaTask struct {
	*BaseAsyncTask
	producer sarama.SyncProducer
	topic    string
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

func NewPushKafkaTask(config *config.Config, client *rpc.Client, db db.IDB, p sarama.SyncProducer) (*PushKafkaTask, error) {
	b := &PushKafkaTask{}

	b.topic = config.PushBlk.Topic

	b.producer = p

	base, err := newBase(PushKafkaTaskName, config, client, db, config.PushBlk.BufferSize,
		b.handleBlock, b.fixHistoryData, b.revertBlock)
	if err != nil {
		return nil, err
	}
	b.BaseAsyncTask = base

	return b, nil
}
func bigIntToStr(v *big.Int) string {
	if v == nil {
		return ""
	}
	return v.String()
}

func blockformat(blk *mtypes.Block, truncateLength int) (bk *ktypes.Block) {
	// defaultTruncateLength is the number of bytes a internal transcation input/output is truncated to.
	var defaultTruncateLength int = 10
	if truncateLength > 0 {
		defaultTruncateLength = truncateLength
	}

	var txsformat = make([]*ktypes.Tx, 0, len(blk.Txs))
	for _, tx := range blk.Txs {
		var eventlogsformat = make([]*ktypes.EventLog, 0, len(tx.EventLogs))
		for _, eventlog := range tx.EventLogs {
			eventlogformat := &ktypes.EventLog{
				Topic0: eventlog.Topic0,
				Topic1: eventlog.Topic1,
				Topic2: eventlog.Topic2,
				Topic3: eventlog.Topic3,
				Data:   eventlog.Data,
				Index:  eventlog.Index,
				Addr:   eventlog.Addr,
			}
			eventlogsformat = append(eventlogsformat, eventlogformat)
		}
		txformat := &ktypes.Tx{
			TxType:               tx.TxType,
			From:                 tx.From,
			To:                   tx.To,
			Hash:                 tx.Hash,
			Index:                tx.Index,
			Value:                bigIntToStr(tx.Value),
			Input:                tx.Input,
			Nonce:                tx.Nonce,
			GasPrice:             bigIntToStr(tx.GasPrice),
			GasLimit:             tx.GasLimit,
			GasUsed:              tx.GasUsed,
			BlockTime:            tx.BlockTime,
			BlockNum:             tx.BlockNum,
			BlockHash:            tx.BlockHash,
			ExecStatus:           tx.ExecStatus,
			EventLogs:            eventlogsformat,
			BaseFee:              bigIntToStr(tx.BaseFee),
			MaxFeePerGas:         bigIntToStr(tx.MaxFeePerGas),         //交易费上限
			MaxPriorityFeePerGas: bigIntToStr(tx.MaxPriorityFeePerGas), //小费上限
			BurntFees:            bigIntToStr(tx.BurntFees),            //baseFee*gasused
		}
		txsformat = append(txsformat, txformat)
	}

	itxs := make([]*ktypes.TxInternal, 0, len(blk.TxInternals))
	for _, itx := range blk.TxInternals {
		input, output := itx.Input, itx.Output
		//if itx.Value.Cmp(big.NewInt(0)) == 1 {
		if len(itx.Input) > defaultTruncateLength {
			input = itx.Input[0:defaultTruncateLength]
		}
		if len(itx.Output) > defaultTruncateLength {
			output = itx.Output[0:defaultTruncateLength]
		}
		kitx := &ktypes.TxInternal{
			Hash:         itx.TxHash,
			From:         itx.From,
			To:           itx.To,
			Value:        itx.Value.String(),
			Success:      itx.Success,
			OPCode:       itx.OPCode,
			Depth:        itx.Depth,
			Gas:          itx.Gas,
			GasUsed:      itx.GasUsed,
			Input:        input,
			Output:       output,
			TraceAddress: itx.TraceAddress,
		}
		itxs = append(itxs, kitx)
		//}
	}

	formatBlock := &ktypes.Block{
		Number:     blk.Number,
		Hash:       blk.Hash,
		Difficulty: blk.Difficulty,
		Nonce:      blk.Nonce,
		ExtraData:  blk.ExtraData,
		GasLimit:   blk.GasLimit,
		GasUsed:    blk.GasUsed,
		Miner:      blk.Miner,
		ParentHash: blk.ParentHash,

		ReceiptsRoot: blk.ReceiptsRoot,
		StateRoot:    blk.StateRoot,
		TxCnt:        blk.TxCnt,
		TimeStamp:    blk.TimeStamp,
		UnclesCnt:    blk.UnclesCnt,
		Size:         blk.Size,
		Txs:          txsformat,
		TxInternals:  itxs,
		BaseFee:      blk.BaseFee,
		BurntFees:    blk.BurntFees,
		State:        blk.State,
	}

	return formatBlock
}

func (b *PushKafkaTask) pushkafka(value string) error {
	msg := &sarama.ProducerMessage{
		Topic: b.topic,
		Value: sarama.ByteEncoder(value),
	}

	length := msg.Value.Length()
	logrus.Infof("before compension push block data size :%d", length)

	pid, offset, err := b.producer.SendMessage(msg)
	if err != nil {
		logrus.Errorf("block-send err:%v", err)
	}
	logrus.Infof("send success pid:%v offset:%v\n", pid, offset)
	return err
}

func (b *PushKafkaTask) fixHistoryData() {
	batch := b.config.PushBlk.BatchBlockCount

	var blk *mtypes.Block
	var blks []*mtypes.Block
	var err error
	if b.curHeight < b.latestHeight-uint64(b.bufferSize) {
		st := time.Now()
		blks, err = b.db.GetBlocksByRange(b.curHeight+1, b.curHeight+1+uint64(batch), mysqldb.DefaultFullBlockFilter)
		logrus.Debugf("block range cost:%d,task:%s,start:%d,batch:%d", time.Since(st)/time.Millisecond, b.name, b.curHeight+1, b.curHeight+1+uint64(batch))
	} else {
		blk, err = b.getBlkInfo(b.curHeight+1, mysqldb.ERC20Filter)
		if blk != nil {
			blks = []*mtypes.Block{blk}
		}
	}

	if len(blks) == 0 || err != nil {
		if err != nil {
			logrus.Errorf("query block info error:%v", err)
		} else {
			logrus.Debugf("push kafka handler. cur height:%v", b.curHeight)
			lastHeight := b.curHeight + 1 + uint64(batch)
			b.curHeight = lastHeight
		}

		time.Sleep(time.Second * 1)
		return
	}
	for _, blk := range blks {
		b.handleBlock(blk)
	}
	height := blks[len(blks)-1].Number
	err = b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, height)
	if err != nil {
		logrus.Errorf("push kafka task update h: %d,err: %v", height, err)
		return
	}
	b.curHeight = height
}

func (b *PushKafkaTask) handleBlock(blk *mtypes.Block) {
	logrus.Infof("push kafka block height :%d", blk.Number)

	if lastpushheight != 0 && lastpushheight+1 != blk.Number {
		logrus.Warnf("push kafka block height :%d error and last push height:%d", blk.Number, lastpushheight)
	}
	lastpushheight = blk.Number
	bk := blockformat(blk, b.config.PushBlk.TruncateLength)
	value, err := json.Marshal(bk)
	if err != nil {
		logrus.Fatalf("block to json err:%v,blk:%+v", err, blk)
	}
	utils.HandleErrorWithRetry(func() error {
		err := b.pushkafka(string(value))
		if err != nil {
			return fmt.Errorf("err:%v,h:%d", err, blk.Number)
		}
		return nil
	}, b.config.OutPut.RetryTimes, b.config.OutPut.RetryInterval)

	height := blk.Number
	// Update the height number of the task -> task.Number
	err = b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, height)
	if err != nil {
		logrus.Errorf("update push kafka task height:%d err:%v", height, err)
		return
	}
	b.curHeight = height
}

func (b *PushKafkaTask) revertBlock(blk *mtypes.Block) {
	b.handleBlock(blk)
	err := b.db.UpdateAsyncTaskNumByName(b.db.GetEngine(), b.name, blk.Number-1)
	if err != nil {
		logrus.Errorf("update balance task height:%d err:%v", blk.Number-1, err)
	}
}
