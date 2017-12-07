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
	"math/big"
	"sync"
)

type OrderManager interface {
	Start()
	Stop()
	MinerOrders(protocol, tokenS, tokenB common.Address, length int, filterOrderhashs []common.Hash) []types.OrderState
	GetOrderBook(protocol, tokenS, tokenB common.Address, length int) ([]types.OrderState, error)
	GetOrders(query map[string]interface{}, pageIndex, pageSize int) (dao.PageResult, error)
	GetOrderByHash(hash common.Hash) (*types.OrderState, error)
	UpdateBroadcastTimeByHash(hash common.Hash, bt int) error
	FillsPageQuery(query map[string]interface{}, pageIndex, pageSize int) (dao.PageResult, error)
	RingMinedPageQuery(query map[string]interface{}, pageIndex, pageSize int) (dao.PageResult, error)
	IsOrderCutoff(owner common.Address, createTime *big.Int) bool
	IsOrderFullFinished(state *types.OrderState) bool
}

type OrderManagerImpl struct {
	options            config.OrderManagerOptions
	commonOpts         *config.CommonOptions
	rds                dao.RdsService
	lock               sync.RWMutex
	processor          *forkProcessor
	um                 usermanager.UserManager
	mc                 *marketcap.MarketCapProvider
	cutoffCache        *CutoffCache
	newOrderWatcher    *eventemitter.Watcher
	ringMinedWatcher   *eventemitter.Watcher
	fillOrderWatcher   *eventemitter.Watcher
	cancelOrderWatcher *eventemitter.Watcher
	cutoffOrderWatcher *eventemitter.Watcher
	forkWatcher        *eventemitter.Watcher
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
	om.cutoffCache = NewCutoffCache(rds)

	return om
}

// Start start orderbook as a service
func (om *OrderManagerImpl) Start() {
	om.newOrderWatcher = &eventemitter.Watcher{Concurrent: false, Handle: om.handleGatewayOrder}
	om.ringMinedWatcher = &eventemitter.Watcher{Concurrent: false, Handle: om.handleRingMined}
	om.fillOrderWatcher = &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderFilled}
	om.cancelOrderWatcher = &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderCancelled}
	om.cutoffOrderWatcher = &eventemitter.Watcher{Concurrent: false, Handle: om.handleOrderCutoff}
	om.forkWatcher = &eventemitter.Watcher{Concurrent: false, Handle: om.handleFork}

	eventemitter.On(eventemitter.OrderManagerGatewayNewOrder, om.newOrderWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorRingMined, om.ringMinedWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorFill, om.fillOrderWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorCancel, om.cancelOrderWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorCutoff, om.cutoffOrderWatcher)
	eventemitter.On(eventemitter.OrderManagerFork, om.forkWatcher)
}

func (om *OrderManagerImpl) Stop() {
	om.lock.Lock()
	defer om.lock.Unlock()

	eventemitter.Un(eventemitter.OrderManagerGatewayNewOrder, om.newOrderWatcher)
	eventemitter.Un(eventemitter.OrderManagerExtractorRingMined, om.ringMinedWatcher)
	eventemitter.Un(eventemitter.OrderManagerExtractorFill, om.fillOrderWatcher)
	eventemitter.Un(eventemitter.OrderManagerExtractorCancel, om.cancelOrderWatcher)
	eventemitter.Un(eventemitter.OrderManagerExtractorCutoff, om.cutoffOrderWatcher)
	eventemitter.Un(eventemitter.OrderManagerFork, om.forkWatcher)
}

