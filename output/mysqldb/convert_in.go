package mysqldb

import (
	"fmt"
	"strings"

	"github.com/chainmonitor/mtypes"
	"github.com/ethereum/go-ethereum/core/types"
)

func ConvertInBlock(block *mtypes.Block) *Block {
	var baseFee, burntFees = "0", "0"
	if block.BaseFee != nil {
		baseFee = block.BaseFee.String()
	}
	if block.BurntFees != nil {
		burntFees = block.BurntFees.String()
	}
	totalDifficulty := "0"
	if block.TotoalDifficulty != nil {
		totalDifficulty = block.TotoalDifficulty.String()
	}
	difficulty := "0"
	if block.Difficulty != nil {
		difficulty = block.Difficulty.String()
	}
	return &Block{
		Number:          block.Number,
		BlockHash:       block.Hash,
		Difficulty:      difficulty,
		ExtraData:       block.ExtraData,
		GasLimit:        block.GasLimit,
		GasUsed:         block.GasUsed,
		Miner:           strings.ToLower(block.Miner),
		ParentHash:      block.ParentHash,
		Size:            block.Size,
		ReceiptsRoot:    block.ReceiptsRoot,
		StateRoot:       block.StateRoot,
		TxsCnt:          block.TxCnt,
		BlockTimestamp:  block.TimeStamp,
		UnclesCnt:       block.UnclesCnt,
		State:           Block_ok,
		BaseFee:         baseFee,
		BurntFees:       burntFees,
		TotalDifficulty: totalDifficulty,
		Nonce:           block.Nonce,
	}
}

func ConvertInTx(blockId int64, block *mtypes.Block, tx *mtypes.Tx) *TxDB {
	txDB := &TxDB{
		TxType:               tx.TxType,
		AddrTo:               strings.ToLower(tx.To),
		AddrFrom:             strings.ToLower(tx.From),
		Hash:                 tx.Hash,
		Index:                tx.Index,
		Value:                tx.Value.String(),
		Input:                tx.Input,
		Nonce:                tx.Nonce,
		GasPrice:             tx.GasPrice.String(),
		GasLimit:             tx.GasLimit,
		GasUsed:              tx.GasUsed,
		IsContract:           tx.IsContract,
		IsContractCreate:     tx.IsContractCreate,
		BlockNum:             block.Number,
		BlockHash:            block.Hash,
		BlockTime:            int64(block.TimeStamp),
		ExecStatus:           tx.ExecStatus,
		BaseFee:              "0",
		MaxFeePerGas:         "0",
		MaxPriorityFeePerGas: "0",
		BurntFees:            "0",
	}
	//eip1559
	if tx.TxType == types.DynamicFeeTxType {
		txDB.BaseFee = tx.BaseFee.String()
		txDB.MaxFeePerGas = tx.MaxFeePerGas.String()
		txDB.MaxPriorityFeePerGas = tx.MaxPriorityFeePerGas.String()
		txDB.BurntFees = tx.BurntFees.String()
	}

	return txDB
}

func ConvertInLog(blockId int64, block *mtypes.Block, tx *mtypes.Tx, tlog *mtypes.EventLog) *TxLog {
	return &TxLog{
		Hash:      tx.Hash,
		Addr:      strings.ToLower(tlog.Addr),
		AddrFrom:  strings.ToLower(tx.From),
		AddrTo:    strings.ToLower(tx.To),
		Topic0:    tlog.Topic0,
		Topic1:    tlog.Topic1,
		Topic2:    tlog.Topic2,
		Topic3:    tlog.Topic3,
		Data:      tlog.Data,
		Index:     tlog.Index,
		BlockNum:  block.Number,
		BlockTime: block.TimeStamp,
	}
}

// transTraceAddressToString transform traceAddress array to Etherscan style, like: "call_0_0_5_2"
func transTraceAddressToString(opcode string, traceAddress []uint64) string {
	var res = strings.ToLower(opcode)
	for _, addr := range traceAddress {
		res = fmt.Sprintf("%s_%d", res, addr)
	}
	return res
}

// ConvertInInternalTx convert rpc internal tx structure into database internal tx structure.
func ConvertInInternalTx(blockId int64, block *mtypes.Block, internalTx *mtypes.TxInternal) *TxInternal {
	var value string
	if internalTx.Value != nil {
		value = internalTx.Value.String()
	}

	return &TxInternal{
		TxHash:       internalTx.TxHash,
		AddrFrom:     strings.ToLower(internalTx.From),
		AddrTo:       strings.ToLower(internalTx.To),
		OPCode:       internalTx.OPCode,
		Value:        value,
		Success:      internalTx.Success,
		Depth:        internalTx.Depth,
		Gas:          internalTx.Gas,
		GasUsed:      internalTx.GasUsed,
		Input:        internalTx.Input,
		Output:       internalTx.Output,
		TraceAddress: transTraceAddressToString(internalTx.OPCode, internalTx.TraceAddress),
		BlockNum:     block.Number,
		BlockTime:    block.TimeStamp,
	}
}

