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
	"encoding/json"
	"fmt"
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
	TX_TYPE_CUTOFF       = 9
	TX_TYPE_CUTOFF_PAIR  = 10
)

type Transaction struct {
	Protocol    common.Address
	Owner       common.Address
	From        common.Address
	To          common.Address
	TxHash      common.Hash
	Content     []byte
	BlockNumber *big.Int
	Value       *big.Int
	LogIndex    int64
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
	case TX_TYPE_CUTOFF:
		ret = "cutoff"
	case TX_TYPE_CUTOFF_PAIR:
		ret = "cutoff_trading_pair"
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
	tx.Content = []byte(src.Hash.Hex())
	tx.BlockNumber = blockNumber
	tx.CreateTime = nowtime
	tx.UpdateTime = nowtime
	return nil
}

func (tx *Transaction) GetOrderContent() (common.Hash, error) {
	if tx.Type == TX_TYPE_BUY || tx.Type == TX_TYPE_SELL {
		return common.HexToHash(string(tx.Content)), nil
	} else {
		return NilHash, fmt.Errorf("get order salt,transaction type error:%d", tx.Type)
	}
}

func (tx *Transaction) FromFillEvent(src *OrderFilledEvent, to common.Address, txtype uint8) error {
	tx.Owner = src.Owner
	tx.From = src.Owner
	tx.To = to
	tx.Type = txtype
	if txtype == TX_TYPE_SELL {
		tx.Value = src.AmountS
	} else {
		tx.Value = src.AmountB
	}

	tx.LogIndex = src.LogIndex
	tx.Content = []byte(src.OrderHash.Hex())
	tx.fullFilled(src.TxInfo)

	return nil
}

func (tx *Transaction) GetFillContent() (common.Hash, error) {
	if tx.Type == TX_TYPE_SELL || tx.Type == TX_TYPE_BUY {
		return common.HexToHash(string(tx.Content)), nil
	} else {
		return NilHash, fmt.Errorf("get fill salt,transaction type error:%d", tx.Type)
	}
}

func (tx *Transaction) FromCancelMethod(src *Order, txhash common.Hash, status uint8, value, blockNumber *big.Int, nowtime int64) error {
	tx.Protocol = src.Protocol
	tx.Owner = src.Owner
	tx.From = src.Owner
	tx.To = src.Protocol
	tx.Type = TX_TYPE_CANCEL_ORDER
	tx.Status = status
	tx.Value = value

	tx.Content = src.Hash.Bytes()
	tx.BlockNumber = blockNumber
	tx.LogIndex = 0
	tx.CreateTime = nowtime
	tx.UpdateTime = nowtime

	return nil
}

func (tx *Transaction) FromCancelEvent(src *OrderCancelledEvent, owner common.Address) error {
	tx.From = src.From
	tx.To = src.To
	tx.Owner = owner
	tx.Type = TX_TYPE_CANCEL_ORDER
	tx.Value = src.AmountCancelled
	tx.Content = src.OrderHash.Bytes()
	tx.LogIndex = src.LogIndex

	tx.fullFilled(src.TxInfo)

	return nil
}

func (tx *Transaction) FromCutoffEvent(src *AllOrdersCancelledEvent) error {
	tx.From = src.From
	tx.To = src.To
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CUTOFF
	tx.Value = src.Cutoff
	tx.LogIndex = src.LogIndex

	tx.fullFilled(src.TxInfo)
	return nil
}

func (tx *Transaction) FromCutoffMethodEvent(src *CutoffMethodEvent) error {
	tx.From = src.From
	tx.To = src.To
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CUTOFF
	tx.Value = src.Value
	tx.LogIndex = 0

	tx.fullFilled(src.TxInfo)
	return nil
}

type CutoffPairSalt struct {
	Token1 common.Address `json:"token1"`
	Token2 common.Address `json:"token2"`
}

func (tx *Transaction) FromCutoffPairEvent(src *OrdersCancelledEvent) error {
	tx.From = src.From
	tx.To = src.To
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CUTOFF_PAIR
	tx.Value = src.Cutoff
	tx.LogIndex = src.LogIndex

	var salt CutoffPairSalt
	salt.Token1 = src.Token1
	salt.Token2 = src.Token2
	if bs, err := json.Marshal(salt); err != nil {
		return err
	} else {
		tx.Content = bs
	}

	tx.fullFilled(src.TxInfo)
	return nil
}

func (tx *Transaction) FromCutoffPairMethodEvent(src *CutoffPairMethodEvent) error {
	tx.From = src.From
	tx.To = src.To
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CUTOFF_PAIR
	tx.Value = src.Value
	tx.LogIndex = 0

	var salt CutoffPairSalt
	salt.Token1 = src.Token1
	salt.Token2 = src.Token2
	if bs, err := json.Marshal(salt); err != nil {
		return err
	} else {
		tx.Content = bs
	}

	tx.fullFilled(src.TxInfo)
	return nil
}

func (tx *Transaction) GetCutoffPairContent() (*CutoffPairSalt, error) {
	if tx.Type != TX_TYPE_CUTOFF_PAIR {
		return nil, fmt.Errorf("cutoff pair salt,transaction type error:%d", tx.Type)
	}
	var cutoffpair CutoffPairSalt
	if err := json.Unmarshal(tx.Content, &cutoffpair); err != nil {
		return nil, err
	}

	return &cutoffpair, nil
}

func (tx *Transaction) FromWethDepositMethod(src *WethDepositMethodEvent) error {
	tx.Owner = src.From
	tx.From = src.From
	tx.To = src.To
	tx.Value = src.Value
	tx.Type = TX_TYPE_WRAP
	tx.LogIndex = 0

	tx.fullFilled(src.TxInfo)

	return nil
}

func (tx *Transaction) FromWethWithdrawalMethod(src *WethWithdrawalMethodEvent) error {
	tx.Owner = src.Owner
	tx.From = src.From
	tx.To = src.To
	tx.Value = src.Value
	tx.Type = TX_TYPE_UNWRAP
	tx.LogIndex = 0

	tx.fullFilled(src.TxInfo)

	return nil
}

func (tx *Transaction) FromApproveMethod(src *ApproveMethodEvent) error {
	tx.Owner = src.From
	tx.From = src.From
	tx.To = src.To
	tx.Value = src.Value
	tx.Type = TX_TYPE_APPROVE
	tx.LogIndex = 0

	tx.fullFilled(src.TxInfo)

	return nil
}

func (tx *Transaction) FromTransferEvent(src *TransferEvent, sendOrReceive uint8) error {
	if sendOrReceive == TX_TYPE_SEND {
		tx.Owner = src.Sender
	} else {
		tx.Owner = src.Receiver
	}
	tx.From = src.Sender
	tx.To = src.Receiver
	tx.Value = src.Value
	tx.Type = sendOrReceive
	tx.LogIndex = src.LogIndex

	tx.fullFilled(src.TxInfo)

	return nil
}

func (tx *Transaction) fullFilled(txinfo TxInfo) {
	tx.Protocol = txinfo.Protocol
	if txinfo.TxFailed {
		tx.Status = TX_STATUS_FAILED
	} else {
		tx.Status = TX_STATUS_SUCCESS
	}
	tx.TxHash = txinfo.TxHash
	tx.BlockNumber = txinfo.BlockNumber
	tx.CreateTime = txinfo.BlockTime
	tx.UpdateTime = txinfo.BlockTime
}
