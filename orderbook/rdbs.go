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
	"errors"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"sync"
)

const (
	FINISH_TABLE_NAME  = "finished"
	PENDING_TABLE_NAME = "pending"
)

type Rdbs struct {
	db                  db.Database
	finishTable         db.Database
	pendingTable        db.Database
	orderChan			chan *types.OrderState
	orderhashIndexTable map[types.Hash]*orderhashIndex
	mtx                 sync.RWMutex
}

type orderhashIndex struct {
	hash   types.Hash
	owner  types.Address
	status types.OrderStatus
}

func NewRdbs(database db.Database) *Rdbs {
	r := &Rdbs{}
	r.db = database
	r.finishTable = db.NewTable(database, FINISH_TABLE_NAME)
	r.pendingTable = db.NewTable(database, PENDING_TABLE_NAME)
	r.orderhashIndexTable = make(map[types.Hash]*orderhashIndex)
	r.orderChan = make(chan *types.OrderState)
	return r
}

func (r *Rdbs) Close() {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.pendingTable.Close()
	r.finishTable.Close()
}

func (r *Rdbs) Reload() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	iterator1 := r.pendingTable.NewIterator(nil, nil)
	for iterator1.Next() {
		state, err := r.reloadOrderState(iterator1.Value())
		if err != nil {
			log.Errorf("orderbook rdbs pending table reload error %s", err.Error())
			continue
		}
		r.orderChan <- state
	}

	iterator2 := r.finishTable.NewIterator(nil, nil)
	for iterator2.Next() {
		_, err := r.reloadOrderState(iterator2.Value())
		if err != nil {
			log.Errorf("orderbook rdbs finish table reload error %s", err.Error())
			continue
		}
	}
	return nil
}

func (r *Rdbs) reloadOrderState(bs []byte) (*types.OrderState, error) {
	var state types.OrderState

	if err := json.Unmarshal(bs, &state); nil != err {
		return nil, err
	}

	version, err := state.LatestVersion()
	if err != nil {
		return nil, err
	}

	r.setOrderhashIndex(state.RawOrder.Hash, state.RawOrder.Owner, version.Status)

	return &state, nil
}

// GetOrder get single order with hash
func (r *Rdbs) GetOrder(id types.Hash) (*types.OrderState, error) {
	var (
		value []byte
		err   error
		ord   types.OrderState
	)

	r.mtx.RLock()
	defer r.mtx.RUnlock()

	// get status from with index
	idx, ok := r.getOrderhashIndex(id)
	if !ok {
		return nil, errors.New("order " + id.Hex() + " do not exist")
	}

	// get bytes from table
	if idx.status == types.ORDER_NEW || idx.status == types.ORDER_PENDING {
		value, err = r.pendingTable.Get(id.Bytes())
	} else {
		value, err = r.finishTable.Get(id.Bytes())
	}
	if err != nil {
		return nil, err
	}

	// marshal bytes to order
	if err = json.Unmarshal(value, &ord); err != nil {
		return nil, err
	}

	return &ord, nil
}

func (r *Rdbs) SetOrder(state *types.OrderState) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	bs, err := json.Marshal(state)

	version, err := state.LatestVersion()
	if err != nil {
		return err
	}

	if err != nil {
		return errors.New("orderbook order" + state.RawOrder.Hash.Hex() + " marshal error")
	}

	if err := r.pendingTable.Put(state.RawOrder.Hash.Bytes(), bs); err != nil {
		return errors.New("orderbook order save error")
	}

	r.setOrderhashIndex(state.RawOrder.Hash, state.RawOrder.Owner, version.Status)

	return nil
}

// moveOrder move order when partial finished order fully exchanged
func (r *Rdbs) MoveOrder(ord *types.OrderState) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	key := ord.RawOrder.Hash.Bytes()
	value, err := json.Marshal(ord)
	if err != nil {
		return err
	}

	if err := r.pendingTable.Delete(key); err != nil {
		return err
	}

	if err := r.finishTable.Put(key, value); err != nil {
		return err
	}
	return nil
}

func (r *Rdbs) CheckOrderStatus(state *types.OrderState) types.OrderStatus {
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	idx, _ := r.getOrderhashIndex(state.RawOrder.Hash)
	return idx.status
}

// GetOrders get orders from persistence database
func (r *Rdbs) GetOrders(owner types.Address, status []types.OrderStatus, timestamp *big.Int) {
	// TODO
}

func (r *Rdbs) setOrderhashIndex(id types.Hash, owner types.Address, status types.OrderStatus) {
	r.orderhashIndexTable[id] = &orderhashIndex{owner: owner, hash: id, status: status}
}

func (r *Rdbs) getOrderhashIndex(id types.Hash) (*orderhashIndex, bool) {
	idx, ok := r.orderhashIndexTable[id]
	if !ok {
		return nil, false
	}

	return idx, true
}
