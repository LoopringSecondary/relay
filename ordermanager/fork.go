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
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"math/big"
	"sort"
)

type forkProcessor struct {
	db dao.RdsService
	mc marketcap.MarketCapProvider
}

func newForkProcess(rds dao.RdsService, mc marketcap.MarketCapProvider) *forkProcessor {
	processor := &forkProcessor{}
	processor.db = rds
	processor.mc = mc

	return processor
}

//	todo(fuk): fork逻辑重构
// 1.从各个事件表中获取所有处于分叉块中的事件(fill,cancel,cutoff,cutoffPair)并按照blockNumber以及logIndex倒序
// 2.对每个事件进行回滚处理,并更新数据库,即便订单已经过期也要对其相关数据进行更新
// 3.删除所有分叉事件,或者对其进行标记
func (p *forkProcessor) fork(event *types.ForkedEvent) error {
	log.Debugf("order manager processing chain fork......")

	from := event.ForkBlock.Int64()
	to := event.DetectedBlock.Int64()

	list, _ := p.GetForkEvents(from, to)
	if list.Len() == 0 {
		log.Errorf("order manager fork error: non fork events")
		return nil
	}

	sort.Sort(list)

	for _, v := range list {
		switch v.Type {
		case FORK_EVT_TYPE_FILL:
			p.RollBackSingleFill(v.Event.(*types.OrderFilledEvent))
		case FORK_EVT_TYPE_CANCEL:
			p.RollBackSingleCancel(v.Event.(*types.OrderCancelledEvent))
		case FORK_EVT_TYPE_CUTOFF:
			p.RollBackSingleCutoff(v.Event.(*types.CutoffEvent))
		case FORK_EVT_TYPE_CUTOFF_PAIR:
			p.RollBackSingleCutoffPair(v.Event.(*types.CutoffPairEvent))
		}
	}

	p.MarkForkEvents(from, to)
	return nil
}

// calculate order's related values and status, update order
func (p *forkProcessor) RollBackSingleFill(evt *types.OrderFilledEvent) error {
	state := &types.OrderState{}
	model, err := p.db.GetOrderByHash(evt.OrderHash)
	if err != nil {
		return err
	}
	if err := model.ConvertUp(state); err != nil {
		return err
	}

	// judge order status
	//if state.Status == types.ORDER_CUTOFF || state.Status == types.ORDER_FINISHED || state.Status == types.ORDER_UNKNOWN {
	//	log.Debugf("order manager,handle order filled event,order %s status is %d ", state.RawOrder.Hash.Hex(), state.Status)
	//	return nil
	//}

	// calculate dealt amount
	state.UpdatedBlock = evt.BlockNumber
	state.DealtAmountS = new(big.Int).Sub(state.DealtAmountS, evt.AmountS)
	state.DealtAmountB = new(big.Int).Sub(state.DealtAmountB, evt.AmountB)
	state.SplitAmountS = new(big.Int).Sub(state.SplitAmountS, evt.SplitS)
	state.SplitAmountB = new(big.Int).Sub(state.SplitAmountB, evt.SplitB)

	log.Debugf("order manager,process fork fill event orderhash:%s,dealAmountS:%s,dealtAmountB:%s", state.RawOrder.Hash.Hex(), state.DealtAmountS.String(), state.DealtAmountB.String())

	// update order status
	settleOrderStatus(state, p.mc)

	// update rds.Order
	if err := model.ConvertDown(state); err != nil {
		log.Errorf(err.Error())
		return err
	}
	if err := p.db.UpdateOrderWhileFill(state.RawOrder.Hash, state.Status, state.DealtAmountS, state.DealtAmountB, state.SplitAmountS, state.SplitAmountB, state.UpdatedBlock); err != nil {
		return err
	}

	return nil
}

func (p *forkProcessor) RollBackSingleCancel(evt *types.OrderCancelledEvent) error {
	return nil
}

func (p *forkProcessor) RollBackSingleCutoff(evt *types.CutoffEvent) error {
	return nil
}

func (p *forkProcessor) RollBackSingleCutoffPair(evt *types.CutoffPairEvent) error {
	return nil
}

// todo(fuk): process error
func (p *forkProcessor) MarkForkEvents(from, to int64) error {
	p.db.RollBackFill(from, to)
	p.db.RollBackCancel(from, to)
	p.db.RollBackCutoff(from, to)
	p.db.RollBackCutoffPair(from, to)

	return nil
}

const (
	FORK_EVT_TYPE_FILL        = "fill"
	FORK_EVT_TYPE_CANCEL      = "cancel"
	FORK_EVT_TYPE_CUTOFF      = "cutoff"
	FORK_EVT_TYPE_CUTOFF_PAIR = "cutoff_pair"
)

func (p *forkProcessor) GetForkEvents(from, to int64) (InnerForkEventList, error) {
	var list InnerForkEventList

	if fillList, _ := p.db.GetFillForkEvents(from, to); len(fillList) > 0 {
		for _, v := range fillList {
			var (
				fill     types.OrderFilledEvent
				innerEvt InnerForkEvent
			)
			v.ConvertUp(&fill)
			innerEvt.LogIndex = fill.LogIndex
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
			innerEvt.LogIndex = cancel.LogIndex
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
			innerEvt.LogIndex = cutoff.LogIndex
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
			innerEvt.LogIndex = cutoffPair.LogIndex
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
