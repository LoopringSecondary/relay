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

package matchengine

import (
	"github.com/Loopring/ringminer/types"
	"strconv"
	"math/rand"
	"math"
	"math/big"
)

//ring相关的，包含链的判定、费用等数据计算等, 分润费用是必须计算的，便于判断等

func IsRing() {

}

//compute availableAmountS of order
func AvailableAmountS(order *types.FilledOrder) error {
	//地址的余额
	//todo:just for test, it should use contract to compute
	//balance := big.NewInt(0)
	//loopring.Tokens[order.OrderState.RawOrder.TokenS].BalanceOf.Call(balance, "", order.OrderState.Owner)
	////订单的剩余金额
	//remainedAmount := big.NewInt(0)
	//loopring.LoopringImpls[order.OrderState.RawOrder.Protocol].RemainAmount.Call(&remainedAmount,"", order.OrderState.OrderHash)
	//if (order.AvailableAmountS.Cmp(balance) > 0) {
	//	order.AvailableAmountS = balance
	//}
	//if (order.AvailableAmountS.Cmp(remainedAmount) > 0) {
	//	order.AvailableAmountS = remainedAmount
	//}

	//订单的可用金额，需要根据BuyNoMoreThanAmountB进行区分
	order.OrderState.RemainedAmountS = big.NewInt(10000)
	order.OrderState.RemainedAmountS = order.OrderState.RawOrder.AmountS
	order.OrderState.RemainedAmountB = order.OrderState.RawOrder.AmountB

	order.AvailableAmountS = big.NewInt(10000)
	order.AvailableAmountS = order.OrderState.RawOrder.AmountS
	return nil
}

