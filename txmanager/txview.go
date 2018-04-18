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
	GetMinedTransactionCount(ownerStr, symbol string) (int, error)
	GetMinedTransactions(owner, symbol string, limit, offset int) ([]TransactionJsonResult, error)
	GetTransactionsByHash(owner string, hashList []string) ([]TransactionJsonResult, error)
}

type TransactionViewImpl struct {
	db dao.RdsService
}

// todo(fuk): 在分布式锁以及wallet_service事件通知推送落地后使用redis缓存(当前版本暂时不考虑redis),考虑到分叉及用户tx总量，
// 缓存应该有三个数据类型:
// 1. user key 存储用户pengding&mined first page transactions,设置过期时间
// 2. user tx number key存储某个用户所有tx数量的key,设置过期时间
// 3. block key 存储某个block涉及到的用户key(用于分叉),设置过期时间
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

func (impl *TransactionViewImpl) GetMinedTransactionCount(ownerStr, symbol string) (int, error) {
	owner := common.HexToAddress(ownerStr)
	status := []types.TxStatus{types.TX_STATUS_SUCCESS, types.TX_STATUS_FAILED}
	symbol = standardSymbol(symbol)
	number, err := impl.db.GetMinedTransactionCount(owner.Hex(), symbol, status)
	if number == 0 || err != nil {
		return 0, ErrNonTransaction
	}
	return number, nil
}

func (impl *TransactionViewImpl) GetMinedTransactions(ownerStr, symbol string, limit, offset int) ([]TransactionJsonResult, error) {
	var list []TransactionJsonResult

	owner := common.HexToAddress(ownerStr)
	symbol = standardSymbol(symbol)
	status := []types.TxStatus{types.TX_STATUS_SUCCESS, types.TX_STATUS_FAILED}

	hashs, err := impl.db.GetMinedTransactionHashs(owner.Hex(), symbol, status, limit, offset)
	if len(hashs) == 0 || err != nil {
		return list, ErrNonTransaction
	}

	txs, err := impl.db.GetTrxByHashes(hashs)
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

// 如果transaction包含多条记录,则将protocol不同的记录放到content里
func assemble(items []dao.Transaction, owner common.Address) []TransactionJsonResult {
	var list []TransactionJsonResult

	transferCombine := make(map[common.Hash]map[int64]Transaction) // map[txhash]map[logindex]TransactionJsonResult
	for _, v := range items {
		var (
			tx  types.Transaction
			res TransactionJsonResult
		)
		v.ConvertUp(&tx)
		symbol := protocolToSymbol(tx.Protocol)
		res.fromTransaction(tx, owner, symbol)

		// 1.同一个logIndex进行过滤
		// 2.同一个tx 如果包含某个transfer 则将其他的transfer打包到content
		if !tx.IsTransfer() {
			list = append(list, res)
			continue
		}

		if _, ok := txs[res.TxHash]; !ok {
			if len(txs[res.TxHash]) == 0 {
				txs[res.TxHash] = make(map[int64]TransactionJsonResult)
				txs[res.TxHash][res.LogIndex] = res
			} else {
				for _, next := range txs[res.TxHash] {
					if res.LogIndex == next.LogIndex {
						continue
					}
					if res.Symbol == next.Symbol && res.From == next.From && res.To == next.To {
						txs[res.TxHash][res.LogIndex].Value += res.Value
					}
				}
			}
		}

		list = append(list, res)
	}
	return list
}
