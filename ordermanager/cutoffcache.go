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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	gocache "github.com/patrickmn/go-cache"
	"math/big"
	"time"
)

type CutoffCache struct {
	cache  *gocache.Cache
	expire time.Duration
	clean  time.Duration
	rds    dao.RdsService
}

func NewCutoffCache(rds dao.RdsService, expire, clean int64) *CutoffCache {
	cache := &CutoffCache{}
	cache.rds = rds
	cache.expire = time.Duration(expire) * time.Second

	if clean > 0 {
		cache.clean = time.Duration(clean) * time.Second
	} else {
		cache.clean = gocache.NoExpiration
	}
	cache.cache = gocache.New(cache.expire, cache.clean)

	return cache
}

// 合约验证的是创建时间
func (c *CutoffCache) IsOrderCutoff(protocol, owner common.Address, createTime *big.Int) bool {
	cutoff, ok := c.Get(protocol, owner)
	if !ok || cutoff.Cmp(createTime) < 0 {
		return false
	}
	return true
}

func (c *CutoffCache) Get(protocol, owner common.Address) (*big.Int, bool) {
	cutoff, ok := c.get(protocol, owner)
	if !ok {
		if entity, err := c.rds.GetCutoffEvent(protocol, owner); err == nil {
			var evt types.AllOrdersCancelledEvent
			entity.ConvertUp(&evt)
			cutoff = evt.Cutoff
			c.set(protocol, owner, cutoff)
		} else {
			return big.NewInt(0), false
		}
	}

	return cutoff, true
}

func (c *CutoffCache) Add(event *types.AllOrdersCancelledEvent) error {
	entity := new(dao.CutOffEvent)
	entity.ConvertDown(event)
	if err := c.rds.Add(entity); err != nil {
		return err
	}

	return c.set(event.ContractAddress, event.Owner, event.Cutoff)
}

func (c *CutoffCache) Del(protocol, owner common.Address) error {
	if err := c.rds.DelCutoffEvent(protocol, owner); err != nil {
		return err
	}
	c.del(protocol, owner)
	return nil
}

func (c *CutoffCache) set(protocol, owner common.Address, cutoff *big.Int) error {
	key := formatKey(protocol, owner)
	return c.cache.Add(key, cutoff, c.expire)
}

func (c *CutoffCache) get(protocol, owner common.Address) (*big.Int, bool) {
	key := formatKey(protocol, owner)
	data, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	return data.(*big.Int), true
}

func (c *CutoffCache) del(protocol, owner common.Address) {
	key := formatKey(protocol, owner)
	c.cache.Delete(key)
}

func formatKey(protocol, owner common.Address) string {
	return protocol.Hex() + "-" + owner.Hex()
}
