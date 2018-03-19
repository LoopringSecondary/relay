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

	TX_TYPE_APPROVE = 1
	TX_TYPE_SEND    = 2 // SEND
	TX_TYPE_RECEIVE = 3
	TX_TYPE_SELL    = 4 // SELL
	TX_TYPE_BUY     = 5
	//TX_TYPE_WRAP         = 6 // WETH DEPOSIT
	TX_TYPE_CONVERT      = 7 // WETH WITHDRAWAL
	TX_TYPE_CANCEL_ORDER = 8
	TX_TYPE_CUTOFF       = 9
	TX_TYPE_CUTOFF_PAIR  = 10
)

// todo(fuk): mark,transaction不包含sell&buy

type TxInfo struct {
	Protocol    common.Address `json:"protocol"`
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	TxHash      common.Hash    `json:"txHash"`
	BlockHash   common.Hash    `json:"symbol"`
	LogIndex    int64          `json:"logIndex"`
	BlockNumber *big.Int       `json:"blockNumber`
	BlockTime   int64          `json:"block_time"`
	TxFailed    bool           `json:"tx_failed"`
	Symbol      string         `json:"symbol"`
	GasLimit    *big.Int       `json:"gas_limit"`
	GasUsed     *big.Int       `json:"gas_used"`
	GasPrice    *big.Int       `json:"gas_price"`
	Nonce       *big.Int       `json:"nonce"`
}

type Transaction struct {
	TxInfo
	Owner      common.Address `json:"owner"`
	Content    []byte         `json:"content"`
	Value      *big.Int       `json:"value"`
	Type       uint8          `json:"type"`
	Status     uint8          `json:"status"`
	CreateTime int64          `json:"createTime"`
	UpdateTime int64          `json:"updateTime"`
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
	case TX_TYPE_CONVERT:
		ret = "convert"
	case TX_TYPE_CANCEL_ORDER:
		ret = "cancel_order"
	case TX_TYPE_CUTOFF:
		ret = "cutoff"
	case TX_TYPE_CUTOFF_PAIR:
		ret = "cutoff_trading_pair"
	}

	return ret
}

// todo(fuk): delete useless function
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

// todo(fuk): delete useless function
func (tx *Transaction) GetOrderContent() (common.Hash, error) {
	if tx.Type == TX_TYPE_BUY || tx.Type == TX_TYPE_SELL {
		return common.HexToHash(string(tx.Content)), nil
	} else {
		return NilHash, fmt.Errorf("get order salt,transaction type error:%d", tx.Type)
	}
}

func (tx *Transaction) FromFillEvent(src *OrderFilledEvent, to common.Address, txtype uint8) error {
	tx.fullFilled(src.TxInfo)
	tx.Owner = src.Owner
	tx.From = src.Owner
	tx.To = to
	tx.Type = txtype
	if txtype == TX_TYPE_SELL {
		tx.Value = src.AmountS
	} else {
		tx.Value = src.AmountB
	}
	tx.Content = []byte(src.OrderHash.Hex())

	return nil
}

func (tx *Transaction) GetFillContent() (common.Hash, error) {
	if tx.Type == TX_TYPE_SELL || tx.Type == TX_TYPE_BUY {
		return common.HexToHash(string(tx.Content)), nil
	} else {
		return NilHash, fmt.Errorf("get fill salt,transaction type error:%d", tx.Type)
	}
}

func (tx *Transaction) FromCancelMethod(txinfo TxInfo, value *big.Int) error {
	tx.fullFilled(txinfo)
	tx.Owner = txinfo.From
	tx.Type = TX_TYPE_CANCEL_ORDER
	tx.Value = value

	return nil
}

func (tx *Transaction) FromCancelEvent(src *OrderCancelledEvent, owner common.Address) error {
	tx.fullFilled(src.TxInfo)
	tx.Owner = owner
	tx.Type = TX_TYPE_CANCEL_ORDER
	tx.Value = src.AmountCancelled
	tx.Content = []byte(src.OrderHash.Hex())

	return nil
}

func (tx *Transaction) FromCutoffEvent(src *CutoffEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CUTOFF
	tx.Value = src.Cutoff

	return nil
}

func (tx *Transaction) FromCutoffMethodEvent(src *CutoffMethodEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CUTOFF
	tx.Value = src.Value

	return nil
}

type CutoffPairSalt struct {
	Token1 common.Address `json:"token1"`
	Token2 common.Address `json:"token2"`
}

func (tx *Transaction) FromCutoffPairEvent(src *CutoffPairEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CUTOFF_PAIR
	tx.Value = src.Cutoff

	var salt CutoffPairSalt
	salt.Token1 = src.Token1
	salt.Token2 = src.Token2
	if bs, err := json.Marshal(salt); err != nil {
		return err
	} else {
		tx.Content = bs
	}

	return nil
}

func (tx *Transaction) FromCutoffPairMethod(src *CutoffPairMethodEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.Owner = src.Owner
	tx.Type = TX_TYPE_CUTOFF_PAIR
	tx.Value = src.Value

	var salt CutoffPairSalt
	salt.Token1 = src.Token1
	salt.Token2 = src.Token2
	if bs, err := json.Marshal(salt); err != nil {
		return err
	} else {
		tx.Content = bs
	}

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
	tx.fullFilled(src.TxInfo)
	tx.Owner = src.From
	tx.Value = src.Value
	tx.Type = TX_TYPE_CONVERT

	return nil
}

func (tx *Transaction) FromWethWithdrawalMethod(src *WethWithdrawalMethodEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.Owner = src.Owner
	tx.Value = src.Value
	tx.Type = TX_TYPE_CONVERT

	return nil
}

func (tx *Transaction) FromApproveMethod(src *ApproveMethodEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.Owner = src.From
	tx.Value = src.Value
	tx.Type = TX_TYPE_APPROVE

	return nil
}

func (tx *Transaction) FromTransferEvent(src *TransferEvent, sendOrReceive uint8) error {
	tx.fullFilled(src.TxInfo)
	if sendOrReceive == TX_TYPE_SEND {
		tx.Owner = src.Sender
	} else {
		tx.Owner = src.Receiver
	}
	tx.Value = src.Value
	tx.Type = sendOrReceive

	return nil
}

func (tx *Transaction) fullFilled(txinfo TxInfo) {
	tx.TxInfo = txinfo
	if txinfo.TxFailed {
		tx.Status = TX_STATUS_FAILED
	} else {
		tx.Status = TX_STATUS_SUCCESS
	}
	tx.CreateTime = txinfo.BlockTime
	tx.UpdateTime = txinfo.BlockTime
}