func (om *OrderManagerImpl) handleFork(input eventemitter.EventData) error {
	om.Stop()

	if err := om.processor.fork(input.(*types.ForkedEvent)); err != nil {
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
	state.CancelledAmountS = big.NewInt(0)
	state.CancelledAmountB = big.NewInt(0)

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

	model := &dao.RingMinedEvent{}
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
	finished := om.IsOrderFullFinished(state)
	state.SettleFinishedStatus(finished)

	// update rds.Order
	if err := model.ConvertDown(state); err != nil {
		log.Errorf(err.Error())
		return err
	}
	// status,dealtAmountS,dealtAmountB,blockNumber,
	if err := om.rds.UpdateOrderWhileFill(state.RawOrder.Hash, state.Status, state.DealtAmountS, state.DealtAmountB, state.BlockNumber); err != nil {
		log.Errorf(err.Error())
		return err
	}

	return nil
}

func (om *OrderManagerImpl) handleOrderCancelled(input eventemitter.EventData) error {
	event := input.(*types.OrderCancelledEvent)

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
	finished := om.IsOrderFullFinished(state)
	state.SettleFinishedStatus(finished)

	// update rds.Order
	if err := model.ConvertDown(state); err != nil {
		return err
	}
	if err := om.rds.UpdateOrderWhileCancel(state.RawOrder.Hash, state.Status, state.CancelledAmountS, state.CancelledAmountB, state.BlockNumber); err != nil {
		return err
	}

	return nil
}

func (om *OrderManagerImpl) handleOrderCutoff(input eventemitter.EventData) error {
	event := input.(*types.CutoffEvent)

	if err := om.cutoffCache.Add(event); err != nil {
		return err
	}
	if err := om.rds.SettleOrdersCutoffStatus(event.Owner, event.Cutoff); err != nil {
		return err
	}

	return nil
}

func (om *OrderManagerImpl) IsOrderFullFinished(state *types.OrderState) bool {
	var valueOfRemainAmount *big.Rat

	if state.RawOrder.BuyNoMoreThanAmountB {
		dealtAndCancelledAmountB := new(big.Int).Add(state.DealtAmountB, state.CancelledAmountB)
		remainAmountB := new(big.Int).Sub(state.RawOrder.AmountB, dealtAndCancelledAmountB)
		ratRemainAmountB := new(big.Rat).SetInt(remainAmountB)
		price := om.mc.GetMarketCap(state.RawOrder.TokenB)
		valueOfRemainAmount = new(big.Rat).Mul(price, ratRemainAmountB)
	} else {
		dealtAndCancelledAmountS := new(big.Int).Add(state.DealtAmountS, state.CancelledAmountS)
		remainAmountS := new(big.Int).Sub(state.RawOrder.AmountS, dealtAndCancelledAmountS)
		ratRemainAmountS := new(big.Rat).SetInt(remainAmountS)
		price := om.mc.GetMarketCap(state.RawOrder.TokenS)
		valueOfRemainAmount = new(big.Rat).Mul(price, ratRemainAmountS)
	}

	// todo: get compare number from config
	if valueOfRemainAmount.Cmp(big.NewRat(1, 1)) > 0 {
		return false
	}

	return true
}

func (om *OrderManagerImpl) MinerOrders(protocol, tokenS, tokenB common.Address, length int, filterOrderhashs []common.Hash) []types.OrderState {
	var (
		list            []types.OrderState
		modelList       []*dao.Order
		currentBlock    *dao.Block
		markBlockNumber *big.Int
		err             error
		orderhashstrs   []string
		filterStatus    = []types.OrderStatus{types.ORDER_FINISHED, types.ORDER_CUTOFF, types.ORDER_CANCEL}
	)

	for _, v := range filterOrderhashs {
		orderhashstrs = append(orderhashstrs, v.Hex())
	}

	// 从数据库中获取最近处理的block，数据库为空表示程序从未运行过，这个时候所有的order.markBlockNumber都为0
	if currentBlock, err = om.rds.FindLatestBlock(); err == nil {
		var b types.Block
		currentBlock.ConvertUp(&b)
		markBlockNumber = b.BlockNumber
	} else {
		markBlockNumber = big.NewInt(0)
	}

	if err = om.rds.MarkMinerOrders(orderhashstrs, markBlockNumber.Int64()); err != nil {
		log.Debugf("order manager,provide orders for miner error:%s", err.Error())
	}

	markBlockNumber = new(big.Int).Sub(markBlockNumber, big.NewInt(int64(om.options.BlockPeriod)))
	if modelList, err = om.rds.GetOrdersForMiner(protocol.Hex(), tokenS.Hex(), tokenB.Hex(), length, filterStatus, markBlockNumber.Int64()); err != nil {
		return list
	}

	for _, v := range modelList {
		var state types.OrderState
		v.ConvertUp(&state)
		if !om.um.InWhiteList(state.RawOrder.TokenS) {
			list = append(list, state)
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

func (om *OrderManagerImpl) IsOrderCutoff(owner common.Address, createTime *big.Int) bool {
	return om.cutoffCache.IsOrderCutoff(owner, createTime)
}
