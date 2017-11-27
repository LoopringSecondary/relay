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
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"sync"
	"github.com/ethereum/go-ethereum/common"
)

type WhiteListCache struct {
	users map[common.Address]types.WhiteListUser
	rds   dao.RdsService
	mtx   sync.Mutex
}

func newWhiteListCache(rds dao.RdsService) *WhiteListCache {
	c := &WhiteListCache{}
	c.rds = rds
	c.users = make(map[common.Address]types.WhiteListUser)

	if list, err := c.rds.GetWhiteList(); err == nil || len(list) == 0 {
		for _, v := range list {
			var user types.WhiteListUser
			if err := v.ConvertUp(&user); err != nil {
				log.Errorf("new white list cache error:%s", err.Error())
				continue
			}
			c.users[user.Owner] = user
		}
	}

	return c
}

func (c *WhiteListCache) AddWhiteListUser(user types.WhiteListUser) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.InWhiteList(user.Owner) {
		log.Debugf("white list user:%s already exist in cache", user.Owner.Hex())
		return nil
	}

	c.users[user.Owner] = user
	model := dao.WhiteList{}
	if err := model.ConvertDown(&user); err != nil {
		return err
	}

	return c.rds.Add(model)
}

func (c *WhiteListCache) DelWhiteListUser(user types.WhiteListUser) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if !c.InWhiteList(user.Owner) {
		log.Debugf("white list user:%s already deleted in cache", user.Owner.Hex())
		return nil
	}

	delete(c.users, user.Owner)
	model := dao.WhiteList{}
	if err := model.ConvertDown(&user); err != nil {
		return err
	}

	return c.rds.Del(model)
}

func (c *WhiteListCache) InWhiteList(user common.Address) bool {
	if _, ok := c.users[user]; !ok {
		return false
	}

	return true
}
