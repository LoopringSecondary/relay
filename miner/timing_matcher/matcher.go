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
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/ordermanager"
	"github.com/ethereum/go-ethereum/common"
	"math/big"

	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	marketLib "github.com/Loopring/relay/market"
	marketUtilLib "github.com/Loopring/relay/market/util"
)

/**
定时从ordermanager中拉取n条order数据进行匹配成环，如果成环则通过调用evaluator进行费用估计，然后提交到submitter进行提交到以太坊
*/

type TimingMatcher struct {
	rounds          *RoundStates
	markets         []*Market
	submitter       *miner.RingSubmitter
	evaluator       *miner.Evaluator
	lastBlockNumber *big.Int
	duration        *big.Int
	roundOrderCount int

	maxCacheRoundsLength int
	delayedNumber        int64
	accountManager       *marketLib.AccountManager

	stopFuncs []func()
}

func NewTimingMatcher(matcherOptions *config.TimingMatcher, submitter *miner.RingSubmitter, evaluator *miner.Evaluator, om ordermanager.OrderManager, accountManager *marketLib.AccountManager) *TimingMatcher {
	matcher := &TimingMatcher{}
	matcher.submitter = submitter
	matcher.evaluator = evaluator
	matcher.accountManager = accountManager
	matcher.roundOrderCount = matcherOptions.RoundOrdersCount
	matcher.rounds = NewRoundStates(matcherOptions.MaxCacheRoundsLength)

	matcher.markets = []*Market{}
	matcher.duration = big.NewInt(matcherOptions.Duration)
	matcher.delayedNumber = matcherOptions.DelayedNumber

	matcher.lastBlockNumber = big.NewInt(0)
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

func (matcher *TimingMatcher) GetAccountAvailableAmount(address common.Address, tokenAddress common.Address) (*big.Rat, error) {
	if balance, allowance, err := matcher.accountManager.GetBalanceByTokenAddress(address, tokenAddress); nil != err {
		return nil, err
	} else {
		availableAmount := new(big.Rat).SetInt(balance)
		allowanceAmount := new(big.Rat).SetInt(allowance)
		if availableAmount.Cmp(allowanceAmount) > 0 {
			availableAmount = allowanceAmount
		}

		matchedAmountS := matcher.rounds.filledAmountS(address, tokenAddress)
		availableAmount.Sub(availableAmount, matchedAmountS)

		return availableAmount, nil
	}
}
