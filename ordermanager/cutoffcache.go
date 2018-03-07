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
	ca "github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"time"
)

type CutoffCache struct {
	cache ca.Cache
	ttl   int64
}

func NewCutoffCache(cache ca.Cache, expire int64) *CutoffCache {
	cutoffcache := &CutoffCache{}
	cutoffcache.cache = cache
	cutoffcache.ttl = expire

	return cutoffcache
}

// 合约验证的是创建时间
func (c *CutoffCache) IsOrderCutoff(order *types.OrderState) bool {
	protocol := order.RawOrder.Protocol
	owner := order.RawOrder.Owner
	validsince := order.RawOrder.ValidSince

	if cutoff := c.GetCutoff(protocol, owner); cutoff.Cmp(validsince) > 0 {
		return true
	}

	token1 := order.RawOrder.TokenS
	token2 := order.RawOrder.TokenB
	if cutoff := c.GetCutoffPair(protocol, owner, token1, token2); cutoff.Cmp(validsince) > 0 {
		return true
	}

	return false
}

func (c *CutoffCache) GetCutoff(protocol, owner common.Address) *big.Int {
	key := formatCutoffKey(protocol, owner)

	if bs, err := c.cache.Get(key); err == nil {
		return bytes2value(bs)
	}

	if cutoff, _ := ethaccessor.GetCutoff(protocol, owner, "latest"); cutoff.Cmp(big.NewInt(0)) > 0 {
		c.UpdateCutoff(protocol, owner, cutoff)
		return cutoff
	}

	return big.NewInt(0)
}

func (c *CutoffCache) GetCutoffPair(protocol, owner, token1, token2 common.Address) *big.Int {
	key := formatCutoffPairKey(protocol, owner, token1, token2)

	if bs, err := c.cache.Get(key); err == nil {
		return bytes2value(bs)
	}

	if cutoff, _ := ethaccessor.GetCutoffPair(protocol, owner, token1, token2, "latest"); cutoff.Cmp(big.NewInt(0)) > 0 {
		c.UpdateCutoffPair(protocol, owner, token1, token2, cutoff)
		return cutoff
	}

	return big.NewInt(0)
}

func (c *CutoffCache) UpdateCutoff(protocol, owner common.Address, cutoff *big.Int) error {
	key := formatCutoffKey(protocol, owner)
	bs := value2bytes(cutoff)

	return c.cache.Set(key, bs, time.Now().Unix()+c.ttl)
}

func (c *CutoffCache) UpdateCutoffPair(protocol, owner, token1, token2 common.Address, cutoff *big.Int) error {
	key := formatCutoffPairKey(protocol, owner, token1, token2)
	bs := value2bytes(cutoff)

	return c.cache.Set(key, bs, time.Now().Unix()+c.ttl)
}

func formatCutoffKey(protocol, owner common.Address) string {
	return protocol.Hex() + "-" + owner.Hex()
}

// todo(fuk): need test
func formatCutoffPairKey(protocol, owner, token1, token2 common.Address) string {
	var bs []byte

	bs1 := token1.Bytes()
	bs2 := token2.Bytes()

	for i := 0; i < len(bs1); i++ {
		bs[i] = bs1[i] ^ bs2[i]
	}

	return protocol.Hex() + "-" + owner.Hex() + "-" + string(bs)
}

func value2bytes(v *big.Int) []byte  { return v.Bytes() }
func bytes2value(bs []byte) *big.Int { return big.NewInt(0).SetBytes(bs) }
