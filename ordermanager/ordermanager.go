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
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"github.com/Loopring/relay/usermanager"
	"github.com/ethereum/go-ethereum/common"
	"gx/ipfs/QmSERhEpow33rKAUMJq8yfJVQjLmdABGg899cXg7GcX1Bk/common/model"
	"math/big"
	"sync"
)

type OrderManager interface {
	Start()
	Stop()
	MinerOrders(tokenS, tokenB common.Address, length int, filterOrderhashs []common.Hash) []types.OrderState
	GetOrderBook(protocol, tokenS, tokenB common.Address, length int) ([]types.OrderState, error)
	GetOrders(query map[string]interface{}, pageIndex, pageSize int) (dao.PageResult, error)
	GetOrderByHash(hash common.Hash) (*types.OrderState, error)
	UpdateBroadcastTimeByHash(hash common.Hash, bt int) error
	FillsPageQuery(query map[string]interface{}, pageIndex, pageSize int) (dao.PageResult, error)
	RingMinedPageQuery(query map[string]interface{}, pageIndex, pageSize int) (dao.PageResult, error)
}

type OrderManagerImpl struct {
	options    config.OrderManagerOptions
	commonOpts *config.CommonOptions
	rds        dao.RdsService
	lock       sync.RWMutex
	processor  *forkProcessor
	provider   *minerOrdersProvider
	um         usermanager.UserManager
	mc         *marketcap.MarketCapProvider
}

func NewOrderManager(options config.OrderManagerOptions,
	commonOpts *config.CommonOptions,
	rds dao.RdsService,
	userManager usermanager.UserManager,
	accessor *ethaccessor.EthNodeAccessor,
	market *marketcap.MarketCapProvider) *OrderManagerImpl {

	om := &OrderManagerImpl{}
	om.options = options
	om.commonOpts = commonOpts
	om.rds = rds
	om.processor = newForkProcess(om.rds, accessor)
	om.um = userManager
	om.mc = market

	// new miner orders provider
	om.provider = newMinerOrdersProvider(options.TickerDuration, options.BlockPeriod, om.commonOpts, om.rds)

	return om
}

// Start start orderbook as a service
func (om *OrderManagerImpl) Start() {
	newOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleGatewayOrder}
	ringMinedWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleRingMined}
	fillOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderFilled}
	cancelOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderCancelled}
	cutoffOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderCutoff}
	forkWatcher := &eventemitter.Watcher{Concurrent: false, Handle: om.handleFork}

	eventemitter.On(eventemitter.OrderManagerGatewayNewOrder, newOrderWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorRingMined, ringMinedWatcher)
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

	if err := om.processor.fork(input.(*ethaccessor.ForkedEvent)); err != nil {
		log.Errorf("order manager,handle fork error:%s", err.Error())
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
	state.DealtAmountB = big.NewInt(0)
	state.DealtAmountS = big.NewInt(0)
	model := &dao.Order{}

	log.Debugf("order manager,handle gateway order,order.hash:%s amountS:%s", state.RawOrder.Hash.Hex(), state.RawOrder.AmountS.String())

	if err := model.ConvertDown(state); err != nil {
		log.Debugf("order manager,handle gateway order,convert order state to order error:%s", err.Error())
		return err
	}
	if err := om.rds.Add(model); err != nil {
		log.Debugf("order manager,handle gateway order,insert order error:%s", err.Error())
		return err
	}

	return nil
}

func (om *OrderManagerImpl) handleRingMined(input eventemitter.EventData) error {
	event := input.(*types.RingMinedEvent)

	model := &dao.RingMined{}
	if err := model.ConvertDown(event); err != nil {
		return err
	}
	if err := om.rds.Add(model); err != nil {
		log.Debugf("order manager,handle ringmined event,event %s has already exist", event.RingIndex.String())
		return err
	}

	return nil
}

