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

package miner

import (
	"errors"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math"
	"math/big"
)

type Evaluator struct {
	marketCapProvider     *marketcap.MarketCapProvider
	rateRatioCVSThreshold int64
}

func availableAmountS(filledOrder *types.FilledOrder) error {
	filledOrder.AvailableAmountS = new(big.Rat).SetInt(filledOrder.OrderState.RemainedAmountS)
	filledOrder.AvailableAmountB = new(big.Rat).SetInt(filledOrder.OrderState.RemainedAmountB)
	sellPrice := new(big.Rat).SetFrac(filledOrder.OrderState.RawOrder.AmountS, filledOrder.OrderState.RawOrder.AmountB)

	if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
		filledOrder.AvailableAmountS.Mul(filledOrder.AvailableAmountB, sellPrice)
	} else {
		filledOrder.AvailableAmountB.Mul(filledOrder.AvailableAmountB, new(big.Rat).Inv(sellPrice))
	}
	return nil
}

func (e *Evaluator) ComputeRing(ringState *types.Ring) error {

	ringState.LegalFee = new(big.Rat)

	productAmountS := big.NewRat(int64(1), int64(1))
	productAmountB := big.NewRat(int64(1), int64(1))

	//compute price
	for _, order := range ringState.Orders {
		amountS := new(big.Rat).SetInt(order.OrderState.RawOrder.AmountS)
		amountB := new(big.Rat).SetInt(order.OrderState.RawOrder.AmountB)

		productAmountS.Mul(productAmountS, amountS)
		productAmountB.Mul(productAmountB, amountB)

		order.SPrice = new(big.Rat)
		order.SPrice.Quo(amountS, amountB)

		order.BPrice = new(big.Rat)
		order.BPrice.Quo(amountB, amountS)
	}

	productPrice := new(big.Rat)
	productPrice.Quo(productAmountS, productAmountB)
	//todo:change pow to big.Int
	priceOfFloat, _ := productPrice.Float64()
	rootOfRing := math.Pow(priceOfFloat, 1/float64(len(ringState.Orders)))
	rate := new(big.Rat).SetFloat64(rootOfRing)
	ringState.ReducedRate = new(big.Rat)
	ringState.ReducedRate.Inv(rate)
	log.Debugf("priceFloat:%f , len:%d, rootOfRing:%f, reducedRate:%d ", priceOfFloat, len(ringState.Orders), rootOfRing, ringState.ReducedRate.RatString())

	//todo:get the fee for select the ring of mix income
	//LRC等比例下降，首先需要计算fillAmountS
	//分润的fee，首先需要计算fillAmountS，fillAmountS取决于整个环路上的完全匹配的订单
	//如何计算最小成交量的订单，计算下一次订单的卖出或买入，然后根据比例替换
	minVolumeIdx := 0

	for idx, filledOrder := range ringState.Orders {
		filledOrder.SPrice.Mul(filledOrder.SPrice, ringState.ReducedRate)

		filledOrder.BPrice.Inv(filledOrder.SPrice)

		//todo:当以Sell为基准时，考虑账户余额、订单剩余金额的最小值
		if err := availableAmountS(filledOrder); nil != err {
			return err
		}
		amountS := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountS)
		amountB := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountB)

		//根据用户设置，判断是以卖还是买为基准
		//买入不超过amountB
		filledOrder.RateAmountS = new(big.Rat).Set(amountS)
		filledOrder.RateAmountS.Mul(amountS, ringState.ReducedRate)
		//if BuyNoMoreThanAmountB , AvailableAmountS need to be reduced by the ratePrice
		if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
			availbleAmountB := new(big.Rat).Set(filledOrder.AvailableAmountB)
			availableAmountS := new(big.Rat).Mul(filledOrder.RateAmountS, availbleAmountB)
			availableAmountS.Quo(availableAmountS, amountB)
			if filledOrder.AvailableAmountB.Cmp(new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountB)) < 0 {
				filledOrder.AvailableAmountS.Set(availableAmountS)
			}
		}

		//与上一订单的买入进行比较
		var lastOrder *types.FilledOrder
		if idx > 0 {
			lastOrder = ringState.Orders[idx-1]
		}

		filledOrder.FillAmountS = new(big.Rat)
		if lastOrder != nil && lastOrder.FillAmountB.Cmp(filledOrder.AvailableAmountS) >= 0 {
			//当前订单为最小订单
			filledOrder.FillAmountS.Set(filledOrder.AvailableAmountS)
			minVolumeIdx = idx
			//根据minVolumeIdx进行最小交易量的计算,两个方向进行
		} else if lastOrder == nil {
			filledOrder.FillAmountS.Set(filledOrder.AvailableAmountS)
		} else {
			//上一订单为最小订单需要对remainAmountS进行折扣计算
			filledOrder.FillAmountS.Set(lastOrder.FillAmountB)
		}
		filledOrder.FillAmountB = new(big.Rat).Mul(filledOrder.FillAmountS, filledOrder.BPrice)
	}

	//compute the volume of the ring by the min volume
	//todo:the first and the last
	//if (ring.RawRing.Orders[len(ring.RawRing.Orders) - 1].FillAmountB.Cmp(ring.RawRing.Orders[0].FillAmountS) < 0) {
	//	minVolumeIdx = len(ring.RawRing.Orders) - 1
	//	for i := minVolumeIdx-1; i >= 0; i-- {
	//		//按照前面的，同步减少交易量
	//		order := ring.RawRing.Orders[i]
	//		var nextOrder *types.FilledOrder
	//		nextOrder = ring.RawRing.Orders[i + 1]
	//		order.FillAmountB = nextOrder.FillAmountS
	//		order.FillAmountS.Mul(order.FillAmountB, order.EnlargedSPrice)
	//	}
	//}

	for i := minVolumeIdx - 1; i >= 0; i-- {
		//按照前面的，同步减少交易量
		order := ringState.Orders[i]
		var nextOrder *types.FilledOrder
		nextOrder = ringState.Orders[i+1]
		order.FillAmountB = nextOrder.FillAmountS
		order.FillAmountS.Mul(order.FillAmountB, order.SPrice)
	}

	for i := minVolumeIdx + 1; i < len(ringState.Orders); i++ {
		order := ringState.Orders[i]
		var lastOrder *types.FilledOrder
		lastOrder = ringState.Orders[i-1]
		order.FillAmountS = lastOrder.FillAmountB
		order.FillAmountB.Mul(order.FillAmountS, order.BPrice)
	}

	//compute the fee of this ring and orders, and set the feeSelection
	e.computeFeeOfRingAndOrder(ringState)

	//cvs
	if cvs, err := PriceRateCVSquare(ringState); nil != err {
		return err
	} else {
		log.Debugf("ringState.length:%d ,  cvs:%s", len(ringState.Orders), cvs.String())
		if cvs.Int64() <= e.rateRatioCVSThreshold {
			return nil
		} else {
			return errors.New("cvs must less than RateRatioCVSThreshold")
		}
	}
}

