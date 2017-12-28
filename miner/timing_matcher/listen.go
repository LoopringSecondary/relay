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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
)

func (matcher *TimingMatcher) listenNewBlock() {
	newBlockChan := make(chan *types.BlockEvent)

	go func() {
		for {
			select {
			case blockEvent := <-newBlockChan:
				if nil != blockEvent {
					nextBlockNumber := new(big.Int).Add(matcher.duration, matcher.lastBlockNumber)
					if nextBlockNumber.Cmp(blockEvent.BlockNumber) <= 0 {
						log.Debugf("miner starts a new match round")
						matcher.lastBlockNumber = blockEvent.BlockNumber
						matcher.rounds.appendNewRoundState(matcher.lastBlockNumber)
						var wg sync.WaitGroup
						for _, market := range matcher.markets {
							wg.Add(1)
							go func(m *Market) {
								defer func() {
									wg.Add(-1)
								}()
								m.match()
							}(market)
						}
						wg.Wait()
					}
				}
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			blockEvent := eventData.(*types.BlockEvent)
			newBlockChan <- blockEvent
			return nil
		},
	}
	eventemitter.On(eventemitter.Block_New, watcher)
	matcher.stopFuncs = append(matcher.stopFuncs, func() {
		close(newBlockChan)
		eventemitter.Un(eventemitter.Block_New, watcher)
	})

}

func (matcher *TimingMatcher) listenSubmitEvent() {
	submitEventChan := make(chan common.Hash)
	go func() {
		for {
			select {
			case ringhash := <-submitEventChan:
				log.Debugf("received mined event, this round will be removed, ringhash:%s", ringhash.Hex())
				matcher.rounds.removeMinedRing(ringhash)
			}
		}
	}()

	submitWatcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			minedEvent := eventData.(*types.RingMinedEvent)
			submitEventChan <- minedEvent.Ringhash
			return nil
		},
	}

	submitFailedWatcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			minedEvent := eventData.(*types.RingSubmitFailedEvent)
			submitEventChan <- minedEvent.RingHash
			return nil
		},
	}

	eventemitter.On(eventemitter.OrderManagerExtractorRingMined, submitWatcher)
	eventemitter.On(eventemitter.Miner_RingSubmitFailed, submitFailedWatcher)
	matcher.stopFuncs = append(matcher.stopFuncs, func() {
		close(submitEventChan)
		eventemitter.Un(eventemitter.OrderManagerExtractorRingMined, submitWatcher)
		eventemitter.Un(eventemitter.Miner_RingSubmitFailed, submitWatcher)
	})
}