func (om *OrderManagerImpl) handleOrderFilled(input eventemitter.EventData) error {
	event := input.(*types.OrderFilledEvent)

	// set miner order provider current block number
	om.provider.setBlockNumber(event.Blocknumber)

	// save event
	_, err := om.rds.FindFillEventByRinghashAndOrderhash(event.Ringhash, event.OrderHash)
	if err == nil {
		return fmt.Errorf("order manager,handle order filled event,fill already exist ringIndex:%s orderHash:", event.RingIndex.String(), event.OrderHash.Hex())
	}

	newFillModel := &dao.FillEvent{}
	if err := newFillModel.ConvertDown(event); err != nil {
		log.Debugf("order manager,handle order filled event error:order %s convert down failed", event.OrderHash.Hex())
		return err
	}
	if err := om.rds.Add(newFillModel); err != nil {
		log.Debugf("order manager,handle order filled event error:order %s insert faild", event.OrderHash.Hex())
		return err
	}

	// get rds.Order and types.OrderState
	state := &types.OrderState{}
	model, err := om.rds.GetOrderByHash(event.OrderHash)
	if err != nil {
		return err
	}
	if err := model.ConvertUp(state); err != nil {
		return err
	}

	// judge order status
	if state.Status == types.ORDER_CUTOFF || state.Status == types.ORDER_FINISHED || state.Status == types.ORDER_UNKNOWN {
		return fmt.Errorf("order manager,handle order filled event error:order %s status is %d ", state.RawOrder.Hash.Hex(), state.Status)
	}

	// calculate dealt amount
	state.BlockNumber = event.Blocknumber
	state.DealtAmountS = new(big.Int).Add(state.DealtAmountS, event.AmountS)
	state.DealtAmountB = new(big.Int).Add(state.DealtAmountB, event.AmountB)

	// update order status
	om.isOrderFullFinished(state)

	// update rds.Order
	if err := model.ConvertDown(state); err != nil {
		return err
	}
	if err := om.rds.Update(state); err != nil {
		return err
	}

	return nil
}

func (om *OrderManagerImpl) handleOrderCancelled(input eventemitter.EventData) error {
	event := input.(*types.OrderCancelledEvent)

	// set miner order provider current block number
	om.provider.setBlockNumber(event.Blocknumber)

	// save event
	_, err := om.rds.FindCancelEvent(event.OrderHash, event.AmountCancelled)
	if err == nil {
		return fmt.Errorf("order manager,handle order cancelled event error:event %s have already exist", event.OrderHash)
	}
	newCancelEventModel := &dao.CancelEvent{}
	if err := newCancelEventModel.ConvertDown(event); err != nil {
		return err
	}
	if err := om.rds.Add(newCancelEventModel); err != nil {
		return err
	}

	// get rds.Order and types.OrderState
	state := &types.OrderState{}
	model, err := om.rds.GetOrderByHash(event.OrderHash)
	if err != nil {
		return err
	}
	if err := model.ConvertUp(state); err != nil {
		return err
	}

	// judge status
	if state.Status == types.ORDER_CUTOFF || state.Status == types.ORDER_FINISHED || state.Status == types.ORDER_UNKNOWN {
		return fmt.Errorf("order manager,handle order filled event error:order %s status is %d ", event.OrderHash.Hex(), state.Status)
	}

	// calculate remainAmount
	if state.RawOrder.BuyNoMoreThanAmountB {
		state.CancelledAmountB = event.AmountCancelled
	} else {
		state.CancelledAmountS = event.AmountCancelled
	}

	// update order status
	om.isOrderFullFinished(state)

	// update rds.Order
	if err := model.ConvertDown(state); err != nil {
		return err
	}
	if err := om.rds.Update(state); err != nil {
		return err
	}

	return nil
}

func (om *OrderManagerImpl) handleOrderCutoff(input eventemitter.EventData) error {
	event := input.(*types.CutoffEvent)

	// set miner order provider current block number
	om.provider.setBlockNumber(event.Blocknumber)

	// save event
	model, err := om.rds.FindCutoffEventByOwnerAddress(event.Owner)
	if err != nil {
		model = &dao.CutOffEvent{}
		if err := model.ConvertDown(event); err != nil {
			return err
		}
		if err := om.rds.Add(model); err != nil {
			return err
		}
	} else {
		if err := model.ConvertDown(event); err != nil {
			return err
		}
		if err := om.rds.Update(model); err != nil {
			return err
		}
	}

	// get order list
	list, err := om.rds.GetCutoffOrders(model.Cutoff)
	if err != nil {
		return err
	}

	// update each order
	var orderhashs []string
	for _, order := range list {
		orderhashs = append(orderhashs, order.OrderHash)
	}
	if err := om.rds.SettleOrdersStatus(orderhashs, types.ORDER_CUTOFF); err != nil {
		return err
	}

	return nil
}

