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
	"errors"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/dao"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/types"
	"gx/ipfs/QmSERhEpow33rKAUMJq8yfJVQjLmdABGg899cXg7GcX1Bk/common/model"
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

type OrderManager interface {
	Start()
	Stop()
}

type OrderManagerImpl struct {
	options   config.OrderBookOptions
	dao       dao.RdsService
	lock      sync.RWMutex
	ticker    *time.Ticker
	processor *forkProcessor
}

func NewOrderManager(options config.OrderBookOptions, dao dao.RdsService) *OrderManagerImpl {
	ob := &OrderManagerImpl{}

	// todo: use config
	ob.options = options
	ob.dao = dao
	ob.ticker = time.NewTicker(1 * time.Second)
	ob.processor = newForkProcess(ob.dao)

	return ob
}

// Start start orderbook as a service
func (om *OrderManagerImpl) Start() {
	newOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleGatewayOrder}
	fillOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderFilled}
	cancelOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderCancelled}
	cutoffOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderCutoff}
	forkWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleFork}

	eventemitter.On(eventemitter.OrderManagerGatewayNewOrder, newOrderWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorFill, fillOrderWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorCancel, cancelOrderWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorCutoff, cutoffOrderWatcher)
	eventemitter.On(eventemitter.Fork, forkWatcher)
}

func (om *OrderManagerImpl) Stop() {
	om.lock.Lock()
	defer om.lock.Unlock()

	om.ticker.Stop()
}

// todo expire time

func (om *OrderManagerImpl) handleFork(input eventemitter.EventData) error {
	return om.processor.fork(input.(*chainclient.ForkedEvent))
}

// 来自ipfs的新订单
// 所有来自ipfs的订单都是新订单
func (om *OrderManagerImpl) handleGatewayOrder(input eventemitter.EventData) error {
	om.lock.Lock()
	defer om.lock.Unlock()

	state := input.(*types.OrderState)
	state.Status = types.ORDER_NEW
	model := &dao.Order{}
	model.ConvertDown(state)

	if err := om.dao.Add(model); err != nil {
		return err
	}

	return nil
}

func (om *OrderManagerImpl) handleOrderFilled(input eventemitter.EventData) error {
	event := input.(*chainclient.OrderFilledEvent)

	// save event
	_, err := om.dao.FindFillByRinghashAndOrderhash(types.BytesToHash(event.Ringhash), types.BytesToHash(event.OrderHash))
	if err != nil {
		newFillModel := &dao.Fill{}
		if err := newFillModel.ConvertDown(event); err != nil {
			return err
		}
		om.dao.Add(newFillModel)
	}

	// get dao.Order and types.OrderState
	state := &types.OrderState{}
	orderhash := types.BytesToHash(event.OrderHash)
	model, err := om.dao.GetOrderByHash(orderhash)
	if err != nil {
		return err
	}
	if err := model.ConvertUp(state); err != nil {
		return err
	}

	// validate orderHash
	rawOrderHashHex := state.RawOrder.Hash.Hex()
	evtOrderHashHex := types.BytesToHash(event.OrderHash).Hex()
	if rawOrderHashHex != evtOrderHashHex {
		return errors.New("raw orderhash hex:" + rawOrderHashHex + "not equal event orderhash hex:" + evtOrderHashHex)
	}

	// calculate orderState.remainAmounts
	state.BlockNumber = event.Blocknumber
	state.RemainedAmountS = event.AmountS
	if state.RawOrder.BuyNoMoreThanAmountB == true && event.AmountB.Cmp(state.RawOrder.AmountB) > 0 {
		state.RemainedAmountB = state.RawOrder.AmountB
	} else {
		state.RemainedAmountB = event.AmountB
	}
	if event.AmountS.Cmp(big.NewInt(0)) < 1 {
		state.RemainedAmountS = big.NewInt(0)
	}

	// update order status
	state.SettleStatus()

	// update dao.Order
	if err := model.ConvertDown(state); err != nil {
		return err
	}
	if err := om.dao.Update(state); err != nil {
		return err
	}

	return nil
}

func (om *OrderManagerImpl) handleOrderCancelled(input eventemitter.EventData) error {
	return nil
}

func (om *OrderManagerImpl) handleOrderCutoff(input eventemitter.EventData) error {
	return nil
}
