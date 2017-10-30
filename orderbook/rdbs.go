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
)

const (
	FINISH_TABLE_NAME  = "finished"
	PENDING_TABLE_NAME = "pending"
)

type Rdbs struct {
	db           db.Database
	finishTable  db.Database
	partialTable db.Database
	idxs         SliceOrderTimestampIndex
}

func NewRdbs(database db.Database) *Rdbs {
	r := &Rdbs{}
	r.db = database
	r.finishTable = db.NewTable(database, FINISH_TABLE_NAME)
	r.partialTable = db.NewTable(database, PENDING_TABLE_NAME)
	return r
}

func (r *Rdbs) Close() {
	r.partialTable.Close()
	r.finishTable.Close()
}

func (r *Rdbs) Scan() error {
	iterator := r.partialTable.NewIterator(nil, nil)
	for iterator.Next() {
		dataBytes := iterator.Value()
		state := &types.OrderState{}
		if err := json.Unmarshal(dataBytes, state); nil != err {
			log.Errorf("err:%s", err.Error())
		} else {
			//sendOrderToMiner(state)
		}
	}
	return nil
}

// GetOrder get single order with hash
func (r *Rdbs) GetOrder(id types.Hash) (*types.OrderState, error) {
	ord, _, err := r.getOrder(id)
	return ord, err
}

func (r *Rdbs) SetOrder(state *types.OrderState) error {
	bs, err := json.Marshal(state)

	if err != nil {
		return errors.New("orderbook order" + state.RawOrder.Hash.Hex() + " marshal error")
	}

	if err := r.partialTable.Put(state.RawOrder.Hash.Bytes(), bs); err != nil {
		return errors.New("orderbook order save error")
	}

	return nil
}

func (r *Rdbs) getOrder(id types.Hash) (*types.OrderState, string, error) {
	var (
		value []byte
		err   error
		tn    string
		ord   types.OrderState
	)

	if value, err = r.partialTable.Get(id.Bytes()); err != nil {
		value, err = r.finishTable.Get(id.Bytes())
		if err != nil {
			return nil, "", errors.New("order do not exist")
		} else {
			tn = FINISH_TABLE_NAME
		}
	} else {
		tn = PENDING_TABLE_NAME
	}

	err = json.Unmarshal(value, &ord)
	if err != nil {
		return nil, tn, err
	}

	return &ord, tn, nil
}

// GetOrders get orders from persistence database
func (r *Rdbs) GetOrders() {
	// todo?
}

// moveOrder move order when partial finished order fully exchanged
func (r *Rdbs) MoveOrder(ord *types.OrderState) error {
	key := ord.RawOrder.Hash.Bytes()
	value, err := json.Marshal(ord)
	if err != nil {
		return err
	}

	if err := r.partialTable.Delete(key); err != nil {
		return err
	}

	if err := r.finishTable.Put(key, value); err != nil {
		return err
	}
	return nil
}
