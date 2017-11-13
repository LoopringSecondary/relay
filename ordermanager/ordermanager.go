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

package ordermanager

import (
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/dao"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"sync"
	"time"
)

/**
todo:
1. filter
2. chain event
3. 事件执行到第几个块等信息数据
4. 订单完成的标志，以及需要发送到miner
*/

type OrderManager interface {
	Start()
	Stop()
}

type OrderManagerImpl struct {
	options config.OrderBookOptions
	dao     dao.RdsService
	lock    sync.RWMutex
	ticker  *time.Ticker
}

func NewOrderManager(options config.OrderBookOptions, dao dao.RdsService) *OrderManagerImpl {
	ob := &OrderManagerImpl{}

	// todo: use config
	ob.options = options
	ob.dao = dao
	ob.ticker = time.NewTicker(1 * time.Second)

	return ob
}

// Start start orderbook as a service
func (ob *OrderManagerImpl) Start() {
	ob.recover()

	peerOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: ob.handleGatewayOrder}
	chainOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: ob.handleExtractorOrder}

	eventemitter.On(eventemitter.OrderBookGateway, peerOrderWatcher)
	eventemitter.On(eventemitter.OrderBookExtractor, chainOrderWatcher)
}

func (ob *OrderManagerImpl) Stop() {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	ob.ticker.Stop()
}

func (ob *OrderManagerImpl) recover() {
	// todo
}

// 来自ipfs的新订单
// 所有来自ipfs的订单都是新订单
func (ob *OrderManagerImpl) handleGatewayOrder(input eventemitter.EventData) error {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	state := input.(*types.OrderState)
	model := &dao.Order{}
	model.ConvertDown(state)

	if err := ob.dao.Add(model); err != nil {
		return err
	}

	return nil
}

// 处理来自eth网络的evt/transaction转换后的orderState
// 订单必须存在，如果不存在则不处理
// 如果之前没有存储，那么应该等到eth网络监听到transaction并解析成相应的order再处理
// 如果之前已经存储，那么应该直接处理并发送到miner
func (ob *OrderManagerImpl) handleExtractorOrder(input eventemitter.EventData) error {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	ord := input.(*types.OrderState)
	vd, err := ord.LatestVersion()
	if err != nil {
		return err
	}

	state := ord

	switch vd.Status {
	case types.ORDER_PENDING:
		log.Debugf("orderbook accept pending order from chain:%s", state.RawOrder.Hash.Hex())
		//ob.afterSendOrderToMiner(state)

	case types.ORDER_FINISHED:
		log.Debugf("orderbook accept finished order from chain:%s", state.RawOrder.Hash.Hex())
		//ob.afterSendOrderToMiner(state)

	case types.ORDER_CANCEL:
		log.Debugf("orderbook accept cancelled order from chain:%s", state.RawOrder.Hash.Hex())
		//ob.afterSendOrderToMiner(state)

	case types.ORDER_REJECT:
		log.Debugf("orderbook accept reject order from chain:%s", state.RawOrder.Hash.Hex())
		//ob.afterSendOrderToMiner(state)

	default:
		log.Errorf("orderbook version data status error:%s", state.RawOrder.Hash.Hex())
	}

	return nil
}