// ConvertInContract convert rpc contract structure into database contract structure.
func ConvertInContract(blockId int64, block *mtypes.Block, contract *mtypes.Contract) *Contract {
	c := &Contract{
		TxHash:      contract.TxHash,
		Addr:        strings.ToLower(contract.Addr),
		CreatorAddr: strings.ToLower(contract.CreatorAddr),
		ExecStatus:  contract.ExecStatus,
		BlockNum:    block.Number,
		BlockState:  0,
	}
	return c
}

func ConvertInBalances(balances []*mtypes.Balance) []*Balance {
	res := make([]*Balance, 0, len(balances))

	for _, b := range balances {
		if b.BalanceType != mtypes.NormalBalance {
			res = append(res, nil)
		}

		b.Value = b.ValueHexBig.ToInt()

		balance := "0"
		if b.Value != nil {
			balance = b.Value.String()
		}

		var origin string
		if len(balance) > 65 {
			origin = balance
			balance = balance[:65]
		}

		res = append(res, &Balance{
			Addr:          strings.ToLower(b.Addr.Hex()),
			Balance:       balance,
			Height:        b.Height.Uint64(),
			BalanceOrigin: origin,
		})
	}

	return res
}

func ConvertInErc20Balances(bs []*mtypes.Balance) []*BalanceErc20 {
	res := make([]*BalanceErc20, 0, len(bs))

	for _, b := range bs {
		if b.BalanceType != mtypes.NormalBalance {
			res = append(res, nil)
		}

		balance := "0"
		if b.Value != nil {
			balance = b.Value.String()
		}

		var origin string
		if len(balance) > 65 {
			origin = balance
			balance = balance[:65]
		}

		res = append(res, &BalanceErc20{
			Addr:          strings.ToLower(b.Addr.Hex()),
			ContractAddr:  strings.ToLower(b.ContractAddr.Hex()),
			Balance:       balance,
			Height:        b.Height.Uint64(),
			BalanceOrigin: origin,
		})
	}

	return res
}

func ConvertInErc20Infos(erc20Infos []*mtypes.Erc20Info) []*Erc20Info {
	ret := make([]*Erc20Info, 0, len(erc20Infos))
	for _, info := range erc20Infos {
		var (
			origin      string
			totalSupply = "0"
		)
		if len(info.TotoalSupply) > 65 {
			origin = info.TotoalSupply
			totalSupply = info.TotoalSupply[:65]
		} else if info.TotoalSupply != "" {
			totalSupply = info.TotoalSupply
			origin = info.TotoalSupplyOrigin
		}
		ret = append(ret, &Erc20Info{
			Addr:               info.Addr,
			Name:               info.Name,
			Symbol:             info.Symbol,
			Decimals:           info.Decimals,
			TotoalSupply:       totalSupply,
			TotoalSupplyOrigin: origin,
		})
	}
	return ret
}

func ConvertInErc721Infos(erc721Infos []*mtypes.Erc721Info) []*Erc721Info {
	ret := make([]*Erc721Info, 0, len(erc721Infos))
	for _, info := range erc721Infos {
		var (
			origin      string
			totalSupply = "0"
		)
		if len(info.TotoalSupply) > 65 {
			origin = info.TotoalSupply
			totalSupply = info.TotoalSupply[:65]
		} else if info.TotoalSupply != "" {
			totalSupply = info.TotoalSupply
		}
		ret = append(ret, &Erc721Info{
			Addr:               info.Addr,
			Name:               info.Name,
			Symbol:             info.Symbol,
			TotoalSupply:       totalSupply,
			TotoalSupplyOrigin: origin,
		})
	}
	return ret
}

func ConvertInTokenPairs(pairs []*mtypes.TokenPair) []*TokenPair {
	ret := make([]*TokenPair, 0, len(pairs))
	for _, pair := range pairs {
		ret = append(ret, &TokenPair{
			Addr:     strings.ToLower(pair.Addr.Hex()),
			Token0:   strings.ToLower(pair.Token0.Hex()),
			Token1:   strings.ToLower(pair.Token1.Hex()),
			Reserve0: pair.Reserve0,
			Reserve1: pair.Reserve1,
			PairName: pair.PairName,
			BlockNum: pair.BlockNum,
		})
	}
	return ret
}
