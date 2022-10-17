package config

var (
	DefaultBaseStorageConfig = BaseStorage{
		BufferSize:    500,
		MaxBlockCount: 100,
		MaxTxCount:    500,
		MaxLogCount:   1000,
		MaxInterval:   100,
	}

	DefaultBalanceConfig = Balance{
		Concurrent:      5,
		BatchBlockCount: 50,
		BufferSize:      500,
		CacheSize:       10000,
	}
	DefaultErc20BalanceConfig = Erc20Balance{
		Concurrent:      5,
		BatchBlockCount: 50,
		BufferSize:      500,
		CacheSize:       10000,
		BatchRPC:        60,
	}

	DefaultFetchConfig = FetchConf{
		FetchBatch:            50,
		FetchTimeoutInt:       3000,
		FetchRetryTimes:       20,
		FetchRetryIntervalInt: 100,

		TxWorker: 15,
		TxBatch:  30,

		BlockWorker:      10,
		BlockBatch:       5,
		BlocksDelay:      3,
		BlocksSafe:       20,
		BlockWaitTimeInt: 3000,

		IsDealInternal:    false,
		ReceiptRetryTimes: 20,
	}

	DefaultOutputConfig = OutputConf{
		SqlBatch:         500,
		RetryIntervalInt: 500,
		RetryTimes:       5,
	}

	DefaultLogConfig = Log{
		Stdout: stdout{
			Enable: true,
			Level:  4,
		},
	}

	DefaultErc721TxConfig = Erc721Tx{
		BatchBlockCount: 50,
		MaxTxCount:      100,
		MaxBlockCount:   50,
	}

	DefaultErc20TxConfig = Erc20Tx{
		BatchBlockCount: 50,
		MaxTxCount:      100,
		MaxBlockCount:   50,
	}

	DefaultDexPairConfig = DexPair{
		BufferSize: 300,
		Concurrent: 5,
		CacheSize:  50000,
		RPCBatch:   64,
	}

	DefaultPushKafkaConfig = PushBlk{
		Topic: "test_topic",
	}

	DefaultErc1155TxConfig = Erc1155Tx{
		BufferSize:      300,
		BatchBlockCount: 50,
		MaxTxCount:      100,
		MaxBlockCount:   50,
	}

	DefaultErc1155BalanceConfig = Erc1155Balance{
		BufferSize:      300,
		BatchBlockCount: 50,
		MaxTxCount:      100,
		MaxBlockCount:   50,
		CacheSize:       10000,
		BatchRPC:        60,
		Concurrent:      5,
	}
	DefaultBlockDelay = BlockDelay{
		Delay:           3,
		BufferSize:      300,
		BatchBlockCount: 50,
	}
)
