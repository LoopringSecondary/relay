package orderbook

import (
	"encoding/json"
	"errors"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"math/big"
	//"sort"
	"sort"
	"sync"
)

const (
	FINISH_TABLE_NAME  = "finished"
	PENDING_TABLE_NAME = "pending"
)

type Rdbs struct {
	db           db.Database
	finishTable  db.Database
	partialTable db.Database
	idxs         SliceOrderIndex
	mtx          sync.Mutex
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
			sendOrderToMiner(state)
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

type OrderIndex struct {
	hash      types.Hash
	timestamp *big.Int
}

type SliceOrderIndex []*OrderIndex

func (s SliceOrderIndex) Len() int {
	return len(s)
}

func (s SliceOrderIndex) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

// asc
func (s SliceOrderIndex) Less(i, j int) bool {
	if s[i].timestamp.Cmp(s[j].timestamp) < 0 {
		return true
	}
	return false
}

func (r *Rdbs) load() {

}

func (r *Rdbs) push(hash types.Hash, timestamp *big.Int) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	idx := &OrderIndex{}
	idx.hash = hash
	idx.timestamp = timestamp

	r.idxs = append(r.idxs, idx)
	sort.Sort(r.idxs)
}

func (r *Rdbs) pop() (*OrderIndex, error) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	if len(r.idxs) < 1 {
		return nil, errors.New("orderbook orderIndex slice is empty")
	}

	return r.idxs[0], nil
}
