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
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"math"
	"math/big"
	"strconv"
	"github.com/ethereum/go-ethereum/common"
)

var (
	RateRatioCVSThreshold int64
	RateProvider *ExchangeRateProvider
)

//compute availableAmountS of order
func AvailableAmountS(filledOrder *types.FilledOrder) error {
	order := filledOrder.OrderState.RawOrder
	balance := &types.Big{}
	allowance := &types.Big{}
	filledOrder.AvailableAmountS = new(big.Int).Set(order.AmountS)
	filledOrder.AvailableAmountB = new(big.Int).Set(order.AmountB)

	LoopringInstance.Tokens[order.TokenS].BalanceOf.Call(balance, "pending", common.BytesToAddress(order.Owner.Bytes()))
	LoopringInstance.Tokens[order.TokenS].Allowance.Call(allowance, "pending", common.BytesToAddress(order.Owner.Bytes()), common.BytesToAddress(order.Protocol.Bytes()))

	println("order.Owner", order.Owner.Hex(), order.TokenS.Hex())
	if balance.BigInt().Cmp(big.NewInt(0)) <= 0 {
		return errors.New("not enough balance")
	} else if allowance.BigInt().Cmp(big.NewInt(0)) <= 0 {
		return errors.New("not enough allowance")
	} else {
		if filledOrder.AvailableAmountS.Cmp(balance.BigInt()) > 0 {
			filledOrder.AvailableAmountS = balance.BigInt()
		}
		if filledOrder.AvailableAmountS.Cmp(allowance.BigInt()) > 0 {
			filledOrder.AvailableAmountS = allowance.BigInt()
		}
	}
	//订单的剩余金额
	filledAmount := &types.Big{}
	//filled buynomorethanb=true保存的为amountb，如果为false保存的为amounts
	LoopringInstance.LoopringImpls[filledOrder.OrderState.RawOrder.Protocol].GetOrderFilled.Call(filledAmount, "pending", common.BytesToHash(order.Hash.Bytes()))
	if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
		remainedAmount := new(big.Int).Set(filledOrder.OrderState.RawOrder.AmountB)
		remainedAmount.Sub(remainedAmount, filledAmount.BigInt())
		if filledOrder.AvailableAmountB.Cmp(remainedAmount) > 0 {
			filledOrder.AvailableAmountB = remainedAmount
		}
	} else {
		remainedAmount := new(big.Int).Set(filledOrder.OrderState.RawOrder.AmountS)
		remainedAmount.Sub(remainedAmount, filledAmount.BigInt())
		//todo:return false when remainedAmount less than amount x
		if filledOrder.AvailableAmountS.Cmp(remainedAmount) > 0 {
			filledOrder.AvailableAmountS = remainedAmount
		}
	}
	return nil
}

var DECIMALS *big.Int = big.NewInt(1000000000000000) //todo:最好采用10的18次方，或者对应token的DECIMALS, 但是注意，计算价格时，目前使用math.pow, 需要转为float64，不要超出范围
var FOURTIMESDECIMALS *types.EnlargedInt = &types.EnlargedInt{Value: big.NewInt(0).Mul(DECIMALS, DECIMALS), Decimals: big.NewInt(0).Mul(DECIMALS, DECIMALS)}

