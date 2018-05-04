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
)

// 用于计算view里存储的amount以及最终展示给用户的数据
type innerFill struct {
	TokenS         common.Address
	TokenB         common.Address
	AmountS        *big.Int
	AmountB        *big.Int
	SplitS         *big.Int
	SplitB         *big.Int
	LrcReward      *big.Int
	LrcFee         *big.Int
	SymbolS        string
	SymbolB        string
	TotalAmountS   *big.Int
	TotalAmountB   *big.Int
	TotalAmountLrc *big.Int
}

func (fill *innerFill) prepare() {
	fill.TotalAmountS = big.NewInt(0)
	fill.TotalAmountB = big.NewInt(0)
	fill.TotalAmountLrc = big.NewInt(0)
}

func (fill *innerFill) FromOrderFilledEvent(src *types.OrderFilledEvent) {
	fill.prepare()
	fill.TokenS = src.TokenS
	fill.TokenB = src.TokenB
	fill.AmountS = src.AmountS
	fill.AmountB = src.AmountB
	fill.SplitS = src.SplitS
	fill.SplitB = src.SplitB
	fill.LrcReward = src.LrcReward
	fill.LrcFee = src.LrcFee
}

func (fill *innerFill) FromOrderFillContent(src *OrderFilledContent) {
	fill.prepare()
	fill.TokenS = common.HexToAddress(src.TokenS)
	fill.TokenB = common.HexToAddress(src.TokenB)
	fill.AmountS, _ = new(big.Int).SetString(src.AmountS, 0)
	fill.AmountB, _ = new(big.Int).SetString(src.AmountB, 0)
	fill.SplitS, _ = new(big.Int).SetString(src.SplitS, 0)
	fill.SplitB, _ = new(big.Int).SetString(src.SplitB, 0)
	fill.LrcReward, _ = new(big.Int).SetString(src.LrcReward, 0)
	fill.LrcFee, _ = new(big.Int).SetString(src.LrcFee, 0)
}

// 将lrc量归纳到totalAmountS中
//
// 基于合约amount不包含lrcFee及lrcReward及split的前提
// lrcFee由用户支出
// lrcReward由用户接收
// split由钱包及miner接收
//
// 如果用户卖的就是lrc,那么tokenS账户支出 = amountS + lrcFee + splitS - lrcReward
// 如果用户卖的就是lrc,那么tokenB账户收入 = amountB + lrcReward - lrcFee - splitB
// 如果用户交易不是lrc,那么lrc账户支出 = lrcFee - lrcReward

func (fill *innerFill) calculateAmountS() {
	fill.TotalAmountS = new(big.Int).Sub(fill.TotalAmountS, fill.AmountS)
	fill.TotalAmountS = new(big.Int).Sub(fill.TotalAmountS, fill.SplitS)
}

func (fill *innerFill) calculateAmountB() {
	fill.TotalAmountB = new(big.Int).Add(fill.TotalAmountB, fill.AmountB)
	fill.TotalAmountB = new(big.Int).Sub(fill.TotalAmountB, fill.SplitB)
}

func (fill *innerFill) calculateLrc() {
	fill.TotalAmountLrc = new(big.Int).Add(fill.TotalAmountLrc, fill.LrcReward)
	fill.TotalAmountLrc = new(big.Int).Sub(fill.TotalAmountLrc, fill.LrcFee)
}

func (fill *innerFill) calculateSellLrc() {
	if fill.SymbolS == SYMBOL_LRC {
		fill.TotalAmountS = new(big.Int).Add(fill.TotalAmountS, fill.TotalAmountLrc)
		fill.TotalAmountLrc = big.NewInt(0)
	}
}

func (fill *innerFill) calculateBuyLrc() {
	if fill.SymbolB == SYMBOL_LRC {
		fill.TotalAmountB = new(big.Int).Add(fill.TotalAmountB, fill.TotalAmountLrc)
		fill.TotalAmountLrc = big.NewInt(0)
	}
}
