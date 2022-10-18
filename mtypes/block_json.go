package mtypes

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type BlockJson struct {
	BaseFeePerGas    string        `json:"baseFeePerGas"`
	Difficulty       string        `json:"difficulty"`
	ExtraData        string        `json:"extraData"`
	GasLimit         string        `json:"gasLimit"`
	GasUsed          string        `json:"gasUsed"`
	Hash             string        `json:"hash"`
	Miner            string        `json:"miner"`
	Nonce            string        `json:"nonce"`
	Number           string        `json:"number"`
	ParentHash       string        `json:"parentHash"`
	ReceiptsRoot     string        `json:"receiptsRoot"`
	Sha3Uncles       string        `json:"sha3Uncles"`
	Size             string        `json:"size"`
	StateRoot        string        `json:"stateRoot"`
	TimeStamp        string        `json:"timestamp"`
	TotalDifficulty  string        `json:"totalDifficulty"`
	Txs              []*TxJson     `json:"transactions"`
	TransactionsRoot string        `json:"transactionsRoot"`
	Uncles           []interface{} `json:"uncles"`
}

func (b *BlockJson) ToBlock() *Block {
	num := hexutil.MustDecodeUint64(b.Number)
	difficulty := hexutil.MustDecodeBig(b.Difficulty)
	if b.TotalDifficulty == "" {
		panic(fmt.Sprintf("totalDifficulty empty bnum:%s", b.Number))
	}
	totoalDifficulty := hexutil.MustDecodeBig(b.TotalDifficulty)
	gasLimit := hexutil.MustDecodeUint64(b.GasLimit)
	gasUsed := hexutil.MustDecodeUint64(b.GasUsed)
	size := hexutil.MustDecodeUint64(b.Size)
	timeStamp := hexutil.MustDecodeUint64(b.TimeStamp)
	var baseFee *big.Int
	if b.BaseFeePerGas != "" {
		baseFee = hexutil.MustDecodeBig(b.BaseFeePerGas)
	}
	block := &Block{
		Number:           num,
		Hash:             b.Hash,
		Difficulty:       difficulty,
		TotoalDifficulty: totoalDifficulty,
		Nonce:            b.Nonce,
		ExtraData:        b.ExtraData,
		GasLimit:         gasLimit,
		GasUsed:          gasUsed,
		Miner:            b.Miner,
		ParentHash:       b.ParentHash,
		ReceiptsRoot:     b.ReceiptsRoot,
		Sha3Uncles:       b.Sha3Uncles,
		Size:             size,
		StateRoot:        b.StateRoot,
		TxCnt:            len(b.Txs),
		TimeStamp:        timeStamp,
		UnclesCnt:        len(b.Uncles),
		BaseFee:          baseFee,
	}

	return block
}

// transaction type
type TxType uint8

const (
	Binary TxType = iota
	LoginCandidate
	LogoutCandidate
	Delegate
	UnDelegate
)

type Number int64

type TxJson struct {
	From                 string `json:"from"`
	Gas                  string `json:"gas"`
	GasPrice             string `json:"gasPrice"`
	Hash                 string `json:"hash"`
	Input                string `json:"input"`
	MaxFeePerGas         string `json:"maxFeePerGas"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas"`
	Nonce                string `json:"nonce"`
	R                    string `json:"r"`
	S                    string `json:"s"`
	To                   string `json:"to"`
	TransactionIndex     string `json:"transactionIndex"`
	Type                 uint8  `json:"type"`
	V                    string `json:"v"`
	Value                string `json:"value"`
}

/*
// todo：优化代码

	func (t *TxJson) UnmarshalJSON(d []byte) error {
		tmp := struct {
			From                 string      `json:"from"`
			Gas                  string      `json:"gas"`
			GasPrice             string      `json:"gasPrice"`
			Hash                 string      `json:"hash"`
			Input                string      `json:"input"`
			MaxFeePerGas         string      `json:"maxFeePerGas"`
			MaxPriorityFeePerGas string      `json:"maxPriorityFeePerGas"`
			Nonce                string      `json:"nonce"`
			R                    string      `json:"r"`
			S                    string      `json:"s"`
			To                   string      `json:"to"`
			TransactionIndex     string      `json:"transactionIndex"`
			Type                 interface{} `json:"Type"`
			V                    string      `json:"v"`
			Value                string      `json:"value"`
		}{}

		if err := json.Unmarshal(d, &tmp); err != nil {
			return err
		}

		t.From = tmp.From
		t.Gas = tmp.Gas
		t.GasPrice = tmp.GasPrice
		t.Hash = tmp.Hash
		t.Input = tmp.Input
		t.MaxFeePerGas = tmp.MaxFeePerGas
		t.MaxPriorityFeePerGas = tmp.MaxPriorityFeePerGas

		t.Nonce = tmp.Nonce
		t.R = tmp.R
		t.S = tmp.S
		t.To = tmp.To
		t.TransactionIndex = tmp.TransactionIndex
		t.V = tmp.V
		t.Value = tmp.Value

		switch v := tmp.Type.(type) {
		case float64:
			t.Type = strconv.Itoa(int(v))
		case string:
			t.Type = v
		default:
			return fmt.Errorf("invalid value for Foo: %v", v)
		}

		return nil
	}
*/
func (t *TxJson) ToTx() *Tx {
	var txtype uint64
	//if t.Type == "" {
	//	txtype = 0
	//} else {
	//	txtype = hexutil.MustDecodeUint64(t.Type)
	//}

	txtype = uint64(t.Type)

	index := hexutil.MustDecodeUint64(t.TransactionIndex)
	value := hexutil.MustDecodeBig(t.Value)
	nonce := hexutil.MustDecodeUint64(t.Nonce)
	gasPrice := hexutil.MustDecodeBig(t.GasPrice)
	gasLimit := hexutil.MustDecodeUint64(t.Gas)
	var maxFeePerGas, maxPriorityFeePerGas *big.Int
	if t.MaxFeePerGas != "" {
		maxFeePerGas = hexutil.MustDecodeBig(t.MaxFeePerGas)
	}
	if t.MaxPriorityFeePerGas != "" {
		maxPriorityFeePerGas = hexutil.MustDecodeBig(t.MaxPriorityFeePerGas)
	}

	tx := &Tx{
		TxType:               uint8(txtype),
		From:                 strings.ToLower(t.From),
		To:                   strings.ToLower(t.To),
		Hash:                 t.Hash,
		Index:                int(index),
		Value:                value,
		Input:                t.Input,
		Nonce:                nonce,
		GasPrice:             gasPrice,
		GasLimit:             gasLimit,
		MaxFeePerGas:         maxFeePerGas,
		MaxPriorityFeePerGas: maxPriorityFeePerGas,
	}
	return tx
}
