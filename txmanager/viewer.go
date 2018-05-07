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
	"github.com/Loopring/relay/market/util"
	txtyp "github.com/Loopring/relay/txmanager/types"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"strconv"
	"strings"
)

func GetPendingTransactions(owner string) ([]txtyp.TransactionJsonResult, error) {
	return impl.GetPendingTransactions(owner)
}
func GetTransactionsByHash(owner string, hashList []string) ([]txtyp.TransactionJsonResult, error) {
	return impl.GetTransactionsByHash(owner, hashList)
}
func GetAllTransactionCount(ownerStr, symbol, status, typ string) (int, error) {
	return impl.GetAllTransactionCount(ownerStr, symbol, status, typ)
}
func GetAllTransactions(owner, symbol, status, typ string, limit, offset int) ([]txtyp.TransactionJsonResult, error) {
	return impl.GetAllTransactions(owner, symbol, status, typ, limit, offset)
}

type TransactionViewer interface {
	GetPendingTransactions(owner string) ([]txtyp.TransactionJsonResult, error)
	GetAllTransactionCount(owner, symbol, status, typ string) (int, error)
	GetAllTransactions(owner, symbol, status, typ string, limit, offset int) ([]txtyp.TransactionJsonResult, error)
	GetTransactionsByHash(owner string, hashList []string) ([]txtyp.TransactionJsonResult, error)
}

var impl TransactionViewer

type TransactionViewerImpl struct {
	db dao.RdsService
}

// todo(fuk): 在分布式锁以及wallet_service事件通知推送落地后使用redis缓存(当前版本暂时不考虑redis),考虑到分叉及用户tx总量，
// 缓存应该有三个数据类型:
// 1. user key 存储用户pengding&mined first page transactions,设置过期时间
// 2. user tx number key存储某个用户所有tx数量的key,设置过期时间
// 3. block key 存储某个block涉及到的用户key(用于分叉),设置过期时间
func NewTxView(db dao.RdsService) {
	tm := &TransactionViewerImpl{}
	tm.db = db
	impl = tm
}

var (
	ErrOwnerAddressInvalid error = errors.New("owner address invalid")
	ErrHashListEmpty       error = errors.New("hash list is empty")
	ErrNonTransaction      error = errors.New("no transaction found")
)

func (impl *TransactionViewerImpl) GetTransactionsByHash(ownerStr string, hashList []string) ([]txtyp.TransactionJsonResult, error) {
	var list []txtyp.TransactionJsonResult

	if !validateTxHashList(hashList) {
		return list, ErrHashListEmpty
	}
	if !validateOwner(ownerStr) {
		return list, ErrOwnerAddressInvalid
	}

	owner := safeOwner(ownerStr)
	txs, _ := impl.db.GetTxViewByOwnerAndHashs(owner, hashList)
	if len(txs) == 0 {
		return list, ErrNonTransaction
	}

	list = impl.assemble(txs)

	return list, nil
}

func (impl *TransactionViewerImpl) GetPendingTransactions(ownerStr string) ([]txtyp.TransactionJsonResult, error) {
	list := make([]txtyp.TransactionJsonResult, 0)

	if !validateOwner(ownerStr) {
		return list, ErrOwnerAddressInvalid
	}

	owner := safeOwner(ownerStr)
	txs, err := impl.db.GetPendingTxViewByOwner(owner)
	if err != nil {
		return list, ErrNonTransaction
	}

	list = impl.assemble(txs)
	return list, nil
}

func (impl *TransactionViewerImpl) GetAllTransactionCount(ownerStr, symbolStr, statusStr, typStr string) (int, error) {
	if !validateOwner(ownerStr) {
		return 0, ErrOwnerAddressInvalid
	}

	owner := common.HexToAddress(ownerStr)
	symbol := safeSymbol(symbolStr)
	status := safeStatus(statusStr)
	typ := safeType(typStr)

	number, err := impl.db.GetTxViewCountByOwner(owner.Hex(), symbol, status, typ)
	if number == 0 || err != nil {
		return 0, ErrNonTransaction
	}

	return number, nil
}

