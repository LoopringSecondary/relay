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
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"math/big"
	"sort"
)

type ForkProcessor struct {
	db dao.RdsService
	mc marketcap.MarketCapProvider
}

func NewForkProcess(rds dao.RdsService, mc marketcap.MarketCapProvider) *ForkProcessor {
	processor := &ForkProcessor{}
	processor.db = rds
	processor.mc = mc

	return processor
}

//	fork process chain fork logic in order manager
// 1.从各个事件表中获取所有处于分叉块中的事件(fill,cancel,cutoff,cutoffPair)并按照blockNumber以及logIndex倒序
// 2.遍历event,处理各个类型event对应的回滚逻辑:
//   a.处理fill,不需关心订单当前状态,减去相应fill量,然后判定订单status为new/partial/finished
//   b.处理cancel,在合约里,订单是可以被持续cancel的,ordermanager跟随合约逻辑,即便订单已经处于finished/cutoff状态,cancel的量也会递增
//     那么,在回滚时,我们可以不关心订单状态(前提是订单只有finished状态,没有cancelled状态,如果前端展示需要cancelled状态,必须根据cancel的量进行计算)
//   c.处理cutoff,合约里cutoff可以重复提交,而在ordermanager中,所有cutoff事件都会被存储,但是更新订单时,同一个订单不会被多次cutoff
//     那么,在回滚时,我们需要知道某一个订单以前是否也cutoff过,在dao/cutoff中我们存储了orderhashList,可以将这些订单取出并按照订单量重置状态
//   d.处理cutoffPair,同cutoff
func (p *ForkProcessor) Fork(event *types.ForkedEvent) error {
	from := event.ForkBlock.Int64()
	to := event.DetectedBlock.Int64()

	list, _ := p.GetForkEvents(from, to)
	if list.Len() == 0 {
		log.Debugf("order manager fork:non fork events")
		return nil
	}

	sort.Sort(list)

	var err error
	for _, v := range list {
		switch v.Type {
		case FORK_EVT_TYPE_FILL:
			err = p.RollBackSingleFill(v.Event.(*types.OrderFilledEvent))
		case FORK_EVT_TYPE_CANCEL:
			err = p.RollBackSingleCancel(v.Event.(*types.OrderCancelledEvent))
		case FORK_EVT_TYPE_CUTOFF:
			err = p.RollBackSingleCutoff(v.Event.(*types.CutoffEvent))
		case FORK_EVT_TYPE_CUTOFF_PAIR:
			err = p.RollBackSingleCutoffPair(v.Event.(*types.CutoffPairEvent))
		}
		if err != nil {
			return err
		}
	}

	return p.MarkForkEvents(from, to)
}

// calculate order's related values and status, update order
func (p *ForkProcessor) RollBackSingleFill(evt *types.OrderFilledEvent) error {
	state := &types.OrderState{}
	model, err := p.db.GetOrderByHash(evt.OrderHash)
	if err != nil {
		log.Debugf("fork fill event,order:%s not exist in dao/fill", evt.OrderHash.Hex())
		return nil
	}
	model.ConvertUp(state)

	// calculate dealt amount
	state.UpdatedBlock = evt.BlockNumber
	state.DealtAmountS = safeSub(state.DealtAmountS, evt.AmountS)
	state.DealtAmountB = safeSub(state.DealtAmountB, evt.AmountB)
	state.SplitAmountS = safeSub(state.SplitAmountS, evt.SplitS)
	state.SplitAmountB = safeSub(state.SplitAmountB, evt.SplitB)

	log.Debugf("fork fill event, orderhash:%s,dealAmountS:%s,dealtAmountB:%s", state.RawOrder.Hash.Hex(), state.DealtAmountS.String(), state.DealtAmountB.String())

	// update order status
	settleOrderStatus(state, p.mc, ORDER_FROM_FILL)

	// update rds.Order
	model.ConvertDown(state)
	if err := p.db.UpdateOrderWhileFill(state.RawOrder.Hash, state.Status, state.DealtAmountS, state.DealtAmountB, state.SplitAmountS, state.SplitAmountB, state.UpdatedBlock); err != nil {
		return err
	}

	return nil
}

