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
	"github.com/patrickmn/go-cache"
	"github.com/Loopring/relay/eventemiter"
)

type Account struct {
	contractVersion string
	address string
	balances []Balance
	blockNumber int
}

type Balance struct {
	token string
	balance string
	allowance string
}

type AccountManager struct {
	c             *cache.Cache
}

func NewAccountManager() AccountManager {

	accountManager := AccountManager{}
	accountManager.c = cache.New(cache.NoExpiration, cache.NoExpiration)
	transferWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.HandleTokenTransfer}
	approveWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.HandleApprove}
	//TODO(xiaolu) change event type
	eventemitter.On(eventemitter.OrderManagerExtractorFill, transferWatcher)
	eventemitter.On(eventemitter.OrderManagerExtractorFill, approveWatcher)

	return accountManager
}

func(a *AccountManager) GetBalance(contractVersion, address string) Account {

	accountInCache, ok := a.c.Get(buildCacheKey(contractVersion, address))
	if ok {
		account := accountInCache.(Account)
		return account
	} else {
		account := Account{contractVersion:contractVersion, address:address, balances:make([]Balance, 0), blockNumber:-1}
		for k, v := range AllTokens {
			amount := a.getBalanceFromAccessor(v)
			balance := Balance{token:k, balance:amount}
			allowance := a.getAllowanceFromAccessor(v)
			balance.allowance = allowance
			account.balances = append(account.balances, balance)
		}
		a.c.Set(buildCacheKey(contractVersion, address), account, cache.NoExpiration)
		return account
	}
}

func(a *AccountManager) getCutoff(address string) {

}

func(a *AccountManager) HandleTokenTransfer(input eventemitter.EventData) (err error) {

}

func(a *AccountManager) HandleApprove(input eventemitter.EventData) (err error) {

}

func(a *AccountManager) getBalanceFromAccessor(token string) string {

	return ""
}

func(a *AccountManager) getAllowanceFromAccessor(token string) string {
	return ""
}

func(a *AccountManager) buildBalanceFromAccessor(token string) Balance {
	return Balance{}
}

func buildCacheKey(version, address string) string {
	return address + "_" + version
}
