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
	"sync"
	"github.com/Loopring/ringminer/types"
	"github.com/Loopring/ringminer/lrcdb"
	"log"
	"os"
	"github.com/Loopring/ringminer/config"
)

type ORDER_STATUS int

const (
	FINISH_TABLE_NAME = "finished"
	PARTIAL_TABLE_NAME = "partial"
)

type OrderBookConfig struct {
	name           string
	cacheCapacity   int
	bufferCapacity  int
}

type OrderBook struct {
	conf         OrderBookConfig
	toml         config.DbOptions
	db           lrcdb.Database
	finishTable  lrcdb.Database
	partialTable lrcdb.Database
	whisper      *types.Whispers
	lock         sync.RWMutex
}

func (ob *OrderBook) loadConfig() {
	// TODO(fk): set path as global variable
	dir := os.Getenv("GOPATH") + "/github.com/Loopring/ringminer/"
	file := dir + ob.toml.Name
	cache := ob.toml.CacheCapacity
	buffer := ob.toml.BufferCapacity

	// TODO(fk): load config from cli or genesis

	ob.conf = OrderBookConfig{file, cache, buffer}
}

func NewOrderBook(whisper *types.Whispers, options config.DbOptions) *OrderBook {
	s := &OrderBook{}

	s.toml = options
	s.loadConfig()

	s.db = lrcdb.NewDB(s.conf.name, s.conf.cacheCapacity, s.conf.bufferCapacity)
	s.finishTable = lrcdb.NewTable(s.db, FINISH_TABLE_NAME)
	s.partialTable = lrcdb.NewTable(s.db, PARTIAL_TABLE_NAME)
	s.whisper = whisper

	return s
}

// Start start orderbook as a service
func (s *OrderBook) Start() {
	go func() {
		for {
			select {
			case ord := <- s.whisper.PeerOrderChan:
				s.peerOrderHook(ord)
			case ord := <- s.whisper.ChainOrderChan:
				s.chainOrderHook(ord)
			}
		}
	}()
}

func (s *OrderBook) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.finishTable.Close()
	s.partialTable.Close()
	s.db.Close()
}

func (ob *OrderBook) peerOrderHook(ord *types.Order) error {

	// TODO(fk): order filtering

	ob.lock.Lock()
	defer ob.lock.Unlock()

	key := ord.GenHash().Bytes()
	value,err := ord.MarshalJson()
	if err != nil {
		return err
	}

	ob.partialTable.Put(key, value)

	// TODO(fk): delete after test
	if input, err := ob.partialTable.Get(key); err != nil {
		panic(err)
	} else {
		var ord types.Order
		ord.UnMarshalJson(input)
		log.Println(ord.TokenS.Str())
		log.Println(ord.TokenB.Str())
		log.Println(ord.AmountS.Uint64())
		log.Println(ord.AmountB.Uint64())
	}

	// TODO(fk): send orderState to matchengine
	//state := ord.Convert()
	//ob.whisper.EngineOrderChan <- state
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
		err error
		ord types.OrderState
	)

	if value, err = ob.partialTable.Get(id.Bytes()); err != nil {
		value, err = ob.finishTable.Get(id.Bytes())
	}
	if err != nil {
		return nil, err
	}

	err = ord.UnMarshalJson(value)
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
	key := odw.OrderHash.Bytes()
	value, err := odw.MarshalJson()
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
