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
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// send/receive/sell/buy/wrap/unwrap/cancelOrder/approve
const (
	TX_STATUS_PENDING = 0
	TX_STATUS_SUCCESS = 1
	TX_STATUS_FAILED  = 2

	TX_TYPE_APPROVE      = 1
	TX_TYPE_SEND         = 2 // SEND
	TX_TYPE_RECEIVE      = 3
	TX_TYPE_SELL         = 4 // SELL
	TX_TYPE_BUY          = 5
	TX_TYPE_WRAP         = 6 // WETH DEPOSIT
	TX_TYPE_UNWRAP       = 7 // WETH WITHDRAWAL
	TX_TYPE_CANCEL_ORDER = 8
)

type Transaction struct {
	Protocol    common.Address
	Owner       common.Address
	From        common.Address
	To          common.Address
	TxHash      common.Hash
	OrderHash   common.Hash
	BlockNumber *big.Int
	Value       *big.Int
	Type        uint8
	Status      uint8
	CreateTime  int64
	UpdateTime  int64
}

func (tx *Transaction) StatusStr() string {
	var ret string
	switch tx.Status {
	case TX_STATUS_PENDING:
		ret = "pending"
	case TX_STATUS_SUCCESS:
		ret = "success"
	case TX_STATUS_FAILED:
		ret = "failed"
	}

	return ret
}

func (tx *Transaction) TypeStr() string {
	var ret string

	switch tx.Type {
	case TX_TYPE_APPROVE:
		ret = "approve"
	case TX_TYPE_SEND:
		ret = "send"
	case TX_TYPE_RECEIVE:
		ret = "receive"
	case TX_TYPE_SELL:
		ret = "sell"
	case TX_TYPE_BUY:
		ret = "buy"
	case TX_TYPE_WRAP:
		ret = "wrap"
	case TX_TYPE_UNWRAP:
		ret = "unwrap"
	case TX_TYPE_CANCEL_ORDER:
		ret = "cancel_order"
	}

	return ret
}

func (tx *Transaction) FromOrder(src *Order, txhash common.Hash, to common.Address, txtype, status uint8, blockNumber *big.Int, nowtime int64) error {
	tx.Protocol = src.Protocol
	tx.Owner = src.Owner
	tx.From = src.Owner
	tx.To = to
	tx.Type = txtype
	tx.Status = status
	if txtype == TX_TYPE_SELL {
		tx.Value = src.AmountS
	} else {
		tx.Value = src.AmountB
	}
	tx.TxHash = txhash
	tx.OrderHash = src.Hash
	tx.BlockNumber = blockNumber
	tx.CreateTime = nowtime
	tx.UpdateTime = nowtime
	return nil
}

func (tx *Transaction) FromFillEvent(src *OrderFilledEvent, to common.Address, txtype, status uint8) error {
	tx.Protocol = src.ContractAddress
	tx.Owner = src.Owner
	tx.From = src.Owner
	tx.To = to
	tx.Type = txtype
	tx.Status = status
	if txtype == TX_TYPE_SELL {
		tx.Value = src.AmountS
	} else {
		tx.Value = src.AmountB
	}
	tx.TxHash = src.TxHash
	tx.OrderHash = src.OrderHash
	tx.BlockNumber = src.Blocknumber
	tx.CreateTime = src.Time.Int64()
	tx.UpdateTime = src.Time.Int64()

	return nil
}

func (tx *Transaction) FromCancelMethod(src *Order, txhash common.Hash, status uint8, value, blockNumber *big.Int, nowtime int64) error {
	tx.Protocol = src.Protocol
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CANCEL_ORDER
	tx.Status = status
	tx.Value = value
	tx.TxHash = txhash
	tx.OrderHash = src.Hash
	tx.BlockNumber = blockNumber
	tx.CreateTime = nowtime
	tx.UpdateTime = nowtime

	return nil
}

func (tx *Transaction) FromCancelEvent(src *OrderCancelledEvent, owner common.Address, status uint8) error {
	tx.Protocol = src.ContractAddress
	tx.Owner = owner
	tx.Type = TX_TYPE_CANCEL_ORDER
	tx.Status = status
	tx.Value = src.AmountCancelled
	tx.TxHash = src.TxHash
	tx.OrderHash = src.OrderHash
	tx.BlockNumber = src.Blocknumber
	tx.CreateTime = src.Time.Int64()
	tx.UpdateTime = src.Time.Int64()

	return nil
}

func (tx *Transaction) FromWethDepositMethod(src *WethDepositMethodEvent, status uint8) error {
	tx.Protocol = src.ContractAddress
	tx.Owner = src.From
	tx.From = src.From
	tx.To = src.To
	tx.Value = src.Value
	tx.Type = TX_TYPE_WRAP
	tx.Status = status
	tx.TxHash = src.TxHash
	tx.BlockNumber = src.Blocknumber
	tx.CreateTime = src.Time.Int64()
	tx.UpdateTime = src.Time.Int64()

	return nil
}

func (tx *Transaction) FromWethWithdrawalMethod(src *WethWithdrawalMethodEvent, status uint8) error {
	tx.Protocol = src.ContractAddress
	tx.Owner = src.From
	tx.From = src.From
	tx.To = src.To
	tx.Value = src.Value
	tx.Type = TX_TYPE_UNWRAP
	tx.Status = status
	tx.TxHash = src.TxHash
	tx.BlockNumber = src.Blocknumber
	tx.CreateTime = src.Time.Int64()
	tx.UpdateTime = src.Time.Int64()

	return nil
}

func (tx *Transaction) FromApproveMethod(src *ApproveMethodEvent, status uint8) error {
	tx.Protocol = src.ContractAddress
	tx.Owner = src.From
	tx.From = src.From
	tx.To = src.To
	tx.Value = src.Value
	tx.Type = TX_TYPE_APPROVE
	tx.Status = status
	tx.TxHash = src.TxHash
	tx.BlockNumber = src.Blocknumber
	tx.CreateTime = src.Time.Int64()
	tx.UpdateTime = src.Time.Int64()

	return nil
}

func (tx *Transaction) FromTransferEvent(src *TransferEvent, txhash common.Hash, sendOrReceive, status uint8) error {
	tx.Protocol = src.ContractAddress
	if sendOrReceive == TX_TYPE_SEND {
		tx.Owner = src.From
		tx.From = src.From
		tx.To = src.To
	} else {
		tx.Owner = src.To
		tx.From = src.From
		tx.To = src.To
	}
	tx.Value = src.Value
	tx.Type = sendOrReceive
	tx.Status = status
	tx.TxHash = txhash
	tx.BlockNumber = src.Blocknumber
	tx.CreateTime = src.Time.Int64()
	tx.UpdateTime = src.Time.Int64()

	return nil
}