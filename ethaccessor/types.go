/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package ethaccessor

import (
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type Block struct {
	Number           types.Big   `json:"number"`
	Hash             common.Hash `json:"hash"`
	ParentHash       common.Hash `json:"parentHash"`
	Nonce            string      `json:"nonce"`
	Sha3Uncles       string      `json:"sha3Uncles"`
	LogsBloom        string      `json:"logsBloom"`
	TransactionsRoot string      `json:"transactionsRoot"`
	ReceiptsRoot     string      `json:"stateRoot"`
	Miner            string      `json:"miner"`
	Difficulty       types.Big   `json:"difficulty"`
	TotalDifficulty  types.Big   `json:"totalDifficulty"`
	ExtraData        string      `json:"extraData"`
	Size             types.Big   `json:"size"`
	GasLimit         types.Big   `json:"gasLimit"`
	GasUsed          types.Big   `json:"gasUsed"`
	Timestamp        types.Big   `json:"timestamp"`
	Uncles           []string    `json:"uncles"`
}

type BlockWithTxObject struct {
	Block
	Transactions []Transaction
}

func (block Block) IsNull() bool {
	return types.IsZeroHash(block.Hash)
}

type BlockWithTxAndReceipt struct {
	Block
	Transactions []Transaction        `json:"transactions"`
	Receipts     []TransactionReceipt `json:"receipts"`
}

type BlockWithTxHash struct {
	Block
	Transactions []string
}

type Transaction struct {
	Hash             string    `json:"hash"`
	Nonce            types.Big `json:"nonce"`
	BlockHash        string    `json:"blockHash"`
	BlockNumber      types.Big `json:"blockNumber"`
	TransactionIndex types.Big `json:"transactionIndex"`
	From             string    `json:"from"`
	To               string    `json:"to"`
	Value            types.Big `json:"value"`
	GasPrice         types.Big `json:"gasPrice"`
	Gas              types.Big `json:"gas"`
	Input            string    `json:"input"`
	R                string    `json:"r"`
	S                string    `json:"s"`
	V                string    `json:"v"`
}

func (tx *Transaction) MethodId() string {
	// filter method input
	input := common.FromHex(tx.Input)
	if len(input) < 4 || len(tx.Input) < 10 {
		return ""
	}

	// filter method id
	return common.ToHex(input[0:4])
}

func (tx *Transaction) IsNull() bool {
	return types.IsZeroHash(common.HexToHash(tx.Hash))
}

func (tx *Transaction) IsPending() bool {
	if tx.BlockNumber.BigInt().Cmp(big.NewInt(0)) <= 0 {
		return true
	} else {
		return false
	}
}

type Log struct {
	LogIndex         types.Big `json:"logIndex"`
	BlockNumber      types.Big `json:"blockNumber"`
	BlockHash        string    `json:"blockHash"`
	TransactionHash  string    `json:"transactionHash"`
	TransactionIndex types.Big `json:"transactionIndex"`
	Address          string    `json:"address"`
	Data             string    `json:"data"`
	Topics           []string  `json:"topics"`
	Removed          bool      `json:"removed"`
}

func (evtlog *Log) EventId() common.Hash {
	if len(evtlog.Topics) == 0 {
		return types.NilHash
	}
	return common.HexToHash(evtlog.Topics[0])
}

type FilterQuery struct {
	FromBlock string           `json:"fromBlock"`
	ToBlock   string           `json:"toBlock"`
	Address   []common.Address `json:"address"`
	Topics    [][]common.Hash  `json:"topics"`
}

type LogParameter struct {
	Topics []string
}

type TransactionReceipt struct {
	BlockHash         string     `json:"blockHash"`
	BlockNumber       types.Big  `json:"blockNumber"`
	ContractAddress   string     `json:"contractAddress"`
	CumulativeGasUsed types.Big  `json:"cumulativeGasUsed"`
	From              string     `json:"from"`
	GasUsed           types.Big  `json:"gasUsed"`
	Logs              []Log      `json:"logs"`
	LogsBloom         string     `json:"logsBloom"`
	Root              string     `json:"root"`
	Status            *types.Big `json:"status"`
	To                string     `json:"to"`
	TransactionHash   string     `json:"transactionHash"`
	TransactionIndex  types.Big  `json:"transactionIndex"`
}

func (receipt *TransactionReceipt) AfterByzantiumFork() bool {
	byzantiumBlock := big.NewInt(4370000)
	return receipt.BlockNumber.BigInt().Cmp(byzantiumBlock) >= 0
}

func (receipt *TransactionReceipt) StatusInvalid() bool {
	if !receipt.AfterByzantiumFork() {
		return false
	}
	return receipt.Status == nil
}

func (receipt *TransactionReceipt) HasNoLog() bool {
	return len(receipt.Logs) == 0
}

func (receipt *TransactionReceipt) Failed(tx *Transaction) bool {
	if receipt.AfterByzantiumFork() && receipt.Status.BigInt().Cmp(big.NewInt(1)) == 0 {
		return false
	}

	// env:local or test net
	if !receipt.AfterByzantiumFork() && !receipt.HasNoLog() {
		return false
	}
	if !receipt.AfterByzantiumFork() && tx.Value.BigInt().Cmp(big.NewInt(0)) > 0 {
		return false
	}

	return true
}

type BlockIterator struct {
	startNumber   *big.Int
	endNumber     *big.Int
	currentNumber *big.Int
	ethClient     *ethNodeAccessor
	withTxData    bool
	confirms      uint64
}

type CallArg struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to"`
	Gas      types.Big      `json:"gas"`
	GasPrice types.Big      `json:"gasPrice"`
	Value    types.Big      `json:"value"`
	Data     string         `json:"data"`
	Nonce    types.Big      `json:"nonce"`
}
