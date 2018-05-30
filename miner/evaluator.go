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
	"math"
	"math/big"

	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

type Evaluator struct {
	marketCapProvider         marketcap.MarketCapProvider
	rateRatioCVSThreshold     int64
	gasUsedWithLength         map[int]*big.Int
	realCostRate, walletSplit *big.Rat

	minGasPrice, maxGasPrice *big.Int
	feeReceipt               common.Address

	matcher Matcher
}

func ReducedRate(ringState *types.Ring) *big.Rat {
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
	reducedRate := new(big.Rat)
	reducedRate.Inv(rate)
	log.Debugf("Miner,rate:%s, priceFloat:%f , len:%d, rootOfRing:%f, reducedRate:%s ", rate.FloatString(2), priceOfFloat, len(ringState.Orders), rootOfRing, reducedRate.FloatString(2))

	return reducedRate
}

func (e *Evaluator) ComputeRing(ringState *types.Ring) error {

	if len(ringState.Orders) <= 1 {
		return fmt.Errorf("length of ringState.Orders must > 1 , ringhash:%s", ringState.Hash.Hex())
	}

	ringState.ReducedRate = ReducedRate(ringState)

	//todo:get the fee for select the ring of mix income
	//LRC等比例下降，首先需要计算fillAmountS
	//分润的fee，首先需要计算fillAmountS，fillAmountS取决于整个环路上的完全匹配的订单
	//如何计算最小成交量的订单，计算下一次订单的卖出或买入，然后根据比例替换
	minVolumeIdx := 0

	for idx, filledOrder := range ringState.Orders {
		filledOrder.SPrice.Mul(filledOrder.SPrice, ringState.ReducedRate)

		filledOrder.BPrice.Inv(filledOrder.SPrice)

		amountS := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountS)
		//amountB := new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountB)

		//根据用户设置，判断是以卖还是买为基准
		//买入不超过amountB
		filledOrder.RateAmountS = new(big.Rat).Set(amountS)
		filledOrder.RateAmountS.Mul(amountS, ringState.ReducedRate)
		//if BuyNoMoreThanAmountB , AvailableAmountS need to be reduced by the ratePrice
		//recompute availabeAmountS and availableAmountB by the latest price
		if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
			//filledOrder.AvailableAmountS = new(big.Rat)
			filledOrder.AvailableAmountS.Mul(filledOrder.SPrice, filledOrder.AvailableAmountB)
		} else {
			filledOrder.AvailableAmountB.Mul(filledOrder.BPrice, filledOrder.AvailableAmountS)
		}
		log.Debugf("orderhash:%s availableAmountS:%s, availableAmountB:%s", filledOrder.OrderState.RawOrder.Hash.Hex(), filledOrder.AvailableAmountS.FloatString(2), filledOrder.AvailableAmountB.FloatString(2))

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
	if err := e.computeFeeOfRingAndOrder(ringState); nil != err {
		return err
	}

	//cvs
	cvs, err := PriceRateCVSquare(ringState)
	if nil != err {
		return err
	} else {
		if cvs.Int64() <= e.rateRatioCVSThreshold {
			return nil
		} else {
			for _, o := range ringState.Orders {
				log.Debugf("cvs bigger than RateRatioCVSThreshold orderhash:%s", o.OrderState.RawOrder.Hash.Hex())
			}
			return errors.New("Miner,cvs must less than RateRatioCVSThreshold")
		}
	}

}

