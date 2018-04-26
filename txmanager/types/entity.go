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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type TransactionEntity struct {
	From        common.Address `json:"from"`
	To          common.Address `json:"to"`
	BlockNumber int64          `json:"block_number"`
	Hash        common.Hash    `json:"hash"`
	LogIndex    int64          `json:"log_index"`
	Value       *big.Int       `json:"value"`
	Content     string         `json:"content"`
	Status      types.TxStatus `json:"status"`
	GasLimit    *big.Int       `json:"gas_limit"`
	GasUsed     *big.Int       `json:"gas_used"`
	GasPrice    *big.Int       `json:"gas_price"`
	Nonce       *big.Int       `json:"nonce"`
	BlockTime   int64          `json:"block_time"`
}

type ApproveContent struct {
	Owner   string `json:"owner"`
	Spender string `json:"spender"`
	Amount  string `json:"amount"`
}

type CancelContent struct {
	OrderHash string `json:"order_hash"`
	Amount    string `json:"amount"`
}

func ParseCancelContent(str string) (CancelContent, error) {
	var content CancelContent
	err := json.Unmarshal([]byte(str), &content)
	return content, err
}

type CutoffContent struct {
	Owner           string `json:"owner"`
	CutoffTimeStamp int64  `json:"cutoff"`
}

type CutoffPairContent struct {
	Owner           string `json:"owner"`
	Token1          string `json:"token1"`
	Token2          string `json:"token2"`
	CutoffTimeStamp int64  `json:"cutoff"`
}

func ParseCutoffPairContent(str string) (CutoffPairContent, error) {
	var content CutoffPairContent
	err := json.Unmarshal([]byte(str), &content)
	return content, err
}

type WethWithdrawalContent struct {
	Src    string `json:"src"`
	Amount string `json:"amount"`
}

type WethDepositContent struct {
	Dst    string `json:"dst"`
	Amount string `json:"amount"`
}

type TransferContent struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Amount   string `json:"amount"`
}

// todo fill
type OrderFilledContent struct {
	RingHash      string `json:"ring_hash"`
	PreOrderHash  string `json:"pre_order_hash"`
	OrderHash     string `json:"order_hash"`
	NextOrderHash string `json:"next_order_hash"`
	Owner         string `json:"owner"`
	TokenS        string `json:"token_s"`
	TokenB        string `json:"token_b"`
	SellTo        string `json:"sell_to"`
	BuyFrom       string `json:"buy_from"`
	RingIndex     string `json:"ring_index"`
	AmountS       string `json:"amount_s"`
	AmountB       string `json:"amount_b"`
	LrcReward     string `json:"lrc_reward"`
	LrcFee        string `json:"lrc_fee"`
	SplitS        string `json:"split_s"`
	SplitB        string `json:"split_b"`
	Market        string `json:"market"`
	FillIndex     string `json:"fill_index"`
}

func (tx *TransactionEntity) FromApproveEvent(src *types.ApprovalEvent) error {
	tx.fullFilled(src.TxInfo)

	var content ApproveContent
	content.Owner = src.Owner.Hex()
	content.Spender = src.Spender.Hex()
	content.Amount = src.Amount.String()

	bs, err := json.Marshal(&content)
	if err != nil {
		return err
	}

	tx.Content = string(bs)
	return nil
}

func (tx *TransactionEntity) FromCancelEvent(src *types.OrderCancelledEvent) error {
	tx.fullFilled(src.TxInfo)

	var content CancelContent
	content.OrderHash = src.OrderHash.Hex()
	content.Amount = src.AmountCancelled.String()

	bs, err := json.Marshal(&content)
	if err != nil {
		return err
	}

	tx.Content = string(bs)
	return nil
}

func (tx *TransactionEntity) FromCutoffEvent(src *types.CutoffEvent) error {
	tx.fullFilled(src.TxInfo)

	var content CutoffContent
	content.Owner = src.Owner.Hex()
	content.CutoffTimeStamp = src.Cutoff.Int64()

	bs, err := json.Marshal(&content)
	if err != nil {
		return err
	}

	tx.Content = string(bs)
	return nil
}

func (tx *TransactionEntity) FromCutoffPairEvent(src *types.CutoffPairEvent) error {
	tx.fullFilled(src.TxInfo)

	var content CutoffPairContent
	content.Owner = src.Owner.Hex()
	content.Token1 = src.Token1.Hex()
	content.Token2 = src.Token2.Hex()
	content.CutoffTimeStamp = src.Cutoff.Int64()

	bs, err := json.Marshal(&content)
	if err != nil {
		return err
	}

	tx.Content = string(bs)
	return nil
}

// 充值和提现from和to都是用户钱包自己的地址，因为合约限制了发送方msg.sender
func (tx *TransactionEntity) FromWethDepositEvent(src *types.WethDepositEvent) error {
	tx.fullFilled(src.TxInfo)

	var content WethDepositContent
	content.Dst = src.Dst.Hex()
	content.Amount = src.Amount.String()

	bs, err := json.Marshal(&content)
	if err != nil {
		return err
	}

	tx.Content = string(bs)
	return nil
}

func (tx *TransactionEntity) FromWethWithdrawalEvent(src *types.WethWithdrawalEvent) error {
	tx.fullFilled(src.TxInfo)

	var content WethWithdrawalContent
	content.Src = src.Src.Hex()
	content.Amount = src.Amount.String()

	bs, err := json.Marshal(&content)
	if err != nil {
		return err
	}

	tx.Content = string(bs)
	return nil
}

func (tx *TransactionEntity) FromTransferEvent(src *types.TransferEvent) error {
	tx.fullFilled(src.TxInfo)

	var content TransferContent
	content.Sender = src.Sender.Hex()
	content.Receiver = src.Receiver.Hex()
	content.Amount = src.Amount.String()

	bs, err := json.Marshal(&content)
	if err != nil {
		return err
	}

	tx.Content = string(bs)
	return nil
}

func (tx *TransactionEntity) FromEthTransferEvent(src *types.TransferEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.Content = ""
	return nil
}

func (entity *TransactionEntity) fullFilled(src types.TxInfo) {
	entity.Hash = src.TxHash
	entity.From = src.From
	entity.To = src.To
	entity.BlockNumber = src.BlockNumber.Int64()
	entity.LogIndex = src.TxLogIndex
	entity.Value = src.Value
	entity.Status = src.Status
	entity.GasLimit = src.GasLimit
	entity.GasUsed = src.GasUsed
	entity.GasPrice = src.GasPrice
	entity.Nonce = src.Nonce
	entity.BlockTime = src.BlockTime
}

// Compare return true: is the same
func (tx *TransactionEntity) Compare(src *TransactionEntity) bool {
	if tx.Hash != src.Hash {
		return false
	}
	if tx.LogIndex != src.LogIndex {
		return false
	}
	if tx.Nonce != src.Nonce {
		return false
	}
	if tx.Status != src.Status {
		return false
	}
	return true
}
