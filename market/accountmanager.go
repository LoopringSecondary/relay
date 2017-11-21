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

package market

import (
	"github.com/robfig/cron"
	"github.com/patrickmn/go-cache"
)

type Account struct {
	contractVersion string
	tokens []Balance
}

type Balance struct {
	token string
	balance string
	allowance string
}

type AccountManager struct {
	c             *cache.Cache
	cacheReady    bool
	cron		  *cron.Cron
}

func(a *AccountManager) getBalance(address string) Account {
	return Account{}
}

func(a *AccountManager) getNonceFromAccessor(address string) {

}

func(a *AccountManager) getCutoff(address string) {

}

func(a *AccountManager) getBalanceFromAccessor(address string) {

}



