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
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"

	"github.com/Loopring/relay/config"
	marketLib "github.com/Loopring/relay/market"
	marketUtilLib "github.com/Loopring/relay/market/util"
)

/**
定时从ordermanager中拉取n条order数据进行匹配成环，如果成环则通过调用evaluator进行费用估计，然后提交到submitter进行提交到以太坊
*/

type TimingMatcher struct {
	MatchedOrders   map[common.Hash]*OrderMatchState
	MinedRings      map[common.Hash]*minedRing
	mtx             sync.RWMutex
	roundMtx        sync.RWMutex
	markets         []*Market
	submitter       *miner.RingSubmitter
	evaluator       *miner.Evaluator
	lastBlockNumber *big.Int
	duration        *big.Int
	roundOrderCount int
	delayedNumber   int64
	accounts        map[common.Address]*miner.Account
	accountManager  *marketLib.AccountManager

	stopFuncs []func()
}

func NewTimingMatcher(matcherOptions *config.TimingMatcher, submitter *miner.RingSubmitter, evaluator *miner.Evaluator, om ordermanager.OrderManager, accountManager *marketLib.AccountManager) *TimingMatcher {
	matcher := &TimingMatcher{}
	matcher.submitter = submitter
	matcher.evaluator = evaluator
	matcher.accountManager = accountManager
	matcher.roundOrderCount = matcherOptions.RoundOrdersCount
	matcher.MatchedOrders = make(map[common.Hash]*OrderMatchState)
	matcher.MinedRings = make(map[common.Hash]*minedRing)
	matcher.markets = []*Market{}
	matcher.duration = big.NewInt(matcherOptions.Duration)
	matcher.delayedNumber = matcherOptions.DelayedNumber

	matcher.lastBlockNumber = big.NewInt(0)
	matcher.mtx = sync.RWMutex{}
	matcher.roundMtx = sync.RWMutex{}
	matcher.stopFuncs = []func(){}

	for _, pair := range marketUtilLib.AllTokenPairs {
		inited := false
		for _, market := range matcher.markets {
			if (market.TokenB == pair.TokenB && market.TokenA == pair.TokenS) ||
				(market.TokenA == pair.TokenB && market.TokenB == pair.TokenS) {
				inited = true
				break
			}
		}
		if !inited {
			for _, protocolAddress := range matcher.submitter.Accessor.ProtocolAddresses {
				m := &Market{}
				m.protocolAddress = protocolAddress.ContractAddress
				m.lrcAddress = protocolAddress.LrcTokenAddress
				m.om = om
				m.matcher = matcher
				m.TokenA = pair.TokenS
				m.TokenB = pair.TokenB
				m.AtoBOrderHashesExcludeNextRound = []common.Hash{}
				m.BtoAOrderHashesExcludeNextRound = []common.Hash{}
				matcher.markets = append(matcher.markets, m)
			}
		}
	}

	return matcher
}

func (matcher *TimingMatcher) Start() {
	matcher.listenNewBlock()
	matcher.listenSubmitEvent()
}

func (matcher *TimingMatcher) Stop() {
	for _, stop := range matcher.stopFuncs {
		stop()
	}
}

func (matcher *TimingMatcher) deleteRoundAfterSubmit(ringHash common.Hash) {
	matcher.mtx.Lock()
	defer matcher.mtx.Unlock()

	if ringState, ok := matcher.MinedRings[ringHash]; ok {
		log.Debugf("MinedRings ringhash:%s will be removed", ringHash.Hex())
		delete(matcher.MinedRings, ringHash)
		for _, orderHash := range ringState.orderHashes {
			if minedState, ok := matcher.MatchedOrders[orderHash]; ok {
				if len(minedState.rounds) <= 1 {
					delete(matcher.MatchedOrders, orderHash)
				} else {
					for idx, s := range minedState.rounds {
						if s.ringHash == ringHash {
							log.Debugf("MatchedOrders ringhash:%s will be removed", ringHash.Hex())
							round1 := append(minedState.rounds[:idx], minedState.rounds[idx+1:]...)
							minedState.rounds = round1
						}
					}
				}
			}
		}
	}
}

func (matcher *TimingMatcher) addMatchedOrder(filledOrder *types.FilledOrder, ringHash common.Hash) {
	matcher.mtx.Lock()
	defer matcher.mtx.Unlock()

	if ring, exists := matcher.MinedRings[ringHash]; !exists {
		ring = &minedRing{ringHash: ringHash, orderHashes: []common.Hash{filledOrder.OrderState.RawOrder.Hash}}
		matcher.MinedRings[ringHash] = ring
	} else {
		ring.orderHashes = append(ring.orderHashes, filledOrder.OrderState.RawOrder.Hash)
	}
	var matchState *OrderMatchState
	if matchState1, ok := matcher.MatchedOrders[filledOrder.OrderState.RawOrder.Hash]; !ok {
		matchState = &OrderMatchState{}
		matchState.orderState = filledOrder.OrderState
		matchState.rounds = []*RoundState{}
	} else {
		matchState = matchState1
	}

	roundState := &RoundState{
		round:          matcher.lastBlockNumber,
		ringHash:       ringHash,
		matchedAmountB: filledOrder.FillAmountB,
		matchedAmountS: filledOrder.FillAmountS,
	}

	matchState.rounds = append(matchState.rounds, roundState)
	matcher.MatchedOrders[filledOrder.OrderState.RawOrder.Hash] = matchState
}

func (matcher *TimingMatcher) getAccountBalance(address common.Address, tokenAddress common.Address) (*miner.TokenBalance, error) {
	matcher.mtx.Lock()
	defer matcher.mtx.Unlock()

	var (
		account *miner.Account
		exists  bool
	)
	if account, exists = matcher.accounts[address]; !exists {
		account = &miner.Account{}
		account.Tokens = make(map[common.Address]*miner.TokenBalance)
		matcher.accounts[address] = account
	}

	if b, e := account.Tokens[tokenAddress]; !e {
		if balance, allowance, err := matcher.accountManager.GetBalanceByTokenAddress(address, tokenAddress); nil != err {
			return nil, err
		} else {
			b = &miner.TokenBalance{}
			b.Allowance = new(big.Int).Set(allowance)
			b.Balance = new(big.Int).Set(balance)
			account.Tokens[tokenAddress] = b
			return b, nil
		}
	} else {
		return b, nil
	}
}
