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
	"github.com/ethereum/go-ethereum/common/hexutil"
)

//通用数据结构
type Block struct {
	Number	hexutil.Big
	Hash	string
	ParentHash	string
	Nonce	string
	Sha3Uncles	string
	LogsBloom	string
	TransactionsRoot	string
	ReceiptsRoot	string
	Miner	string
	Difficulty	hexutil.Big
	TotalDifficulty	hexutil.Big
	ExtraData	string
	Size	hexutil.Big
	GasLimit	hexutil.Big
	GasUsed	hexutil.Big
	Timestamp	hexutil.Big
	Uncles	[]string
}

type BlockWithTxObject struct {
	Block
	Transactions	[]Transaction
}

type BlockWithTxHash struct {
	Block
	Transactions	[]string
}

type Transaction struct {
	Hash	string
	Nonce	hexutil.Big
	BlockHash	string
	BlockNumber	hexutil.Big
	TransactionIndex	hexutil.Big
	From	string
	To	string
	Value	hexutil.Big
	GasPrice	hexutil.Big
	Gas	hexutil.Big
	Input	string
}

type Log struct {
	LogIndex hexutil.Big
	BlockNumber hexutil.Big
	BlockHash	string
	TransactionHash	string
	Address	string
}

type LogParameter struct {
	Topics	[]string
}
