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
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"math/big"
	"sync"
	"time"
)

/**
定时从ordermanager中拉取n条order数据进行匹配成环，如果成环则通过调用evaluator进行费用估计，然后提交到submitter进行提交到以太坊
*/

type minedRing struct {
	ringHash    types.Hash
	orderHashes []types.Hash
}

type RoundState struct {
	round          int64
	ringHash       types.Hash
	matchedAmountS *big.Rat
	matchedAmountB *big.Rat
}

type OrderMatchState struct {
	orderState types.OrderState
	round      []*RoundState
}

type TimingMatcher struct {
	MatchedOrders map[types.Hash]*OrderMatchState
	MinedRings    map[types.Hash]*minedRing
	mtx           sync.RWMutex
	StopChan      chan bool
	round         int64
	markets       []*Market
}

type Market struct {
	matcher      *TimingMatcher
	ordermanager ordermanager.OrderManager

	TokenA     types.Address
	TokenB     types.Address
	AtoBOrders map[types.Hash]*types.OrderState
	BtoAOrders map[types.Hash]*types.OrderState

	AtoBNotMatchedOrderHashes []types.Hash
	BtoANotMatchedOrderHashes []types.Hash
}

func NewTimingMatcher() *TimingMatcher {
	matcher := &TimingMatcher{}
	matcher.MatchedOrders = make(map[types.Hash]*OrderMatchState)
	//todo:get markets from market.Allmarket
	m := &Market{}
	m.matcher = matcher
	m.AtoBNotMatchedOrderHashes = []types.Hash{}
	m.BtoANotMatchedOrderHashes = []types.Hash{}
	matcher.markets = []*Market{m}

	return matcher
}

func (matcher *TimingMatcher) Start() {
	watcher := &eventemitter.Watcher{Concurrent: false, Handle: matcher.afterSubmit}
	//todo:the topic should contain submit success
	eventemitter.On(eventemitter.RingMined, watcher)

	go func() {
		for {
			select {
			case <-time.After(1 * time.Second):
				var wg sync.WaitGroup
				matcher.round += 1
				for _, market := range matcher.markets {
					wg.Add(1)
					go func(market *Market) {
						defer func() {
							wg.Add(-1)
						}()
						market.match()
					}(market)
				}
				wg.Wait()
			case <-matcher.StopChan:
				break
			}
		}
	}()
}

func (market *Market) getOrdersForMatching() {
	market.AtoBOrders = make(map[types.Hash]*types.OrderState)
	market.BtoAOrders = make(map[types.Hash]*types.OrderState)

	atoBOrders := market.ordermanager.MinerOrders(market.TokenA, market.TokenB, market.AtoBNotMatchedOrderHashes)
	btoAOrders := market.ordermanager.MinerOrders(market.TokenB, market.TokenA, market.BtoANotMatchedOrderHashes)

	for _, order := range atoBOrders {
		market.AtoBOrders[order.RawOrder.Hash] = &order
	}

	for _, order := range btoAOrders {
		market.BtoAOrders[order.RawOrder.Hash] = &order
	}

	market.AtoBNotMatchedOrderHashes = []types.Hash{}
	market.BtoANotMatchedOrderHashes = []types.Hash{}

	//it should sub the matched amount in last round.
	market.reduceRemainedAmountBeforeMatch()
}

func (market *Market) reduceRemainedAmountBeforeMatch() {
	for orderHash, orderState := range market.AtoBOrders {
		if matchedOrder, ok := market.matcher.MatchedOrders[orderHash]; ok {
			if len(matchedOrder.round) <= 0 {
				delete(market.AtoBOrders, orderHash)
			} else {
				for _, matchedRound := range matchedOrder.round {
					orderState.RemainedAmountB.Sub(orderState.RemainedAmountB, intFromRat(matchedRound.matchedAmountB))
					orderState.RemainedAmountS.Sub(orderState.RemainedAmountS, intFromRat(matchedRound.matchedAmountS))
				}
			}
		}
	}
}

func (market *Market) reduceRemainedAmountAfterFilled(filledOrder *types.FilledOrder) {
	filledOrderState := filledOrder.OrderState
	if filledOrderState.RawOrder.TokenS == market.TokenA {
		orderState := market.AtoBOrders[filledOrderState.RawOrder.Hash]
		orderState.RemainedAmountB.Sub(orderState.RemainedAmountB, intFromRat(filledOrder.FillAmountB))
		orderState.RemainedAmountS.Sub(orderState.RemainedAmountS, intFromRat(filledOrder.FillAmountS))
	} else {
		orderState := market.BtoAOrders[filledOrderState.RawOrder.Hash]
		orderState.RemainedAmountB.Sub(orderState.RemainedAmountB, intFromRat(filledOrder.FillAmountB))
		orderState.RemainedAmountS.Sub(orderState.RemainedAmountS, intFromRat(filledOrder.FillAmountS))
	}
}

