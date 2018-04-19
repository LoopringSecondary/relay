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

package timing_matcher

import (
	"fmt"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sort"
)

type Market struct {
	matcher      *TimingMatcher
	om           ordermanager.OrderManager
	protocolImpl *ethaccessor.ProtocolAddress

	TokenA     common.Address
	TokenB     common.Address
	AtoBOrders map[common.Hash]*types.OrderState
	BtoAOrders map[common.Hash]*types.OrderState

	AtoBOrderHashesExcludeNextRound []common.Hash
	BtoAOrderHashesExcludeNextRound []common.Hash
}

func (market *Market) match() {
	market.getOrdersForMatching(market.protocolImpl.ContractAddress)
	matchedOrderHashes := make(map[common.Hash]bool) //true:fullfilled, false:partfilled
	ringSubmitInfos := []*types.RingSubmitInfo{}
	candidateRingList := CandidateRingList{}

	//step 1: evaluate received
	for _, a2BOrder := range market.AtoBOrders {
		for _, b2AOrder := range market.BtoAOrders {
			if miner.PriceValid(a2BOrder, b2AOrder) {
				if candidateRing, err := market.GenerateCandidateRing(a2BOrder, b2AOrder); nil != err {
					log.Errorf("err:%s", err.Error())
					continue
				} else {
					if candidateRing.received.Sign() > 0 {
						candidateRingList = append(candidateRingList, *candidateRing)
					} else {
						log.Debugf("timing_matchher, market ringForSubmit received not enough, received:%s, cost:%s ", candidateRing.received.FloatString(0), candidateRing.cost.FloatString(0))
					}
				}
			}
		}
	}

	log.Debugf("match round:%s, market: %s -> %s , candidateRingList.length:%d", market.matcher.lastRoundNumber, market.TokenA.Hex(), market.TokenB.Hex(), len(candidateRingList))
	//the ring that can get max received
	list := candidateRingList
	for {
		if len(list) <= 0 {
			break
		}

		sort.Sort(list)
		candidateRing := list[0]
		list = list[1:]
		orders := []*types.OrderState{}
		for hash, _ := range candidateRing.filledOrders {
			if o, exists := market.AtoBOrders[hash]; exists {
				orders = append(orders, o)
			} else {
				orders = append(orders, market.BtoAOrders[hash])
			}
		}
		if ringForSubmit, err := market.generateRingSubmitInfo(orders...); nil != err {
			log.Debugf("generate RingSubmitInfo err:%s", err.Error())
			continue
		} else {
			//todo:for test, release this limit
			if ringForSubmit.Received.Sign() > 0 {
				for _, filledOrder := range ringForSubmit.RawRing.Orders {
					orderState := market.reduceAmountAfterFilled(filledOrder)
					isFullFilled := market.om.IsOrderFullFinished(orderState)
					matchedOrderHashes[filledOrder.OrderState.RawOrder.Hash] = isFullFilled
					//market.matcher.rounds.AppendFilledOrderToCurrent(filledOrder, ringForSubmit.RawRing.Hash)

					list = market.reduceReceivedOfCandidateRing(list, filledOrder, isFullFilled)
				}
				AddMinedRing(ringForSubmit)
				ringSubmitInfos = append(ringSubmitInfos, ringForSubmit)
			} else {
				log.Debugf("ring:%s will not be submitted,because of received:%s", ringForSubmit.RawRing.Hash.Hex(), ringForSubmit.Received.String())
			}
		}
	}

	for orderHash, _ := range market.AtoBOrders {
		if fullFilled, exists := matchedOrderHashes[orderHash]; exists || fullFilled {
			market.AtoBOrderHashesExcludeNextRound = append(market.AtoBOrderHashesExcludeNextRound, orderHash)
		}
	}

	for orderHash, _ := range market.BtoAOrders {
		if fullFilled, exists := matchedOrderHashes[orderHash]; exists || fullFilled {
			market.BtoAOrderHashesExcludeNextRound = append(market.BtoAOrderHashesExcludeNextRound, orderHash)
		}
	}
	if len(ringSubmitInfos) > 0 {
		eventemitter.Emit(eventemitter.Miner_NewRing, ringSubmitInfos)
	}
}

