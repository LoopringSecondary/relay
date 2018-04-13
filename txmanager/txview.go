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
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/go-crypto/keys/tx"
)

type TransactionJsonResult struct {
	Protocol    common.Address     `json:"protocol"`
	From        common.Address     `json:"from"`
	To          common.Address     `json:"to"`
	TxHash      common.Hash        `json:"txHash"`
	Symbol      string             `json:"symbol"`
	Content     TransactionContent `json:"content"`
	BlockNumber int64              `json:"blockNumber"`
	Value       string             `json:"value"`
	LogIndex    int64              `json:"logIndex"`
	Type        string             `json:"type"`
	Status      string             `json:"status"`
	CreateTime  int64              `json:"createTime"`
	UpdateTime  int64              `json:"updateTime"`
	Nonce       string             `json:"nonce"`
}

type TransactionContent struct {
	Market    string `json:"market"`
	OrderHash string `json:"orderHash"`
}

func (dst TransactionJsonResult) fromTransaction(tx types.Transaction) {
	dst.Protocol = tx.Protocol
	dst.From = tx.From
	dst.To = tx.To
	dst.TxHash = tx.TxHash
	dst.BlockNumber = tx.BlockNumber.Int64()
	dst.LogIndex = tx.LogIndex
	dst.Type = tx.TypeStr()
	dst.Status = tx.StatusStr()
	dst.CreateTime = tx.CreateTime
	dst.UpdateTime = tx.UpdateTime
	dst.Symbol = tx.Symbol
	dst.Nonce = tx.TxInfo.Nonce.String()

	// fill content
	if tx.Type == types.TX_TYPE_CUTOFF_PAIR {
		if ctx, err := tx.GetCutoffPairContent(); err == nil {
			if mkt, err := util.WrapMarketByAddress(ctx.Token1.Hex(), ctx.Token2.Hex()); err == nil {
				dst.Content = TransactionContent{Market: mkt}
			}
		}
	} else if tx.Type == types.TX_TYPE_CANCEL_ORDER {
		if ctx, err := tx.GetCancelOrderHash(); err == nil {
			dst.Content = TransactionContent{OrderHash: ctx}
		}
	}

	// set value
	if tx.Value == nil {
		dst.Value = "0"
	} else {
		dst.Value = tx.Value.String()
	}
}

// copy create copy while special type
func copy(src types.Transaction, owner common.Address) []types.Transaction {
	var (
		list []types.Transaction
		dst  = src
	)

	if src.Type == types.TX_TYPE_TRANSFER {
		if owner == src.From {
			dst.Type = types.TX_TYPE_SEND
		}
		if owner == src.To {
			dst.Type = types.TX_TYPE_RECEIVE
		}
		list = append(list, dst)
	}
	if src.Type == types.TX_TYPE_DEPOSIT {
		src.Type = types.TX_TYPE_CONVERT_INCOME
		dst.Type = types.TX_TYPE_CONVERT_OUTCOME
		dst.Symbol = ETH_SYMBOL
		dst.Protocol = types.NilAddress
		list = append(list, dst)
	}
	if src.Type == types.TX_TYPE_WITHDRAWAL {
		src.Type = types.TX_TYPE_CONVERT_OUTCOME
		dst.Type = types.TX_TYPE_CONVERT_INCOME
		dst.Symbol = ETH_SYMBOL
		dst.Protocol = types.NilAddress
	}

	list = append(list, src)

	return list
}

type TransactionView interface {
	GetPendingTransactions(owner common.Address) []*types.Transaction
	GetMinedTransactions(owner common.Address, symbol string, page, size int) []*types.Transaction
	GetTransactionsByHash(hashList []common.Hash) []*types.Transaction
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

	return dbListToView(txs)
}

func dbItemToView(src dao.Transaction) TransactionJsonResult {
	var dst TransactionJsonResult
	tx := &types.Transaction{}
	src.ConvertUp(tx)
	dst.fromTransaction(tx)
	return dst
}

func dbListToView(items []dao.Transaction) []TransactionJsonResult {
	var list []TransactionJsonResult
	for _, v := range items {
		res := dbItemToView(v)
		list = append(list, res)
	}
	return list
}
