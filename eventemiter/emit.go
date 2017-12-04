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
	"sync"
)

//todo:more stronger if it has cache, but, the more the nearer to eventsourcing

type Topic string

const (
	OrderCanceled                  = "OrderCanceled"
	OrderFilled                    = "OrderFilled"
	ExtractorFork                  = "ExtractorFork" //chain forked
	OrderManagerFork               = "OrderManagerFork"
	RingSubmitFailed               = "RingSubmitFailed" //submit ring failed
	Transaction                    = "Transaction"
	Gateway                        = "Gateway"
	AccountTransfer                = "AccountTransfer"
	AccountApproval                = "AccountApproval"
	TokenRegistered                = "TokenRegistered"
	TokenUnRegistered              = "TokenUnRegistered"
	RingHashSubmitted              = "RingHashSubmitted"
	AddressAuthorized              = "AddressAuthorized"
	AddressDeAuthorized            = "AddressDeAuthorized"
	OrderManagerGatewayNewOrder    = "OrderManagerGatewayNewOrder"
	OrderManagerExtractorRingMined = "OrderManagerRingMined"
	OrderManagerExtractorFill      = "OrderManagerExtractorFill"
	OrderManagerExtractorCancel    = "OrderManagerExtractorCancel"
	OrderManagerExtractorCutoff    = "OrderManagerExtractorCutoff"
	MinedOrderState                = "MinedOrderState" //orderbook send orderstate to miner

	//Miner
	Miner_DeleteOrderState = "Miner_DeleteOrderState"
	Miner_NewOrderState    = "Miner_NewOrderState"
	Miner_NewRing          = "Miner_NewRing"
	Miner_RingMined        = "Miner_RingMined"
	Miner_RingSubmitFailed        = "Miner_RingSubmitFailed"

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
				ob.Handle(eventData)
			}(ob)
		}
	}
	wg.Wait()
}

func init() {
	watchers = make(map[string][]*Watcher)
	mtx = &sync.Mutex{}
}
