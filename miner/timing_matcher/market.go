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
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"math/rand"
	"sort"
)

type Market struct {
	matcher         *TimingMatcher
	om              ordermanager.OrderManager
	protocolAddress common.Address
	lrcAddress      common.Address

	TokenA     common.Address
	TokenB     common.Address
	AtoBOrders map[common.Hash]*types.OrderState
	BtoAOrders map[common.Hash]*types.OrderState

	AtoBOrderHashesExcludeNextRound []common.Hash
	BtoAOrderHashesExcludeNextRound []common.Hash
}

//todo:miner 需要有足够的lrc，否则费用选择分润时，收不到任何收益
func (market *Market) match() {
	market.getOrdersForMatching(market.protocolAddress)
	matchedOrderHashes := make(map[common.Hash]bool) //true:fullfilled, false:partfilled
	ringSubmitInfos := []*types.RingSubmitInfo{}
	candidateRingList := CandidateRingList{}

	for _, a2BOrder := range market.AtoBOrders {
		for _, b2AOrder := range market.BtoAOrders {
			if miner.PriceValid(a2BOrder, b2AOrder) {
				if ringForSubmit, err := market.generateRingSubmitInfo(a2BOrder, b2AOrder); nil != err {
					log.Errorf("err:%s", err.Error())
					continue
				} else {
					log.Debugf("ringForSubmit: %s , Received: %s , protocolGas: %s , protocolGasPrice: %s", ringForSubmit.Ringhash.Hex(), ringForSubmit.Received.String(), ringForSubmit.ProtocolGas.String(), ringForSubmit.ProtocolGasPrice.String())
					if ringForSubmit.Received.Cmp(big.NewRat(int64(0), int64(1))) > 0 {
						candidateRingList = append(candidateRingList, CandidateRing{received: ringForSubmit.Received, orderHashes: []common.Hash{a2BOrder.RawOrder.Hash, b2AOrder.RawOrder.Hash}})
					} else {
						log.Debugf("timing_matchher,market ringForSubmit received not enough, received:%s, gas:%s, gasPrice:%s ", ringForSubmit.Received.String(), ringForSubmit.ProtocolGas.String(), ringForSubmit.ProtocolGasPrice.String())
					}
				}
			}
		}
	}

	sort.Sort(candidateRingList)
	for _, candidateRing := range candidateRingList {
		var (
			a2BOrder, b2AOrder *types.OrderState
			exists             bool
		)
		if a2BOrder, exists = market.AtoBOrders[candidateRing.orderHashes[0]]; exists {
			b2AOrder = market.BtoAOrders[candidateRing.orderHashes[1]]
		} else {
			a2BOrder = market.AtoBOrders[candidateRing.orderHashes[1]]
			b2AOrder = market.BtoAOrders[candidateRing.orderHashes[0]]
		}
		if ringForSubmit, err := market.generateRingSubmitInfo(a2BOrder, b2AOrder); nil != err {
			log.Errorf("filledOrderfilledOrderfilledOrder  err:%s", err.Error())
			continue
		} else {
			if ringForSubmit.Received.Cmp(big.NewRat(int64(0), int64(1))) > 0 {
				for _, filledOrder := range ringForSubmit.RawRing.Orders {
					orderState := market.reduceAmountAfterFilled(filledOrder)
					matchedOrderHashes[filledOrder.OrderState.RawOrder.Hash] = market.om.IsOrderFullFinished(orderState)
					market.matcher.addMatchedOrder(filledOrder, ringForSubmit.RawRing.Hash)
				}
				ringSubmitInfos = append(ringSubmitInfos, ringForSubmit)
			}
		}
	}

	for orderHash, _ := range market.AtoBOrders {
		if fullFilled, exists := matchedOrderHashes[orderHash]; !exists || fullFilled {
			market.AtoBOrderHashesExcludeNextRound = append(market.AtoBOrderHashesExcludeNextRound, orderHash)
		}
	}

	for orderHash, _ := range market.BtoAOrders {
		if fullFilled, exists := matchedOrderHashes[orderHash]; !exists || fullFilled {
			market.BtoAOrderHashesExcludeNextRound = append(market.BtoAOrderHashesExcludeNextRound, orderHash)
		}
	}
	eventemitter.Emit(eventemitter.Miner_NewRing, ringSubmitInfos)
}

