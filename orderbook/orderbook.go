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
	"encoding/json"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"sync"
)

/**
todo:
1. filter
2. chain event
3. 事件执行到第几个块等信息数据
4. 订单完成的标志，以及需要发送到miner
*/

const (
	FINISH_TABLE_NAME  = "finished"
	PARTIAL_TABLE_NAME = "partial"
)

type Whisper struct {
	PeerOrderChan   chan *types.Order
	EngineOrderChan chan *types.OrderState
	ChainOrderChan  chan *types.OrderMined
}

type OrderBook struct {
	options       config.OrderBookOptions
	filters       []Filter
	db            db.Database
	finishTable   db.Database
	partialTable  db.Database
	runtimeTables map[string]db.Database
	whisper       *Whisper
	lock          sync.RWMutex
	minAmount     *big.Int
}

func NewOrderBook(options config.OrderBookOptions, database db.Database, whisper *Whisper) *OrderBook {
	ob := &OrderBook{}

	ob.options = options
	ob.db = database
	ob.finishTable = db.NewTable(database, FINISH_TABLE_NAME)
	ob.partialTable = db.NewTable(database, PARTIAL_TABLE_NAME)
	ob.whisper = whisper

	//todo:filters init
	filters := []Filter{}
	baseFilter := &BaseFilter{MinLrcFee: big.NewInt(options.Filters.BaseFilter.MinLrcFee)}
	filters = append(filters, baseFilter)
	tokenSFilter := &TokenSFilter{}
	tokenBFilter := &TokenBFilter{}

	filters = append(filters, tokenSFilter)
	filters = append(filters, tokenBFilter)

	return ob
}

func (ob *OrderBook) recoverOrder() error {
	iterator := ob.partialTable.NewIterator(nil, nil)
	for iterator.Next() {
		dataBytes := iterator.Value()
		state := &types.OrderState{}
		if err := json.Unmarshal(dataBytes, state); nil != err {
			log.Errorf("err:%s", err.Error())
		} else {
			ob.whisper.EngineOrderChan <- state
		}
	}
	return nil
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
	ob.recoverOrder()

	go func() {
		for {
			select {
			case ord := <-ob.whisper.PeerOrderChan:
				log.Debugf("accept data from peer:%s", ord.Protocol.Hex())
				if valid, err := ob.filter(ord); valid {
					if err := ob.peerOrderHook(ord); nil != err {
						log.Errorf("err:", err.Error())
					}
				} else {
					log.Errorf("receive order but valid failed:%s", err.Error())
				}
			case ord := <-ob.whisper.ChainOrderChan:
				ob.chainOrderHook(ord)
			}
		}
	}()
}

func (ob *OrderBook) Stop() {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	ob.finishTable.Close()
	ob.partialTable.Close()
	//ob.db.Close()
}

func (ob *OrderBook) peerOrderHook(ord *types.Order) error {

	ob.lock.Lock()
	defer ob.lock.Unlock()

	// TODO(fk): order filtering

	state := &types.OrderState{}
	state.RawOrder = *ord
	state.RawOrder.Hash = ord.GenerateHash()

	//todo:it should not query db everytime.
	if input, err := ob.partialTable.Get(state.RawOrder.Hash.Bytes()); err != nil {
		panic(err)
	} else if len(input) == 0 {
		if inpupt1, err1 := ob.finishTable.Get(state.RawOrder.Hash.Bytes()); nil != err1 {
			panic(err1)
		} else if len(inpupt1) == 0 {
			state.Status = types.ORDER_NEW
			state.RemainedAmountS = state.RawOrder.AmountS
			state.RemainedAmountB = state.RawOrder.AmountB
		} else {
			state.Status = types.ORDER_FINISHED
		}
	} else {
		state.Status = types.ORDER_PARTIAL
	}

	//do nothing when types.ORDER_NEW != state.Status
	if types.ORDER_NEW == state.Status {

		log.Debugf("state hash:%s", state.RawOrder.Hash.Hex())

		//save to db
		dataBytes, err := json.Marshal(state)
		if err != nil {
			return err
		}
		ob.partialTable.Put(state.RawOrder.Hash.Bytes(), dataBytes)

		//send to miner
		ob.whisper.EngineOrderChan <- state
	}

	return nil
}

func (ob *OrderBook) chainOrderHook(ord *types.OrderMined) error {
	ob.lock.Lock()
	defer ob.lock.Unlock()

	return nil
}

// GetOrder get single order with hash
func (ob *OrderBook) GetOrder(id types.Hash) (*types.OrderState, error) {
	var (
		value []byte
		err   error
		ord   types.OrderState
	)

	if value, err = ob.partialTable.Get(id.Bytes()); err != nil {
		value, err = ob.finishTable.Get(id.Bytes())
	}

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(value, &ord)
	if err != nil {
		return nil, err
	}

	return &ord, nil
}

// GetOrders get orders from persistence database
func (ob *OrderBook) GetOrders() {

}

// moveOrder move order when partial finished order fully exchanged
func (ob *OrderBook) moveOrder(odw *types.OrderState) error {
	key := odw.RawOrder.Hash.Bytes()
	value, err := json.Marshal(odw)
	if err != nil {
		return err
	}
	ob.partialTable.Delete(key)
	ob.finishTable.Put(key, value)
	return nil
}

// isFinished judge order state
func isFinished(odw *types.OrderState) bool {
	//if odw.RawOrder.
	return true
}