func (p *ForkProcessor) RollBackSingleCancel(evt *types.OrderCancelledEvent) error {
	// get rds.Order and types.OrderState
	state := &types.OrderState{}
	model, err := p.db.GetOrderByHash(evt.OrderHash)
	if err != nil {
		log.Debugf("fork order cancelled event,order:%s not exist in dao/order", evt.OrderHash.Hex())
		return nil
	}
	model.ConvertUp(state)

	// calculate remainAmount and cancelled amount should be saved whether order is finished or not
	if state.RawOrder.BuyNoMoreThanAmountB {
		state.CancelledAmountB = safeSub(state.CancelledAmountB, evt.AmountCancelled)
		log.Debugf("fork order cancelled event,order:%s cancelled amountb:%s", state.RawOrder.Hash.Hex(), state.CancelledAmountB.String())
	} else {
		state.CancelledAmountS = safeSub(state.CancelledAmountS, evt.AmountCancelled)
		log.Debugf("fork order cancelled event,order:%s cancelled amounts:%s", state.RawOrder.Hash.Hex(), state.CancelledAmountS.String())
	}

	// update order status
	settleOrderStatus(state, p.mc, ORDER_FROM_FILL)
	state.UpdatedBlock = evt.BlockNumber

	// update rds.Order
	model.ConvertDown(state)
	if err := p.db.UpdateOrderWhileCancel(state.RawOrder.Hash, state.Status, state.CancelledAmountS, state.CancelledAmountB, state.UpdatedBlock); err != nil {
		return fmt.Errorf("fork cancel event,error:%s", err.Error())
	}

	return nil
}

func (p *ForkProcessor) RollBackSingleCutoff(evt *types.CutoffEvent) error {
	if len(evt.OrderHashList) == 0 {
		log.Debugf("fork cutoff event,tx:%s,no order cutoff", evt.TxHash.Hex())
		return nil
	}

	for _, orderhash := range evt.OrderHashList {
		state := &types.OrderState{}
		model, err := p.db.GetOrderByHash(orderhash)
		if err != nil {
			log.Debugf("fork cutoff event,order:%s not exist in dao/order", orderhash.Hex())
			continue
		}
		model.ConvertUp(state)

		// update order status
		settleOrderStatus(state, p.mc, ORDER_FROM_FILL)

		if err := p.db.UpdateOrderWhileRollbackCutoff(orderhash, state.Status, evt.BlockNumber); err != nil {
			return fmt.Errorf("fork cutoff event,error:%s", err.Error())
		}

		log.Debugf("fork cutoff event,order:%s", orderhash.Hex())
	}

	return nil
}

func (p *ForkProcessor) RollBackSingleCutoffPair(evt *types.CutoffPairEvent) error {
	if len(evt.OrderHashList) == 0 {
		log.Debugf("fork cutoffPair event,tx:%s,no order cutoff", evt.TxHash.Hex())
		return nil
	}

	for _, orderhash := range evt.OrderHashList {
		state := &types.OrderState{}
		model, err := p.db.GetOrderByHash(orderhash)
		if err != nil {
			log.Debugf("fork cutoffPair event,order:%s not exist in dao/order", orderhash.Hex())
			continue
		}
		model.ConvertUp(state)

		// update order status
		// 在ordermanager 已完成的订单不会再更新,因此,cutoff事件发生之前,从钱包的角度来看只会有fillEvent,默认cancel取消所有的量
		settleOrderStatus(state, p.mc, ORDER_FROM_FILL)

		if err := p.db.UpdateOrderWhileRollbackCutoff(orderhash, state.Status, evt.BlockNumber); err != nil {
			return fmt.Errorf("fork cutoffPair event,error:%s", err.Error())
		}

		log.Debugf("fork cutoff pair event,order:%s", orderhash.Hex())
	}

	return nil
}