func (impl *TransactionViewerImpl) GetAllTransactions(ownerStr, symbolStr, statusStr, typStr string, limit, offset int) ([]txtyp.TransactionJsonResult, error) {
	list := make([]txtyp.TransactionJsonResult, 0)

	if !validateOwner(ownerStr) {
		return list, ErrOwnerAddressInvalid
	}

	owner := safeOwner(ownerStr)
	symbol := safeSymbol(symbolStr)
	status := safeStatus(statusStr)
	typ := safeType(typStr)

	views, err := impl.db.GetTxViewByOwner(owner, symbol, status, typ, limit, offset)
	if err != nil {
		return list, ErrNonTransaction
	}

	list = impl.assemble(views)

	return list, nil
}

// 如果transaction包含多条记录,则将protocol不同的记录放到content里
func (impl *TransactionViewerImpl) assemble(daoviews []dao.TransactionView) []txtyp.TransactionJsonResult {
	list := make([]txtyp.TransactionJsonResult, 0)

	// get dao.TransactionEntity
	entitymap := GetEntityCache(impl.db, daoviews)

	for _, v := range daoviews {
		var (
			view   txtyp.TransactionView
			entity txtyp.TransactionEntity
			model  dao.TransactionEntity
			ok     bool
		)

		// get entity from map
		if model, ok = entitymap.getEntity(v.TxHash, v.LogIndex); !ok {
			continue
		}

		// convert data struct
		v.ConvertUp(&view)
		model.ConvertUp(&entity)

		// convert txtyp.TransactionView & txtyp.TransactionEntity to txtyp.TransactionJsonResult
		if res, err := getTransactionJsonResult(&view, &entity); err == nil {
			list = append(list, res)
		}
	}

	return list
}

func getTransactionJsonResult(view *txtyp.TransactionView, entity *txtyp.TransactionEntity) (txtyp.TransactionJsonResult, error) {
	res := txtyp.NewResult(view)

	if entity.Content == "" {
		res.FromOtherEntity(entity)
		return res, nil
	}

	var err error

	switch view.Type {
	case txtyp.TX_TYPE_APPROVE:
		err = res.FromApproveEntity(entity)

	case txtyp.TX_TYPE_CANCEL_ORDER:
		err = res.FromCancelEntity(entity)

	case txtyp.TX_TYPE_CUTOFF:
		err = res.FromCutoffEntity(entity)

	case txtyp.TX_TYPE_CUTOFF_PAIR:
		err = res.FromCutoffPairEntity(entity)

	case txtyp.TX_TYPE_CONVERT_INCOME:
		if view.Symbol == txtyp.SYMBOL_WETH {
			err = res.FromWethDepositEntity(entity)
		} else {
			err = res.FromWethWithdrawalEntity(entity)
		}

	case txtyp.TX_TYPE_CONVERT_OUTCOME:
		if view.Symbol == txtyp.SYMBOL_WETH {
			err = res.FromWethWithdrawalEntity(entity)
		} else {
			err = res.FromWethDepositEntity(entity)
		}

	case txtyp.TX_TYPE_SEND, txtyp.TX_TYPE_RECEIVE:
		err = res.FromTransferEntity(entity)

	case txtyp.TX_TYPE_SELL, txtyp.TX_TYPE_BUY, txtyp.TX_TYPE_LRC_FEE, txtyp.TX_TYPE_LRC_REWARD:
		err = res.FromFillEntity(entity)
	}

	return res, err
}

func validateOwner(ownerStr string) bool {
	if ownerStr == "" {
		return false
	}
	return true
}

func validateTxHashList(list []string) bool {
	if len(list) == 0 {
		return false
	}
	return true
}

func safeOwner(ownerStr string) string           { return common.HexToAddress(ownerStr).Hex() }
func safeStatus(statusStr string) types.TxStatus { return types.StrToTxStatus(statusStr) }
func safeType(typStr string) txtyp.TxType        { return txtyp.StrToTxType(typStr) }
func safeSymbol(symbol string) string            { return strings.ToUpper(symbol) }

func protocolToSymbol(address common.Address) string {
	if address == types.NilAddress {
		return txtyp.SYMBOL_ETH
	}
	symbol := util.AddressToAlias(address.Hex())
	return safeSymbol(symbol)
}

func symbolToProtocol(symbol string) common.Address {
	symbol = safeSymbol(symbol)
	if symbol == txtyp.SYMBOL_ETH {
		return types.NilAddress
	}
	return util.AliasToAddress(symbol)
}

func txLogIndexStr(txhash string, logindex int64) string {
	logindexstr := strconv.Itoa(int(logindex))
	return txhash + "-" + logindexstr
}