func (e *Evaluator) computeFeeOfRingAndOrder(ringState *types.Ring) {

	//the ring use the min MarginSplitPercentage as self's MarginSplitPercentage
	minShareRate := new(big.Rat).SetInt(big.NewInt(0))
	for _, order := range ringState.Orders {
		percentage := new(big.Rat).SetInt64(int64(order.OrderState.RawOrder.MarginSplitPercentage))
		if minShareRate.Cmp(percentage) > 0 {
			minShareRate = percentage
		}
	}

	for _, filledOrder := range ringState.Orders {
		lrcAddress := &common.Address{}

		lrcAddress.SetBytes([]byte(e.marketCapProvider.LRC_ADDRESS))
		//todo:成本节约
		legalAmountOfSaving := new(big.Rat)
		if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
			amountS := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountS)
			amountB := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountB)
			sPrice := new(big.Rat)
			sPrice.Quo(amountS, amountB)
			savingAmount := new(big.Rat)
			savingAmount.Mul(filledOrder.FillAmountB, sPrice)
			savingAmount.Sub(savingAmount, filledOrder.FillAmountS)
			filledOrder.FeeS = savingAmount
			legalAmountOfSaving.Mul(filledOrder.FeeS, e.marketCapProvider.GetMarketCap(filledOrder.OrderState.RawOrder.TokenS))
			log.Debugf("savingAmount:%s", savingAmount.FloatString(10))
		} else {
			savingAmount := new(big.Rat).Set(filledOrder.FillAmountB)
			savingAmount.Mul(savingAmount, ringState.ReducedRate)
			savingAmount.Sub(filledOrder.FillAmountB, savingAmount)
			filledOrder.FeeS = savingAmount
			//todo:address of buy token
			legalAmountOfSaving.Mul(filledOrder.FeeS, e.marketCapProvider.GetMarketCap(filledOrder.OrderState.RawOrder.TokenB))
		}

		//compute lrcFee
		rate := new(big.Rat).Quo(filledOrder.AvailableAmountS, new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountS))
		filledOrder.LrcFee = new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.LrcFee)
		filledOrder.LrcFee.Mul(filledOrder.LrcFee, rate)

		legalAmountOfLrc := new(big.Rat).Mul(e.marketCapProvider.GetMarketCap(*lrcAddress), filledOrder.LrcFee)

		//the lrcreward should be set when select  MarginSplit as the selection of fee
		if legalAmountOfLrc.Cmp(legalAmountOfSaving) > 0 {
			filledOrder.FeeSelection = 0
			filledOrder.LegalFee = legalAmountOfLrc
		} else {
			filledOrder.FeeSelection = 1
			legalAmountOfSaving.Mul(legalAmountOfSaving, minShareRate)
			filledOrder.LegalFee = legalAmountOfSaving
			lrcReward := new(big.Rat).Set(legalAmountOfSaving)
			lrcReward.Quo(lrcReward, new(big.Rat).SetInt64(int64(2)))
			lrcReward.Quo(lrcReward, e.marketCapProvider.GetMarketCap(*lrcAddress))
			log.Debugf("lrcReward:%s  legalFee:%s", lrcReward.FloatString(10), filledOrder.LegalFee.FloatString(10))
			filledOrder.LrcReward = lrcReward
		}
		ringState.LegalFee.Add(ringState.LegalFee, filledOrder.LegalFee)
	}
}