func (market *Market) reduceReceivedOfCandidateRing(list CandidateRingList, filledOrder *types.FilledOrder, isFullFilled bool) CandidateRingList {
	resList := CandidateRingList{}
	hash := filledOrder.OrderState.RawOrder.Hash
	for _, ring := range list {
		if amountS, exists := ring.filledOrders[hash]; exists {
			if isFullFilled {
				continue
			}
			availableAmountS := new(big.Rat)
			availableAmountS.Sub(filledOrder.AvailableAmountS, filledOrder.FillAmountS)
			if availableAmountS.Sign() > 0 {
				var remainedAmountS *big.Rat
				if amountS.Cmp(availableAmountS) >= 0 {
					remainedAmountS = availableAmountS
				} else {
					remainedAmountS = amountS
				}
				log.Debugf("reduceReceivedOfCandidateRing, filledOrder.availableAmountS:%s, filledOrder.FillAmountS:%s, amountS:%s", filledOrder.AvailableAmountS.FloatString(3), filledOrder.FillAmountS.FloatString(3), amountS.FloatString(3))
				rate := new(big.Rat)
				rate.Quo(remainedAmountS, amountS)
				remainedReceived := new(big.Rat).Add(ring.received, ring.cost)
				remainedReceived.Mul(remainedReceived, rate).Sub(remainedReceived, ring.cost)
				//todo:for test, release this limit
				//if remainedReceived.Sign() <= 0 {
				//	continue
				//}
				for hash, amount := range ring.filledOrders {
					ring.filledOrders[hash] = amount.Mul(amount, rate)
				}
				resList = append(resList, ring)
			}
		} else {
			resList = append(resList, ring)
		}
	}
	return resList
}

/**
get orders from ordermanager
*/
func (market *Market) getOrdersForMatching(protocolAddress common.Address) {
	market.AtoBOrders = make(map[common.Hash]*types.OrderState)
	market.BtoAOrders = make(map[common.Hash]*types.OrderState)

	// log.Debugf("timing matcher,market tokenA:%s, tokenB:%s, atob hash length:%d, btoa hash length:%d", market.TokenA.Hex(), market.TokenB.Hex(), len(market.AtoBOrderHashesExcludeNextRound), len(market.BtoAOrderHashesExcludeNextRound))
	currentRoundNumber := market.matcher.lastRoundNumber.Int64()
	deleyedNumber := market.matcher.delayedNumber + currentRoundNumber

	atoBOrders := market.om.MinerOrders(protocolAddress, market.TokenA, market.TokenB, market.matcher.roundOrderCount, int64(0), currentRoundNumber, &types.OrderDelayList{OrderHash: market.AtoBOrderHashesExcludeNextRound, DelayedCount: deleyedNumber})

	if len(atoBOrders) < market.matcher.roundOrderCount {
		orderCount := market.matcher.roundOrderCount - len(atoBOrders)
		orders := market.om.MinerOrders(protocolAddress, market.TokenA, market.TokenB, orderCount, currentRoundNumber+1, currentRoundNumber+market.matcher.delayedNumber)
		atoBOrders = append(atoBOrders, orders...)
	}

	btoAOrders := market.om.MinerOrders(protocolAddress, market.TokenB, market.TokenA, market.matcher.roundOrderCount, int64(0), currentRoundNumber, &types.OrderDelayList{OrderHash: market.BtoAOrderHashesExcludeNextRound, DelayedCount: deleyedNumber})
	if len(btoAOrders) < market.matcher.roundOrderCount {
		orderCount := market.matcher.roundOrderCount - len(btoAOrders)
		orders := market.om.MinerOrders(protocolAddress, market.TokenB, market.TokenA, orderCount, currentRoundNumber+1, currentRoundNumber+market.matcher.delayedNumber)
		btoAOrders = append(btoAOrders, orders...)
	}

	//log.Debugf("#### %s,%s %d,%d %d",market.TokenA.Hex(),market.TokenB.Hex(), len(atoBOrders), len(btoAOrders),market.matcher.roundOrderCount)
	market.AtoBOrderHashesExcludeNextRound = []common.Hash{}
	market.BtoAOrderHashesExcludeNextRound = []common.Hash{}

	for _, order := range atoBOrders {
		market.reduceRemainedAmountBeforeMatch(order)
		if !market.om.IsOrderFullFinished(order) {
			market.AtoBOrders[order.RawOrder.Hash] = order
		} else {
			market.AtoBOrderHashesExcludeNextRound = append(market.AtoBOrderHashesExcludeNextRound, order.RawOrder.Hash)
		}
		log.Debugf("order status in this new round, orderhash:%s, DealtAmountS:%s, ", order.RawOrder.Hash.Hex(), order.DealtAmountS.String())
	}

	for _, order := range btoAOrders {
		market.reduceRemainedAmountBeforeMatch(order)
		if !market.om.IsOrderFullFinished(order) {
			market.BtoAOrders[order.RawOrder.Hash] = order
		} else {
			market.BtoAOrderHashesExcludeNextRound = append(market.BtoAOrderHashesExcludeNextRound, order.RawOrder.Hash)
		}
		log.Debugf("order status in this new round, orderhash:%s, DealtAmountS:%s", order.RawOrder.Hash.Hex(), order.DealtAmountS.String())
	}
}

