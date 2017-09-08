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

// 旷工在成本节约和fee上二选一，撮合者计算出:
// 1.fee(lrc)的市场价(法币交易价格)
// 2.成本节约(savingShare)的市场价(法币交易价格)
// 撮合者在fee和savingShare中二选一，作为自己的利润，
// 如果撮合者选择fee，则成本节约分给订单发起者，如果选择成本节约，则需要返还给用户一定的lrc
// 这样一来，撮合者的利润判断公式应该是max(fee, savingShare - fee * s),s为固定比例
// 此外，在选择最优环路的时候，撮合者会在确定了选择fee/savingShare后，选择某个具有最大利润的环路
// 但是，根据谷歌竞拍法则(A出价10,B出价20,最终成交价为10)，撮合者最终获得的利润只能是利润最小的环路利润
type Ring struct {
	Orders []*FilledOrder
	Miner Address
	FeeRecepient Address
	ThrowIfTokenAllowanceOrBalanceIsInsuffcient bool
	V                 uint8   `json:"v"`
	R                 Sign    `json:"r"`
	S                 Sign    `json:"s"`
}

type RingState struct {
	RawRing     *Ring
	Hash        Hash    `json:"hash"` // 订单链id
	ReducedRate *EnlargedInt              //成环之后，折价比例
	LegalFee    *EnlargedInt //法币计算的fee
	FeeMode     int                   //收费方式，0 lrc 1 share
}

// TODO(fukun): 添加状态判断是否成环
//type OrderRing struct {
//	Ring         `json:"ring"`     // 订单链
//	Closure bool `json:"closure"`  // 是否闭合
//}