//成环之后才可计算能否成交，否则不需计算，判断是否能够成交，不能使用除法计算
func PriceValid(a2BOrder *types.OrderState, b2AOrder *types.OrderState) bool {
	amountS := new(big.Int).Mul(a2BOrder.RawOrder.AmountS, b2AOrder.RawOrder.AmountS)
	amountB := new(big.Int).Mul(a2BOrder.RawOrder.AmountB, b2AOrder.RawOrder.AmountB)
	return amountS.Cmp(amountB) >= 0
}

func PriceRateCVSquare(ringState *types.Ring) (*big.Int, error) {
	rateRatios := []*big.Int{}
	scale, _ := new(big.Int).SetString("10000", 0)
	for _, filledOrder := range ringState.Orders {
		rawOrder := filledOrder.OrderState.RawOrder
		log.Debugf("rawOrder.AmountS:%s, filledOrder.RateAmountS:%s", rawOrder.AmountS.String(), filledOrder.RateAmountS.FloatString(10))
		s1b0, _ := new(big.Int).SetString(filledOrder.RateAmountS.FloatString(0), 10)
		//s1b0 = s1b0.Mul(s1b0, rawOrder.AmountB)

		s0b1 := new(big.Int).SetBytes(rawOrder.AmountS.Bytes())
		//s0b1 = s0b1.Mul(s0b1, rawOrder.AmountB)
		if s1b0.Cmp(s0b1) > 0 {
			return nil, errors.New("rateAmountS must less than amountS")
		}
		ratio := new(big.Int).Set(scale)
		ratio.Mul(ratio, s1b0).Div(ratio, s0b1)
		log.Debugf("ratio:%s", ratio.String())
		rateRatios = append(rateRatios, ratio)
	}
	return CVSquare(rateRatios, scale), nil
}

func CVSquare(rateRatios []*big.Int, scale *big.Int) *big.Int {
	avg := big.NewInt(0)
	length := big.NewInt(int64(len(rateRatios)))
	length1 := big.NewInt(int64(len(rateRatios) - 1))
	for _, ratio := range rateRatios {
		avg.Add(avg, ratio)
	}
	avg = avg.Div(avg, length)

	cvs := big.NewInt(0)
	for _, ratio := range rateRatios {
		sub := big.NewInt(0)
		sub.Sub(ratio, avg)

		subSquare := new(big.Int).Mul(sub, sub)
		cvs.Add(cvs, subSquare)
	}

	return cvs.Mul(cvs, scale).Div(cvs, avg).Mul(cvs, scale).Div(cvs, avg).Div(cvs, length1)
}

func NewEvaluator(marketCapProvider *marketcap.MarketCapProvider, rateRatioCVSThreshold int64) *Evaluator {
	return &Evaluator{marketCapProvider: marketCapProvider, rateRatioCVSThreshold: rateRatioCVSThreshold}
}
