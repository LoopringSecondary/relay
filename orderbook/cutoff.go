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
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"sync"
)

// 将cutoff事件存储到db中，启动时load到map内
// orderbook添加filter，新来的订单如果在map
// 中能找到，则直接过滤

const CUTOFF_EVENT_TABLE_NAME = "cutoff_events"

//go:generate gencodec -type CutoffIndex -field-override cutoffIndexMarshaling -out gen_cutoffindex_json.go
type CutoffIndex struct {
	Address     types.Address `json:"address" gencodec:"required"`     // owner地址
	Timestamp   *big.Int      `json:"timestamp" gencodec:"required"`   // 事件发生时间
	Cutoff      *big.Int      `json:"cutoff" gencodec:"required"`      // 批量删除时间
	BlockNumber *big.Int      `json:"blocknumber" gencodec:"required"` // 事件发生区块
}

type cutoffIndexMarshaling struct {
	Timestamp   *types.Big
	Cutoff      *types.Big
	BlockNumber *types.Big
}

type CutoffIndexCache struct {
	indexMap    map[types.Address]CutoffIndex
	persistence db.Database
	mtx         sync.Mutex
}

func NewCutoffIndexCache(database db.Database) *CutoffIndexCache {
	c := &CutoffIndexCache{}
	c.persistence = db.NewTable(database, CUTOFF_EVENT_TABLE_NAME)
	c.indexMap = make(map[types.Address]CutoffIndex)

	return c
}

func (c *CutoffIndexCache) Load() {

}

// todo filter

func (c *CutoffIndexCache) Add(address types.Address, timestamp, cutoff, blocknumber *big.Int) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	entity := CutoffIndex{Address: address, Timestamp: timestamp, Cutoff: cutoff, BlockNumber: blocknumber}

	bs, err := entity.MarshalJSON()
	if err != nil {
		return err
	}

	if err := c.persistence.Put(address.Bytes(), bs); err != nil {
		return err
	}

	c.indexMap[address] = entity

	return nil
}

// todo: 限制map大小,移除过期数据
func (c *CutoffIndexCache) Del() {
	c.mtx.Lock()
	defer c.mtx.Unlock()

}
