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

package eventemitter

import (
	"github.com/Loopring/relay/log"
	"sync"
)

//todo:more stronger if it has cache, but, the more the nearer to eventsourcing

type Topic string

const (
	// Extractor
	SyncChainComplete = "SyncChainComplete"
	ChainForkDetected = "ChainForkDetected"
	ChainForkProcess  = "ChainForkProcess"
	ExtractorFork     = "ExtractorFork" //chain forked

	// Block
	NewBlock = "NewBlock"

	// gateway
	NewOrder = "NewOrder"

	// loopring
	RingMined           = "RingMined"
	OrderFilledEvent    = "OrderFilledEvent"
	OrderCancelledEvent = "OrderCancelledEvent"
	CutoffAllEvent      = "CutoffAllEvent"
	CutoffPairEvent     = "CutoffPairEvent"
	TokenRegistered     = "TokenRegistered"
	TokenUnRegistered   = "TokenUnRegistered"
	AddressAuthorized   = "AddressAuthorized"
	AddressDeAuthorized = "AddressDeAuthorized"

	// erc20
	ApprovalEvent = "ApprovalEvent"
	TransferEvent = "TransferEvent"

	// weth
	WethDepositEvent    = "WethDepositEvent"
	WethWithdrawalEvent = "WethWithdrawalEvent"

	// Transaction
	MinedTransactionEvent   = "MinedTransactionEvent"
	PendingTransactionEvent = "PendingTransactionEvent"
	EthTransferEvent        = "EthTransferEvent"

	// miner
	Miner_NewRing = "Miner_NewRing"
)

var watchers map[string][]*Watcher
var mtx *sync.Mutex

type EventData interface{}

type Watcher struct {
	Concurrent bool
	Handle     func(eventData EventData) error
}

func Un(topic string, watcher *Watcher) {
	mtx.Lock()
	defer mtx.Unlock()
	watchersTmp := []*Watcher{}
	for _, w := range watchers[topic] {
		if w != watcher {
			watchersTmp = append(watchersTmp, w)
		}
	}
	watchers[topic] = watchersTmp
}

func On(topic string, watcher *Watcher) {
	mtx.Lock()
	defer mtx.Unlock()
	if _, ok := watchers[topic]; !ok {
		watchers[topic] = make([]*Watcher, 0)
	}
	watchers[topic] = append(watchers[topic], watcher)
}

func Emit(topic string, eventData EventData) {
	//should limit the count of watchers
	var wg sync.WaitGroup
	for _, ob := range watchers[topic] {
		if ob.Concurrent {
			go ob.Handle(eventData)
		} else {
			wg.Add(1)
			go func(ob *Watcher) {
				//
				defer func() {
					wg.Add(-1)
				}()
				if err := ob.Handle(eventData); err != nil {
					log.Errorf(err.Error())
				}
			}(ob)
		}
	}
	wg.Wait()
}

//todo: impl it
func NewSerialWatcher(topic string, handle func(e EventData) error) (stopFunc func(), err error) {
	dataChan := make(chan EventData)
	go func() {
		for {
			select {
			case event := <-dataChan:
				handle(event)
			}
		}
	}()

	watcher := &Watcher{
		Concurrent: false,
		Handle: func(eventData EventData) error {
			dataChan <- eventData
			return nil
		},
	}
	On(topic, watcher)

	return func() {
		close(dataChan)
		Un(topic, watcher)
	}, nil
}

func init() {
	watchers = make(map[string][]*Watcher)
	mtx = &sync.Mutex{}
}
