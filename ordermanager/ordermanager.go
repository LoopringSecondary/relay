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
	"github.com/Loopring/relay/chainclient"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"math/big"
	"sync"
	"time"
)

type OrderManager interface {
	Start()
	Stop()
	MinerOrders(tokenS, tokenB types.Address, filterOrderhashs []types.Hash) []types.OrderState
}

type OrderManagerImpl struct {
	options   config.OrderManagerOptions
	dao       dao.RdsService
	lock      sync.RWMutex
	processor *forkProcessor
	provider  *minerOrdersProvider
}

func NewOrderManager(options config.OrderManagerOptions, dao dao.RdsService) *OrderManagerImpl {
	ob := &OrderManagerImpl{}

	ob.options = options
	ob.dao = dao
	ob.processor = newForkProcess(ob.dao)

	// new miner orders provider
	duration := time.Duration(options.TickerDuration)
	blockPeriod := types.NewBigPtr(big.NewInt(int64(options.BlockPeriod)))
	ob.provider = newMinerOrdersProvider(duration, blockPeriod)

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

	om.provider.start()
}

func (om *OrderManagerImpl) Stop() {
	om.lock.Lock()
	defer om.lock.Unlock()

	om.provider.stop()
}

func (om *OrderManagerImpl) handleFork(input eventemitter.EventData) error {
	om.Stop()

	if err := om.processor.fork(input.(*chainclient.ForkedEvent)); err != nil {
		log.Errorf("order manager handle fork error:%s", err.Error())
	}

	om.Start()

	return nil
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
	event := input.(*types.OrderFilledEvent)

	// set miner order provider current block number
	om.provider.setBlockNumber(event.Blocknumber)

	// save event
	_, err := om.dao.FindFillEventByRinghashAndOrderhash(event.Ringhash, event.OrderHash)
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
	orderhash := event.OrderHash
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
	state.BlockNumber = event.Blocknumber.BigInt()
	state.RemainedAmountS = event.AmountS.BigInt()
	if state.RawOrder.BuyNoMoreThanAmountB == true && event.AmountB.BigInt().Cmp(state.RawOrder.AmountB) > 0 {
		state.RemainedAmountB = state.RawOrder.AmountB
	} else {
		state.RemainedAmountB = event.AmountB.BigInt()
	}
	if event.AmountS.BigInt().Cmp(big.NewInt(0)) < 1 {
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
	event := input.(*types.OrderCancelledEvent)
	orderhash := event.OrderHash

	// set miner order provider current block number
	om.provider.setBlockNumber(event.Blocknumber)

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
		state.RemainedAmountB = new(big.Int).Sub(state.RemainedAmountB, event.AmountCancelled.BigInt())
		if state.RemainedAmountB.Cmp(big.NewInt(0)) < 0 {
			log.Errorf("order:%s cancel amountB:%s error", orderhash.Hex(), event.AmountCancelled.BigInt().String())
			state.RemainedAmountB = big.NewInt(0)
		}
	} else {
		state.RemainedAmountS = new(big.Int).Sub(state.RemainedAmountS, event.AmountCancelled.BigInt())
		if state.RemainedAmountS.Cmp(big.NewInt(0)) < 0 {
			log.Errorf("order:%s cancel amountS:%s error", orderhash.Hex(), event.AmountCancelled.BigInt().String())
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
	event := input.(*types.CutoffEvent)

	// set miner order provider current block number
	om.provider.setBlockNumber(event.Blocknumber)

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

func (om *OrderManagerImpl) MinerOrders(tokenS, tokenB types.Address, filterOrderhashs []types.Hash) []types.OrderState {
	if err := om.provider.markOrders(filterOrderhashs); err != nil {
		log.Debugf("get miner orders error:%s", err.Error())
	}

	return om.provider.getOrders(tokenS, tokenB, filterOrderhashs)
}
