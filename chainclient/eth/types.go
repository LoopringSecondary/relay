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

package eth

import (
	"github.com/Loopring/ringminer/types"
)

//todo:need to modify

type Block struct {
	Number           types.Big  `json:"number"`
	Hash             types.Hash `json:"hash"`
	ParentHash       types.Hash `json:"parentHash"`
	Nonce            string     `json:"nonce"`
	Sha3Uncles       string     `json:"sha3Uncles"`
	LogsBloom        string     `json:"logsBloom"`
	TransactionsRoot string     `json:"transactionsRoot"`
	ReceiptsRoot     string     `json:"stateRoot"`
	Miner            string     `json:"miner"`
	Difficulty       types.Big  `json:"difficulty"`
	TotalDifficulty  types.Big  `json:"totalDifficulty"`
	ExtraData        string     `json:"extraData"`
	Size             types.Big  `json:"size"`
	GasLimit         types.Big  `json:"gasLimit"`
	GasUsed          types.Big  `json:"gasUsed"`
	Timestamp        types.Big  `json:"timestamp"`
	Uncles           []string   `json:"uncles"`
}

type BlockWithTxObject struct {
	Block
	Transactions []Transaction
}

type BlockWithTxHash struct {
	Block
	Transactions []string
}

type Transaction struct {
	Hash             string
	Nonce            types.Big
	BlockHash        string
	BlockNumber      types.Big
	TransactionIndex types.Big
	From             string
	To               string
	Value            types.Big
	GasPrice         types.Big
	Gas              types.Big
	Input            string
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
}

type FilterQuery struct {
	FromBlock string          `json:"fromBlock"`
	ToBlock   string          `json:"toBlock"`
	Address   []types.Address `json:"address"`
	Topics    [][]types.Hash  `json:"topics"`
}

type LogParameter struct {
	Topics []string
}

type TransactionReceipt struct {
	TransactionHash   types.Hash    `json:"transactionHash"`
	TransactionIndex  types.Big     `json:"transactionIndex"`
	BlockHash         types.Hash    `json:"blockHash"`
	BlockNumber       types.Big     `json:"blockNumber"`
	CumulativeGasUsed types.Big     `json:"cumulativeGasUsed"`
	GasUsed           types.Big     `json:"gasUsed"`
	ContractAddress   types.Address `json:"contractAddress"`
	Logs              []Log         `json:"logs"`
}
