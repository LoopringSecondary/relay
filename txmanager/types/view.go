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

package types

import (
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type TransactionView struct {
	Symbol      string         `json:"symbol"`
	Owner       common.Address `json:"owner"` // 用户地址
	To          common.Address `json:"to"`    // 如果是转账相关则为receiver,如果是打个合约则为to
	TxHash      common.Hash    `json:"tx_hash"`
	BlockNumber int64          `json:"block_number"`
	LogIndex    int64          `json:"log_index"`
	Amount      *big.Int       `json:"amount"`
	Nonce       *big.Int       `json:"nonce"`
	Type        TxType         `json:"type"`
	Status      types.TxStatus `json:"status"`
	CreateTime  int64          `json:"create_time"`
	UpdateTime  int64          `json:"update_time"`
}

func ApproveView(src *types.ApprovalEvent) (TransactionView, error) {
	var (
		tx  TransactionView
		err error
	)

	if tx.Symbol, err = util.GetSymbolWithAddress(src.To); err != nil {
		return tx, err
	}
	tx.fullFilled(src.TxInfo)

	tx.Owner = src.Owner
	tx.To = src.Spender
	tx.Amount = src.Amount
	tx.Type = TX_TYPE_APPROVE

	return tx, nil
}

// 从entity中获取amount&orderHash
func CancelView(src *types.OrderCancelledEvent) TransactionView {
	var tx TransactionView

	tx.Symbol = ETH_SYMBOL
	tx.fullFilled(src.TxInfo)

	tx.Owner = src.From
	tx.To = src.To
	tx.Type = TX_TYPE_CANCEL_ORDER

	return tx
}

func CutoffView(src *types.CutoffEvent) TransactionView {
	var tx TransactionView

	tx.fullFilled(src.TxInfo)
	tx.Symbol = ETH_SYMBOL
	tx.Owner = src.Owner
	tx.To = src.To
	tx.Amount = src.Cutoff
	tx.Type = TX_TYPE_CUTOFF

	return tx
}

// 从entity中获取token1,token2
func CutoffPairView(src *types.CutoffPairEvent) TransactionView {
	var tx TransactionView

	tx.fullFilled(src.TxInfo)
	tx.Symbol = ETH_SYMBOL
	tx.Amount = src.Cutoff
	tx.Owner = src.Owner
	tx.To = src.To
	tx.Type = TX_TYPE_CUTOFF_PAIR

	return tx
}

func WethDepositView(src *types.WethDepositEvent) []TransactionView {
	var (
		list     []TransactionView
		tx1, tx2 TransactionView
	)

	tx1.fullFilled(src.TxInfo)
	tx1.Owner = src.Dst
	tx1.To = src.To
	tx1.Amount = src.Amount
	tx1.Symbol = ETH_SYMBOL
	tx1.Type = TX_TYPE_CONVERT_OUTCOME

	tx2 = tx1
	tx2.Symbol = WETH_SYMBOL
	tx2.Type = TX_TYPE_CONVERT_INCOME

	list = append(list, tx1, tx2)
	return list
}

func WethWithdrawalView(src *types.WethWithdrawalEvent) []TransactionView {
	var (
		list     []TransactionView
		tx1, tx2 TransactionView
	)

	tx1.fullFilled(src.TxInfo)
	tx1.Owner = src.Src
	tx1.To = src.To
	tx1.Amount = src.Amount
	tx1.Symbol = ETH_SYMBOL
	tx1.Type = TX_TYPE_CONVERT_INCOME

	tx2 = tx1
	tx2.Symbol = WETH_SYMBOL
	tx2.Type = TX_TYPE_CONVERT_OUTCOME

	list = append(list, tx1, tx2)

	return list
}

func TransferView(src *types.TransferEvent) ([]TransactionView, error) {
	var (
		list     []TransactionView
		tx1, tx2 TransactionView
		err      error
	)

	if tx1.Symbol, err = util.GetSymbolWithAddress(src.To); err != nil {
		return list, err
	}
	tx1.fullFilled(src.TxInfo)
	tx1.Amount = src.Amount

	tx1.Owner = src.Sender
	tx1.Type = TX_TYPE_SEND

	tx2 = tx1
	tx2.Owner = src.Receiver
	tx2.Type = TX_TYPE_RECEIVE

	list = append(list, tx1, tx2)
	return list, nil
}

func EthTransferView(src *types.TransferEvent) []TransactionView {
	var (
		list     []TransactionView
		tx1, tx2 TransactionView
	)

	tx1.fullFilled(src.TxInfo)
	tx1.Amount = src.Value
	tx1.Symbol = ETH_SYMBOL

	if src.Value.Cmp(big.NewInt(0)) > 0 {
		tx1.Owner = src.From
		tx1.Type = TX_TYPE_SEND

		tx2 = tx1
		tx2.Owner = src.To
		tx2.Type = TX_TYPE_RECEIVE
	} else {
		tx1.Type = TX_TYPE_UNSUPPORTED_CONTRACT
		tx1.Owner = src.From

		tx2 = tx1
		tx2.Owner = src.To
	}

	return list
}

func RelatedOwners(viewList []TransactionView) []common.Address {
	var list []common.Address
	for _, v := range viewList {
		list = append(list, v.Owner)
	}
	return list
}

func (tx *TransactionView) fullFilled(src types.TxInfo) {
	tx.TxHash = src.TxHash
	tx.BlockNumber = src.BlockNumber.Int64()
	tx.LogIndex = src.TxLogIndex
	tx.Status = src.Status
	tx.Nonce = src.Nonce
	tx.CreateTime = src.BlockTime
	tx.UpdateTime = src.BlockTime
}

// todo fill
