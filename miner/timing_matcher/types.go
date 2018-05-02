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
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

//type OrderMatchedState struct {
//	ringHash      common.Hash `json:"ringhash"`
//	filledAmountS *big.Rat    `json:"filled_amount_s"`
//	filledAmountB *big.Rat    `json:"filled_amount_b"`
//}
//
//type OrderRoundState struct {
//	owner  common.Address `json:"owner"`
//	tokenS common.Address `json:"token_s"`
//	rings  map[common.Hash]*OrderMatchedState `json:"order_matched_state"`
//}
//
//type RoundState struct {
//	mtx             sync.RWMutex
//	round           *big.Int
//	orderStates     map[common.Hash]*OrderRoundState
//	matchedBalances map[common.Address][]common.Hash
//}
//
//func NewRoundState(round *big.Int) *RoundState {
//	rs := &RoundState{}
//	rs.mtx = sync.RWMutex{}
//	rs.round = new(big.Int).Set(round)
//	rs.orderStates = make(map[common.Hash]*OrderRoundState)
//	rs.matchedBalances = make(map[common.Address][]common.Hash)
//
//	return rs
//}
//
//func (rs *RoundState) FilledAmountS(owner common.Address, token common.Address) (amountS *big.Rat) {
//	rs.mtx.RLock()
//	defer rs.mtx.RUnlock()
//
//	amountS = new(big.Rat).SetInt64(int64(0))
//	if orderhashes, exists := rs.matchedBalances[owner]; exists {
//		for _, orderhash := range orderhashes {
//			if roundState, exists := rs.orderStates[orderhash]; exists && token == roundState.tokenS {
//				for _, matchedState := range roundState.rings {
//					amountS.Add(amountS, matchedState.filledAmountS)
//				}
//			}
//		}
//	}
//	return amountS
//}
//
//func (rs *RoundState) DealtAmount(orderhash common.Hash) (amountS *big.Rat, amountB *big.Rat) {
//	rs.mtx.RLock()
//	defer rs.mtx.RUnlock()
//
//	amountS = new(big.Rat).SetInt64(int64(0))
//	amountB = new(big.Rat).SetInt64(int64(0))
//
//	if order, exists := rs.orderStates[orderhash]; exists {
//		log.Debugf("state.orderStatesstate.orderStates :%d", len(order.rings))
//		for _, matched := range order.rings {
//			amountB.Add(amountB, matched.filledAmountB)
//			amountS.Add(amountS, matched.filledAmountS)
//		}
//	}
//	return amountS, amountB
//}
//
//func (rs *RoundState) removeMinedOrder(orderhash common.Hash) {
//	if len(rs.orderStates[orderhash].rings) <= 0 {
//		delete(rs.orderStates, orderhash)
//		balancesMap := make(map[common.Address][]common.Hash)
//		for addr, hashes := range rs.matchedBalances {
//			hashes1 := []common.Hash{}
//			for _, orderhash1 := range hashes {
//				if orderhash != orderhash1 {
//					hashes1 = append(hashes1, orderhash)
//				}
//			}
//			if len(hashes1) > 0 {
//				balancesMap[addr] = hashes1
//			}
//		}
//		rs.matchedBalances = balancesMap
//	}
//}
//
//func (rs *RoundState) RemoveMinedRing(ringhash common.Hash) {
//	rs.mtx.Lock()
//	defer rs.mtx.Unlock()
//
//	orderhashes := []common.Hash{}
//	for orderhash, orderState := range rs.orderStates {
//		if _, exists := orderState.rings[ringhash]; exists {
//			orderhashes = append(orderhashes, orderhash)
//		}
//	}
//	for _, hash := range orderhashes {
//		delete(rs.orderStates[hash].rings, ringhash)
//		if len(rs.orderStates[hash].rings) <= 0 {
//			rs.removeMinedOrder(hash)
//		}
//	}
//}
//
//func (rs *RoundState) AddMatchedOrders(filledOrder *types.FilledOrder, ringHash common.Hash) {
//	rs.mtx.Lock()
//	defer rs.mtx.Unlock()
//
//	owner := filledOrder.OrderState.RawOrder.Owner
//	orderhash := filledOrder.OrderState.RawOrder.Hash
//	orderMatchedState := &OrderMatchedState{}
//	orderMatchedState.ringHash = ringHash
//	orderMatchedState.filledAmountB = filledOrder.FillAmountB
//	orderMatchedState.filledAmountS = filledOrder.FillAmountS
//	var (
//		orderRoundState *OrderRoundState
//		exists          bool
//	)
//	if orderRoundState, exists = rs.orderStates[orderhash]; !exists {
//		orderRoundState = &OrderRoundState{}
//		orderRoundState.owner = owner
//		orderRoundState.rings = make(map[common.Hash]*OrderMatchedState)
//		rs.orderStates[orderhash] = orderRoundState
//	}
//	rs.orderStates[orderhash].rings[ringHash] = orderMatchedState
//
//	if _, exists := rs.matchedBalances[owner]; !exists {
//		rs.matchedBalances[owner] = []common.Hash{}
//	}
//
//	rs.matchedBalances[owner] = append(rs.matchedBalances[owner], orderhash)
//}
//
//
//const (
//	ROUNDSTATE = "roundstate_"
//
//)
//
//type RoundStates struct {
//	mtx            sync.RWMutex
//	states         []*RoundState
//	maxCacheLength int
//	//orderRound map[common.Hash][]*big.Int
//	//ringRound map[common.Hash][]*big.Int
//	//balanceRound map[common.Address][]*big.Int
//}
//
//func NewRoundStates(maxCacheLength int) *RoundStates {
//	r := &RoundStates{}
//	r.mtx = sync.RWMutex{}
//	r.states = []*RoundState{}
//	r.maxCacheLength = maxCacheLength
//	return r
//}
//
//func (r *RoundStates) appendNewRoundState(round *big.Int) {
//	r.mtx.Lock()
//	defer r.mtx.Unlock()
//
//	state := NewRoundState(round)
//	if (2 * r.maxCacheLength) <= len(r.states) {
//		r.states = r.states[(len(r.states) - 2*r.maxCacheLength):]
//	}
//	r.states = append(r.states, state)
//}
//
//func (r *RoundStates) AppendFilledOrderToCurrent(filledOrder *types.FilledOrder, ringHash common.Hash) {
//	r.mtx.Lock()
//	defer r.mtx.Unlock()
//
//	r.states[len(r.states)-1].AddMatchedOrders(filledOrder, ringHash)
//}
//
//func (r *RoundStates) maxCacheRounds() []*RoundState {
//	startIdx := len(r.states) - r.maxCacheLength
//	if startIdx < 0 {
//		startIdx = 0
//	}
//	return r.states[startIdx:]
//}
//
//func (r *RoundStates) DealtAmount(orderhash common.Hash) (amountS *big.Rat, amountB *big.Rat) {
//	r.mtx.RLock()
//	defer r.mtx.RUnlock()
//
//	amountS = new(big.Rat).SetInt64(int64(0))
//	amountB = new(big.Rat).SetInt64(int64(0))
//	for _, state := range r.maxCacheRounds() {
//		amountS1, amountB1 := state.DealtAmount(orderhash)
//		amountS.Add(amountS, amountS1)
//		amountB.Add(amountB, amountB1)
//	}
//
//	return amountS, amountB
//}
//
//func (r *RoundStates) FilledAmountS(owner common.Address, token common.Address) (amountS *big.Rat) {
//	r.mtx.RLock()
//	defer r.mtx.RUnlock()
//
//	amountS = new(big.Rat).SetInt64(int64(0))
//	for _, state := range r.maxCacheRounds() {
//		amountS.Add(amountS, state.FilledAmountS(owner, token))
//	}
//	return amountS
//}
//
//func (r *RoundStates) RemoveMinedRing(ringhash common.Hash) {
//	r.mtx.Lock()
//	defer r.mtx.Unlock()
//
//	for _, state := range r.states {
//		state.RemoveMinedRing(ringhash)
//	}
//}

type CandidateRing struct {
	filledOrders map[common.Hash]*big.Rat
	received     *big.Rat
	cost         *big.Rat
}

type CandidateRingList []CandidateRing

func (ringList CandidateRingList) Len() int {
	return len(ringList)
}
func (ringList CandidateRingList) Swap(i, j int) {
	ringList[i], ringList[j] = ringList[j], ringList[i]
}
func (ringList CandidateRingList) Less(i, j int) bool {
	return ringList[i].received.Cmp(ringList[j].received) > 0
}
