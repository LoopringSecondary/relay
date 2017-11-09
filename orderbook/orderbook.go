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

package orderbook

import (
	"errors"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"math/big"
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

type OrderBook struct {
	options     config.OrderBookOptions
	commOpts    config.CommonOptions
	filters     []Filter
	rdbs        *Rdbs
	ordTimeList *OrderTimestampList
	cutoffcache *CutoffIndexCache
	lock        sync.RWMutex
	minAmount   *big.Int
	ticker      *time.Ticker
}

func NewOrderBook(options config.OrderBookOptions, commOpts config.CommonOptions, database db.Database) *OrderBook {
	ob := &OrderBook{}

	ob.options = options
	ob.commOpts = commOpts
	ob.rdbs = NewRdbs(database)

	// todo: use config
	ob.ticker = time.NewTicker(1 * time.Second)
	ob.ordTimeList = &OrderTimestampList{}
	ob.cutoffcache = NewCutoffIndexCache(database)

	return ob
}

func (ob *OrderBook) filter(o *types.Order) (bool, error) {
	valid := true
	var err error
	for _, filter := range ob.filters {
		valid, err = filter.filter(o)
		if !valid {
			return valid, err
		}
	}
	return valid, nil
}

// Start start orderbook as a service
func (ob *OrderBook) Start() {
	ob.recover()

	peerOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: ob.handlePeerOrder}
	chainOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: ob.handleChainOrder}

	eventemitter.On(eventemitter.OrderBookPeer, peerOrderWatcher)
	eventemitter.On(eventemitter.OrderBookChain, chainOrderWatcher)

	go func() {
		for {
			select {
			case <-ob.ticker.C:
				ob.sendOrderToMiner()
			}
		}
	}()
}

func (ob *OrderBook) recover() {
	for {
		state := <-ob.rdbs.orderChan
		ob.beforeSendOrderToMiner(state)
	}
}

func (ob *OrderBook) Stop() {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	// todo
	ob.rdbs.Close()
	ob.ticker.Stop()
}

// 来自ipfs的新订单
// 所有来自ipfs的订单都是新订单
func (ob *OrderBook) handlePeerOrder(input eventemitter.EventData) error {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	ord := input.(*types.Order)

	if valid, err := ob.filter(ord); !valid {
		return err
	}

	orderhash := ord.GenerateHash()
	state := &types.OrderState{}
	state.RawOrder = *ord
	state.RawOrder.Hash = orderhash
	state.RawOrder.GeneratePrice()

	log.Debugf("orderbook accept new order hash:%s", orderhash.Hex())
	log.Debugf("orderbook accept new order amountS:%s", ord.AmountS.String())
	log.Debugf("orderbook accept new order amountB:%s", ord.AmountB.String())

	// 之前从未存储过
	if _, err := ob.rdbs.GetOrder(orderhash); err == nil {
		return errors.New("order " + orderhash.Hex() + " already exist")
	}

	state.AddVersion(types.VersionData{})
	ob.beforeSendOrderToMiner(state)

	return nil
}

// 处理来自eth网络的evt/transaction转换后的orderState
// 订单必须存在，如果不存在则不处理
// 如果之前没有存储，那么应该等到eth网络监听到transaction并解析成相应的order再处理
// 如果之前已经存储，那么应该直接处理并发送到miner
func (ob *OrderBook) handleChainOrder(input eventemitter.EventData) error {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	ord := input.(*types.OrderState)

	if valid, err := ob.filter(&ord.RawOrder); !valid {
		return err
	}

	vd, err := ord.LatestVersion()
	if err != nil {
		return err
	}

	state := ord

	switch vd.Status {
	case types.ORDER_NEW:
		log.Debugf("orderbook accept new order from chain:%s", state.RawOrder.Hash.Hex())
		ob.beforeSendOrderToMiner(state)

	case types.ORDER_PENDING:
		log.Debugf("orderbook accept pending order from chain:%s", state.RawOrder.Hash.Hex())
		ob.afterSendOrderToMiner(state)

	case types.ORDER_FINISHED:
		log.Debugf("orderbook accept finished order from chain:%s", state.RawOrder.Hash.Hex())
		ob.afterSendOrderToMiner(state)

	case types.ORDER_CANCEL:
		log.Debugf("orderbook accept cancelled order from chain:%s", state.RawOrder.Hash.Hex())
		ob.afterSendOrderToMiner(state)

	case types.ORDER_REJECT:
		log.Debugf("orderbook accept reject order from chain:%s", state.RawOrder.Hash.Hex())
		ob.afterSendOrderToMiner(state)

	default:
		log.Errorf("orderbook version data status error:%s", state.RawOrder.Hash.Hex())
	}

	return nil
}

// beforeSendOrderToMiner push order state index to rdbs sliceOrderIndex
func (ob *OrderBook) beforeSendOrderToMiner(state *types.OrderState) {
	ob.rdbs.SetOrder(state)
	ob.ordTimeList.Push(state.RawOrder.Hash, state.RawOrder.Timestamp)
}

func (ob *OrderBook) sendOrderToMiner() error {
	hash, err := ob.ordTimeList.Pop()
	if err != nil {
		return nil
	}

	state, err := ob.rdbs.GetOrder(hash)
	if err != nil {
		return err
	}

	expiretime := big.NewInt(0).Add(state.RawOrder.Timestamp, state.RawOrder.Ttl)
	nowtime := big.NewInt(time.Now().Unix())
	if nowtime.Cmp(expiretime) > 0 {
		ob.rdbs.MoveOrder(state)
		return errors.New("orderbook order:" + state.RawOrder.Hash.Hex() + " ready to send is expired")
	}

	eventemitter.Emit(eventemitter.MinedOrderState, state)

	return nil
}

func (ob *OrderBook) afterSendOrderToMiner(state *types.OrderState) error {
	return ob.rdbs.MoveOrder(state)
}

// isFinished judge order state
func (ob *OrderBook) isFullFilled(odw *types.OrderState) bool {
	//if odw.RawOrder.
	return true
}

//////////////////////////////////////////////////////////////
//
// 调用内部方法实现
//
//////////////////////////////////////////////////////////////
func (ob *OrderBook) GetOrder(id types.Hash) (*types.OrderState, error) { return ob.rdbs.GetOrder(id) }
func (ob *OrderBook) AddCutoffEvent(address types.Address, timestamp, cutoff, blocknumber *big.Int) error {
	return ob.cutoffcache.Add(address, timestamp, cutoff, blocknumber)
}
