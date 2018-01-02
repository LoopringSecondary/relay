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

package usermanager

import (
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	gocache "github.com/patrickmn/go-cache"
	"time"
)

type WhiteListCache struct {
	cache  *gocache.Cache
	rds    dao.RdsService
	expire time.Duration
}

func newWhiteListCache(options *config.UserManagerOptions, rds dao.RdsService) *WhiteListCache {
	c := &WhiteListCache{}
	c.rds = rds

	expire := time.Duration(options.WhiteListCacheExpireTime) * time.Second
	cleanUp := time.Duration(options.WhiteListCacheCleanTime) * time.Second
	c.cache = gocache.New(expire, cleanUp)
	c.expire = expire

	c.refreshWhiteList()

	return c
}

func (c WhiteListCache) syncWhiteList() {
	if list, err := c.rds.GetWhiteList(); err == nil || len(list) == 0 {
		for _, v := range list {
			var user types.WhiteListUser
			if err := v.ConvertUp(&user); err != nil {
				log.Errorf("new white list cache error:%s", err.Error())
				continue
			}
			c.set(&user)
		}
	}
}
func (c WhiteListCache) refreshWhiteList() {
	c.syncWhiteList()
	go func() {
		for {
			select {
			case <-time.After(time.Second * 60):
				c.syncWhiteList()
			}
		}
	}()
}

func (c *WhiteListCache) AddWhiteListUser(user types.WhiteListUser) error {
	if c.InWhiteList(user.Owner) {
		log.Debugf("white list user:%s already exist in cache", user.Owner.Hex())
		return nil
	}

	c.set(&user)
	model := dao.WhiteList{}
	if err := model.ConvertDown(&user); err != nil {
		return err
	}

	return c.rds.Add(model)
}

func (c *WhiteListCache) DelWhiteListUser(user types.WhiteListUser) error {
	if !c.InWhiteList(user.Owner) {
		log.Debugf("white list user:%s already deleted in cache", user.Owner.Hex())
		return nil
	}

	c.del(user.Owner)
	model := dao.WhiteList{}
	if err := model.ConvertDown(&user); err != nil {
		return err
	}

	return c.rds.Del(model)
}

func (c *WhiteListCache) InWhiteList(address common.Address) bool {
	_, ok := c.get(address)
	return ok
}

// get get value from gocache
func (c *WhiteListCache) get(address common.Address) (*types.WhiteListUser, bool) {
	data, ok := c.cache.Get(address.Hex())
	if !ok {
		return nil, false
	}

	user := data.(*types.WhiteListUser)

	return user, true
}

// set set key-value in gocache
func (c *WhiteListCache) set(user *types.WhiteListUser) {
	address := user.Owner.Hex()
	c.cache.Set(address, user, c.expire)
}

// del delete key from gocache
func (c *WhiteListCache) del(address common.Address) {
	c.cache.Delete(address.Hex())
}
