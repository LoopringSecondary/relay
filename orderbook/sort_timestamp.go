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
	"errors"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"sort"
	"sync"
	"time"
)

type OrderTimestampIndex struct {
	hash      types.Hash
	timestamp *big.Int
}

type OrderTimestampList struct {
	list SliceOrderTimestampIndex
	mtx  sync.Mutex
}

type SliceOrderTimestampIndex []*OrderTimestampIndex

func (s SliceOrderTimestampIndex) Len() int {
	return len(s)
}

func (s SliceOrderTimestampIndex) Swap(i, j int) {
	tmp := s[i]
	s[i] = s[j]
	s[j] = tmp
}

// asc
func (s SliceOrderTimestampIndex) Less(i, j int) bool {
	if s[i].timestamp.Cmp(s[j].timestamp) < 0 {
		return true
	}
	return false
}

// todo
func (l *OrderTimestampList) load() {

}

func (l *OrderTimestampList) Push(hash types.Hash, timestamp *big.Int) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	idx := &OrderTimestampIndex{}
	idx.hash = hash
	idx.timestamp = timestamp

	l.list = append(l.list, idx)
	sort.Sort(l.list)
}

func (l *OrderTimestampList) Pop() (types.Hash, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	n := types.HexToHash("")

	if len(l.list) < 1 {
		return n, errors.New("orderbook orderIndex slice is empty")
	}

	unixtime := time.Now().Unix()
	if l.list[0].timestamp.Cmp(big.NewInt(unixtime)) < 0 {
		return n, errors.New("orderbook orderIndex slice not ready")
	}

	idx := l.list[0]
	l.list = l.list[1:]

	return idx.hash, nil
}
