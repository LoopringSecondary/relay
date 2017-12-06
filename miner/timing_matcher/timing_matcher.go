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
	marketLib "github.com/Loopring/relay/market/util"
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
	rounds     []*RoundState
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

	AtoBOrderHashesExcludeNextRound []common.Hash
	BtoAOrderHashesExcludeNextRound []common.Hash
}

func NewTimingMatcher(submitter *miner.RingSubmitter, evaluator *miner.Evaluator, om ordermanager.OrderManager) *TimingMatcher {
	matcher := &TimingMatcher{submitter: submitter, evaluator: evaluator}
	matcher.MatchedOrders = make(map[common.Hash]*OrderMatchState)
	matcher.markets = []*Market{}
	pairs := make(map[common.Address]common.Address)
	for _, pair := range marketLib.AllTokenPairs {
		if addr, ok := pairs[pair.TokenS]; !ok || addr != pair.TokenB {
			if addr1, ok1 := pairs[pair.TokenB]; !ok1 || addr1 != pair.TokenS {
				pairs[pair.TokenS] = pair.TokenB
				m := &Market{}
				m.om = om
				m.matcher = matcher
				m.TokenA = pair.TokenS
				m.TokenB = pair.TokenB
				m.AtoBOrderHashesExcludeNextRound = []common.Hash{}
				m.BtoAOrderHashesExcludeNextRound = []common.Hash{}
				matcher.markets = append(matcher.markets, m)
			}
		} else {
			log.Debugf("miner,timing matcher cann't find tokenPair tokenS:%s, tokenB:%s", pair.TokenS.Hex(), pair.TokenB.Hex())
		}
	}

	return matcher
}

