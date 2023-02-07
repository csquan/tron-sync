package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	remote "github.com/shima-park/agollo/viper-remote"
	"github.com/spf13/viper"
)

type OutputConf struct {
	DB               string `mapstructure:"db"`             //DB 连接信息
	SqlBatch         int    `mapstructure:"sql_batch"`      //  批量插入sql时，每次批量插入的最大值
	RetryIntervalInt int    `mapstructure:"retry_interval"` // 存储失败时重试的间隔 ms
	RetryTimes       int    `mapstructure:"retry_times"`    // 存储失败，重试次数
	RetryInterval    time.Duration
}

type MonitorConf struct {
	DB string `mapstructure:"db"` //DB 连接信息
}

type EryConf struct {
	PUB string `mapstructure:"pub"` //DB 连接信息
}

func (o *OutputConf) init() {
	o.RetryInterval = time.Duration(o.RetryIntervalInt) * time.Millisecond
}

type FetchConf struct {
	IsDealInternal bool `mapstructure:"is_deal_internal"`

	RpcURL            string `mapstructure:"rpc_url"`
	MultiCallContract string `mapstructure:"multi_call_contract"`

	BlockWorker int `mapstructure:"block_worker"`
	TxWorker    int `mapstructure:"tx_worker"`
	TxBatch     int `mapstructure:"tx_batch"`
	FetchBatch  int `mapstructure:"fetch_batch"`

	FetchTimeoutInt       int `mapstructure:"fetch_timeout"`        //ms
	FetchRetryIntervalInt int `mapstructure:"fetch_retry_interval"` //ms

	FetchTimeout       time.Duration
	FetchRetryInterval time.Duration

	FetchRetryTimes   int `mapstructure:"fetch_retry_times"`
	ReceiptRetryTimes int `mapstructure:"receipt_retry_times"`

	BlockBatch  uint64 `mapstructure:"block_batch"`
	BlocksDelay uint64 `mapstructure:"blocks_delay"`
	BlocksSafe  uint64 `mapstructure:"block_safe"`

	BlockWaitTimeInt int `mapstructure:"block_wait_time"` //ms
	BlockWaitTime    time.Duration

	StartHeight uint64 `mapstructure:"start_height"`
	EndHeight   uint64 `mapstructure:"end_height"`
}

func (f *FetchConf) init() {
	f.FetchTimeout = time.Duration(f.FetchTimeoutInt) * time.Millisecond
	f.FetchRetryInterval = time.Duration(f.FetchRetryIntervalInt) * time.Millisecond
	f.BlockWaitTime = time.Duration(f.BlockWaitTimeInt) * time.Millisecond
}

type Config struct {
	ProfPort int    `mapstructure:"prof_port"`
	AppName  string `mapstructure:"app_name"`

	OutPut  OutputConf  `mapstructure:"output"`
	Monitor MonitorConf `mapstructure:"monitor"`
	Ery     EryConf     `mapstructure:"ery"`

	Fetch          FetchConf      `mapstructure:"fetch"`
	Tasks          []string       `mapstructure:"tasks"`
	LogConf        Log            `mapstructure:"log"`
	Balance        Balance        `mapstructure:"balance"`
	BaseStorage    BaseStorage    `mapstructure:"base_storage"`
	Erc20Balance   Erc20Balance   `mapstructure:"erc20_balance"`
	Erc721Tx       Erc721Tx       `mapstructure:"erc721_tx"`
	Erc1155Tx      Erc1155Tx      `mapstructure:"erc1155_tx"`
	Erc1155Balance Erc1155Balance `mapstructure:"erc1155_balance"`
	Erc20Tx        Erc20Tx        `mapstructure:"erc20_tx"`
	DexPair        DexPair        `mapstructure:"dex_pair"`
	PushBlk        PushBlk        `mapstructure:"push_blk"`
	BlockDelay     BlockDelay     `mapstructure:"block_delay"`

	Kafka Kafka `mapstructure:"kafka"`
}

