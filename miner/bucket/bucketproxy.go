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
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
	"sync"
	"github.com/Loopring/ringminer/eventemiter"
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

type BucketProxy struct {
	ringChan             chan *types.RingState
	orderStateChan       chan *types.OrderState
	ringSubmitFailedChan chan *types.RingState
	buckets              map[types.Address]Bucket
	submitClient         *miner.RingSubmitClient
	mtx                  *sync.RWMutex
	options              config.MinerOptions
}

func NewBucketProxy(submitClient *miner.RingSubmitClient) miner.Proxy {
	var proxy miner.Proxy
	bp := &BucketProxy{}

	bp.ringChan = make(chan *types.RingState, 100)

	bp.ringSubmitFailedChan = make(chan *types.RingState, 100)

	bp.orderStateChan = make(chan *types.OrderState, 100)

	bp.mtx = &sync.RWMutex{}

	bp.buckets = make(map[types.Address]Bucket)
	bp.submitClient = submitClient
	proxy = bp
	return proxy
}

func (bp *BucketProxy) Start() {
	bp.submitClient.Start()

	//miner.RateProvider.Start()

	go bp.listenOrderState()

	//go bp.listenRingSubmit()

	go func() {
		for {
			select {
			case orderRing := <-bp.ringChan:
				if err := bp.submitClient.NewRing(orderRing); nil != err {
					log.Errorf("err:%s", err.Error())
				} else {
					//this should call deleteOrder if the order was fullfilled, and do nothing else.
					for _, order := range orderRing.RawRing.Orders {
						//不应该调用orderbook而是加上当前已经被匹配过后的金额
						if order.IsFullFilled() {
							bp.deleteOrder(&order.OrderState)
						}
					}
				}
			}
		}
	}()

}

func (bp *BucketProxy) Stop() {
	close(bp.ringChan)
	close(bp.orderStateChan)
	close(bp.ringSubmitFailedChan)
	for _, bucket := range bp.buckets {
		bucket.Stop()
	}
}

func (bp *BucketProxy) newOrder(order *types.OrderState) {
	bp.mtx.RLock()
	defer bp.mtx.RUnlock()
	//if bp.buckets doesn't contains the bucket named by tokenS, create it.
	if _, ok := bp.buckets[order.RawOrder.TokenS]; !ok {
		bucket := NewBucket(order.RawOrder.TokenS, bp.ringChan)
		bp.buckets[order.RawOrder.TokenS] = *bucket
	}

	//it is unnecessary actually
	if _, ok := bp.buckets[order.RawOrder.TokenB]; !ok {
		bucket := NewBucket(order.RawOrder.TokenB, bp.ringChan)
		bp.buckets[order.RawOrder.TokenB] = *bucket
	}

	for _, b := range bp.buckets {
		b.NewOrder(*order)
	}
}

func (bp *BucketProxy) deleteOrder(order *types.OrderState) {
	for _, bucket := range bp.buckets {
		bucket.deleteOrder(*order)
		log.Debugf("tokenS:%s, order len:%d, semiRing len:%d", bucket.token.Hex(), len(bucket.orders), len(bucket.semiRings))
	}
}

func (bp *BucketProxy) AddFilter() {

}

func (bp *BucketProxy) listenRingSubmit() {
	watcher := &eventemitter.Watcher{
		Concurrent:false,
		Handle:func (e eventemitter.EventData) error {
			submitFailed := e.(*miner.RingSubmitFailed)
			bp.ringSubmitFailedChan <- submitFailed.RingState
			return nil
		},
	}
	eventemitter.On(eventemitter.RingSubmitFailed, watcher)

	for {
		select {
		case ringState,isClose := <-bp.ringSubmitFailedChan:
			if isClose {
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
}

func (bp *BucketProxy) listenOrderState() {
	watcher := &eventemitter.Watcher{
		Concurrent:false,
		Handle:func (e eventemitter.EventData) error {
			orderState := e.(*types.OrderState)
			bp.orderStateChan <- orderState
			return nil
		},
	}
	//todo:topic
	eventemitter.On(eventemitter.MinedOrderState, watcher)

	for {
		select {
		case orderState := <-bp.orderStateChan:
			vd, _ := orderState.LatestVersion()
			if types.ORDER_NEW == vd.Status {
				miner.LoopringInstance.AddToken(orderState.RawOrder.TokenS)
				miner.LoopringInstance.AddToken(orderState.RawOrder.TokenB)
				bp.newOrder(orderState)
			} else if types.ORDER_CANCEL == vd.Status || types.ORDER_FINISHED == vd.Status {
				//todo:process the case of cancel partable
				bp.deleteOrder(orderState)
			}
		}
	}
}
