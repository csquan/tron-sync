DROP TABLE IF EXISTS `async_task`;
CREATE TABLE `async_task`
(
    `id`      bigint(20) unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `num`     bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '开始block number',
    `end_num` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT '结束block number',
    `name`    varchar(256) NOT NULL DEFAULT '' COMMENT 'task name'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;

DROP TABLE IF EXISTS `balance`;
CREATE TABLE `balance`
(
    `id`             bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `addr`           char(42)       NOT NULL DEFAULT '' COMMENT 'address',
    `balance`        decimal(65, 0) NOT NULL DEFAULT '0' COMMENT '账户数额',
    `height`         bigint(20) NOT NULL DEFAULT '0' COMMENT '更新时区块高度',
    `balance_origin` varchar(256)   NOT NULL DEFAULT '' COMMENT 'balance 原始数据',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    UNIQUE KEY `uk_addr` (`addr`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='账户表';

DROP TABLE IF EXISTS `balance_erc20`;
CREATE TABLE `balance_erc20`
(
    `id`             bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `addr`           char(42)       NOT NULL DEFAULT '' COMMENT '账户地址',
    `contract_addr`  char(42)       NOT NULL DEFAULT '' COMMENT 'erc20合约地址',
    `balance`        decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'balance',
    `height`         bigint(20) NOT NULL DEFAULT '0' COMMENT 'block height',
    `balance_origin` varchar(256)   NOT NULL DEFAULT '' COMMENT 'balance origin value',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    UNIQUE KEY `addr_contract` (`addr`,`contract_addr`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='erc20 账户表';

DROP TABLE IF EXISTS `block`;
CREATE TABLE `block` (
  `num` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'block number',
  `block_hash` char(66) DEFAULT NULL COMMENT 'block hash',
  `difficulty` int(11) NOT NULL DEFAULT '0' COMMENT '区块难度',
  `extra_data` text DEFAULT NULL COMMENT 'extra data',
  `gas_limit` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'gas limit',
  `gas_used` bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'gas used',
  `miner` char(42) NOT NULL DEFAULT '' COMMENT '矿工',
  `parent_hash` char(66) NOT NULL DEFAULT '' COMMENT 'parent hash',
  `size` int(11) NOT NULL DEFAULT '0' COMMENT 'block size',
  `receipts_root` char(66) NOT NULL DEFAULT '' COMMENT '收据树默克尔root hash',
  `state_root` char(66) NOT NULL DEFAULT '' COMMENT '状态树root hash',
  `txs_cnt` int(11) NOT NULL DEFAULT '0' COMMENT '交易数量',
  `block_timestamp` int(11) NOT NULL DEFAULT '0' COMMENT '打包时间',
  `uncles_cnt` int(11) NOT NULL DEFAULT '0' COMMENT '叔区块个数',
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `state` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:ok 1:rollback',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `total_difficulty` int(11) NOT NULL DEFAULT '0' COMMENT 'total difficulty',
  `base_fee` decimal(65,0) NOT NULL DEFAULT '0' COMMENT 'base fee',
  `burnt_fees` decimal(65,0) NOT NULL DEFAULT '0' COMMENT 'burnt fee',
  `nonce` varchar(64) DEFAULT NULL COMMENT 'nonce',
  PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
  KEY `idx_num` (`num`),
  KEY `idx_block_hash` (`block_hash`),
  KEY `idx_state_num` (`state`,`num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='区块表';

DROP TABLE IF EXISTS `contract`;
CREATE TABLE `contract`
(
    `id`           bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `tx_hash`      char(66)  NOT NULL DEFAULT '' COMMENT 'transaction hash',
    `addr`         char(42)  NOT NULL DEFAULT '' COMMENT '合约地址',
    `creator_addr` char(42)  NOT NULL DEFAULT '' COMMENT '创建者地址',
    `exec_status`  tinyint(4) NOT NULL DEFAULT '0' COMMENT '1:ok 0:fail',
    `block_num`    bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'block num',
    `block_state`  tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:ok 1:rollback',
    `create_time`  timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    UNIQUE KEY `uk_addr` (`addr`),
    KEY            `idx_txhash` (`tx_hash`),
    KEY            `idx_blocknum` (`block_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin  COMMENT='合约创建';

DROP TABLE IF EXISTS `erc20_info`;
CREATE TABLE `erc20_info`
(
    `id`                  bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `addr`                char(42)       NOT NULL DEFAULT '' COMMENT '合约地址',
    `name`                blob                    DEFAULT NULL COMMENT 'erc20 name',
    `symbol`              blob                    DEFAULT NULL COMMENT 'erc20 symbol',
    `decimals`            int(10) unsigned NOT NULL DEFAULT '0' COMMENT 'erc20 decimals',
    `total_supply`        decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'total supply',
    `total_supply_origin` varchar(256)            DEFAULT NULL COMMENT 'total supply origin value',
    `create_time`         timestamp      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    UNIQUE KEY `uk_addr` (`addr`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='erc20 合约 info';

DROP TABLE IF EXISTS `erc721_info`;
CREATE TABLE `erc721_info`
(
    `id`                  bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `addr`                char(42)       NOT NULL DEFAULT '' COMMENT '合约地址',
    `name`                blob                    DEFAULT NULL COMMENT 'erc20 name',
    `symbol`              blob                    DEFAULT NULL COMMENT 'erc20 symbol',
    `total_supply`        decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'total supply',
    `total_supply_origin` varchar(256)            DEFAULT NULL COMMENT 'total supply origin value',
    `create_time`         timestamp      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    UNIQUE KEY `uk_addr` (`addr`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='erc721 合约 info';

DROP TABLE IF EXISTS `token_erc721`;
CREATE TABLE `token_erc721`
(
    `id`             bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `contract_addr`  char(42)     NOT NULL COMMENT 'erc721合约地址',
    `token_id`       varchar(256) NOT NULL DEFAULT '0' COMMENT 'token id',
    `owner_addr`     char(42)     NOT NULL DEFAULT '' COMMENT '所有者的地址',
    `token_uri`      varchar(256) COMMENT '',
    `token_metadata` blob COMMENT '',
    `height`         bigint(20) NOT NULL DEFAULT '0' COMMENT 'block height',
    `create_time`    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time`    timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    UNIQUE KEY `addr_token_id` (`contract_addr`,`token_id`),
    KEY              `idx_owner_addr` (`owner_addr`),
    KEY              `idx_contract_addr` (`contract_addr`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin  COMMENT='erc721 代币表';

DROP TABLE IF EXISTS `tx`;
CREATE TABLE `tx`
(
    `id`                       bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `addr_to`                  char(42)       NOT NULL DEFAULT '' COMMENT '接收地址',
    `addr_from`                char(42)       NOT NULL DEFAULT '' COMMENT '发送地址',
    `tx_hash`                  char(66)       NOT NULL DEFAULT '' COMMENT 'transaction hash',
    `tx_index`                 int(11) NOT NULL DEFAULT '0' COMMENT 'transaction index',
    `tx_value`                 decimal(65, 0)          DEFAULT NULL COMMENT 'transaction value',
    `input`                    longtext                DEFAULT NULL COMMENT 'transaction input',
    `nonce`                    int(11) NOT NULL DEFAULT '0',
    `gas_price`                decimal(65, 0)          DEFAULT NULL COMMENT 'gas price',
    `gas_limit`                bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'gas limit',
    `gas_used`                 bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'gas used',
    `is_contract`              tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否调用合约交易',
    `is_contract_create`       tinyint(4) NOT NULL DEFAULT '0' COMMENT '是否创建合约交易',
    `block_time`               int(11) NOT NULL DEFAULT '0' COMMENT '打包时间',
    `block_num`                bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'block number',
    `block_hash`               char(66)       NOT NULL DEFAULT '' COMMENT 'block hash',
    `exec_status`              tinyint(4) NOT NULL DEFAULT '0' COMMENT 'transaction 执行结果',
    `create_time`              timestamp      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `block_state`              tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:ok 1:fail',
    `max_fee_per_gas`          decimal(65, 0) NOT NULL DEFAULT '0' COMMENT '最高交易小费',
    `max_priority_fee_per_gas` decimal(65, 0) NOT NULL DEFAULT '0' COMMENT '最高有限小费',
    `burnt_fees`               decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'burnt fees',
    `base_fee`                 decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'base fee',
    `tx_type`                  tinyint(4) NOT NULL DEFAULT '0' COMMENT '交易类型',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    KEY                        `idx_txhash_blocknum` (`tx_hash`,`block_num`),
    KEY                        `idx_addr_to` (`addr_to`),
    KEY                        `idx_addr_from` (`addr_from`),
    KEY                        `idx_block_num` (`block_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='交易表';

DROP TABLE IF EXISTS `tx_erc20`;
CREATE TABLE `tx_erc20`
(
    `id`               bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `tx_hash`          char(66)       NOT NULL DEFAULT '' COMMENT 'transaction hash',
    `addr`             char(42)       NOT NULL DEFAULT '' COMMENT '合约地址',
    `sender`           char(42)       NOT NULL DEFAULT '' COMMENT '发送地址',
    `receiver`         char(42)       NOT NULL DEFAULT '' COMMENT '接收地址',
    `token_cnt`        decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'token 个数',
    `log_index`        int(11) DEFAULT NULL COMMENT 'transaction log index',
    `token_cnt_origin` varchar(78)    NOT NULL DEFAULT '' COMMENT 'token cnt origin value',
    `create_time`      timestamp      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `block_state`      tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:ok 1:rollback',
    `block_num`        bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'block number',
    `block_time`       int(11) NOT NULL DEFAULT '0' COMMENT '打包时间',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    KEY                `idx_txhash` (`tx_hash`),
    KEY                `idx_addr` (`addr`),
    KEY                `idx_sender` (`sender`),
    KEY                `idx_receiver` (`receiver`),
    KEY                `idx_block_num` (`block_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='erc20交易';

DROP TABLE IF EXISTS `tx_erc721`;
CREATE TABLE `tx_erc721`
(
    `id`          bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `tx_hash`     char(66)     NOT NULL DEFAULT '' COMMENT 'transaction hash',
    `addr`        char(42)     NOT NULL DEFAULT '' COMMENT '合约地址',
    `sender`      char(42)     NOT NULL DEFAULT '' COMMENT '发送地址',
    `receiver`    char(42)     NOT NULL DEFAULT '' COMMENT '接收地址',
    `token_id`    varchar(256) NOT NULL DEFAULT '0' COMMENT '接收地址',
    `log_index`   int(11) DEFAULT NULL COMMENT 'transaction log index',
    `create_time` timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `block_state` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:ok 1:rollback',
    `block_num`   bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'block number',
    `block_time`  int(11) NOT NULL DEFAULT '0' COMMENT '打包时间',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    KEY           `idx_txhash` (`tx_hash`),
    KEY           `idx_addr` (`addr`),
    KEY           `idx_sender` (`sender`),
    KEY           `idx_receiver` (`receiver`),
    KEY           `idx_token_id` (`token_id`),
    KEY           `idx_block_num` (`block_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='erc721交易';

DROP TABLE IF EXISTS `tx_log`;
CREATE TABLE `tx_log`
(
    `id`          bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `tx_hash`     char(66)  NOT NULL DEFAULT '' COMMENT 'transaction hash',
    `addr_to`     char(42)  NOT NULL DEFAULT '' COMMENT '接收地址',
    `topic0`      char(66)  NOT NULL DEFAULT '' COMMENT 'topic0',
    `log_data`    longtext           DEFAULT NULL COMMENT 'log data',
    `log_index`   int(11) NOT NULL DEFAULT '0' COMMENT 'log index',
    `addr_from`   char(42)  NOT NULL DEFAULT '' COMMENT '发送地址',
    `addr`        char(42)  NOT NULL DEFAULT '' COMMENT '产生日志的合约地址',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `block_state` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:ok 1:rollback',
    `topic1`      char(66)  NOT NULL DEFAULT '' COMMENT 'topic1',
    `topic2`      char(66)  NOT NULL DEFAULT '' COMMENT 'topic2',
    `topic3`      char(66)  NOT NULL DEFAULT '' COMMENT 'topic3',
    `block_num`   bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'block number',
    `block_time`  int(11) NOT NULL DEFAULT '0' COMMENT '打包时间',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    KEY           `idx_tx_hash_topic0` (`tx_hash`,`topic0`),
    KEY           `idx_block_num` (`block_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='交易log';

DROP TABLE IF EXISTS `tx_internal`;
CREATE TABLE `tx_internal`
(
    `id`          bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `tx_hash`     char(66)       NOT NULL DEFAULT '' COMMENT 'transaction hash',
    `addr_from`   char(42)       NOT NULL DEFAULT '' COMMENT '发送地址',
    `addr_to`     char(42)       NOT NULL DEFAULT '' COMMENT '接收地址',
    `op_code`     varchar(50)    NOT NULL DEFAULT '' COMMENT '操作码',
    `value`       decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'transaction value',
    `success`     tinyint(4) NOT NULL DEFAULT '0' COMMENT '执行结果',
    `depth`       int(11) NOT NULL DEFAULT '0' COMMENT '调用深度',
    `gas`         bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'gas',
    `gas_used`    bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'gas used',
    `input`       text    NOT NULL COMMENT 'input',
    `output`      text    NOT NULL COMMENT 'output',
    `trace_address` varchar(255) NOT NULL DEFAULT '' COMMENT 'trace address',
    `create_time` timestamp      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `block_state` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:ok 1:rollback',
    `block_num`   bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'block number',
    `block_time`  int(11) NOT NULL DEFAULT '0' COMMENT '打包时间',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    KEY           `idx_txhash` (`tx_hash`),
    KEY           `idx_addrto` (`addr_to`),
    KEY           `idx_addrfrom` (`addr_from`),
    KEY           `idx_block_num` (`block_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='内部交易';


DROP TABLE IF EXISTS `token_pair`;
CREATE TABLE `token_pair` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT comment 'cluster id',
  `addr` char(42)  NOT NULL DEFAULT '' comment 'addr',
  `token0` char(42)  NOT NULL DEFAULT '' comment 'token0',
  `token1` char(42)  NOT NULL DEFAULT '' comment 'token1',
  `reserve0` decimal(65,0) NOT NULL DEFAULT '0' comment 'reserve0',
  `reserve1` decimal(65,0) NOT NULL DEFAULT '0' comment 'reserve1',
  `tx_hash` char(66)  NOT NULL DEFAULT '' comment 'tx hash',
  `updated` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'update time',
  `block_num` int(11) NOT NULL DEFAULT '0' comment 'block in',
  `pair_name` varchar(256)  NOT NULL DEFAULT '' comment 'contract name',
  `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP comment 'create time',
  PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
  UNIQUE KEY `addr` (`addr`),
  KEY `idx_token0`(`token0`),
  KEY `idx_token1`(`token1`)
) DEFAULT CHARSET=utf8mb4 comment 'dex token pair';

DROP TABLE IF EXISTS `tx_erc1155`;
CREATE TABLE `tx_erc1155`
(
    `id`          bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `tx_hash`     char(66)       NOT NULL DEFAULT '' COMMENT 'transaction hash',
    `addr`        char(42)       NOT NULL DEFAULT '' COMMENT '合约地址',
    `operator`    char(42)       NOT NULL DEFAULT '' COMMENT '操作者地址',
    `sender`      char(42)       NOT NULL DEFAULT '' COMMENT '金额减少者地址',
    `receiver`    char(42)       NOT NULL DEFAULT '' COMMENT '接收地址',
    `token_id`    varchar(256)   NOT NULL DEFAULT '0' COMMENT '接收地址',
    `token_cnt`   decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'token 个数',
    `log_index`   int(11) DEFAULT NULL COMMENT 'transaction log index',
    `create_time` timestamp      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `block_state` tinyint(4) NOT NULL DEFAULT '0' COMMENT '0:ok 1:rollback',
    `block_num`   bigint(20) unsigned NOT NULL DEFAULT '0' COMMENT 'block number',
    `block_time`  int(11) NOT NULL DEFAULT '0' COMMENT '打包时间',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    KEY           `idx_txhash` (`tx_hash`),
    KEY           `idx_addr_token_id` (`addr`, `token_id`),
    KEY           `idx_sender` (`sender`),
    KEY           `idx_receiver` (`receiver`),
    KEY           `idx_block_num` (`block_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='erc1155交易';

DROP TABLE IF EXISTS `balance_erc1155`;
CREATE TABLE `balance_erc1155`
(
    `id`             bigint(20) unsigned NOT NULL AUTO_INCREMENT,
    `addr`           char(42)       NOT NULL DEFAULT '' COMMENT '账户地址',
    `contract_addr`  char(42)       NOT NULL DEFAULT '' COMMENT 'erc1155合约地址',
    `token_id`       varchar(256)   NOT NULL DEFAULT '0' COMMENT 'token_id',
    `balance`        decimal(65, 0) NOT NULL DEFAULT '0' COMMENT 'balance',
    `height`         bigint(20) NOT NULL DEFAULT '0' COMMENT 'block height',
    `balance_origin` varchar(256)   NOT NULL DEFAULT '' COMMENT 'balance origin value',
    PRIMARY KEY (`id`) /*T![clustered_index] CLUSTERED */,
    UNIQUE KEY `addr_contract` (`addr`,`contract_addr`,`token_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='erc1155 账户表';
