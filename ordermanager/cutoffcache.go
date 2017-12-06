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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
	"time"
)

type CutoffCache struct {
	cache map[common.Address]*big.Int
	rds   dao.RdsService
	mtx   sync.Mutex
}

func NewCutoffCache(rds dao.RdsService) *CutoffCache {
	cache := &CutoffCache{}
	cache.rds = rds
	cache.cache = make(map[common.Address]*big.Int)

	if cutoffEvents, err := rds.FindValidCutoffEvents(); err == nil {
		for _, v := range cutoffEvents {
			cache.cache[v.Owner] = v.Cutoff
		}
	}

	return cache
}

func (c *CutoffCache) Add(event *types.CutoffEvent) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	nowtime := time.Now().Unix()
	if event.Cutoff.Cmp(big.NewInt(nowtime)) < 0 {
		return fmt.Errorf("cutoff cache,invalid cutoff time:%d", nowtime)
	}

	var (
		model dao.CutOffEvent
		err   error
	)

	model.ConvertDown(event)
	if _, ok := c.cache[event.Owner]; !ok {
		err = c.rds.Add(model)
	} else {
		err = c.rds.Update(model)
	}

	if err != nil {
		return err
	}

	c.cache[event.Owner] = event.Cutoff

	return nil
}

// 合约验证的是创建时间
func (c *CutoffCache) IsOrderCutoff(owner common.Address, createTime *big.Int) bool {
	cutoffTime, ok := c.cache[owner]
	if !ok || (ok && cutoffTime.Cmp(createTime) < 0) {
		return false
	}

	return true
}