type BaseStorage struct {
	BufferSize    int `mapstructure:"buffer_size"`     //上下游通知中的管道容量
	MaxBlockCount int `mapstructure:"max_block_count"` //批量插入时，每批block最大数量
	MaxTxCount    int `mapstructure:"max_tx_count"`    //批量插入时，每批tx最大数量
	MaxLogCount   int `mapstructure:"max_log_count"`   //批量插入时，每批log最大数量
	MaxInterval   int `mapstructure:"max_interval"`    //ms 收集block信息的时候，最大停顿等待时间
}

type Kafka struct {
	// BatchBlockCount int      `mapstructure:"batch_block_size"`
	ProducerMax int `mapstructure:"producer_max"`
	// Key         string   `mapstructure:"key"` //生产消费约定的key
	Brokers []string `mapstructure:"kafka_servers"`
	Topic   string   `mapstructure:"topic"`
}

type PushBlk struct {
	Topic           string `mapstructure:"topic"`
	BufferSize      int    `mapstructure:"buffer_size"`      //上下游通知中的管道容量,内存中缓存的block数量
	BatchBlockCount int    `mapstructure:"batch_block_size"` //db 批量获取size
	TruncateLength  int    `mapstructure:"truncate_length"`
}

type Balance struct {
	Concurrent      int `mapstructure:"concurrent_count"` //rpc并发数量
	BatchBlockCount int `mapstructure:"batch_block_size"` // 每个rpc每次批量获取的最大值
	BufferSize      int `mapstructure:"buffer_size"`      //上下游通知中的管道容量,内存中缓存的block数量
	CacheSize       int `mapstructure:"cache_size"`       //内存中缓存地址更新信息的最大值
}

type Erc20Balance struct {
	Concurrent      int `mapstructure:"concurrent_count"`
	BatchBlockCount int `mapstructure:"batch_block_size"`
	BufferSize      int `mapstructure:"buffer_size"`
	CacheSize       int `mapstructure:"cache_size"`
	BatchRPC        int `mapstructure:"batch_rpc"`
}

type Erc1155Tx struct {
	BatchBlockCount int `mapstructure:"batch_block_size"`
	MaxBlockCount   int `mapstructure:"max_block_count"`
	MaxTxCount      int `mapstructure:"max_tx_count"`
	BufferSize      int `mapstructure:"buffer_size"`
	CacheSize       int `mapstructure:"cache_size"`
}

type Erc1155Balance struct {
	Concurrent      int `mapstructure:"concurrent_count"`
	BatchBlockCount int `mapstructure:"batch_block_size"`
	MaxBlockCount   int `mapstructure:"max_block_count"`
	MaxTxCount      int `mapstructure:"max_tx_count"`
	BufferSize      int `mapstructure:"buffer_size"`
	CacheSize       int `mapstructure:"cache_size"`
	BatchRPC        int `mapstructure:"batch_rpc"`
}

type DexPair struct {
	BufferSize int `mapstructure:"buffer_size"`
	Concurrent int `mapstructure:"concurrent_count"` //rpc并发数量
	CacheSize  int `mapstructure:"cache_size"`       //内存中缓存地址更新信息的最大值
	RPCBatch   int `mapstructure:"rpc_batch"`
}

type Erc20Tx struct {
	BatchBlockCount int `mapstructure:"batch_block_size"`
	MaxBlockCount   int `mapstructure:"max_block_count"`
	MaxTxCount      int `mapstructure:"max_tx_count"`
}

type Erc721Tx struct {
	BatchBlockCount int `mapstructure:"batch_block_size"`
	MaxBlockCount   int `mapstructure:"max_block_count"`
	MaxTxCount      int `mapstructure:"max_tx_count"`
}

type Log struct {
	Stdout stdout `mapstructure:"stdout"`
	File   file   `mapstructure:"file"`
	Kafka  kafka  `mapstructure:"kafka"`
}

type BlockDelay struct {
	Topic           string `mapstructure:"topic"`
	Delay           int    `mapstructure:"delay"`
	BufferSize      int    `mapstructure:"buffer_size"`
	BatchBlockCount int    `mapstructure:"batch_block_size"`
	TruncateLength  int    `mapstructure:"truncate_length"`
}