//sub the matched amount in new round.
func (market *Market) reduceRemainedAmountBeforeMatch(orderState *types.OrderState) {
	orderHash := orderState.RawOrder.Hash

	if amountS, amountB, err := DealtAmount(orderHash); nil != err {
		log.Errorf("err:%s", err.Error())
	} else {
		log.Debugf("reduceRemainedAmountBeforeMatch:%s, %s, %s", orderState.RawOrder.Owner.Hex(), amountS.String(), amountB.String())
		orderState.DealtAmountB.Add(orderState.DealtAmountB, ratToInt(amountB))
		orderState.DealtAmountS.Add(orderState.DealtAmountS, ratToInt(amountS))
	}
}

func (market *Market) reduceAmountAfterFilled(filledOrder *types.FilledOrder) *types.OrderState {
	filledOrderState := filledOrder.OrderState
	var orderState *types.OrderState

	//only one of DealtAmountB and DealtAmountS is precise
	if filledOrderState.RawOrder.TokenS == market.TokenA {
		orderState = market.AtoBOrders[filledOrderState.RawOrder.Hash]
		orderState.DealtAmountB.Add(orderState.DealtAmountB, ratToInt(filledOrder.FillAmountB))
		orderState.DealtAmountS.Add(orderState.DealtAmountS, ratToInt(filledOrder.FillAmountS))
	} else {
		orderState = market.BtoAOrders[filledOrderState.RawOrder.Hash]
		orderState.DealtAmountB.Add(orderState.DealtAmountB, ratToInt(filledOrder.FillAmountB))
		orderState.DealtAmountS.Add(orderState.DealtAmountS, ratToInt(filledOrder.FillAmountS))
	}
	log.Debugf("order status after matched, orderhash:%s,filledAmountS:%s, DealtAmountS:%s, ", orderState.RawOrder.Hash.Hex(), filledOrder.FillAmountS.String(), orderState.DealtAmountS.String())
	//reduced account balance

	return orderState
}

