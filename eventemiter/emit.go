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
	NewOrder = "NewOrder"

	// Methods
	WethDeposit      = "WethDepositEvent"
	WethWithdrawal   = "WethWithdrawalEvent"
	Approve          = "ApproveMethod"
	Transfer         = "Transfer"
	EthTransferEvent = "EthTransferEvent"

	RingMined           = "RingMined"
	OrderFilled         = "OrderFilled"
	CancelOrder         = "CancelOrder"
	CutoffAll           = "Cutoff"
	CutoffPair          = "CutoffPair"
	TokenRegistered     = "TokenRegistered"
	TokenUnRegistered   = "TokenUnRegistered"
	RingHashSubmitted   = "RingHashSubmitted"
	AddressAuthorized   = "AddressAuthorized"
	AddressDeAuthorized = "AddressDeAuthorized"

	MinedOrderState            = "MinedOrderState" //orderbook send orderstate to miner
	WalletTransactionSubmitted = "WalletTransactionSubmitted"

	ExtractorFork   = "ExtractorFork" //chain forked
	Transaction     = "Transaction"
	GatewayNewOrder = "GatewayNewOrder"

	//Miner
	Miner_DeleteOrderState           = "Miner_DeleteOrderState"
	Miner_NewOrderState              = "Miner_NewOrderState"
	Miner_NewRing                    = "Miner_NewRing"
	Miner_RingMined                  = "Miner_RingMined"
	Miner_RingSubmitResult           = "Miner_RingSubmitResult"
	Miner_SubmitRing_Method          = "Miner_SubmitRing_Method"
	Miner_SubmitRingHash_Method      = "Miner_SubmitRingHash_Method"
	Miner_BatchSubmitRingHash_Method = "Miner_BatchSubmitRingHash_Method"

	// Block
	Block_New = "Block_New"
	Block_End = "Block_End"

	// Extractor
	SyncChainComplete = "SyncChainComplete"
	ChainForkDetected = "ChainForkDetected"
	ExtractorWarning  = "ExtractorWarning"

	// Transaction
	TransactionEvent   = "TransactionEvent"
	PendingTransaction = "PendingTransaction"

	// socketio notify event types
	LoopringTickerUpdated = "LoopringTickerUpdated"
	TrendUpdated          = "TrendUpdated"
	PortfolioUpdated      = "PortfolioUpdated"
	BalanceUpdated        = "BalanceUpdated"
	DepthUpdated          = "DepthUpdated"
	TransactionUpdated    = "TransactionUpdated"
)

//change map to sync.Map
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
