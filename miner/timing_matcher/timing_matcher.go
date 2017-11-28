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
	"sync"
	"time"
)

/**
定时从ordermanager中拉取n条order数据进行匹配成环，如果成环则通过调用evaluator进行费用估计，然后提交到submitter进行提交到以太坊
*/

type minedRing struct {
	ringHash    common.Hash
	orderHashes []common.Hash
}

type RoundState struct {
	round          int64
	ringHash       common.Hash
	matchedAmountS *big.Rat
	matchedAmountB *big.Rat
}

type OrderMatchState struct {
	orderState types.OrderState
	round      []*RoundState
}

type TimingMatcher struct {
	MatchedOrders map[common.Hash]*OrderMatchState
	MinedRings    map[common.Hash]*minedRing
	mtx           sync.RWMutex
	StopChan      chan bool
	round         int64
	markets       []*Market
	submitter     *miner.RingSubmitter
	evaluator     *miner.Evaluator
}

type Market struct {
	matcher *TimingMatcher
	om      ordermanager.OrderManager

	TokenA     common.Address
	TokenB     common.Address
	AtoBOrders map[common.Hash]*types.OrderState
	BtoAOrders map[common.Hash]*types.OrderState

	AtoBNotMatchedOrderHashes []common.Hash
	BtoANotMatchedOrderHashes []common.Hash
}

func NewTimingMatcher(submitter *miner.RingSubmitter, evaluator *miner.Evaluator) *TimingMatcher {
	matcher := &TimingMatcher{submitter: submitter, evaluator: evaluator}
	matcher.MatchedOrders = make(map[common.Hash]*OrderMatchState)
	//todo:get markets from market.Allmarket
	m := &Market{}
	m.matcher = matcher
	m.AtoBNotMatchedOrderHashes = []common.Hash{}
	m.BtoANotMatchedOrderHashes = []common.Hash{}
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
	market.AtoBOrders = make(map[common.Hash]*types.OrderState)
	market.BtoAOrders = make(map[common.Hash]*types.OrderState)

	atoBOrders := market.om.MinerOrders(market.TokenA, market.TokenB, market.AtoBNotMatchedOrderHashes)
	btoAOrders := market.om.MinerOrders(market.TokenB, market.TokenA, market.BtoANotMatchedOrderHashes)
	//atoBOrders := []types.OrderState{}
	//btoAOrders := []types.OrderState{}
	for _, order := range atoBOrders {
		if market.reduceRemainedAmountBeforeMatch(&order) {
			market.AtoBOrders[order.RawOrder.Hash] = &order
		}
	}

	for _, order := range btoAOrders {
		if market.reduceRemainedAmountBeforeMatch(&order) {
			market.BtoAOrders[order.RawOrder.Hash] = &order
		}
	}

	market.AtoBNotMatchedOrderHashes = []common.Hash{}
	market.BtoANotMatchedOrderHashes = []common.Hash{}

	//it should sub the matched amount in last round.
}

func (market *Market) reduceRemainedAmountBeforeMatch(orderState *types.OrderState) bool {
	orderHash := orderState.RawOrder.Hash
	if matchedOrder, ok := market.matcher.MatchedOrders[orderHash]; ok {
		if len(matchedOrder.round) <= 0 {
			delete(market.AtoBOrders, orderHash)
		} else {
			for _, matchedRound := range matchedOrder.round {
				orderState.RemainedAmountB.Sub(orderState.RemainedAmountB, intFromRat(matchedRound.matchedAmountB))
				orderState.RemainedAmountS.Sub(orderState.RemainedAmountS, intFromRat(matchedRound.matchedAmountS))
			}
			if orderState.RemainedAmountB.Cmp(big.NewInt(int64(0))) <= 0 && orderState.RemainedAmountS.Cmp(big.NewInt(int64(0))) <= 0 {
				//todo
				return true
			}
		}
	}
	return true
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

	matchedOrderHashes := make(map[common.Hash]bool)
	ringStates := []*types.RingForSubmit{}
	for _, a2BOrder := range market.AtoBOrders {
		var ringForSubmit *types.RingForSubmit
		for _, b2AOrder := range market.BtoAOrders {
			if miner.PriceValid(a2BOrder, b2AOrder) {
				ringTmp := &types.Ring{}
				ringTmp.Orders = []*types.FilledOrder{convertOrderStateToFilledOrder(a2BOrder), convertOrderStateToFilledOrder(b2AOrder)}

				market.matcher.evaluator.ComputeRing(ringTmp)
				ringForSubmitTmp, err := market.matcher.submitter.GenerateRingSubmitArgs(ringTmp)
				if nil != err {
					log.Errorf("err: %s", err.Error())
				}
				if nil == ringForSubmit || ringForSubmit.RawRing.LegalFee.Cmp(ringTmp.LegalFee) < 0 {
					//todo: 需要确定花费的gas等是多少，来确定是否生成该环路
					ringForSubmit = ringForSubmitTmp
				}

			}
		}

		//对每个order标记已匹配以及减去已匹配的金额
		if nil != ringForSubmit {
			for _, filledOrder := range ringForSubmit.RawRing.Orders {
				market.reduceRemainedAmountAfterFilled(filledOrder)
				matchedOrderHashes[filledOrder.OrderState.RawOrder.Hash] = true
				market.matcher.addMatchedOrder(filledOrder, ringForSubmit.RawRing.Hash)
			}
			ringStates = append(ringStates, ringForSubmit)
		}
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
	ringHash := eventData.(common.Hash)
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

func (matcher *TimingMatcher) addMatchedOrder(filledOrder *types.FilledOrder, ringiHash common.Hash) {
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