func (om *OrderManagerImpl) isOrderFullFinished(state *types.OrderState) {
	var valueOfRemainAmount *big.Rat

	if state.RawOrder.BuyNoMoreThanAmountB {
		dealtAndCancelledAmountB := new(big.Int).Add(state.DealtAmountB, state.CancelledAmountB)
		remainAmountB := new(big.Int).Sub(state.RawOrder.AmountB, dealtAndCancelledAmountB)
		price := om.mc.GetMarketCap(state.RawOrder.TokenB)
		ratRemainAmountB := new(big.Rat).SetInt(remainAmountB)
		valueOfRemainAmount = new(big.Rat).Mul(price, ratRemainAmountB)
	} else {
		dealtAndCancelledAmountS := new(big.Int).Add(state.DealtAmountS, state.CancelledAmountS)
		remainAmountS := new(big.Int).Sub(state.RawOrder.AmountS, dealtAndCancelledAmountS)
		price := om.mc.GetMarketCap(state.RawOrder.TokenS)
		ratRemainAmountS := new(big.Rat).SetInt(remainAmountS)
		valueOfRemainAmount = new(big.Rat).Mul(price, ratRemainAmountS)
	}

	// todo: get compare number from config
	if valueOfRemainAmount.Cmp(big.NewRat(10, 1)) <= 0 {
		state.Status = types.ORDER_FINISHED
	} else {
		state.Status = types.ORDER_PARTIAL
	}
}

func (om *OrderManagerImpl) MinerOrders(tokenS, tokenB common.Address, length int, filterOrderhashs []common.Hash) []types.OrderState {
	var list []types.OrderState

	if err := om.provider.markOrders(filterOrderhashs); err != nil {
		log.Debugf("order manager,provide orders for miner error:%s", err.Error())
	}

	filterList := om.provider.getOrders(tokenS, tokenB, length, filterOrderhashs)

	for _, v := range filterList {
		if !om.um.InWhiteList(v.RawOrder.TokenS) {
			list = append(list, v)
		}
	}

	return list
}

func (om *OrderManagerImpl) GetOrderBook(protocol, tokenS, tokenB common.Address, length int) ([]types.OrderState, error) {
	var list []types.OrderState
	models, err := om.rds.GetOrderBook(protocol, tokenS, tokenB, length)
	if err != nil {
		return list, err
	}

	for _, v := range models {
		var state types.OrderState
		if err := v.ConvertUp(&state); err != nil {
			continue
		}
		list = append(list, state)
	}

	return list, nil
}

func (om *OrderManagerImpl) GetOrders(query map[string]interface{}, pageIndex, pageSize int) (dao.PageResult, error) {
	var (
		pageRes dao.PageResult
	)
	tmp, err := om.rds.OrderPageQuery(query, pageIndex, pageSize)

	if err != nil {
		return pageRes, err
	}
	pageRes.PageIndex = tmp.PageIndex
	pageRes.PageSize = tmp.PageSize
	pageRes.Total = tmp.Total

	for _, v := range tmp.Data {
		var state types.OrderState
		model := v.(dao.Order)
		if err := model.ConvertUp(&state); err != nil {
			continue
		}
		pageRes.Data = append(pageRes.Data, state)
	}

	return pageRes, nil
}

func (om *OrderManagerImpl) GetOrderByHash(hash common.Hash) (orderState *types.OrderState, err error) {
	var result types.OrderState
	order, err := om.rds.GetOrderByHash(hash)
	if err != nil {
		return nil, err
	}

	if err := order.ConvertUp(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (om *OrderManagerImpl) UpdateBroadcastTimeByHash(hash common.Hash, bt int) error {
	return om.rds.UpdateBroadcastTimeByHash(hash.Str(), bt)
}

func (om *OrderManagerImpl) FillsPageQuery(query map[string]interface{}, pageIndex, pageSize int) (result dao.PageResult, err error) {
	return om.rds.FillsPageQuery(query, pageIndex, pageSize)
}

func (om *OrderManagerImpl) RingMinedPageQuery(query map[string]interface{}, pageIndex, pageSize int) (result dao.PageResult, err error) {
	return om.rds.RingMinedPageQuery(query, pageIndex, pageSize)
}
