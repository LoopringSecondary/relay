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
	"github.com/Loopring/relay/ethaccessor"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"errors"
)

type Account struct {
	address string
	balances map[string]Balance
	allowances map[string]Allowance
}

type Balance struct {
	token     string
	balance   *big.Int
}

type Allowance struct {
	contractVersion string
	token string
	allowance *big.Int
}

type AccountManager struct {
	c             *cache.Cache
	accessor *ethaccessor.EthNodeAccessor
	newestBlockNumber types.Big
}

type Token struct {
	token string
	balance string
	allowance string
}

type AccountJson struct {
	contractVersion string
	address string
	tokens []Token
}

func NewAccountManager(accessor *ethaccessor.EthNodeAccessor) AccountManager {

	accountManager := AccountManager{accessor:accessor}
	var blockNumber types.Big
	err := accessor.Call(&blockNumber, "eth_blockNumber")
	if err != nil {
		log.Fatal("init account manager failed, can't get newest block number")
		return accountManager
	}
	accountManager.newestBlockNumber = blockNumber
	accountManager.c = cache.New(cache.NoExpiration, cache.NoExpiration)
	transferWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.HandleTokenTransfer}
	approveWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.HandleApprove}
	eventemitter.On(eventemitter.AccountTransfer, transferWatcher)
	eventemitter.On(eventemitter.AccountApproval, approveWatcher)

	return accountManager
}

func(a *AccountManager) GetBalance(contractVersion, address string) Account {

	accountInCache, ok := a.c.Get(address)
	if ok {
		account := accountInCache.(Account)
		return account
	} else {
		account := Account{address:address, balances:make(map[string]Balance), allowances:make(map[string]Allowance)}
		for k, v := range AllTokens {
			balance := Balance{token:k}

			amount, err := a.GetBalanceFromAccessor(v, address)
			if err != nil {
				log.Info("get balance failed , token : " + v)
			} else {
				balance.balance = amount
				account.balances[k] = balance
			}

			allowance := Allowance{contractVersion:contractVersion, token:k}

			allowanceAmount, err := a.GetAllowanceFromAccessor(v, address, contractVersion)
			if err != nil {
				log.Info("get allowance failed , token : " + v)
			} else {
				allowance.allowance = allowanceAmount
				account.allowances[buildAllowanceKey(contractVersion, k)] = allowance
			}

		}
		a.c.Set(address, account, cache.NoExpiration)
		return account
	}
}

func (a *AccountManager) GetCutoff(contract, address string) (int , error) {
	return a.accessor.GetCutoff(common.StringToAddress(contract), common.StringToAddress(address), "latest")
}

func(a *AccountManager) HandleTokenTransfer(input eventemitter.EventData) (err error) {
	event := input.(types.TransferEvent)
	if event.Blocknumber.BigInt().Cmp(a.newestBlockNumber.BigInt()) < 0 {
		log.Info("the eth network may be forked. flush all cache")
		a.c.Flush()
		a.newestBlockNumber = *types.NewBigPtr(big.NewInt(-1))
	} else {
		a.updateBalance(event, true)
		a.updateBalance(event, false)
	}
	return nil
}

func(a *AccountManager) HandleApprove(input eventemitter.EventData) (err error) {
	event := input.(types.ApprovalEvent)
	if event.Blocknumber.BigInt().Cmp(a.newestBlockNumber.BigInt()) < 0 {
		log.Info("the eth network may be forked. flush all cache")
		a.c.Flush()
		a.newestBlockNumber = *types.NewBigPtr(big.NewInt(-1))
	} else {
		err = a.updateAllowance(event)
	}
	return
}

func(a *AccountManager) GetBalanceFromAccessor(token string, owner string) (*big.Int, error) {
	return a.accessor.Erc20Balance(common.StringToAddress(AllTokens[token]), common.StringToAddress(owner), "latest")
}

func(a *AccountManager) GetAllowanceFromAccessor(token, owner, spender string) (*big.Int, error) {
	return a.accessor.Erc20Allowance(common.StringToAddress(AllTokens[token]), common.StringToAddress(owner), common.StringToAddress(spender), "latest")
}

func buildAllowanceKey(version, token string) string {
	return version + "_" + token
}

func (a *AccountManager) updateBalance(event types.TransferEvent, isAdd bool) error {
	tokenAlias := AddressToAlias(event.ContractAddress.String())

	var address string
	if !isAdd {
		address = event.From.String()
	} else {
		address = event.To.String()
	}

	if tokenAlias == "" {
		return errors.New("unsupported token type : " + tokenAlias)
	}

	v, ok := a.c.Get(address)
	if ok {
		account := v.(Account)
		balance, ok := account.balances[tokenAlias]
		if !ok {
			balance = Balance{token:tokenAlias}
			amount, err := a.GetBalanceFromAccessor(event.ContractAddress.String(), tokenAlias)
			if err != nil {
				log.Error("get balance failed from accessor")
			} else {
				balance.balance = amount
			}
			account.balances[tokenAlias] = balance
		} else {
			oldBalance := balance.balance
			if isAdd {
				balance.balance = oldBalance.Sub(oldBalance, event.Value.BigInt())
			} else {
				balance.balance = oldBalance.Add(oldBalance, event.Value.BigInt())
			}
			account.balances[tokenAlias] = balance
		}
		a.c.Set(address, account, cache.NoExpiration)
	}
	return nil
}


func (a *AccountManager) updateAllowance(event types.ApprovalEvent) error {
	tokenAlias := AddressToAlias(event.ContractAddress.String())
	spender := event.Spender.String()
	address := event.Owner.String()

	if !IsSupportedContract(spender) {
		return errors.New("unsupported contract address : " + spender)
	}

	v, ok := a.c.Get(address)
	if ok {
		account := v.(Account)
		allowance := Allowance{contractVersion:spender, token:tokenAlias, allowance:event.Value.BigInt()}
		account.allowances[buildAllowanceKey(spender, tokenAlias)] = allowance
		a.c.Set(address, account, cache.NoExpiration)
	}
	return nil
}

func (account *Account) ToJsonObject(contractVersion string) AccountJson {

	var accountJson AccountJson
	accountJson.address = account.address
	accountJson.contractVersion = contractVersion
	accountJson.tokens = make([]Token, 0)
	for _, v := range account.balances {
		allowance := account.allowances[buildAllowanceKey(contractVersion, v.token)]
		accountJson.tokens = append(accountJson.tokens, Token{v.token, v.balance.String(), allowance.allowance.String()})
	}
	return accountJson
}
