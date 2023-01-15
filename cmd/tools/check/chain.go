package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/chainmonitor/mtypes"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const erc20Transfer = `0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef`

type DataChain struct {
	Block      *mtypes.Block
	Txs        []*mtypes.Tx
	TxErc20s   map[string][]mtypes.Erc20Transfer
	TxInternal map[string][]*mtypes.TxInternal
	TxLogs     map[string][]mtypes.EventLog
	Contracs   []*mtypes.Contract
}

type Chain struct {
	c        *rpc.Client
	Client   *ethclient.Client
	internal bool
	name     string
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func (c *Chain) getBlockByNum(num uint64) (*types.Block, error) {
	return c.Client.BlockByNumber(context.Background(), big.NewInt(int64(num)))
}

func (c *Chain) getBlockByNumv2(num uint64) (*mtypes.BlockJson, error) {
	bjson := &mtypes.BlockJson{}
	bum := new(big.Int).SetUint64(num)
	err := c.c.Call(bjson, "eth_getBlockByNumber", toBlockNumArg(bum), true)
	if err != nil {
		return nil, err
	}
	return bjson, nil
}

func (c *Chain) getTxInternalsByBlock(num *big.Int) ([]*mtypes.TxInternal, []*mtypes.Contract) {
	start := time.Now()
	arg := map[string]interface{}{
		// "MinValue": 1,
	}
	ret := make([]*mtypes.TxForInternal, 0)
	err := c.c.CallContext(context.Background(), &ret, "debug_traceActionByBlockNumber", toBlockNumArg(num), arg)
	for err != nil {
		log.Printf("get txinternals err:%v,bnum:%d", err, num.Uint64())
		time.Sleep(50 * time.Millisecond)
		err = c.c.CallContext(context.Background(), &ret, "debug_traceActionByBlockNumber", toBlockNumArg(num), arg)
	}
	txInternals := make([]*mtypes.TxInternal, 0)
	contracts := make([]*mtypes.Contract, 0)
	for _, v := range ret {
		txhash := v.TransactionHash

		for _, tilog := range v.Logs {
			var isCreate bool
			if tilog.OPCode == "Create" || tilog.OPCode == "Create2" {
				if tilog.Success {
					var status uint64 = 1
					if tilog.To == `0x0000000000000000000000000000000000000000` {
						log.Fatalf("internal tx empty txhash:%s,from:%s,to:%s", txhash, tilog.From, tilog.To)
					}
					contracts = append(contracts, &mtypes.Contract{
						TxHash:      txhash,
						Addr:        strings.ToLower(tilog.To),
						CreatorAddr: strings.ToLower(tilog.From),
						ExecStatus:  status,
					})
					isCreate = true
				}

			}
			var value *big.Int
			if tilog.Value != nil {
				value = tilog.Value
			}
			if value != nil && value.Uint64() != 0 || isCreate {
				ti := &mtypes.TxInternal{
					TxHash:  txhash,
					From:    tilog.From,
					To:      tilog.To,
					Value:   value,
					Success: tilog.Success,
					OPCode:  tilog.OPCode,
					Depth:   tilog.Depth,
					GasUsed: tilog.GasUsed,
				}
				txInternals = append(txInternals, ti)
			}
		}

	}
	log.Printf("get internals cost:%d,size:%d", time.Since(start)/time.Millisecond, len(ret))
	return txInternals, contracts
}

func (c *Chain) getTxReceipts(txhashs []string) map[string]*types.Receipt {
	if len(txhashs) == 0 {
		log.Printf("txhashs empty")
		return nil
	}
	elems := make([]rpc.BatchElem, 0)
	ret := make(map[string]*types.Receipt)
	for _, txhash := range txhashs {
		receipt := &types.Receipt{}
		elem := rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{txhash},
			Result: receipt,
		}
		ret[txhash] = receipt
		elems = append(elems, elem)
	}
	err := c.c.BatchCallContext(context.Background(), elems)
	for err != nil {
		log.Printf("getTxReceipts err:%v", err)
		time.Sleep(50 * time.Millisecond)
		err = c.c.BatchCallContext(context.Background(), elems)
	}
	return ret
}