type stdout struct {
	Enable bool `mapstructure:"enable"`
	Level  int  `mapstructure:"level"`
}

type file struct {
	Enable bool   `mapstructure:"enable"`
	Path   string `mapstructure:"path"`
	Level  int    `mapstructure:"level"`
}

type kafka struct {
	Enable  bool     `mapstructure:"enable"`
	Level   int      `mapstructure:"level"`
	Brokers []string `mapstructure:"kafka_servers"`
	Topic   string   `mapstructure:"topic"`
}

func LoadConf(fpath string, env string) (*Config, error) {
	if fpath == "" {
		return nil, fmt.Errorf("fpath empty")
	}

	if !strings.HasSuffix(strings.ToLower(fpath), ".yaml") {
		return nil, fmt.Errorf("fpath must has suffix of .yaml")
	}

	//load default config first
	conf := &Config{
		OutPut:         DefaultOutputConfig,
		Fetch:          DefaultFetchConfig,
		Balance:        DefaultBalanceConfig,
		BaseStorage:    DefaultBaseStorageConfig,
		LogConf:        DefaultLogConfig,
		Erc20Balance:   DefaultErc20BalanceConfig,
		Erc721Tx:       DefaultErc721TxConfig,
		Erc20Tx:        DefaultErc20TxConfig,
		DexPair:        DefaultDexPairConfig,
		Erc1155Tx:      DefaultErc1155TxConfig,
		Erc1155Balance: DefaultErc1155BalanceConfig,
		BlockDelay:     DefaultBlockDelay,
	}

	vip := viper.New()
	vip.SetConfigType("yaml")

	if !strings.HasPrefix(strings.ToLower(fpath), "remote") {
		fmt.Println("read configuration from local yaml file :", fpath)
		err := LocalConfig(fpath, vip)
		if err != nil {
			return nil, err
		}

	} else {
		//has prefix of 'remote', get configuration from remote apollo server
		fmt.Println("read configuration from remote apollo server :", fpath)
		err := RemoteConfig(env, fpath, vip)
		if err != nil {
			return nil, err
		}
		//fmt.Println("config from apollo : ", vip)

	}
	//vip.Unmarshal uses "mapstructure" as struct tag for parsing
	err := vip.Unmarshal(conf)
	if err != nil {
		return nil, err
	}

	conf.Fetch.init()
	conf.OutPut.init()

	//printConf(conf)
	return conf, nil
}

// LocalConfig build viper from the local disk
func LocalConfig(filename string, v *viper.Viper) error {
	path, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("err : %v", err)
	}

	v.AddConfigPath(path) //设置读取的文件路径

	v.SetConfigName(filename) //设置读取的文件名

	err = v.ReadInConfig()
	if err != nil {
		return fmt.Errorf("read conf file err : %v", err)
	}

	return err
}

const (
	appID            string = "heco-sync"
	apolloServerTest string = "http://apollo-config.system-service.huobiapps.com"
	apolloServerProd string = "http://apollo-config.system-service.apne-1.huobiapps.com"
)

// RemoteConfig build viper from apollo server
func RemoteConfig(env string, namespace string, v *viper.Viper) error {
	var apolloServer string
	remote.SetAppID(appID)
	switch env {
	case "test":
		apolloServer = apolloServerTest
	case "prod":
		apolloServer = apolloServerProd
	default:
		apolloServer = apolloServerProd
	}
	err := v.AddRemoteProvider("apollo", apolloServer, namespace)
	if err != nil {
		return fmt.Errorf("add remote provider error : %v", err)
	}

	err = v.ReadRemoteConfig()
	if err != nil {
		return fmt.Errorf("read remote config error : %v", err)
	}

	if v.Get("app_name") == nil {
		return fmt.Errorf("read remote config error : app_name not found! This namespace might not exsit!")
	}

	err = v.WatchRemoteConfigOnChannel() // 启动一个goroutine来同步配置更改

	return err
}
