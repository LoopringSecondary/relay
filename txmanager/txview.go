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
	"errors"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

type TransactionView interface {
	GetPendingTransactions(owner string) ([]TransactionJsonResult, error)
	GetMinedTransactions(owner, symbol string, pageIndex, pageSize int) ([]TransactionJsonResult, error)
	GetTransactionsByHash(owner string, hashList []string) ([]TransactionJsonResult, error)
}

type TransactionViewImpl struct {
	db dao.RdsService
}

func NewTxView(db dao.RdsService) *TransactionViewImpl {
	var tm TransactionViewImpl
	tm.db = db

	return &tm
}

var (
	ErrOwnerAddressInvalid error = errors.New("owner address invalid")
	ErrHashListEmpty       error = errors.New("hash list is empty")
	ErrNonTransaction      error = errors.New("no transaction found")
)

// todo pagination
func (impl *TransactionViewImpl) GetPendingTransactions(ownerStr string) ([]TransactionJsonResult, error) {
	var list []TransactionJsonResult

	if ownerStr == "" {
		return list, ErrOwnerAddressInvalid
	}

	owner := common.HexToAddress(ownerStr)
	txs, err := impl.db.GetPendingTransactions(owner.Hex(), types.TX_STATUS_PENDING)
	if err != nil {
		return list, ErrNonTransaction
	}

	list = assemble(txs, owner)
	return list, nil
}

func (impl *TransactionViewImpl) GetMinedTransactions(ownerStr, symbol string, pageIndex, pageSize int) ([]TransactionJsonResult, error) {
	//trxQuery := make(map[string]interface{})
	//
	//if query.Symbol != "" {
	//	trxQuery["symbol"] = query.Symbol
	//}
	//
	//if query.Owner != "" {
	//	trxQuery["owner"] = query.Owner
	//}
	//
	//if query.ThxHash != "" {
	//	trxQuery["tx_hash"] = query.ThxHash
	//}
	//
	//if txStatusToUint8(query.Status) > 0 {
	//	trxQuery["status"] = uint8(txStatusToUint8(query.Status))
	//}
	//
	//if txTypeToUint8(query.TxType) > 0 {
	//	trxQuery["tx_type"] = uint8(txTypeToUint8(query.TxType))
	//}
	//
	//pageIndex := query.PageIndex
	//pageSize := query.PageSize
	//
	//daoPr, err := w.rds.TransactionPageQuery(trxQuery, pageIndex, pageSize)
	//
	//if err != nil {
	//	return pr, err
	//}
	//
	//rst := PageResult{Total: daoPr.Total, PageIndex: daoPr.PageIndex, PageSize: daoPr.PageSize, Data: make([]interface{}, 0)}
	//
	//for _, d := range daoPr.Data {
	//	o := d.(dao.Transaction)
	//	tr := types.Transaction{}
	//	err = o.ConvertUp(&tr)
	//	rst.Data = append(rst.Data, toTxJsonResult(tr))
	//}
	//return rst, nil

	var list []TransactionJsonResult

	owner := common.HexToAddress(ownerStr)
	protocol := symbolToProtocol(symbol)
	status := []uint8{types.TX_STATUS_SUCCESS, types.TX_STATUS_FAILED}
	limit, offset := pagination(pageIndex, pageSize)

	txs, err := impl.db.GetMinedTransactions(owner.Hex(), protocol.Hex(), status, limit, offset)
	if len(txs) == 0 || err != nil {
		return list, ErrNonTransaction
	}

	list = assemble(txs, owner)
	return list, nil
}

func (impl *TransactionViewImpl) GetTransactionsByHash(ownerStr string, hashList []string) ([]TransactionJsonResult, error) {
	var (
		list    []TransactionJsonResult
		hashstr []string
	)
	if len(hashList) == 0 {
		return list, ErrHashListEmpty
	}

	for _, v := range hashList {
		hashstr = append(hashstr, common.HexToHash(v).Hex())
	}
	txs, err := impl.db.GetTrxByHashes(hashstr)

	if len(txs) == 0 || err != nil {
		return list, ErrNonTransaction
	}

	owner := common.HexToAddress(ownerStr)
	list = assemble(txs, owner)

	return list, nil
}

func assemble(items []dao.Transaction, owner common.Address) []TransactionJsonResult {
	var list []TransactionJsonResult
	for _, v := range items {
		var (
			tx  types.Transaction
			res TransactionJsonResult
		)
		v.ConvertUp(&tx)
		symbol := protocolToSymbol(tx.Protocol)
		res.fromTransaction(tx, owner, symbol)
		list = append(list, res)
	}
	return list
}

func pagination(page, size int) (int, int) {
	limit := size
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * size

	return limit, offset
}
