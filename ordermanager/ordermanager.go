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
	eventemitter.On(eventemitter.OrderManagerFork, forkWatcher)
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
	_, err := om.dao.FindFillEventByRinghashAndOrderhash(types.BytesToHash(event.Ringhash), types.BytesToHash(event.OrderHash))
	if err != nil {
		newFillModel := &dao.FillEvent{}
		if err := newFillModel.ConvertDown(event); err != nil {
			return err
		}
		if err := om.dao.Add(newFillModel); err != nil {
			return err
		}
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

	// judge status
	state.SettleStatus()
	if state.Status == types.ORDER_CUTOFF || state.Status == types.ORDER_FINISHED || state.Status == types.ORDER_UNKNOWN {
		return errors.New("order manager handle order filled event error:order status is " + state.Status.Name())
	}

	// validate cutoff
	if cutoffModel, err := om.dao.FindCutoffEventByOwnerAddress(state.RawOrder.TokenS); err == nil {
		if beenCutoff := om.dao.CheckOrderCutoff(orderhash.Hex(), cutoffModel.Cutoff); beenCutoff {
			if err := om.dao.SettleOrdersStatus([]string{orderhash.Hex()}, types.ORDER_CUTOFF); err != nil {
				return err
			} else {
				return errors.New("order manager handle order filled event error:order have been cutoff")
			}
		}
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
	event := input.(*chainclient.OrderCancelledEvent)
	orderhash := types.BytesToHash(event.OrderHash)

	// save event
	_, err := om.dao.FindCancelEventByOrderhash(orderhash)
	if err != nil {
		newCancelEventModel := &dao.CancelEvent{}
		if err := newCancelEventModel.ConvertDown(event); err != nil {
			return err
		}
		if err := om.dao.Add(newCancelEventModel); err != nil {
			return err
		}
	}

	// get dao.Order and types.OrderState
	state := &types.OrderState{}
	model, err := om.dao.GetOrderByHash(orderhash)
	if err != nil {
		return err
	}
	if err := model.ConvertUp(state); err != nil {
		return err
	}

	// judge status
	state.SettleStatus()
	if state.Status == types.ORDER_CUTOFF || state.Status == types.ORDER_FINISHED || state.Status == types.ORDER_UNKNOWN {
		return errors.New("order manager handle order filled event error:order status is " + state.Status.Name())
	}

	// calculate remainAmount
	if state.RawOrder.BuyNoMoreThanAmountB {
		state.RemainedAmountB = new(big.Int).Sub(state.RemainedAmountB, event.AmountCancelled)
		if state.RemainedAmountB.Cmp(big.NewInt(0)) < 0 {
			log.Errorf("order:%s cancel amountB:%s error", orderhash.Hex(), event.AmountCancelled.String())
			state.RemainedAmountB = big.NewInt(0)
		}
	} else {
		state.RemainedAmountS = new(big.Int).Sub(state.RemainedAmountS, event.AmountCancelled)
		if state.RemainedAmountS.Cmp(big.NewInt(0)) < 0 {
			log.Errorf("order:%s cancel amountS:%s error", orderhash.Hex(), event.AmountCancelled.String())
			state.RemainedAmountS = big.NewInt(0)
		}
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

func (om *OrderManagerImpl) handleOrderCutoff(input eventemitter.EventData) error {
	event := input.(*chainclient.CutoffTimestampChangedEvent)

	// save event
	model, err := om.dao.FindCutoffEventByOwnerAddress(event.Owner)
	if err != nil {
		model = &dao.CutOffEvent{}
		if err := model.ConvertDown(event); err != nil {
			return err
		}
		if err := om.dao.Add(model); err != nil {
			return err
		}
	} else {
		if err := model.ConvertDown(event); err != nil {
			return err
		}
		if err := om.dao.Update(model); err != nil {
			return err
		}
	}

	// get order list
	list, err := om.dao.GetCutoffOrders(model.Cutoff)
	if err != nil {
		return err
	}

	// update each order
	var orderhashs []string
	for _, order := range list {
		orderhashs = append(orderhashs, order.OrderHash)
	}
	if err := om.dao.SettleOrdersStatus(orderhashs, types.ORDER_CUTOFF); err != nil {
		return err
	}

	return nil
}