//
func ComputeRing(ringState *types.RingState) error {
	FOURTIMESDECIMALS.Value.Mul(FOURTIMESDECIMALS.Value, DECIMALS)
	FOURTIMESDECIMALS.Decimals.Mul(FOURTIMESDECIMALS.Decimals, DECIMALS)

	ringState.LegalFee = &types.EnlargedInt{Value: big.NewInt(0), Decimals: big.NewInt(1)}

	productPrice := &types.EnlargedInt{}
	productEnlargedAmountS := &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}
	productAmountB := &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}

	//compute price
	for _, order := range ringState.RawRing.Orders {
		enlargedAmountS := &types.EnlargedInt{Value: big.NewInt(1).Mul(order.OrderState.RawOrder.AmountS, DECIMALS), Decimals: big.NewInt(1).Set(DECIMALS)}
		enlargedAmountB := &types.EnlargedInt{Value: big.NewInt(1).Mul(order.OrderState.RawOrder.AmountB, DECIMALS), Decimals: big.NewInt(1).Set(DECIMALS)}

		productEnlargedAmountS.Mul(productEnlargedAmountS, enlargedAmountS)
		productAmountB.MulBigInt(productAmountB, order.OrderState.RawOrder.AmountB)

		enlargedSPrice := &types.EnlargedInt{}
		enlargedSPrice.DivBigInt(enlargedAmountS, order.OrderState.RawOrder.AmountB)
		order.EnlargedSPrice = enlargedSPrice

		enlargedBPrice := &types.EnlargedInt{}
		enlargedBPrice.DivBigInt(enlargedAmountB, order.OrderState.RawOrder.AmountS)
		order.EnlargedBPrice = enlargedBPrice
	}

	productPrice.Div(productEnlargedAmountS, productAmountB)
	//todo:change pow to big.Int
	priceOfFloat, _ := strconv.ParseFloat(productPrice.RealValue().String(), 64)
	rootOfRing := math.Pow(priceOfFloat, 1/float64(len(ringState.RawRing.Orders)))
	v := big.NewInt(int64((float64(DECIMALS.Int64()) / rootOfRing)))
	ringState.ReducedRate = &types.EnlargedInt{Value: v, Decimals: big.NewInt(1).Set(DECIMALS)}
	log.Debugf("priceFloat:%f , len:%d, rootOfRing:%f, reducedRate:%d ", priceOfFloat, len(ringState.RawRing.Orders), rootOfRing, ringState.ReducedRate.RealValue().Int64())


	//todo:get the fee for select the ring of mix income
	//LRC等比例下降，首先需要计算fillAmountS
	//分润的fee，首先需要计算fillAmountS，fillAmountS取决于整个环路上的完全匹配的订单
	//如何计算最小成交量的订单，计算下一次订单的卖出或买入，然后根据比例替换
	minVolumeIdx := 0

	for idx, filledOrder := range ringState.RawRing.Orders {
		filledOrder.EnlargedSPrice.Mul(filledOrder.EnlargedSPrice, ringState.ReducedRate)

		filledOrder.EnlargedBPrice.Div(FOURTIMESDECIMALS, filledOrder.EnlargedSPrice)
		enlargedAmountS := &types.EnlargedInt{Value: big.NewInt(0).Mul(filledOrder.OrderState.RawOrder.AmountS, DECIMALS), Decimals: DECIMALS}

		//todo:当以Sell为基准时，考虑账户余额、订单剩余金额的最小值
		if err := AvailableAmountS(filledOrder);nil != err {
			return err
		}

		//根据用户设置，判断是以卖还是买为基准
		//买入不超过amountB
		savingAmount := &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)} //节省的金额
		rate1 := &types.EnlargedInt{Decimals: big.NewInt(100), Value: big.NewInt(100)}
		rate2 := rate1.Sub(rate1, ringState.ReducedRate)
		savingAmount.Mul(enlargedAmountS, rate2)

		filledOrder.RateAmountS = &types.EnlargedInt{Value: new(big.Int).Set(filledOrder.OrderState.RawOrder.AmountS), Decimals: big.NewInt(1)}
		filledOrder.RateAmountS.Sub(filledOrder.RateAmountS, savingAmount)
		//if BuyNoMoreThanAmountB , AvailableAmountS need to be reduced by the ratePrice
		if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
			enlargedAvailableAmountB := &types.EnlargedInt{Value: big.NewInt(0).Mul(filledOrder.AvailableAmountB, DECIMALS), Decimals: DECIMALS}
			availableAmountS := &types.EnlargedInt{Value: big.NewInt(0), Decimals: big.NewInt(1)}

			//compute availableAmountS
			rate := &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}
			rate.DivBigInt(enlargedAvailableAmountB, filledOrder.OrderState.RawOrder.AmountB)

			availableAmountS.Mul(rate, filledOrder.RateAmountS)
			if availableAmountS.CmpBigInt(filledOrder.AvailableAmountS) < 0 {
				filledOrder.AvailableAmountS = availableAmountS.RealValue()
			}
		}

		//与上一订单的买入进行比较
		var lastOrder *types.FilledOrder
		if idx > 0 {
			lastOrder = ringState.RawRing.Orders[idx-1]
		}

		if lastOrder != nil && lastOrder.FillAmountB.CmpBigInt(filledOrder.AvailableAmountS) >= 0 {
			//当前订单为最小订单
			filledOrder.FillAmountS = &types.EnlargedInt{Value: filledOrder.AvailableAmountS, Decimals: big.NewInt(1)}
			minVolumeIdx = idx
			//根据minVolumeIdx进行最小交易量的计算,两个方向进行
		} else if lastOrder == nil {
			filledOrder.FillAmountS = &types.EnlargedInt{Value: filledOrder.AvailableAmountS, Decimals: big.NewInt(1)}
		} else {
			//上一订单为最小订单需要对remainAmountS进行折扣计算
			filledOrder.FillAmountS = lastOrder.FillAmountB
		}
		filledOrder.FillAmountB = &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}
		filledOrder.FillAmountB.Mul(filledOrder.FillAmountS, filledOrder.EnlargedBPrice)

	}

	//compute the volume of the ring by the min volume
	//todo:the first and the last
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

	for i := minVolumeIdx - 1; i >= 0; i-- {
		//按照前面的，同步减少交易量
		order := ringState.RawRing.Orders[i]
		var nextOrder *types.FilledOrder
		nextOrder = ringState.RawRing.Orders[i+1]
		order.FillAmountB = nextOrder.FillAmountS
		order.FillAmountS.Mul(order.FillAmountB, order.EnlargedSPrice)
	}

	for i := minVolumeIdx + 1; i < len(ringState.RawRing.Orders); i++ {
		order := ringState.RawRing.Orders[i]
		var lastOrder *types.FilledOrder
		lastOrder = ringState.RawRing.Orders[i-1]
		order.FillAmountS = lastOrder.FillAmountB
		order.FillAmountB.Mul(order.FillAmountS, order.EnlargedBPrice)
	}

	//compute the fee of this ring and orders, and set the feeSelection
	computeFeeOfRingAndOrder(ringState)

	//cvs
	if cvs, err := PriceRateCVSquare(ringState); nil != err {
		return err
	} else {
		log.Debugf("ringState.length:%d ,  cvs:%s", len(ringState.RawRing.Orders), cvs.String())
		if cvs.Int64() <= RateRatioCVSThreshold {
			return nil
		} else {
			return errors.New("cvs must less than RateRatioCVSThreshold")
		}
	}
}

