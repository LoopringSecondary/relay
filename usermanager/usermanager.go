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
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

type UserManager interface {
	AddWhiteListUser(user types.WhiteListUser) error
	DelWhiteListUser(user types.WhiteListUser) error
	InWhiteList(owner common.Address) bool
	IsWhiteListOpen() bool
}

type UserManagerImpl struct {
	rds       dao.RdsService
	options   *config.UserManagerOptions
	whiteList *WhiteListCache
}

func NewUserManager(options *config.UserManagerOptions, rds dao.RdsService) *UserManagerImpl {
	impl := &UserManagerImpl{}
	impl.rds = rds
	impl.options = options

	if options.WhiteListOpen {
		impl.whiteList = newWhiteListCache(impl.options, impl.rds)
	}

	return impl
}

func (m *UserManagerImpl) InWhiteList(owner common.Address) bool {
	if !m.options.WhiteListOpen {
		return true
	}

	return m.whiteList.InWhiteList(owner)
}

func (m *UserManagerImpl) AddWhiteListUser(user types.WhiteListUser) error {
	if !m.options.WhiteListOpen {
		return fmt.Errorf("wihte list is closed")
	}
	return m.whiteList.AddWhiteListUser(user)
}
func (m *UserManagerImpl) DelWhiteListUser(user types.WhiteListUser) error {
	if !m.options.WhiteListOpen {
		return fmt.Errorf("wihte list is closed")
	}
	return m.whiteList.DelWhiteListUser(user)
}
func (m *UserManagerImpl) IsWhiteListOpen() bool {
	return m.options.WhiteListOpen
}
