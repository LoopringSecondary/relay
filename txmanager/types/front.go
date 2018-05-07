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
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

type TransactionJsonResult struct {
	Protocol    common.Address     `json:"protocol"`
	Owner       common.Address     `json:"owner"`
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
	GasPrice    string             `json:"gas_price"`
	GasLimit    string             `json:"gas_limit"`
	GasUsed     string             `json:"gas_used"`
	Nonce       string             `json:"nonce"`
}

// todo(后续版本更改)
type TransactionContent struct {
	Market    string `json:"market"`
	OrderHash string `json:"orderHash"`
	Fill      string `json:"fill"`
}

func NewResult(tx *TransactionView) TransactionJsonResult {
	var res TransactionJsonResult

	res.Protocol = types.NilAddress
	res.Owner = tx.Owner
	res.TxHash = tx.TxHash
	res.Symbol = tx.Symbol
	res.BlockNumber = tx.BlockNumber
	res.Value = tx.Amount.String()
	res.LogIndex = tx.LogIndex
	res.Type = TypeStr(tx.Type)
	res.Status = types.StatusStr(tx.Status)
	res.CreateTime = tx.CreateTime
	res.UpdateTime = tx.UpdateTime
	res.Nonce = tx.Nonce.String()

	return res
}

func (r *TransactionJsonResult) FromApproveEntity(entity *TransactionEntity) error {
	var content ApproveContent
	if err := json.Unmarshal([]byte(entity.Content), &content); err != nil {
		return err
	}

	r.beforeConvert(entity)
	r.From = common.HexToAddress(content.Owner)
	r.To = common.HexToAddress(content.Spender)

	return nil
}

func (r *TransactionJsonResult) FromCancelEntity(entity *TransactionEntity) error {
	var content CancelContent
	if err := json.Unmarshal([]byte(entity.Content), &content); err != nil {
		return err
	}

	r.beforeConvert(entity)
	r.Content.OrderHash = content.OrderHash

	return nil
}

func (r *TransactionJsonResult) FromCutoffEntity(entity *TransactionEntity) error {
	var content CutoffContent
	if err := json.Unmarshal([]byte(entity.Content), &content); err != nil {
		return err
	}

	r.beforeConvert(entity)

	return nil
}

func (r *TransactionJsonResult) FromCutoffPairEntity(entity *TransactionEntity) error {
	var content CutoffPairContent
	if err := json.Unmarshal([]byte(entity.Content), &content); err != nil {
		return err
	}

	r.beforeConvert(entity)

	if market, err := util.WrapMarket(content.Token1, content.Token2); err == nil {
		r.Content.Market = market
	} else {
		return err
	}

	return nil
}

func (r *TransactionJsonResult) FromWethDepositEntity(entity *TransactionEntity) error {
	var content WethDepositContent
	if err := json.Unmarshal([]byte(entity.Content), &content); err != nil {
		return err
	}

	r.beforeConvert(entity)
	r.From = common.HexToAddress(content.Dst)
	r.To = common.HexToAddress(content.Dst)

	return nil
}

func (r *TransactionJsonResult) FromWethWithdrawalEntity(entity *TransactionEntity) error {
	var content WethWithdrawalContent
	if err := json.Unmarshal([]byte(entity.Content), &content); err != nil {
		return err
	}

	r.beforeConvert(entity)
	r.From = common.HexToAddress(content.Src)
	r.To = common.HexToAddress(content.Src)

	return nil
}

func (r *TransactionJsonResult) FromTransferEntity(entity *TransactionEntity) error {
	var content TransferContent
	if err := json.Unmarshal([]byte(entity.Content), &content); err != nil {
		return err
	}

	r.beforeConvert(entity)
	r.From = common.HexToAddress(content.Sender)
	r.To = common.HexToAddress(content.Receiver)

	return nil
}

func (r *TransactionJsonResult) FromFillEntity(entity *TransactionEntity) error {
	type frontfill struct {
		RingHash  string `json:"ring_hash"`
		OrderHash string `json:"order_hash"`
		Owner     string `json:"owner"`
		SymbolS   string `json:"symbol_s"`
		SymbolB   string `json:"symbol_b"`
		RingIndex string `json:"ring_index"`
		FillIndex string `json:"fill_index"`
		AmountS   string `json:"amount_s"`
		AmountB   string `json:"amount_b"`
		LrcReward string `json:"lrc_reward"`
		LrcFee    string `json:"lrc_fee"`
		SplitS    string `json:"split_s"`
		SplitB    string `json:"split_b"`
		Market    string `json:"market"`
	}

	var (
		content OrderFilledContent
		fill    frontfill
	)
	if err := json.Unmarshal([]byte(entity.Content), &content); err != nil {
		return err
	}

	r.beforeConvert(entity)
	fill.RingHash = content.RingHash
	fill.OrderHash = content.OrderHash
	fill.Owner = content.Owner
	fill.SymbolS = util.AddressToAlias(content.TokenS)
	fill.SymbolB = util.AddressToAlias(content.TokenB)
	fill.RingIndex = content.RingIndex
	fill.FillIndex = content.FillIndex
	fill.AmountS = content.AmountS
	fill.AmountB = content.AmountB
	fill.LrcReward = content.LrcReward
	fill.LrcFee = content.LrcFee
	fill.SplitS = content.SplitS
	fill.SplitB = content.SplitB
	fill.Market = content.Market

	if bs, err := json.Marshal(&fill); err != nil {
		return err
	} else {
		r.Content.Fill = string(bs)
	}

	return nil
}

// 普通的eth转账及其他合约无需转换
func (r *TransactionJsonResult) FromOtherEntity(entity *TransactionEntity) error {
	r.beforeConvert(entity)
	return nil
}

func (r *TransactionJsonResult) beforeConvert(entity *TransactionEntity) {
	var res TransactionContent
	r.Protocol = entity.Protocol
	r.From = entity.From
	r.To = entity.To
	r.Content = res
	r.GasPrice = entity.GasPrice.String()
	r.GasLimit = entity.GasLimit.String()
	r.GasUsed = entity.GasUsed.String()
}