//费用、收取费用方式、折扣率等一切计算，在此完成
//计算匹配比例
//todo:折扣
func ComputeRing(ring *types.RingState) {
	DECIMALS := big.NewInt(1000000000) //todo:最好采用10的18次方，或者对应token的DECIMALS, 但是注意，计算价格时，目前使用math.pow, 需要转为float64，不要超出范围
	PERCENT := big.NewInt(100)

	ring.LegalFee = &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}

	//根据订单原始金额，计算成交量、成交价
	productPrice := &types.EnlargedInt{}
	productEnlargedAmountS := &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}
	productAmountB := &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}

	for _, order := range ring.RawRing.Orders {

		enlargedAmountS := &types.EnlargedInt{Value:big.NewInt(1).Mul(order.OrderState.RawOrder.AmountS, DECIMALS), Decimals:DECIMALS}
		enlargedAmountB := &types.EnlargedInt{Value:big.NewInt(1).Mul(order.OrderState.RawOrder.AmountB, DECIMALS), Decimals:DECIMALS}

		productEnlargedAmountS.Mul(productEnlargedAmountS, enlargedAmountS)
		productAmountB.MulBigInt(productAmountB, order.OrderState.RawOrder.AmountB)

		//todo：计算price，为了计算交易量
		enlargedSPrice := &types.EnlargedInt{}
		enlargedSPrice.DivBigInt(enlargedAmountS, order.OrderState.RawOrder.AmountB)
		order.EnlargedSPrice = enlargedSPrice

		enlargedBPrice := &types.EnlargedInt{}
		enlargedBPrice.DivBigInt(enlargedAmountB, order.OrderState.RawOrder.AmountS)
		order.EnlargedBPrice = enlargedBPrice
	}

	productPrice.Div(productEnlargedAmountS, productAmountB)
	priceOfFloat,_ := strconv.ParseFloat(productPrice.Value.String(), 0)
	rootOfRing := math.Pow(priceOfFloat, 1/float64(len(ring.RawRing.Orders)))

	ring.ReducedRate = &types.EnlargedInt{Value:big.NewInt(int64((float64(DECIMALS.Int64()) / rootOfRing) * float64(PERCENT.Int64()))), Decimals:PERCENT}

	shareRate := 100

	//todo:计算fee，为了取出最大的环路
	//LRC等比例下降，首先需要计算fillAmountS
	//分润的fee，首先需要计算fillAmountS，fillAmountS取决于整个环路上的完全匹配的订单
	//如何计算最小成交量的订单，计算下一次订单的卖出或买入，然后根据比例替换
	minVolumeIdx := 0

	for idx, order := range ring.RawRing.Orders {

		order.EnlargedSPrice.Mul(order.EnlargedSPrice, ring.ReducedRate)
		doubleDecimals := &types.EnlargedInt{Value:big.NewInt(1).Mul(DECIMALS, DECIMALS), Decimals:big.NewInt(1).Mul(DECIMALS, DECIMALS)}
		order.EnlargedBPrice.Div(doubleDecimals, order.EnlargedSPrice)

		enlargedAmountS := &types.EnlargedInt{Value:big.NewInt(0).Mul(order.OrderState.RawOrder.AmountS, DECIMALS), Decimals:DECIMALS}

		//指定feeSelection
		//todo:当以Sell为基准时，考虑账户余额、订单剩余金额的最小值
		AvailableAmountS(order)

		//根据用户设置，判断是以卖还是买为基准
		//买入不超过amountB
		if (order.OrderState.RawOrder.BuyNoMoreThanAmountB) {
			savingAmount := &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}	//节省的金额
			rate1 := &types.EnlargedInt{Decimals:big.NewInt(100), Value:big.NewInt(100)}
			savingAmount.Mul(enlargedAmountS, rate1.Sub(rate1, ring.ReducedRate))

			order.RateAmountS = &big.Int{}
			order.RateAmountS.Sub(order.OrderState.RawOrder.AmountS, savingAmount.RealValue())

			//todo:计算availableAmountS
			enlargedRemainAmountB := &types.EnlargedInt{Value:big.NewInt(0).Mul(order.OrderState.RemainedAmountB, DECIMALS), Decimals:DECIMALS}
			availableAmountS := &types.EnlargedInt{Value:big.NewInt(0), Decimals:big.NewInt(1)}

			//BuyNoMoreThanAmountB，根据剩余的买入量以及价格重新计算卖出
			rate := &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}
			rate.DivBigInt(enlargedRemainAmountB, order.OrderState.RawOrder.AmountB)

			availableAmountS.MulBigInt(rate, order.RateAmountS)
			if (availableAmountS.CmpBigInt(order.AvailableAmountS) < 0) {
				order.AvailableAmountS = availableAmountS.RealValue()
			}
		} else {
			//rateAmountB 应该是需要的，与孔亮确认
			order.RateAmountS = order.OrderState.RawOrder.AmountS
		}

		//与上一订单的买入进行比较
		var lastOrder *types.FilledOrder
		if (idx > 0) {
			lastOrder = ring.RawRing.Orders[idx - 1]
		}


		if (lastOrder != nil && lastOrder.FillAmountB.CmpBigInt(order.AvailableAmountS) >= 0) {
			//当前订单为最小订单
			order.FillAmountS = &types.EnlargedInt{Value:order.AvailableAmountS,Decimals:big.NewInt(1)}
			minVolumeIdx = idx
			//根据minVolumeIdx进行最小交易量的计算,两个方向进行
		} else if (lastOrder == nil) {
			order.FillAmountS = &types.EnlargedInt{Value:order.AvailableAmountS, Decimals:big.NewInt(1)}
		} else {
			//上一订单为最小订单需要对remainAmountS进行折扣计算
			order.FillAmountS = lastOrder.FillAmountB
		}
		order.FillAmountB = &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}
		order.FillAmountB.Mul(order.FillAmountS, order.EnlargedBPrice);

	}

	//todo:第一个与最后一个判断
	//if (ring.RawRing.Orders[len(ring.RawRing.Orders) - 1].FillAmountB.Cmp(ring.RawRing.Orders[0].FillAmountS) < 0) {
	//	minVolumeIdx = len(ring.RawRing.Orders) - 1
	//	for i := minVolumeIdx-1; i >= 0; i-- {
	//
	//		//按照前面的，同步减少交易量
	//		order := ring.RawRing.Orders[i]
	//		var nextOrder *types.FilledOrder
	//		nextOrder = ring.RawRing.Orders[i + 1]
	//		order.FillAmountB = nextOrder.FillAmountS
	//		order.FillAmountS.Mul(order.FillAmountB, order.EnlargedSPrice)
	//
	//
	//	}
	//}


	for i := minVolumeIdx-1; i >= 0; i-- {

		//按照前面的，同步减少交易量
		order := ring.RawRing.Orders[i]
		var nextOrder *types.FilledOrder
		nextOrder = ring.RawRing.Orders[i + 1]
		order.FillAmountB = nextOrder.FillAmountS
		order.FillAmountS.Mul(order.FillAmountB, order.EnlargedSPrice)

	}

	for i := minVolumeIdx + 1; i < len(ring.RawRing.Orders); i++ {
		order := ring.RawRing.Orders[i]
		var lastOrder *types.FilledOrder

		lastOrder = ring.RawRing.Orders[i - 1]
		order.FillAmountS = lastOrder.FillAmountB
		order.FillAmountB.Mul(order.FillAmountS, order.EnlargedBPrice)
	}

	//计算ring以及各个订单的费用，以及费用支付方式
	for _, order := range ring.RawRing.Orders {
		lrcAddress := &types.Address{}
		lrcAddress.SetBytes([]byte(LRC_ADDRESS))
		//todo:成本节约
		legalAmountOfSaving := &types.EnlargedInt{Value:big.NewInt(1),Decimals:big.NewInt(1)}
		if (order.OrderState.RawOrder.BuyNoMoreThanAmountB) {
			savingAmount := &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}
			enlargedFillAmountS := &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}
			enlargedFillAmountS.MulBigInt(order.FillAmountS, DECIMALS)
			savingAmount.Mul(enlargedFillAmountS, ring.ReducedRate)
			order.FeeS = savingAmount
			//todo:address of sell token
			legalAmountOfSaving.Mul(order.FeeS, GetLegalRate(CNY, *lrcAddress))
		} else {
			savingAmount := &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}
			enlargedFillAmountB := &types.EnlargedInt{Value:big.NewInt(1), Decimals:big.NewInt(1)}
			enlargedFillAmountB.MulBigInt(order.FillAmountB, DECIMALS)
			savingAmount.Mul(enlargedFillAmountB, ring.ReducedRate)
			order.FeeS = savingAmount
			//todo:address of buy token
			legalAmountOfSaving.Mul(order.FeeS, GetLegalRate(CNY, *lrcAddress))
		}

		//lrcFee等比例
		order.LrcFee = &types.EnlargedInt{Value:big.NewInt(1),Decimals:big.NewInt(1)}
		rate := &types.EnlargedInt{Value:big.NewInt(1),Decimals:big.NewInt(1)}
		enlargedAvailabeAmountS := &types.EnlargedInt{Value:big.NewInt(1).Mul(order.AvailableAmountS, DECIMALS), Decimals:DECIMALS}
		rate.DivBigInt(enlargedAvailabeAmountS, order.OrderState.RawOrder.AmountS)
		order.LrcFee.MulBigInt(rate, order.OrderState.RawOrder.LrcFee)

		legalAmountOfLrc := &types.EnlargedInt{Value:big.NewInt(1),Decimals:big.NewInt(1)}

		legalAmountOfLrc.Mul(GetLegalRate(CNY, *lrcAddress), order.LrcFee)

		//todo：比例以及lrc需要*2
		if (legalAmountOfLrc.Cmp(legalAmountOfSaving) > 0) {
			order.FeeSelection = 0
			order.LegalFee = legalAmountOfLrc
		} else {
			order.FeeSelection = 1
			//todo:根据分润比例计算收益
			if (shareRate > order.OrderState.RawOrder.SavingSharePercentage) {
				shareRate = order.OrderState.RawOrder.SavingSharePercentage
			}
			legalAmountOfSaving.Value.Mul(legalAmountOfSaving.Value, big.NewInt(int64(shareRate)))
			legalAmountOfSaving.DivBigInt(legalAmountOfSaving, PERCENT)
			order.LegalFee = legalAmountOfSaving
			lrcReward := &types.EnlargedInt{Value:legalAmountOfSaving.Value, Decimals:legalAmountOfSaving.Decimals}
			lrcReward.DivBigInt(lrcReward, big.NewInt(2))
			lrcReward.Div(lrcReward, GetLegalRate(CNY, *lrcAddress))
			order.LrcReward = lrcReward
		}
		ring.LegalFee.Add(ring.LegalFee, order.LegalFee)
	}

}

func Hash(ring *types.RingState) types.Hash {
	h := &types.Hash{}
	//todo:just for test, it should use contract to compute the hash of ring.
	h.SetBytes([]byte(strconv.Itoa(rand.Int())))
	//loopring.LoopringFingerprints[&types.Address{}].GetRingHash.Call(h, "")
	return *h
}

//成环之后才可计算能否成交，否则不需计算，判断是否能够成交，不能使用除法计算
func PriceValid(ring *types.RingState) bool {
	amountS := big.NewInt(1)
	amountB := big.NewInt(1)
	for _, order := range ring.RawRing.Orders {
		amountS.Mul(amountS, order.OrderState.RawOrder.AmountS)
		amountB.Mul(amountB, order.OrderState.RawOrder.AmountB)
	}
	return amountS.Cmp(amountB) >= 0
}

