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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"fmt"
)

type TransactionEntity struct {
	From        common.Address
	To          common.Address
	BlockNumber int64
	Hash        common.Hash
	LogIndex    int64
	Value       *big.Int
	Content     string
	Status      types.TxStatus
	GasLimit    *big.Int
	GasUsed     *big.Int
	GasPrice    *big.Int
	Nonce       *big.Int
	CreateTime  int64
	UpdateTime  int64
}

func (tx *Transaction) FromApproveEvent(src *ApprovalEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.Owner
	tx.To = src.Spender
	tx.Value = src.Value
	tx.Type = TX_TYPE_APPROVE

	return nil
}

func (entity *TransactionEntity )FromApproveEvent(src *types.ApprovalEvent)  {
	entity.
}

func (entity *TransactionEntity) fullFilled(src types.TxInfo) {
	entity.Hash = src.TxHash
	entity.From = src.From
}

func (tx *Transaction) FromFillEvent(src *OrderFilledEvent, txtype TxType) error {
	tx.fullFilled(src.TxInfo)
	tx.Type = txtype
	if txtype == TX_TYPE_SELL {
		tx.From = src.BuyFrom
		tx.To = src.SellTo
		tx.Value = src.AmountS
	} else {
		tx.From = src.SellTo
		tx.To = src.BuyFrom
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

func (tx *Transaction) FromCancelMethod(src *OrderCancelledEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.From
	tx.To = src.To
	tx.Type = TX_TYPE_CANCEL_ORDER
	tx.Value = src.AmountCancelled
	tx.Content = []byte(src.OrderHash.Hex())

	return nil
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


func (tx *Transaction) FromTransferEvent(src *TransferEvent) error {
	tx.fullFilled(src.TxInfo)
	tx.From = src.Sender
	tx.To = src.Receiver
	tx.Value = src.Value
	tx.Type = TX_TYPE_TRANSFER

	return nil
}
