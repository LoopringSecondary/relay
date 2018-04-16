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

package txmanager

import (
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

type TransactionView interface {
	GetPendingTransactions(owner common.Address) []TransactionJsonResult
	GetMinedTransactions(owner common.Address, symbol string, page, size int) []TransactionJsonResult
	GetTransactionsByHash(hashList []common.Hash) []TransactionJsonResult
}

type TransactionViewImpl struct {
	db dao.RdsService
}

func NewTxView(db dao.RdsService) *TransactionViewImpl {
	var tm TransactionViewImpl
	tm.db = db

	return &tm
}

func (impl *TransactionViewImpl) GetPendingTransactions(owner common.Address) []TransactionJsonResult {
	var list []TransactionJsonResult

	txs, err := impl.db.GetPendingTransactions(owner.Hex(), types.TX_STATUS_PENDING)
	if err != nil {
		return list
	}

	return assemble(txs, owner)
}

func (impl *TransactionViewImpl) GetMinedTransactions(owner common.Address, symbol string, page, size int) []TransactionJsonResult {
	var list []TransactionJsonResult

	protocol := symbolToProtocol(symbol)
	status := []uint8{types.TX_STATUS_SUCCESS, types.TX_STATUS_FAILED}
	limit, offset := pagination(page, size)

	txs, err := impl.db.GetMinedTransactions(owner.Hex(), protocol.Hex(), status, limit, offset)
	if len(txs) == 0 || err != nil {
		return list
	}

	return assemble(txs, owner)
}

func (impl *TransactionViewImpl) GetTransactionsByHash(hashList []common.Hash) []TransactionJsonResult {
	var (
		list    []TransactionJsonResult
		hashstr []string
	)
	if len(hashList) == 0 {
		return list
	}

	for _, v := range hashList {
		hashstr = append(hashstr, v.Hex())
	}
	txs, err := impl.db.GetTrxByHashes(hashstr)

	// todo: handle error code
	if len(txs) == 0 || err != nil {
		return list
	}

	return list
}

func assemble(items []dao.Transaction, owner common.Address) []TransactionJsonResult {
	var list []TransactionJsonResult
	for _, v := range items {
		res := fromDb(v, owner)
		list = append(list, res)
	}
	return list
}

func fromDb(src dao.Transaction, owner common.Address) TransactionJsonResult {
	var (
		tx  types.Transaction
		res TransactionJsonResult
	)
	src.ConvertUp(&tx)
	symbol := protocolToSymbol(tx.Protocol)
	res.fromTransaction(tx, owner, symbol)

	return res
}

func pagination(page, size int) (int, int) {
	limit := size
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * size

	return limit, offset
}