func (market *Market) GenerateCandidateRing(orders ...*types.OrderState) (*CandidateRing, error) {
	filledOrders := []*types.FilledOrder{}
	//miner will received nothing, if miner set FeeSelection=1 and he doesn't have enough lrc
	for _, order := range orders {

		if filledOrder, err := market.generateFilledOrder(order); nil != err {
			log.Errorf("err:%s", err.Error())
			return nil, err
		} else {
			filledOrders = append(filledOrders, filledOrder)
		}
	}

	ringTmp := miner.NewRing(filledOrders)
	if err := market.matcher.evaluator.ComputeRing(ringTmp); nil != err {
		return nil, err
	} else {
		_, _, costLegal, received := market.matcher.evaluator.EvaluateReceived(ringTmp)
		candidateRing := &CandidateRing{cost: costLegal, received: received, filledOrders: make(map[common.Hash]*big.Rat)}
		for _, filledOrder := range ringTmp.Orders {
			log.Debugf("match, filledOrder.FilledAmountS:%s", filledOrder.FillAmountS.FloatString(3))
			candidateRing.filledOrders[filledOrder.OrderState.RawOrder.Hash] = filledOrder.FillAmountS
		}
		return candidateRing, nil
	}
}

func (market *Market) generateFilledOrder(order *types.OrderState) (*types.FilledOrder, error) {

	lrcTokenBalance, err := market.matcher.GetAccountAvailableAmount(order.RawOrder.Owner, market.protocolImpl.LrcTokenAddress, market.protocolImpl.DelegateAddress)
	if nil != err {
		return nil, err
	}

	tokenSBalance, err := market.matcher.GetAccountAvailableAmount(order.RawOrder.Owner, order.RawOrder.TokenS, market.protocolImpl.DelegateAddress)
	if nil != err {
		return nil, err
	}
	if tokenSBalance.Sign() <= 0 {
		return nil, fmt.Errorf("owner:%s token:%s balance or allowance is zero", order.RawOrder.Owner.Hex(), order.RawOrder.TokenS.Hex())
	}
	//todo:
	if market.om.IsValueDusted(order.RawOrder.TokenS, tokenSBalance) {
		return nil, fmt.Errorf("owner:%s token:%s balance or allowance is not enough", order.RawOrder.Owner.Hex(), order.RawOrder.TokenS.Hex())
	}
	return types.ConvertOrderStateToFilledOrder(*order, lrcTokenBalance, tokenSBalance, market.protocolImpl.LrcTokenAddress), nil
}

func (market *Market) generateRingSubmitInfo(orders ...*types.OrderState) (*types.RingSubmitInfo, error) {
	filledOrders := []*types.FilledOrder{}
	//miner will received nothing, if miner set FeeSelection=1 and he doesn't have enough lrc
	for _, order := range orders {
		if filledOrder, err := market.generateFilledOrder(order); nil != err {
			log.Errorf("err:%s", err.Error())
		} else {
			filledOrders = append(filledOrders, filledOrder)
		}
	}

	ringTmp := miner.NewRing(filledOrders)
	if err := market.matcher.evaluator.ComputeRing(ringTmp); nil != err {
		return nil, err
	} else {
		gas, gasPrice, _, received := market.matcher.evaluator.EvaluateReceived(ringTmp)
		res, err := market.matcher.submitter.GenerateRingSubmitInfo(ringTmp, gas, gasPrice, received)
		return res, err
	}
}

func NewMarket(protocolAddress *ethaccessor.ProtocolAddress, tokenS, tokenB common.Address, matcher *TimingMatcher, om ordermanager.OrderManager) *Market {

	m := &Market{}
	m.om = om
	m.protocolImpl = protocolAddress
	m.matcher = matcher
	m.TokenA = tokenS
	m.TokenB = tokenB
	m.AtoBOrderHashesExcludeNextRound = []common.Hash{}
	m.BtoAOrderHashesExcludeNextRound = []common.Hash{}
	return m
}

func ratToInt(rat *big.Rat) *big.Int {
	return new(big.Int).Div(rat.Num(), rat.Denom())
}
