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
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	marketLib "github.com/Loopring/relay/market"
	marketUtilLib "github.com/Loopring/relay/market/util"
	"strings"
)

/**
定时从ordermanager中拉取n条order数据进行匹配成环，如果成环则通过调用evaluator进行费用估计，然后提交到submitter进行提交到以太坊
*/

type TimingMatcher struct {
	//rounds          *RoundStates
	markets         []*Market
	submitter       *miner.RingSubmitter
	evaluator       *miner.Evaluator
	lastRoundNumber *big.Int
	duration        *big.Int
	lagBlocks       int64
	roundOrderCount int
	reservedTime    int64
	maxFailedCount  int64

	maxCacheRoundsLength int
	delayedNumber        int64
	accountManager       *marketLib.AccountManager
	isOrdersReady        bool
	db                   dao.RdsService

	stopFuncs []func()
}

func NewTimingMatcher(matcherOptions *config.TimingMatcher, submitter *miner.RingSubmitter, evaluator *miner.Evaluator, om ordermanager.OrderManager, accountManager *marketLib.AccountManager, rds dao.RdsService) *TimingMatcher {
	matcher := &TimingMatcher{}
	matcher.submitter = submitter
	matcher.evaluator = evaluator
	matcher.accountManager = accountManager
	matcher.roundOrderCount = matcherOptions.RoundOrdersCount
	//matcher.rounds = NewRoundStates(matcherOptions.MaxCacheRoundsLength)
	matcher.isOrdersReady = false
	matcher.db = rds
	matcher.lagBlocks = matcherOptions.LagForCleanSubmitCacheBlocks
	if matcherOptions.ReservedSubmitTime > 0 {
		matcher.reservedTime = matcherOptions.ReservedSubmitTime
	} else {
		matcherOptions.ReservedSubmitTime = 45
	}
	if matcherOptions.MaxSumitFailedCount > 0 {
		matcher.maxFailedCount = matcherOptions.MaxSumitFailedCount
	} else {
		matcher.maxFailedCount = 3
	}

	matcher.markets = []*Market{}
	matcher.duration = big.NewInt(matcherOptions.Duration)
	matcher.delayedNumber = matcherOptions.DelayedNumber

	matcher.lastRoundNumber = big.NewInt(0)
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
			for _, protocolAddress := range ethaccessor.ProtocolAddresses() {
				m := &Market{}
				m.protocolImpl = protocolAddress
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

func (matcher *TimingMatcher) cleanMissedCache() {
	//如果程序不正确的停止，清除错误的缓存数据
	if ringhashes, err := CachedRinghashes(); nil == err {
		for _, ringhash := range ringhashes {

			if submitInfo, err1 := matcher.db.GetRingForSubmitByHash(ringhash); nil == err1 {
				if submitInfo.ID <= 0 {
					RemoveMinedRingAndReturnOrderhashes(ringhash)
					//cache.Del(RingHashPrefix + strings.ToLower(ringhash.Hex()))
				}
			} else {
				if strings.Contains(err1.Error(), "record not found") {
					RemoveMinedRingAndReturnOrderhashes(ringhash)
				}
				log.Errorf("err:%s", err1.Error())
			}
		}
	} else {
		log.Errorf("err:%s", err.Error())
	}
}

func (matcher *TimingMatcher) Start() {
	matcher.listenSubmitEvent()
	matcher.listenOrderReady()
	matcher.listenTimingRound()
	matcher.cleanMissedCache()

	//syncWatcher := &eventemitter.Watcher{Concurrent: false, Handle: func(eventData eventemitter.EventData) error {
	//	log.Debugf("TimingMatcher Start......")
	//	matcher.listenTimingRound()
	//	return nil
	//}}
	//eventemitter.On(eventemitter.SyncChainComplete, syncWatcher)
	//matcher.stopFuncs = append(matcher.stopFuncs, func() {
	//	eventemitter.Un(eventemitter.SyncChainComplete, syncWatcher)
	//})
}

func (matcher *TimingMatcher) Stop() {
	for _, stop := range matcher.stopFuncs {
		stop()
	}
}

func (matcher *TimingMatcher) GetAccountAvailableAmount(address, tokenAddress, spender common.Address) (*big.Rat, error) {
	//log.Debugf("address: %s , token: %s , spender: %s", address.Hex(), tokenAddress.Hex(), spender.Hex())
	if balance, allowance, err := matcher.accountManager.GetBalanceAndAllowance(address, tokenAddress, spender); nil != err {
		return nil, err
	} else {
		availableAmount := new(big.Rat).SetInt(balance)
		allowanceAmount := new(big.Rat).SetInt(allowance)
		if availableAmount.Cmp(allowanceAmount) > 0 {
			availableAmount = allowanceAmount
		}
		matchedAmountS, _ := FilledAmountS(address, tokenAddress)
		log.Debugf("owner:%s, token:%s, spender:%s, availableAmount:%s, balance:%s, allowance:%s, matchedAmountS:%s", address.Hex(), tokenAddress.Hex(), spender.Hex(), availableAmount.FloatString(2), balance.String(), allowance.String(), matchedAmountS.FloatString(2))

		availableAmount.Sub(availableAmount, matchedAmountS)

		return availableAmount, nil
	}
}
