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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

type UserManager interface {
	AddWhiteListUser(user types.WhiteListUser) error
	DelWhiteListUser(user types.WhiteListUser) error
	InWhiteList(owner common.Address) bool
}

type UserManagerImpl struct {
	rds       dao.RdsService
	whiteList *WhiteListCache
}

func NewUserManager(rds dao.RdsService) *UserManagerImpl {
	impl := &UserManagerImpl{}
	impl.rds = rds
	impl.whiteList = newWhiteListCache(impl.rds)

	return impl
}

func (m *UserManagerImpl) InWhiteList(owner common.Address) bool { return m.whiteList.InWhiteList(owner) }
func (m *UserManagerImpl) AddWhiteListUser(user types.WhiteListUser) error {
	return m.whiteList.AddWhiteListUser(user)
}
func (m *UserManagerImpl) DelWhiteListUser(user types.WhiteListUser) error {
	return m.whiteList.DelWhiteListUser(user)
}
