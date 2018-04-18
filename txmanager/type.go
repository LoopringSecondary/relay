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
	"encoding/json"
	"fmt"
)

const (
	ETH_SYMBOL = "ETH"
	WETH_SYMBOL = "WETH"
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

// 过滤老版本重复数据
func filter(tx types.Transaction, owner common.Address, symbol string) error {
	askSymbol := strings.ToUpper(symbol)
	answerSymbol := strings.ToUpper(tx.Symbol)

	switch tx.Type {
	case types.TX_TYPE_SEND:
		if tx.From != owner {
			return fmt.Errorf("transaction view:filter old version compeated send tx:%s, from:%s, to:%s, owner:%s", tx.TxHash.Hex(), tx.From.Hex(), tx.To.Hex(), owner.Hex())
		}

	case types.TX_TYPE_RECEIVE:
		if tx.To != owner {
			return fmt.Errorf("transaction view:filter old version compeated receive tx:%s, from:%s, to:%s, owner:%s", tx.TxHash.Hex(), tx.From.Hex(), tx.To.Hex(), owner.Hex())
		}

	case types.TX_TYPE_CONVERT_INCOME:
		if askSymbol == ETH_SYMBOL && askSymbol != answerSymbol {
			return fmt.Errorf("transaction view:filter old version compeated weth deposit tx:%s, ask symbol:%s, answer symbol:%s", tx.TxHash, askSymbol, answerSymbol)
		}
		if askSymbol == WETH_SYMBOL && askSymbol != answerSymbol {
			return fmt.Errorf("transaction view:filter old version compeated weth withdrawal tx:%s, ask symbol:%s, answer symbol:%s", tx.TxHash, askSymbol, answerSymbol)
		}

	case types.TX_TYPE_CONVERT_OUTCOME:
		if askSymbol == ETH_SYMBOL && askSymbol != answerSymbol {
			return fmt.Errorf("transaction view:filter old version compeated weth deposit tx:%s, ask symbol:%s, answer symbol:%s", tx.TxHash, askSymbol, answerSymbol)
		}
		if askSymbol == WETH_SYMBOL && askSymbol != answerSymbol {
			return fmt.Errorf("transaction view:filter old version compeated weth withdrawal tx:%s, ask symbol:%s, answer symbol:%s", tx.TxHash, askSymbol, answerSymbol)
		}
	}

	return nil
}

func (dst *TransactionJsonResult) fromTransaction(tx types.Transaction, owner common.Address, symbol string) error {
	symbol = strings.ToUpper(symbol)

	// todo(fuk): 数据稳定后可以删除该代码或者加开关过滤该代码
	if err := filter(tx, owner, symbol); err != nil {
		return err
	}

	switch tx.Type {
	case types.TX_TYPE_TRANSFER:
		if tx.From == owner {
			tx.Type = types.TX_TYPE_SEND
		} else {
			tx.Type = types.TX_TYPE_RECEIVE
		}

	case types.TX_TYPE_DEPOSIT:
		if symbol == ETH_SYMBOL {
			tx.Type = types.TX_TYPE_CONVERT_OUTCOME
			tx.Protocol = types.NilAddress
		} else {
			tx.Type = types.TX_TYPE_CONVERT_INCOME
		}

	case types.TX_TYPE_WITHDRAWAL:
		if symbol == ETH_SYMBOL {
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

	return nil
}

//// 将同一个tx里的transfer事件按照symbol&from&to进行整合
//// 1.同一个logIndex进行过滤
//// 2.同一个tx 如果包含某个transfer 则将其他的transfer打包到content
func combineTransferEvents(src []TransactionJsonResult, owner common.Address) []TransactionJsonResult {
	var (
		list        []TransactionJsonResult
		singleTxMap = make(map[common.Hash]TransactionJsonResult)
		contentMap  = make(map[common.Hash][]TransactionJsonResult)
	)

	for _, current := range src {
		if _, ok := singleTxMap[current.TxHash]; !ok {
			singleTxMap[current.TxHash] = current
			contentMap[current.TxHash] = make([]TransactionJsonResult, 0)
		} else {
			contentMap[current.TxHash] = append(contentMap[current.TxHash], current)
		}
	}

	for _, tx := range singleTxMap {
		if len(contentMap[tx.TxHash]) == 0 {
			list = append(list, tx)
			continue
		}
		var (
			combineContentArray []TransactionJsonResult
			combineContentMap = make(map[int64]TransactionJsonResult)
		)
		for _, evt := range contentMap[tx.TxHash] {
			
		}
	}

	return list
}

func (tx *TransactionJsonResult) IsTransfer() bool {
	if tx.Type == types.TypeStr(types.TX_TYPE_SEND) || tx.Type == types.TypeStr(types.TX_TYPE_RECEIVE) {
		return true
	}
	return false
}

func standardSymbol(symbol string) string {
	return strings.ToUpper(symbol)
}

func protocolToSymbol(address common.Address) string {
	if address == types.NilAddress {
		return ETH_SYMBOL
	}
	symbol := util.AddressToAlias(address.Hex())
	return symbol
}

func symbolToProtocol(symbol string) common.Address {
	symbol = standardSymbol(symbol)
	if symbol == ETH_SYMBOL {
		return types.NilAddress
	}
	return util.AliasToAddress(symbol)
}