/**
get orders from ordermanager
*/
func (market *Market) getOrdersForMatching(protocolAddress common.Address) {
	market.AtoBOrders = make(map[common.Hash]*types.OrderState)
	market.BtoAOrders = make(map[common.Hash]*types.OrderState)

	// log.Debugf("timing matcher,market tokenA:%s, tokenB:%s, atob hash length:%d, btoa hash length:%d", market.TokenA.Hex(), market.TokenB.Hex(), len(market.AtoBOrderHashesExcludeNextRound), len(market.BtoAOrderHashesExcludeNextRound))

	deleyedNumber := market.matcher.delayedNumber + rand.Int63n(market.matcher.delayedNumber)
	deleyedNumber = deleyedNumber / 2

	atoBOrders := market.om.MinerOrders(protocolAddress, market.TokenA, market.TokenB, market.matcher.roundOrderCount, &types.OrderDelayList{OrderHash: market.AtoBOrderHashesExcludeNextRound, DelayedCount: deleyedNumber})
	btoAOrders := market.om.MinerOrders(protocolAddress, market.TokenB, market.TokenA, market.matcher.roundOrderCount, &types.OrderDelayList{OrderHash: market.BtoAOrderHashesExcludeNextRound, DelayedCount: deleyedNumber})

	market.AtoBOrderHashesExcludeNextRound = []common.Hash{}
	market.BtoAOrderHashesExcludeNextRound = []common.Hash{}

	for _, order := range atoBOrders {
		market.reduceRemainedAmountBeforeMatch(order)
		if !market.om.IsOrderFullFinished(order) {
			market.AtoBOrders[order.RawOrder.Hash] = order
		} else {
			market.AtoBOrderHashesExcludeNextRound = append(market.AtoBOrderHashesExcludeNextRound, order.RawOrder.Hash)
		}
		log.Debugf("remain status in this round, orderhash:%s, DealtAmountS:%s, ", order.RawOrder.Hash.Hex(), order.DealtAmountS.String())
	}

	for _, order := range btoAOrders {
		market.reduceRemainedAmountBeforeMatch(order)
		if !market.om.IsOrderFullFinished(order) {
			market.BtoAOrders[order.RawOrder.Hash] = order
		} else {
			market.BtoAOrderHashesExcludeNextRound = append(market.BtoAOrderHashesExcludeNextRound, order.RawOrder.Hash)
		}
		log.Debugf("remain status in this round, orderhash:%s, DealtAmountS:%s", order.RawOrder.Hash.Hex(), order.DealtAmountS.String())
	}

}

//sub the matched amount in new round.
func (market *Market) reduceRemainedAmountBeforeMatch(orderState *types.OrderState) {
	orderHash := orderState.RawOrder.Hash

	if matchedOrder, ok := market.matcher.MatchedOrders[orderHash]; ok {
		//if len(matchedOrder.rounds) <= 0 {
		//	delete(market.AtoBOrders, orderHash)
		//	delete(market.BtoAOrders, orderHash)
		//} else {
		for _, matchedRound := range matchedOrder.rounds {
			orderState.DealtAmountB.Add(orderState.DealtAmountB, ratToInt(matchedRound.matchedAmountB))
			orderState.DealtAmountS.Add(orderState.DealtAmountS, ratToInt(matchedRound.matchedAmountS))
		}
		//}
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
	log.Debugf("order status after match another order, orderhash:%s, DealtAmountS:%s, ", orderState.RawOrder.Hash.Hex(), orderState.DealtAmountS.String())
	return orderState
}

func (market *Market) generateRingSubmitInfo(a2BOrder, b2AOrder *types.OrderState) (*types.RingSubmitInfo, error) {
	filledOrders := []*types.FilledOrder{}
	var (
		a2BLrcToken *miner.TokenBalance
		a2BTokenS   *miner.TokenBalance
		b2ALrcToken *miner.TokenBalance
		b2ATokenS   *miner.TokenBalance
		err         error
	)
	if a2BLrcToken, err = market.matcher.getAccountBalance(a2BOrder.RawOrder.Owner, market.lrcAddress); nil != err {
		return nil, err
	}
	if a2BTokenS, err = market.matcher.getAccountBalance(a2BOrder.RawOrder.Owner, a2BOrder.RawOrder.TokenS); nil != err {
		return nil, err
	}
	filledOrders = append(filledOrders, miner.ConvertOrderStateToFilledOrder(*a2BOrder, a2BLrcToken.Available(), a2BTokenS.Available()))

	if b2ALrcToken, err = market.matcher.getAccountBalance(b2AOrder.RawOrder.Owner, market.lrcAddress); nil != err {
		return nil, err
	}
	if b2ATokenS, err = market.matcher.getAccountBalance(b2AOrder.RawOrder.Owner, b2AOrder.RawOrder.TokenS); nil != err {
		return nil, err
	}
	filledOrders = append(filledOrders, miner.ConvertOrderStateToFilledOrder(*b2AOrder, b2ALrcToken.Available(), b2ATokenS.Available()))

	ringTmp := miner.NewRing(filledOrders)
	market.matcher.evaluator.ComputeRing(ringTmp)

	return market.matcher.submitter.GenerateRingSubmitInfo(ringTmp)
}

func ratToInt(rat *big.Rat) *big.Int {
	return new(big.Int).Div(rat.Num(), rat.Denom())
}