func computeFeeOfRingAndOrder(ringState *types.RingState) {

	//the ring use the min MarginSplitPercentage as self's MarginSplitPercentage
	minShareRate := &types.EnlargedInt{Value: big.NewInt(100), Decimals: big.NewInt(100)}
	for _, order := range ringState.RawRing.Orders {
		percentage := int64(order.OrderState.RawOrder.MarginSplitPercentage)
		if minShareRate.Value.Int64() > percentage {
			minShareRate.Value = big.NewInt(percentage)
		}
	}

	for _, order := range ringState.RawRing.Orders {
		lrcAddress := &types.Address{}

		lrcAddress.SetBytes([]byte(RateProvider.LRC_ADDRESS))
		//todo:成本节约
		legalAmountOfSaving := &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}
		if order.OrderState.RawOrder.BuyNoMoreThanAmountB {
			enlargedAmountS := &types.EnlargedInt{Value: big.NewInt(1).Mul(order.OrderState.RawOrder.AmountS, DECIMALS), Decimals: big.NewInt(1).Set(DECIMALS)}
			enlargedSPrice := &types.EnlargedInt{}
			enlargedSPrice.DivBigInt(enlargedAmountS, order.OrderState.RawOrder.AmountB)
			savingAmount := &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}
			savingAmount.Mul(order.FillAmountB, enlargedSPrice)
			savingAmount.Sub(savingAmount, order.FillAmountS)
			order.FeeS = savingAmount
			legalAmountOfSaving.Mul(order.FeeS, RateProvider.GetLegalRate(order.OrderState.RawOrder.TokenS))
			log.Debugf("savingAmount:%d", savingAmount.RealValue().Int64())
		} else {
			savingAmount := &types.EnlargedInt{Value: big.NewInt(0).Set(order.FillAmountB.Value), Decimals: big.NewInt(0).Set(order.FillAmountB.Decimals)}
			savingAmount.Mul(savingAmount, ringState.ReducedRate)
			savingAmount.Sub(order.FillAmountB, savingAmount)
			order.FeeS = savingAmount
			//todo:address of buy token
			legalAmountOfSaving.Mul(order.FeeS, RateProvider.GetLegalRate(order.OrderState.RawOrder.TokenB))
		}

		//compute lrcFee
		order.LrcFee = &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}
		rate := &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}
		enlargedAvailabeAmountS := &types.EnlargedInt{Value: big.NewInt(1).Mul(order.AvailableAmountS, DECIMALS), Decimals: DECIMALS}
		rate.DivBigInt(enlargedAvailabeAmountS, order.OrderState.RawOrder.AmountS)
		order.LrcFee.MulBigInt(rate, order.OrderState.RawOrder.LrcFee)

		legalAmountOfLrc := &types.EnlargedInt{Value: big.NewInt(1), Decimals: big.NewInt(1)}

		legalAmountOfLrc.Mul(RateProvider.GetLegalRate(*lrcAddress), order.LrcFee)

		//the lrcreward should be set when select  MarginSplit as the selection of fee
		if legalAmountOfLrc.Cmp(legalAmountOfSaving) > 0 {
			order.FeeSelection = 0
			order.LegalFee = legalAmountOfLrc
		} else {
			order.FeeSelection = 1
			legalAmountOfSaving.Mul(legalAmountOfSaving, minShareRate)
			order.LegalFee = legalAmountOfSaving
			lrcReward := &types.EnlargedInt{Value: big.NewInt(0).Set(legalAmountOfSaving.Value), Decimals: big.NewInt(0).Set(legalAmountOfSaving.Decimals)}
			lrcReward.DivBigInt(lrcReward, big.NewInt(2))
			lrcReward.Div(lrcReward, RateProvider.GetLegalRate(*lrcAddress))
			log.Debugf("lrcReward:%s  legalFee:%s", lrcReward.RealValue().String(), order.LegalFee.RealValue().String())
			order.LrcReward = lrcReward
		}
		ringState.LegalFee.Add(ringState.LegalFee, order.LegalFee)
	}
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

func PriceRateCVSquare(ringState *types.RingState) (*big.Int, error) {
	rateRatios := []*big.Int{}
	scale, _ := new(big.Int).SetString("10000", 0)
	for _, filledOrder := range ringState.RawRing.Orders {
		rawOrder := filledOrder.OrderState.RawOrder
		log.Debugf("rawOrder.AmountS:%s, filledOrder.RateAmountS:%s", rawOrder.AmountS.String(), filledOrder.RateAmountS.RealValue().String())
		s1b0 := new(big.Int).Set(filledOrder.RateAmountS.RealValue())
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

		subSquare := big.NewInt(1)
		subSquare.Mul(sub, sub)
		cvs.Add(cvs, subSquare)
	}

	return cvs.Mul(cvs, scale).Div(cvs, avg).Mul(cvs, scale).Div(cvs, avg).Div(cvs, length1)
}
