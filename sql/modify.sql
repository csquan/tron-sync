alter table tx drop index uk_tx_blockid;
alter table tx add index idx_txhash_blocknum(`tx_hash`,`block_num`);
alter table tx drop index idx_blockid;

alter table tx_log drop index idx_blockid;
alter table tx_log drop index uk_txhash_logindex_block;

alter table tx_erc20 add index idx_txhash(`tx_hash`);
alter table tx_erc20 drop index uk_txhash_logindex_block;
alter table tx_erc20 drop index idx_blockid;