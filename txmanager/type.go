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
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"strings"
)

const (
	ETH_SYMBOL = "ETH"
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

func (dst TransactionJsonResult) fromTransaction(tx types.Transaction, owner common.Address, symbol string) {
	switch tx.Type {
	case types.TX_TYPE_TRANSFER:
		if tx.From == owner {
			tx.Type = types.TX_TYPE_SEND
		} else {
			tx.Type = types.TX_TYPE_RECEIVE
		}

	case types.TX_TYPE_DEPOSIT:
		if strings.ToUpper(symbol) == strings.ToUpper(ETH_SYMBOL) {
			tx.Type = types.TX_TYPE_CONVERT_OUTCOME
			tx.Protocol = types.NilAddress
		} else {
			tx.Type = types.TX_TYPE_CONVERT_INCOME
		}

	case types.TX_TYPE_WITHDRAWAL:
		if strings.ToUpper(symbol) == strings.ToUpper(ETH_SYMBOL) {
			tx.Type = types.TX_TYPE_CONVERT_INCOME
			tx.Protocol = types.NilAddress
		} else {
			tx.Type = types.TX_TYPE_CONVERT_OUTCOME
		}

	case types.TX_TYPE_CUTOFF_PAIR:
		if ctx, err := tx.GetCutoffPairContent(); err == nil {
			if mkt, err := util.WrapMarketByAddress(ctx.Token1.Hex(), ctx.Token2.Hex()); err == nil {
				dst.Content = TransactionContent{Market: mkt}
			}
		}

	case types.TX_TYPE_CANCEL_ORDER:
		if ctx, err := tx.GetCancelOrderHash(); err == nil {
			dst.Content = TransactionContent{OrderHash: ctx}
		}
	}

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

	// set value
	if tx.Value == nil {
		dst.Value = "0"
	} else {
		dst.Value = tx.Value.String()
	}
}

func protocolToSymbol(address common.Address) string {
	if address == types.NilAddress {
		return ETH_SYMBOL
	}
	symbol := util.AddressToAlias(address.Hex())
	return symbol
}

func symbolToProtocol(symbol string) common.Address {
	if symbol == ETH_SYMBOL {
		return types.NilAddress
	}
	return util.AliasToAddress(symbol)
}