func (e *Evaluator) computeFeeOfRingAndOrder(ringState *types.Ring) error {

	var err error
	var feeReceiptLrcAvailableAmount *big.Rat
	var lrcAddress common.Address
	if impl, exists := ethaccessor.ProtocolAddresses()[ringState.Orders[0].OrderState.RawOrder.Protocol]; exists {
		var err error
		lrcAddress = impl.LrcTokenAddress
		//todo:the address transfer lrcreward should be msg.sender not feeReceipt
		if feeReceiptLrcAvailableAmount, err = e.matcher.GetAccountAvailableAmount(e.feeReceipt, lrcAddress, impl.DelegateAddress); nil != err {
			return err
		}
	} else {
		return errors.New("not support this protocol: " + ringState.Orders[0].OrderState.RawOrder.Protocol.Hex())
	}

	ringState.LegalFee = big.NewRat(int64(0), int64(1))
	for _, filledOrder := range ringState.Orders {
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
			legalAmountOfSaving, err = e.getLegalCurrency(filledOrder.OrderState.RawOrder.TokenS, filledOrder.FeeS)
			if nil != err {
				return err
			}
		} else {
			savingAmount := new(big.Rat).Set(filledOrder.FillAmountB)
			savingAmount.Mul(savingAmount, ringState.ReducedRate)
			savingAmount.Sub(filledOrder.FillAmountB, savingAmount)
			filledOrder.FeeS = savingAmount
			legalAmountOfSaving, err = e.getLegalCurrency(filledOrder.OrderState.RawOrder.TokenB, filledOrder.FeeS)
			if nil != err {
				return err
			}
		}

		//compute lrcFee
		rate := new(big.Rat).Quo(filledOrder.FillAmountS, new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.AmountS))
		filledOrder.LrcFee = new(big.Rat).SetInt(filledOrder.OrderState.RawOrder.LrcFee)
		filledOrder.LrcFee.Mul(filledOrder.LrcFee, rate)

		if filledOrder.AvailableLrcBalance.Cmp(filledOrder.LrcFee) <= 0 {
			filledOrder.LrcFee = filledOrder.AvailableLrcBalance
		}

		legalAmountOfLrc, err1 := e.getLegalCurrency(lrcAddress, filledOrder.LrcFee)
		if nil != err1 {
			return err1
		}

		filledOrder.LegalLrcFee = legalAmountOfLrc
		splitPer := new(big.Rat)
		if filledOrder.OrderState.RawOrder.MarginSplitPercentage > 100 {
			splitPer.SetFrac64(int64(100), int64(100))
		} else {
			splitPer.SetFrac64(int64(filledOrder.OrderState.RawOrder.MarginSplitPercentage), int64(100))
		}
		legalAmountOfSaving.Mul(legalAmountOfSaving, splitPer)
		filledOrder.LegalFeeS = legalAmountOfSaving
		log.Debugf("orderhash:%s, raw.lrc:%s, AvailableLrcBalance:%s, lrcFee:%s, feeS:%s, legalAmountOfLrc:%s,  legalAmountOfSaving:%s, minerLrcAvailable:%s",
			filledOrder.OrderState.RawOrder.Hash.Hex(),
			filledOrder.OrderState.RawOrder.LrcFee.String(),
			filledOrder.AvailableLrcBalance.FloatString(2),
			filledOrder.LrcFee.FloatString(2),
			filledOrder.FeeS.FloatString(2),
			legalAmountOfLrc.FloatString(2), legalAmountOfSaving.FloatString(2), feeReceiptLrcAvailableAmount.FloatString(2))

		lrcFee := new(big.Rat).SetInt(big.NewInt(int64(2)))
		lrcFee.Mul(lrcFee, filledOrder.LegalLrcFee)
		if lrcFee.Cmp(filledOrder.LegalFeeS) < 0 && feeReceiptLrcAvailableAmount.Cmp(filledOrder.LrcFee) > 0 {
			filledOrder.FeeSelection = 1
			filledOrder.LegalFeeS.Sub(filledOrder.LegalFeeS, filledOrder.LegalLrcFee)
			filledOrder.LrcReward = filledOrder.LegalLrcFee
			ringState.LegalFee.Add(ringState.LegalFee, filledOrder.LegalFeeS)

			feeReceiptLrcAvailableAmount.Sub(feeReceiptLrcAvailableAmount, filledOrder.LrcFee)
			//log.Debugf("Miner,lrcReward:%s  legalFee:%s", lrcReward.FloatString(10), filledOrder.LegalFee.FloatString(10))
		} else {
			filledOrder.FeeSelection = 0
			filledOrder.LegalFeeS = filledOrder.LegalLrcFee
			filledOrder.LrcReward = new(big.Rat).SetInt(big.NewInt(int64(0)))
			ringState.LegalFee.Add(ringState.LegalFee, filledOrder.LegalLrcFee)
		}
	}

	e.evaluateReceived(ringState)

	//legalFee := new(big.Rat).SetInt(big.NewInt(int64(0)))
	//feeSelections := []uint8{}
	//legalFees := []*big.Rat{}
	//lrcRewards := []*big.Rat{}
	//
	//for _,filledOrder := range ringState.Orders {
	//	lrcFee := new(big.Rat).SetInt(big.NewInt(int64(2)))
	//	lrcFee.Mul(lrcFee, filledOrder.LegalLrcFee)
	//	log.Debugf("lrcFee:%s, filledOrder.LegalFeeS:%s, minerLrcBalance:%s, filledOrder.LrcFee:%s", lrcFee.FloatString(3), filledOrder.LegalFeeS.FloatString(3), minerLrcBalance.FloatString(3), filledOrder.LrcFee.FloatString(3))
	//	if lrcFee.Cmp(filledOrder.LegalFeeS) < 0 && minerLrcAvailableAmount.Cmp(filledOrder.LrcFee) > 0 {
	//		feeSelections = append(feeSelections, 1)
	//		fee := new(big.Rat).Set(filledOrder.LegalFeeS)
	//		fee.Sub(fee, filledOrder.LegalLrcFee)
	//		legalFees = append(legalFees, fee)
	//		lrcRewards = append(lrcRewards, filledOrder.LegalLrcFee)
	//		legalFee.Add(legalFee, fee)
	//
	//		minerLrcAvailableAmount.Sub(minerLrcAvailableAmount, filledOrder.LrcFee)
	//		//log.Debugf("Miner,lrcReward:%s  legalFee:%s", lrcReward.FloatString(10), filledOrder.LegalFee.FloatString(10))
	//	} else {
	//		feeSelections = append(feeSelections, 0)
	//		legalFees = append(legalFees, filledOrder.LegalLrcFee)
	//		lrcRewards = append(lrcRewards, new(big.Rat).SetInt(big.NewInt(int64(0))))
	//		legalFee.Add(legalFee, filledOrder.LegalLrcFee)
	//	}
	//}

	return nil
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
		s1b0, _ := new(big.Int).SetString(filledOrder.RateAmountS.FloatString(0), 10)
		//s1b0 = s1b0.Mul(s1b0, rawOrder.AmountB)

		s0b1 := new(big.Int).SetBytes(rawOrder.AmountS.Bytes())
		//s0b1 = s0b1.Mul(s0b1, rawOrder.AmountB)
		if s1b0.Cmp(s0b1) > 0 {
			return nil, errors.New("Miner,rateAmountS must less than amountS")
		}
		ratio := new(big.Int).Set(scale)
		ratio.Mul(ratio, s1b0).Div(ratio, s0b1)
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
	log.Debugf("CVSquare, scale:%s, avg:%s, length1:%s", scale.String(), avg.String(), length1.String())
	if avg.Sign() <= 0 {
		return new(big.Int).SetInt64(math.MaxInt64)
	}
	//todo:avg may be zero??
	return cvs.Mul(cvs, scale).Div(cvs, avg).Mul(cvs, scale).Div(cvs, avg).Div(cvs, length1)
}