func (market *Market) match() {

	market.getOrdersForMatching()

	matchedOrderHashes := make(map[types.Hash]bool)
	ringStates := []*types.RingState{}
	for _, a2BOrder := range market.AtoBOrders {
		var ringState *types.RingState
		for _, b2AOrder := range market.BtoAOrders {
			if miner.PriceValid(a2BOrder, b2AOrder) {
				ringTmp := &types.Ring{}

				ringTmp.Orders = []*types.FilledOrder{convertOrderStateToFilledOrder(a2BOrder), convertOrderStateToFilledOrder(b2AOrder)}

				ringStateTmp := &types.RingState{}
				ringStateTmp.RawRing = ringTmp
				miner.ComputeRing(ringStateTmp)
				if nil == ringState || ringState.LegalFee.Cmp(ringStateTmp.LegalFee) < 0 {
					ringState = ringStateTmp
				}
			}
		}

		//对每个order标记已匹配以及减去已匹配的金额
		if nil != ringState {
			for _, filledOrder := range ringState.RawRing.Orders {
				market.reduceRemainedAmountAfterFilled(filledOrder)
				matchedOrderHashes[filledOrder.OrderState.RawOrder.Hash] = true
				market.matcher.addMatchedOrder(filledOrder, ringState.RawRing.Hash)
			}
		}
		ringStates = append(ringStates, ringState)
	}

	for orderHash, _ := range market.AtoBOrders {
		if _, exists := matchedOrderHashes[orderHash]; !exists {
			market.AtoBNotMatchedOrderHashes = append(market.AtoBNotMatchedOrderHashes, orderHash)
		}
	}
	for orderHash, _ := range market.BtoAOrders {
		if _, exists := matchedOrderHashes[orderHash]; !exists {
			market.BtoANotMatchedOrderHashes = append(market.BtoANotMatchedOrderHashes, orderHash)
		}
	}

	eventemitter.Emit(eventemitter.Miner_NewRing, ringStates)
}

func (matcher *TimingMatcher) afterSubmit(eventData eventemitter.EventData) error {
	matcher.mtx.Lock()
	defer matcher.mtx.Unlock()
	ringHash := eventData.(types.Hash)
	if ringState, ok := matcher.MinedRings[ringHash]; ok {
		delete(matcher.MinedRings, ringHash)
		for _, orderHash := range ringState.orderHashes {
			if minedState, ok := matcher.MatchedOrders[orderHash]; ok {
				if len(minedState.round) <= 1 {
					delete(matcher.MatchedOrders, orderHash)
				} else {
					for idx, s := range minedState.round {
						if s.ringHash == ringHash {
							round1 := append(minedState.round[:idx], minedState.round[idx+1:]...)
							minedState.round = round1
						}
					}
				}
			}
		}
	}
	return nil
}

func (matcher *TimingMatcher) Stop() {
	matcher.StopChan <- true
}

func (matcher *TimingMatcher) addMatchedOrder(filledOrder *types.FilledOrder, ringiHash types.Hash) {
	matcher.mtx.Lock()
	defer matcher.mtx.Unlock()

	var matchState *OrderMatchState
	if matchState1, ok := matcher.MatchedOrders[filledOrder.OrderState.RawOrder.Hash]; !ok {
		matchState = &OrderMatchState{}
		matchState.orderState = filledOrder.OrderState
		matchState.round = []*RoundState{}
	} else {
		matchState = matchState1
	}

	roundState := &RoundState{
		round:          matcher.round,
		ringHash:       ringiHash,
		matchedAmountB: filledOrder.FillAmountB,
		matchedAmountS: filledOrder.FillAmountS,
	}

	matchState.round = append(matchState.round, roundState)
	matcher.MatchedOrders[filledOrder.OrderState.RawOrder.Hash] = matchState
}

func intFromRat(rat *big.Rat) *big.Int {
	return new(big.Int).Div(rat.Num(), rat.Denom())
}

func convertOrderStateToFilledOrder(order *types.OrderState) *types.FilledOrder {
	filledOrder := &types.FilledOrder{}
	filledOrder.OrderState = *order
	return filledOrder
}