func (matcher *TimingMatcher) Start() {
	watcher := &eventemitter.Watcher{Concurrent: false, Handle: matcher.afterSubmit}
	//todo:the topic should contain submit success
	eventemitter.On(eventemitter.Miner_RingMined, watcher)
	eventemitter.On(eventemitter.Miner_RingSubmitFailed, watcher)

	go func() {
		for {
			select {
			case <-time.After(30 * time.Second):
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

	// log.Debugf("timing matcher,market tokenA:%s, tokenB:%s, atob hash length:%d, btoa hash length:%d", market.TokenA.Hex(), market.TokenB.Hex(), len(market.AtoBOrderHashesExcludeNextRound), len(market.BtoAOrderHashesExcludeNextRound))

	atoBOrders := market.om.MinerOrders(market.TokenA, market.TokenB, 50, market.AtoBOrderHashesExcludeNextRound)
	btoAOrders := market.om.MinerOrders(market.TokenB, market.TokenA, 50, market.BtoAOrderHashesExcludeNextRound)

	for _, order := range atoBOrders {
		market.reduceRemainedAmountBeforeMatch(&order)
		if !market.om.IsOrderFullFinished(&order) {
			market.AtoBOrders[order.RawOrder.Hash] = &order
		}
	}

	for _, order := range btoAOrders {
		market.reduceRemainedAmountBeforeMatch(&order)
		if !market.om.IsOrderFullFinished(&order) {
			market.BtoAOrders[order.RawOrder.Hash] = &order
		}
	}

	market.AtoBOrderHashesExcludeNextRound = []common.Hash{}
	market.BtoAOrderHashesExcludeNextRound = []common.Hash{}
	//it should sub the matched amount in next round.
}

func (market *Market) reduceRemainedAmountBeforeMatch(orderState *types.OrderState) {
	orderHash := orderState.RawOrder.Hash
	if matchedOrder, ok := market.matcher.MatchedOrders[orderHash]; ok {
		if len(matchedOrder.rounds) <= 0 {
			delete(market.AtoBOrders, orderHash)
			delete(market.BtoAOrders, orderHash)
		} else {
			for _, matchedRound := range matchedOrder.rounds {
				orderState.DealtAmountB.Sub(orderState.DealtAmountB, intFromRat(matchedRound.matchedAmountB))
				orderState.DealtAmountS.Sub(orderState.DealtAmountS, intFromRat(matchedRound.matchedAmountS))
			}
		}
	}
}

func (market *Market) reduceRemainedAmountAfterFilled(filledOrder *types.FilledOrder) *types.OrderState {
	filledOrderState := filledOrder.OrderState
	var orderState *types.OrderState
	if filledOrderState.RawOrder.TokenS == market.TokenA {
		orderState = market.AtoBOrders[filledOrderState.RawOrder.Hash]
		orderState.DealtAmountB.Sub(orderState.DealtAmountB, intFromRat(filledOrder.FillAmountB))
		orderState.DealtAmountS.Sub(orderState.DealtAmountS, intFromRat(filledOrder.FillAmountS))
	} else {
		orderState = market.BtoAOrders[filledOrderState.RawOrder.Hash]
		orderState.DealtAmountB.Sub(orderState.DealtAmountB, intFromRat(filledOrder.FillAmountB))
		orderState.DealtAmountS.Sub(orderState.DealtAmountS, intFromRat(filledOrder.FillAmountS))
	}
	return orderState
}

func (market *Market) match() {
	market.getOrdersForMatching()
	matchedOrderHashes := make(map[common.Hash]bool)  //true:fullmatched, false:partmatched
	ringStates := []*types.RingSubmitInfo{}
	for _, a2BOrder := range market.AtoBOrders {
		var ringForSubmit *types.RingSubmitInfo
		for _, b2AOrder := range market.BtoAOrders {
			if miner.PriceValid(a2BOrder, b2AOrder) {
				ringTmp := types.NewRing([]types.OrderState{*a2BOrder, *b2AOrder})
				market.matcher.evaluator.ComputeRing(ringTmp)
				ringForSubmitTmp, err := market.matcher.submitter.GenerateRingSubmitInfo(ringTmp)
				if nil != err {
					log.Errorf("err: %s", err.Error())
				} else {
					if nil == ringForSubmit || ringForSubmit.Received.Cmp(ringForSubmitTmp.Received) < 0 {
						ringForSubmit = ringForSubmitTmp
					}
				}
			}
		}

		//对每个order标记已匹配以及减去已匹配的金额
		if nil != ringForSubmit {
			for _, filledOrder := range ringForSubmit.RawRing.Orders {
				orderState := market.reduceRemainedAmountAfterFilled(filledOrder)
				matchedOrderHashes[filledOrder.OrderState.RawOrder.Hash] = market.om.IsOrderFullFinished(orderState)
				market.matcher.addMatchedOrder(filledOrder, ringForSubmit.RawRing.Hash)
			}
			ringStates = append(ringStates, ringForSubmit)
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
	eventemitter.Emit(eventemitter.Miner_NewRing, ringStates)
}

func (matcher *TimingMatcher) afterSubmit(eventData eventemitter.EventData) error {
	matcher.mtx.Lock()
	defer matcher.mtx.Unlock()

	e := eventData.(*types.RingMinedEvent)
	ringHash := e.Ringhash
	if ringState, ok := matcher.MinedRings[ringHash]; ok {
		delete(matcher.MinedRings, ringHash)
		for _, orderHash := range ringState.orderHashes {
			if minedState, ok := matcher.MatchedOrders[orderHash]; ok {
				if len(minedState.rounds) <= 1 {
					delete(matcher.MatchedOrders, orderHash)
				} else {
					for idx, s := range minedState.rounds {
						if s.ringHash == ringHash {
							round1 := append(minedState.rounds[:idx], minedState.rounds[idx+1:]...)
							minedState.rounds = round1
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
		matchState.rounds = []*RoundState{}
	} else {
		matchState = matchState1
	}

	roundState := &RoundState{
		round:          matcher.round,
		ringHash:       ringiHash,
		matchedAmountB: filledOrder.FillAmountB,
		matchedAmountS: filledOrder.FillAmountS,
	}

	matchState.rounds = append(matchState.rounds, roundState)
	matcher.MatchedOrders[filledOrder.OrderState.RawOrder.Hash] = matchState
}

func intFromRat(rat *big.Rat) *big.Int {
	return new(big.Int).Div(rat.Num(), rat.Denom())
}
