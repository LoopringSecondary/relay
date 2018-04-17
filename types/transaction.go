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

type (
	TxStatus uint8
	TxType   uint8
)

const (
	TX_STATUS_UNKNOWN TxStatus = 0
	TX_STATUS_PENDING TxStatus = 1
	TX_STATUS_SUCCESS TxStatus = 2
	TX_STATUS_FAILED  TxStatus = 3

	TX_TYPE_UNKNOWN              TxType = 0
	TX_TYPE_APPROVE              TxType = 1
	TX_TYPE_SEND                 TxType = 2 // SEND
	TX_TYPE_RECEIVE              TxType = 3
	TX_TYPE_SELL                 TxType = 4 // SELL
	TX_TYPE_BUY                  TxType = 5
	TX_TYPE_CONVERT_INCOME       TxType = 7 // WETH WITHDRAWAL
	TX_TYPE_CONVERT_OUTCOME      TxType = 8
	TX_TYPE_CANCEL_ORDER         TxType = 9
	TX_TYPE_CUTOFF               TxType = 10
	TX_TYPE_CUTOFF_PAIR          TxType = 11
	TX_TYPE_UNSUPPORTED_CONTRACT TxType = 12
	TX_TYPE_TRANSFER             TxType = 101
	TX_TYPE_DEPOSIT              TxType = 102
	TX_TYPE_WITHDRAWAL           TxType = 103
)

// todo(fuk): mark,transaction不包含sell&buy

type TxInfo struct {
	Protocol    common.Address `json:"protocol"`
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	TxHash      common.Hash    `json:"txHash"`
	BlockHash   common.Hash    `json:"symbol"`
	TxIndex     int64          `json:txIndex`
	LogIndex    int64          `json:"logIndex"`
	BlockNumber *big.Int       `json:"blockNumber"`
	BlockTime   int64          `json:"block_time"`
	Status      TxStatus       `json:"status"`
	GasLimit    *big.Int       `json:"gas_limit"`
	GasUsed     *big.Int       `json:"gas_used"`
	GasPrice    *big.Int       `json:"gas_price"`
	Nonce       *big.Int       `json:"nonce"`
}

type Transaction struct {
	TxInfo
	Symbol     string         `json:"symbol"`
	Protocol   common.Address `json:"protocol"`
	Content    []byte         `json:"content"`
	Value      *big.Int       `json:"value"`
	Type       TxType         `json:"type"`
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
	default:
		ret = "unknown"
	}

	return ret
}

func StrToTxStatus(txType string) TxStatus {
	var ret TxStatus
	switch txType {
	case "pending":
		ret = TX_STATUS_PENDING
	case "success":
		ret = TX_STATUS_SUCCESS
	case "failed":
		ret = TX_STATUS_FAILED
	default:
		ret = TX_STATUS_UNKNOWN
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
	case TX_TYPE_CONVERT_INCOME:
		ret = "convert_income"
	case TX_TYPE_CONVERT_OUTCOME:
		ret = "convert_outcome"
	case TX_TYPE_CANCEL_ORDER:
		ret = "cancel_order"
	case TX_TYPE_CUTOFF:
		ret = "cutoff"
	case TX_TYPE_CUTOFF_PAIR:
		ret = "cutoff_trading_pair"
	case TX_TYPE_UNSUPPORTED_CONTRACT:
		ret = "unsupported_contract"
	case TX_TYPE_TRANSFER:
		ret = "transfer"
	case TX_TYPE_DEPOSIT:
		ret = "deposit"
	case TX_TYPE_WITHDRAWAL:
		ret = "withdrawal"
	default:
		ret = "unknown"
	}

	return ret
}

func StrToTxType(status string) TxType {
	var ret TxType

	switch status {
	case "approve":
		ret = TX_TYPE_APPROVE
	case "send":
		ret = TX_TYPE_SEND
	case "receive":
		ret = TX_TYPE_RECEIVE
	case "sell":
		ret = TX_TYPE_SELL
	case "buy":
		ret = TX_TYPE_BUY
	case "convert_income":
		ret = TX_TYPE_CONVERT_INCOME
	case "convert_outcome":
		ret = TX_TYPE_CONVERT_OUTCOME
	case "cancel_order":
		ret = TX_TYPE_CANCEL_ORDER
	case "cutoff":
		ret = TX_TYPE_CUTOFF
	case "cutoff_trading_pair":
		ret = TX_TYPE_CUTOFF_PAIR
	case "unsupported_contract":
		ret = TX_TYPE_UNSUPPORTED_CONTRACT
	case "transfer":
		ret = TX_TYPE_TRANSFER
	case "deposit":
		ret = TX_TYPE_DEPOSIT
	case "withdrawal":
		ret = TX_TYPE_WITHDRAWAL
	default:
		ret = TX_TYPE_UNKNOWN
	}

	return ret
}

func (tx *Transaction) FromCancelEvent(src *OrderCancelledEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.From
	tx.To = src.To
	tx.Type = TX_TYPE_CANCEL_ORDER
	tx.Value = src.AmountCancelled
	tx.Content = []byte(src.OrderHash.Hex())

	return nil
}

func (tx *Transaction) FromCutoffEvent(src *CutoffEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.Owner
	tx.To = src.To
	tx.Type = TX_TYPE_CUTOFF
	tx.Value = src.Cutoff

	return nil
}

type CutoffPairSalt struct {
	Token1 common.Address `json:"token1"`
	Token2 common.Address `json:"token2"`
}

func (tx *Transaction) FromCutoffPairEvent(src *CutoffPairEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.Owner
	tx.To = src.To
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

func (tx *Transaction) GetCancelOrderHash() (string, error) {
	if tx.Type != TX_TYPE_CANCEL_ORDER {
		return "", fmt.Errorf("cutoff pair salt,transaction type error:%d", tx.Type)
	}
	return string(tx.Content), nil
}

// 充值和提现from和to都是用户钱包自己的地址，因为合约限制了发送方msg.sender
func (tx *Transaction) FromWethDepositEvent(src *WethDepositEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.Dst
	tx.To = src.Dst
	tx.Value = src.Value
	tx.Type = TX_TYPE_DEPOSIT

	return nil
}

func (tx *Transaction) FromWethWithdrawalEvent(src *WethWithdrawalEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.Src
	tx.To = src.Src
	tx.Value = src.Value
	tx.Type = TX_TYPE_WITHDRAWAL

	return nil
}

func (tx *Transaction) FromApproveEvent(src *ApprovalEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.Owner
	tx.To = src.Spender
	tx.Value = src.Value
	tx.Type = TX_TYPE_APPROVE

	return nil
}

func (tx *Transaction) FromTransferEvent(src *TransferEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.Sender
	tx.To = src.Receiver
	tx.Value = src.Value
	tx.Type = TX_TYPE_TRANSFER

	return nil
}

func (tx *Transaction) fullFilled(txinfo TxInfo) {
	tx.TxInfo = txinfo
	tx.Protocol = txinfo.Protocol
	tx.CreateTime = txinfo.BlockTime
	tx.UpdateTime = txinfo.BlockTime
}