func getTxFrom(tx *types.Transaction) string {
	// var signer types.Signer = types.FrontierSigner{}
	// if tx.Protected() {
	// 	signer = types.NewEIP155Signer(tx.ChainId())
	// }
	var signer types.Signer
	if tx.Protected() {
		signer = types.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = types.HomesteadSigner{}
	}
	from, err := types.Sender(signer, tx)
	if err != nil {
		panic(fmt.Sprintf("get tx from err:%v,tx:%s", err, tx.Hash()))
	}
	return from.Hex()
}

func (c *Chain) GetData(num uint64) *DataChain {
	ret := &DataChain{}
	log.Printf("get block num start")
	var block *mtypes.Block
	var txhashs []string
	txs := make([]*mtypes.Tx, 0)
	eventLogs := make([]mtypes.EventLog, 0)
	txErc20s := make([]mtypes.Erc20Transfer, 0)
	retContracts := make([]*mtypes.Contract, 0)

	if c.name != "poly" && c.name != "heco" {
		b, err := c.getBlockByNum(num)
		log.Printf("get block num end")
		if err != nil {
			log.Fatalf("get block err:%v", err)
		}
		block = &mtypes.Block{
			Number:           b.Number().Uint64(),
			Hash:             b.Hash().Hex(),
			Difficulty:       b.Difficulty(),
			TotoalDifficulty: big.NewInt(0),
			ExtraData:        hexutil.Encode(b.Header().Extra),
			GasLimit:         b.GasLimit(),
			GasUsed:          b.GasUsed(),
			Miner:            strings.ToLower(b.Header().Coinbase.Hex()),
			ParentHash:       b.ParentHash().Hex(),
			Size:             uint64(b.Size()),
			TxCnt:            len(b.Transactions()),
			TimeStamp:        b.Header().Time,
			UnclesCnt:        len(b.Uncles()),
			ReceiptsRoot:     b.ReceiptHash().Hex(),
			StateRoot:        b.Root().Hex(),
			Nonce:            hexutil.Encode(b.Header().Nonce[:]),
		}
		if c.name == "eth" && block.Number >= 12_965_000 {
			block.BaseFee = b.Header().BaseFee
			burnt := new(big.Int)
			burnt = burnt.Mul(b.Header().BaseFee, new(big.Int).SetUint64(block.GasUsed))
			block.BurntFees = burnt
		}
		for _, tx := range b.Transactions() {
			txhashs = append(txhashs, tx.Hash().Hex())
		}
		receipts := c.getTxReceipts(txhashs)
		for _, tx := range b.Transactions() {
			receipt := receipts[tx.Hash().Hex()]
			var toHex string
			if tx.To() != nil {
				toHex = tx.To().Hex()
			}
			txtemp := &mtypes.Tx{
				From:       strings.ToLower(getTxFrom(tx)),
				To:         strings.ToLower(toHex),
				Hash:       tx.Hash().Hex(),
				Index:      int(receipt.TransactionIndex),
				Value:      tx.Value(),
				Input:      hexutil.Encode(tx.Data()),
				Nonce:      tx.Nonce(),
				GasPrice:   tx.GasPrice(),
				GasLimit:   tx.Gas(),
				GasUsed:    receipt.GasUsed,
				ExecStatus: receipt.Status,
				TxType:     tx.Type(),
			}
			// eip 1559
			if tx.Type() == types.DynamicFeeTxType {
				txtemp.MaxFeePerGas = tx.GasFeeCap()
				txtemp.MaxPriorityFeePerGas = tx.GasTipCap()
				temp := new(big.Int).SetUint64(txtemp.GasUsed)
				txtemp.BurntFees = temp.Mul(temp, block.BaseFee)
			}

			for _, txlog := range receipt.Logs {
				if len(txlog.Topics) > 0 && txlog.Topics[0].Hex() == erc20Transfer {
					if len(txlog.Topics) < 3 {
						log.Printf("topics size not 3 txhash:%s,index:%d", tx.Hash(), txlog.Index)
						continue
					}
					sender := common.HexToAddress(txlog.Topics[1].Hex())
					receiver := common.HexToAddress(txlog.Topics[2].Hex())

					tokens := new(big.Int)
					tokens.SetBytes(txlog.Data)
					//erc20转账数据
					erc20AddrHex := txlog.Address.Hex()
					erc20Tranfer := mtypes.Erc20Transfer{
						TxHash:   tx.Hash().Hex(),
						Addr:     strings.ToLower(erc20AddrHex),
						Sender:   strings.ToLower(sender.Hex()),
						Receiver: strings.ToLower(receiver.Hex()),
						Tokens:   tokens,
						LogIndex: int(txlog.Index),
					}
					txErc20s = append(txErc20s, erc20Tranfer)

				}

				item := txlog
				elog := mtypes.EventLog{}
				var topic0, topic1, topic2, topic3 string
				switch len(item.Topics) {
				case 1:
					topic0 = item.Topics[0].Hex()
				case 2:
					topic0 = item.Topics[0].Hex()
					topic1 = item.Topics[1].Hex()
				case 3:
					topic0 = item.Topics[0].Hex()
					topic1 = item.Topics[1].Hex()
					topic2 = item.Topics[2].Hex()
				case 4:
					topic0 = item.Topics[0].Hex()
					topic1 = item.Topics[1].Hex()
					topic2 = item.Topics[2].Hex()
					topic3 = item.Topics[3].Hex()
				}
				elog.TxHash = tx.Hash().Hex()
				elog.Topic0 = topic0
				elog.Topic1 = topic1
				elog.Topic2 = topic2
				elog.Topic3 = topic3

				elog.Addr = strings.ToLower(txlog.Address.Hex())
				elog.Data = hexutil.Encode(txlog.Data) //TODO是否hash编码
				elog.Index = txlog.Index
				eventLogs = append(eventLogs, elog)
			}

			if tx.To() == nil || receipt.ContractAddress.Hex() != `0x0000000000000000000000000000000000000000` {
				if receipt.Status == 1 {
					contract := &mtypes.Contract{
						TxHash:      tx.Hash().Hex(),
						Addr:        strings.ToLower(receipt.ContractAddress.Hex()),
						CreatorAddr: getTxFrom(tx),
						ExecStatus:  receipt.Status,
					}
					retContracts = append(retContracts, contract)
				}
			}
			txs = append(txs, txtemp)
		}
		ret.Txs = txs
	} else {
		blockjson, err := c.getBlockByNumv2(num)
		for _, tx := range blockjson.Txs {
			txhashs = append(txhashs, tx.Hash)
		}

		receipts := c.getTxReceipts(txhashs)

		if err != nil {
			log.Fatalf("get block v2 err:%v", err)
		}
		block = blockjson.ToBlock()

		for _, tx := range blockjson.Txs {
			receipt := receipts[tx.Hash]
			var (
				toHex            string = tx.To
				isContractCreate bool
			)
			if toHex == "" || toHex == "0x" || receipt.ContractAddress.Hex() != `0x0000000000000000000000000000000000000000` {
				if receipt.Status == 1 {
					isContractCreate = true
					// toHex = receipt.ContractAddress.Hex()
					contract := &mtypes.Contract{
						TxHash:      tx.Hash,
						Addr:        strings.ToLower(receipt.ContractAddress.Hex()),
						CreatorAddr: tx.From,
						ExecStatus:  receipt.Status,
					}
					retContracts = append(retContracts, contract)
				}
			}

			var isContract bool
			if len(tx.Input) != 0 && tx.Input != "0x" { //是否调用合约
				isContract = true
			}
			txtemp := tx.ToTx()
			txtemp.GasUsed = receipt.GasUsed
			txtemp.IsContract = isContract
			txtemp.IsContractCreate = isContractCreate
			txtemp.ExecStatus = receipt.Status

			// erc20s := make([]mtypes.Erc20Transfer, 0)
			// erc20Infos := make([]*mtypes.Erc20Info, 0)

			// eventLogs := make([]mtypes.EventLog, 0)
			for _, item := range receipt.Logs {
				txlog := item
				if len(txlog.Topics) > 0 && txlog.Topics[0].Hex() == erc20Transfer {
					if len(txlog.Topics) < 3 {
						log.Printf("topics size not 3 txhash:%s,index:%d", tx.Hash, txlog.Index)
						continue
					}
					sender := common.HexToAddress(txlog.Topics[1].Hex())
					receiver := common.HexToAddress(txlog.Topics[2].Hex())

					tokens := new(big.Int)
					tokens.SetBytes(txlog.Data)
					var (
						receiverBalance *big.Int
						senderBalance   *big.Int
					)
					//erc20转账数据
					erc20AddrHex := txlog.Address.Hex()
					erc20Tranfer := mtypes.Erc20Transfer{
						Addr:            strings.ToLower(erc20AddrHex),
						Sender:          strings.ToLower(sender.Hex()),
						Receiver:        strings.ToLower(receiver.Hex()),
						Tokens:          tokens,
						LogIndex:        int(txlog.Index),
						SenderBalance:   senderBalance,
						ReceiverBalance: receiverBalance,
						TxHash:          tx.Hash,
					}
					txErc20s = append(txErc20s, erc20Tranfer)

				}

				elog := mtypes.EventLog{}
				var topic0, topic1, topic2, topic3 string
				switch len(item.Topics) {
				case 1:
					topic0 = item.Topics[0].Hex()
				case 2:
					topic0 = item.Topics[0].Hex()
					topic1 = item.Topics[1].Hex()
				case 3:
					topic0 = item.Topics[0].Hex()
					topic1 = item.Topics[1].Hex()
					topic2 = item.Topics[2].Hex()
				case 4:
					topic0 = item.Topics[0].Hex()
					topic1 = item.Topics[1].Hex()
					topic2 = item.Topics[2].Hex()
					topic3 = item.Topics[3].Hex()
				}
				elog.Topic0 = topic0
				elog.Topic1 = topic1
				elog.Topic2 = topic2
				elog.Topic3 = topic3
				elog.TxHash = tx.Hash

				elog.Addr = strings.ToLower(txlog.Address.Hex())
				elog.Data = hexutil.Encode(txlog.Data) //TODO是否hash编码
				elog.Index = txlog.Index
				eventLogs = append(eventLogs, elog)
			}

			if tx.To == "" || tx.To == "0x" || receipt.ContractAddress.Hex() != `0x0000000000000000000000000000000000000000` {
				if receipt.Status == 1 {
					contract := &mtypes.Contract{
						TxHash:      tx.Hash,
						Addr:        strings.ToLower(receipt.ContractAddress.Hex()),
						CreatorAddr: tx.From,
						ExecStatus:  receipt.Status,
					}
					retContracts = append(retContracts, contract)
				}
			}
			txs = append(txs, txtemp)
		}
		log.Printf("poly txs cnt:%d", len(txs))
		ret.Txs = txs

	}
	ret.Block = block

	logM := make(map[string][]mtypes.EventLog)
	for _, elog := range eventLogs {
		if v, ok := logM[elog.TxHash]; ok {
			v = append(v, elog)
			logM[elog.TxHash] = v
		} else {
			v = make([]mtypes.EventLog, 0)
			v = append(v, elog)
			logM[elog.TxHash] = v
		}
	}
	ret.TxLogs = logM

	erc20M := make(map[string][]mtypes.Erc20Transfer)
	for _, erc20 := range txErc20s {
		if v, ok := erc20M[erc20.TxHash]; ok {
			v = append(v, erc20)
			erc20M[erc20.TxHash] = v
		} else {
			v = make([]mtypes.Erc20Transfer, 0)
			v = append(v, erc20)
			erc20M[erc20.TxHash] = v
		}
	}

	ret.TxErc20s = erc20M
	ret.Contracs = retContracts
	if c.internal {
		txInternals, contracts := c.getTxInternalsByBlock(big.NewInt(int64(num)))
		internalMap := make(map[string][]*mtypes.TxInternal)
		for _, ti := range txInternals {
			if v, ok := internalMap[ti.TxHash]; ok {
				v = append(v, ti)
				internalMap[ti.TxHash] = v
			} else {
				v = make([]*mtypes.TxInternal, 0)
				v = append(v, ti)
				internalMap[ti.TxHash] = v
			}
		}
		ret.TxInternal = internalMap
		ret.Contracs = append(ret.Contracs, contracts...)
	}
	cm := make(map[string]*mtypes.Contract)
	for _, c := range ret.Contracs {
		cm[c.Addr] = c
	}
	contracts := make([]*mtypes.Contract, 0)
	for _, v := range cm {
		contracts = append(contracts, v)
	}
	ret.Contracs = contracts
	return ret
}
