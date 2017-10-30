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

package bucket

import (
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
	"sync"
)

/**
暂时不处理以下情况
todo：此时环路的撮合驱动是由新订单的到来进行驱动，但是新订单并不是一直到达的，因此为了不浪费计算量以及增加匹配度，在没有新订单到达时，需要进行下一个长度的匹配
 bucket在解决限定长度的，新订单的快速匹配较好，但是在订单不频繁时，如何挖掘现有的订单进行处理？
 如何进行新的匹配
 首先是需要跨bucket的，进行整合的
 bucket中的更改如何反映到现有的,如何进行semiRing的遍历
 需要一个pool，对bucket进行抽象，由realtime调用pool接口，进行实时计算
 可能需要使用归约订单的结构
*/

/**
思路：设计符合要求的数据格式，
负责协调各个bucket，将ring发送到区块链，
该处负责接受neworder, cancleorder等事件，并把事件广播给所有的bucket，同时调用client将已形成的环路发送至区块链，发送时需要再次查询订单的最新状态，保证无错，一旦出错需要更改ring的各种数据，如交易量、费用分成等
*/

type BucketMatcher struct {
	newRingChan          chan *types.RingState
	orderStateChan       chan *types.OrderState
	ringSubmitFailedChan chan *types.RingState
	buckets              map[types.Address]Bucket
	submitter            *miner.RingSubmitter
	mtx                  *sync.RWMutex
	ringLength           int
	options              config.MinerOptions
}

func NewBucketMatcher(submitter *miner.RingSubmitter, ringLength int) miner.Matcher {
	var matcher miner.Matcher
	bp := &BucketMatcher{}

	bp.newRingChan = make(chan *types.RingState, 100)

	bp.ringSubmitFailedChan = make(chan *types.RingState, 100)

	bp.orderStateChan = make(chan *types.OrderState)

	bp.mtx = &sync.RWMutex{}

	bp.buckets = make(map[types.Address]Bucket)
	bp.ringLength = ringLength
	bp.submitter = submitter
	matcher = bp
	return matcher
}

func (bp *BucketMatcher) Start() {

	bp.listenOrderState()

	//go bp.listenRingSubmit()

	bp.listenNewRing()

}

func (bp *BucketMatcher) Stop() {
	close(bp.newRingChan)
	close(bp.orderStateChan)
	close(bp.ringSubmitFailedChan)
	for _, bucket := range bp.buckets {
		bucket.stop()
	}
}

func (bp *BucketMatcher) listenNewRing() {
	go func() {
		for {
			select {
			case ringState := <-bp.newRingChan:
				if err := bp.submitter.NewRing(ringState); nil != err {
					log.Errorf("err:%s", err.Error())
				} else {
					//this should call deleteOrder if the order was fullfilled, and do nothing else.
					for _, order := range ringState.RawRing.Orders {
						//不应该调用orderbook而是加上当前已经被匹配过后的金额
						if order.IsFullFilled() {
							bp.deleteOrder(&order.OrderState)
						}
					}
				}
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(e eventemitter.EventData) error {
			ringState := e.(*types.RingState)
			bp.newRingChan <- ringState
			return nil
		},
	}
	eventemitter.On(eventemitter.Miner_NewRing, watcher)
}

func (bp *BucketMatcher) newOrder(orderState *types.OrderState) {
	bp.mtx.RLock()
	defer bp.mtx.RUnlock()

	miner.MinerInstance.Loopring.AddToken(orderState.RawOrder.TokenS)
	miner.MinerInstance.Loopring.AddToken(orderState.RawOrder.TokenB)

	//if bp.buckets doesn't contains the bucket named by tokenS, create it.
	if _, ok := bp.buckets[orderState.RawOrder.TokenS]; !ok {
		bucket := NewBucketAndStart(orderState.RawOrder.TokenS, bp.ringLength)
		bp.buckets[orderState.RawOrder.TokenS] = *bucket
	}

	//it is unnecessary actually
	if _, ok := bp.buckets[orderState.RawOrder.TokenB]; !ok {
		bucket := NewBucketAndStart(orderState.RawOrder.TokenB, bp.ringLength)
		bp.buckets[orderState.RawOrder.TokenB] = *bucket
	}

	eventemitter.Emit(eventemitter.Miner_NewOrderState, orderState)
}

func (bp *BucketMatcher) deleteOrder(orderState *types.OrderState) {
	eventemitter.Emit(eventemitter.Miner_DeleteOrderState, orderState)
}

func (bp *BucketMatcher) listenRingSubmit() {
	go func() {
		for {
			select {
			case ringState, isClose := <-bp.ringSubmitFailedChan:
				if !isClose {
					break
				}
				for _, order := range ringState.RawRing.Orders {
					//todo:查询orderbook获取最新值, 是否已被匹配过
					if true {
						bp.orderStateChan <- &order.OrderState
					}
				}
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(e eventemitter.EventData) error {
			submitFailed := e.(*miner.RingSubmitFailed)
			bp.ringSubmitFailedChan <- submitFailed.RingState
			return nil
		},
	}
	eventemitter.On(eventemitter.Miner_NewRing, watcher)

}

func (bp *BucketMatcher) listenOrderState() {

	go func() {
		for {
			select {
			case orderState, isClose := <-bp.orderStateChan:

				if !isClose {
					log.Debugf("bp.orderStateChan closed")
					break
				}
				vd, _ := orderState.LatestVersion()
				if types.ORDER_NEW == vd.Status {
					bp.newOrder(orderState)
				} else if types.ORDER_CANCEL == vd.Status || types.ORDER_FINISHED == vd.Status {
					//todo:process the case of cancel partable
					bp.deleteOrder(orderState)
				}

			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(e eventemitter.EventData) error {
			orderState := e.(*types.OrderState)
			bp.orderStateChan <- orderState
			return nil
		},
	}
	eventemitter.On(eventemitter.MinedOrderState, watcher)

}
