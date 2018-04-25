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
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

func GetPendingTransactions(owner string) ([]TransactionJsonResult, error) {
	return impl.GetPendingTransactions(owner)
}
func GetTransactionsByHash(owner string, hashList []string) ([]TransactionJsonResult, error) {
	return impl.GetTransactionsByHash(owner, hashList)
}
func GetAllTransactionCount(ownerStr, symbol, status, typ string) (int, error) {
	return impl.GetAllTransactionCount(ownerStr, symbol, status, typ)
}
func GetAllTransactions(owner, symbol, status, typ string, limit, offset int) ([]TransactionJsonResult, error) {
	return impl.GetAllTransactions(owner, symbol, status, typ, limit, offset)
}

type TransactionView interface {
	GetPendingTransactions(owner string) ([]TransactionJsonResult, error)
	GetAllTransactionCount(owner, symbol, status, typ string) (int, error)
	GetAllTransactions(owner, symbol, status, typ string, limit, offset int) ([]TransactionJsonResult, error)
	GetTransactionsByHash(owner string, hashList []string) ([]TransactionJsonResult, error)
}

var impl TransactionView

type TransactionViewImpl struct {
	db dao.RdsService
}

// todo(fuk): 在分布式锁以及wallet_service事件通知推送落地后使用redis缓存(当前版本暂时不考虑redis),考虑到分叉及用户tx总量，
// 缓存应该有三个数据类型:
// 1. user key 存储用户pengding&mined first page transactions,设置过期时间
// 2. user tx number key存储某个用户所有tx数量的key,设置过期时间
// 3. block key 存储某个block涉及到的用户key(用于分叉),设置过期时间
func NewTxView(db dao.RdsService) {
	tm := &TransactionViewImpl{}
	tm.db = db
	impl = tm
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
	txs, err := impl.db.GetPendingTransactionsByOwner(owner.Hex())
	if err != nil {
		return list, ErrNonTransaction
	}

	list = assemble(txs, owner)
	return list, nil
}

func (impl *TransactionViewImpl) GetAllTransactionCount(ownerStr, symbolStr, statusStr, typStr string) (int, error) {
	owner := common.HexToAddress(ownerStr)
	symbol := standardSymbol(symbolStr)
	statusList := statusStringToList(statusStr)
	typList := typeStringToList(typStr)

	number, err := impl.db.GetTransactionCount(owner.Hex(), symbol, statusList, typList)
	if number == 0 || err != nil {
		return 0, ErrNonTransaction
	}

	return number, nil
}

func (impl *TransactionViewImpl) GetAllTransactions(ownerStr, symbolStr, statusStr, typStr string, limit, offset int) ([]TransactionJsonResult, error) {
	var list []TransactionJsonResult

	owner := common.HexToAddress(ownerStr)
	symbol := standardSymbol(symbolStr)
	statusList := statusStringToList(statusStr)
	typList := typeStringToList(typStr)

	hashs, err := impl.db.GetTransactionHashs(owner.Hex(), symbol, statusList, typList, limit, offset)
	if len(hashs) == 0 || err != nil {
		return list, ErrNonTransaction
	}

	txs, err := impl.db.GetTrxByHashes(hashs)
	if len(txs) == 0 || err != nil {
		return list, ErrNonTransaction
	}

	list = assemble(txs, owner)
	//list = collector(list)
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

	txs, _ := impl.db.GetTrxByHashes(hashstr)
	if len(txs) == 0 {
		return list, ErrNonTransaction
	}

	owner := common.HexToAddress(ownerStr)
	list = assemble(txs, owner)

	return list, nil
}

// 如果transaction包含多条记录,则将protocol不同的记录放到content里
func assemble(items []dao.Transaction, owner common.Address) []TransactionJsonResult {
	var list []TransactionJsonResult

	for _, v := range items {
		var (
			tx  types.Transaction
			res TransactionJsonResult
		)
		v.ConvertUp(&tx)
		symbol := protocolToSymbol(tx.Protocol)

		// todo(fuk): 数据稳定后可以删除该代码或者加开关过滤该代码
		if err := filter(&tx, owner, symbol); err != nil {
			log.Debugf(err.Error())
			continue
		}

		res.fromTransaction(&tx, owner, symbol)
		list = append(list, res)
	}

	return list
}

func statusStringToList(statusStr string) []types.TxStatus {
	var list []types.TxStatus

	status := types.StrToTxStatus(statusStr)
	if status == types.TX_STATUS_UNKNOWN {
		return list
	}
	list = append(list, status)
	return list
}

func typeStringToList(typStr string) []types.TxType {
	var list []types.TxType

	typ := types.StrToTxType(typStr)
	if typ == types.TX_TYPE_UNKNOWN {
		return list
	}

	list = append(list, typ)
	return list
}