func (p *ForkProcessor) MarkForkEvents(from, to int64) error {
	if err := p.db.RollBackRingMined(from, to); err != nil {
		return fmt.Errorf("fork rollback ringmined events error:%s", err.Error())
	}
	if err := p.db.RollBackFill(from, to); err != nil {
		return fmt.Errorf("fork rollback fill events error:%s", err.Error())
	}
	if err := p.db.RollBackCancel(from, to); err != nil {
		return fmt.Errorf("fork rollback cancel events error:%s", err.Error())
	}
	if err := p.db.RollBackCutoff(from, to); err != nil {
		return fmt.Errorf("fork rollback cutoff events error:%s", err.Error())
	}
	if err := p.db.RollBackCutoffPair(from, to); err != nil {
		return fmt.Errorf("fork rollback cutoffPair events error:%s", err.Error())
	}

	return nil
}

const (
	FORK_EVT_TYPE_FILL        = "fill"
	FORK_EVT_TYPE_CANCEL      = "cancel"
	FORK_EVT_TYPE_CUTOFF      = "cutoff"
	FORK_EVT_TYPE_CUTOFF_PAIR = "cutoff_pair"
)

func (p *ForkProcessor) GetForkEvents(from, to int64) (InnerForkEventList, error) {
	var list InnerForkEventList

	if fillList, _ := p.db.GetFillForkEvents(from, to); len(fillList) > 0 {
		for _, v := range fillList {
			var (
				fill     types.OrderFilledEvent
				innerEvt InnerForkEvent
			)
			v.ConvertUp(&fill)
			innerEvt.LogIndex = fill.TxLogIndex
			innerEvt.BlockNumber = fill.BlockNumber.Int64()
			innerEvt.Type = FORK_EVT_TYPE_FILL
			innerEvt.Event = &fill
			list = append(list, innerEvt)
		}
	}

	if cancelList, _ := p.db.GetCancelForkEvents(from, to); len(cancelList) > 0 {
		for _, v := range cancelList {
			var (
				cancel   types.OrderCancelledEvent
				innerEvt InnerForkEvent
			)
			v.ConvertUp(&cancel)
			innerEvt.LogIndex = cancel.TxLogIndex
			innerEvt.BlockNumber = cancel.BlockNumber.Int64()
			innerEvt.Type = FORK_EVT_TYPE_CANCEL
			innerEvt.Event = &cancel
			list = append(list, innerEvt)
		}
	}

	if cutoffList, _ := p.db.GetCutoffForkEvents(from, to); len(cutoffList) > 0 {
		for _, v := range cutoffList {
			var (
				cutoff   types.CutoffEvent
				innerEvt InnerForkEvent
			)
			v.ConvertUp(&cutoff)
			innerEvt.LogIndex = cutoff.TxLogIndex
			innerEvt.BlockNumber = cutoff.BlockNumber.Int64()
			innerEvt.Type = FORK_EVT_TYPE_CUTOFF
			innerEvt.Event = &cutoff
			list = append(list, innerEvt)
		}
	}

	if cutoffPairList, _ := p.db.GetCutoffPairForkEvents(from, to); len(cutoffPairList) > 0 {
		for _, v := range cutoffPairList {
			var (
				cutoffPair types.CutoffPairEvent
				innerEvt   InnerForkEvent
			)
			v.ConvertUp(&cutoffPair)
			innerEvt.LogIndex = cutoffPair.TxLogIndex
			innerEvt.BlockNumber = cutoffPair.BlockNumber.Int64()
			innerEvt.Type = FORK_EVT_TYPE_CUTOFF_PAIR
			innerEvt.Event = &cutoffPair
			list = append(list, innerEvt)
		}
	}

	return list, nil
}

type InnerForkEvent struct {
	Type        string
	BlockNumber int64
	LogIndex    int64
	Event       interface{}
}

type InnerForkEventList []InnerForkEvent

func (l InnerForkEventList) Len() int {
	return len(l)
}

func (l InnerForkEventList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l InnerForkEventList) Less(i, j int) bool {
	if l[i].BlockNumber == l[j].BlockNumber {
		return l[i].LogIndex > l[j].LogIndex
	} else {
		return l[i].BlockNumber > l[j].BlockNumber
	}
}

func safeSub(x, y *big.Int) *big.Int {
	zero := big.NewInt(0)
	ret := new(big.Int).Sub(x, y)
	if ret.Cmp(zero) >= 0 {
		return ret
	} else {
		return zero
	}
}