func (e *Evaluator) getLegalCurrency(tokenAddress common.Address, amount *big.Rat) (*big.Rat, error) {
	return e.marketCapProvider.LegalCurrencyValue(tokenAddress, amount)
}

func (e *Evaluator) evaluateReceived(ringState *types.Ring) {
	ringState.Received = big.NewRat(int64(0), int64(1))
	ringState.GasPrice = ethaccessor.EstimateGasPrice(e.minGasPrice, e.maxGasPrice)
	//log.Debugf("len(ringState.Orders):%d", len(ringState.Orders))
	ringState.Gas = new(big.Int)
	ringState.Gas.Set(e.gasUsedWithLength[len(ringState.Orders)])
	protocolCost := new(big.Int)
	protocolCost.Mul(ringState.Gas, ringState.GasPrice)

	costEth := new(big.Rat).SetInt(protocolCost)
	ringState.LegalCost, _ = e.marketCapProvider.LegalCurrencyValueOfEth(costEth)

	log.Debugf("legalFee:%s, cost:%s, realCostRate:%s, protocolCost:%s, gas:%s, gasPrice:%s", ringState.LegalFee.FloatString(2), ringState.LegalCost.FloatString(2), e.realCostRate.FloatString(2), protocolCost.String(), ringState.Gas.String(), ringState.GasPrice.String())
	ringState.LegalCost.Mul(ringState.LegalCost, e.realCostRate)
	log.Debugf("legalFee:%s, cost:%s, realCostRate:%s", ringState.LegalFee.FloatString(2), ringState.LegalCost.FloatString(2), e.realCostRate.FloatString(2))
	ringState.Received.Sub(ringState.LegalFee, ringState.LegalCost)
	ringState.Received.Mul(ringState.Received, e.walletSplit)
	return
}

func NewEvaluator(marketCapProvider marketcap.MarketCapProvider, minerOptions config.MinerOptions) *Evaluator {
	gasUsedMap := make(map[int]*big.Int)
	gasUsedMap[2] = big.NewInt(500000)
	//todo:confirm this value
	gasUsedMap[3] = big.NewInt(500000)
	gasUsedMap[4] = big.NewInt(500000)
	e := &Evaluator{marketCapProvider: marketCapProvider, rateRatioCVSThreshold: minerOptions.RateRatioCVSThreshold, gasUsedWithLength: gasUsedMap}
	e.realCostRate = new(big.Rat)
	if int64(minerOptions.Subsidy) >= 1 {
		e.realCostRate.SetInt64(int64(0))
	} else {
		e.realCostRate.SetFloat64(float64(1.0) - minerOptions.Subsidy)
	}
	e.feeReceipt = common.HexToAddress(minerOptions.FeeReceipt)
	e.walletSplit = new(big.Rat)
	e.walletSplit.SetFloat64(minerOptions.WalletSplit)
	e.minGasPrice = big.NewInt(minerOptions.MinGasLimit)
	e.maxGasPrice = big.NewInt(minerOptions.MaxGasLimit)
	return e
}

func (e *Evaluator) SetMatcher(matcher Matcher) {
	e.matcher = matcher
}
