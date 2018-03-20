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
	"errors"
	rcache "github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/patrickmn/go-cache"
	"math/big"
	"strings"
)

var RedisCachePlaceHolder = make([]byte, 0)

const DefaultUnlockTtl = 3600 * 24 * 30
const UnlockCachePreKey = "Unlocked_Address_"

type Account struct {
	Address    string
	Balances   map[string]Balance
	Allowances map[string]Allowance
}

type Balance struct {
	Token   string
	Balance *big.Int
}

type Allowance struct {
	//contractVersion string
	token     string
	allowance *big.Int
}

type AccountManager struct {
	c                 *cache.Cache
	newestBlockNumber types.Big
}

type Token struct {
	Token     string `json:"symbol"`
	Balance   string `json:"balance"`
	Allowance string `json:"allowance"`
}

type AccountJson struct {
	ContractVersion string  `json:"contractVersion"`
	Address         string  `json:"owner"`
	Tokens          []Token `json:"tokens"`
}

func NewAccountManager() AccountManager {

	accountManager := AccountManager{}
	var blockNumber types.Big
	err := ethaccessor.BlockNumber(&blockNumber)
	if err != nil {
		log.Fatal("init account manager failed, can't get newest block number")
		return accountManager
	}
	accountManager.newestBlockNumber = blockNumber
	accountManager.c = cache.New(cache.NoExpiration, cache.NoExpiration)
	transferWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.HandleTokenTransfer}
	approveWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.HandleApprove}
	wethDepositWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.HandleWethDeposit}
	wethWithdrawalWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.HandleWethWithdrawal}
	eventemitter.On(eventemitter.AccountTransfer, transferWatcher)
	eventemitter.On(eventemitter.AccountApproval, approveWatcher)
	eventemitter.On(eventemitter.WethDepositMethod, wethDepositWatcher)
	eventemitter.On(eventemitter.WethWithdrawalMethod, wethWithdrawalWatcher)

	return accountManager
}

func (a *AccountManager) GetBalance(contractVersion, address string) Account {
	address = strings.ToLower(address)
	accountInCache, ok := a.c.Get(address)
	if ok {
		account := accountInCache.(Account)
		return account
	} else {
		account := Account{Address: address, Balances: make(map[string]Balance), Allowances: make(map[string]Allowance)}
		for k, v := range util.AllTokens {
			balance := Balance{Token: k}

			amount, err := a.GetBalanceFromAccessor(v.Symbol, address)
			if err != nil {
				log.Infof("get balance failed, token:%s", v.Symbol)
			} else {
				balance.Balance = amount
				account.Balances[k] = balance
			}

			allowance := Allowance{
				//contractVersion: contractVersion,
				token: k}

			allowanceAmount, err := a.GetAllowanceFromAccessor(v.Symbol, address, contractVersion)
			if err != nil {
				log.Errorf("get allowance failed, token:%s, address:%s, spender:%s", v.Symbol, address, contractVersion)
			} else {
				allowance.allowance = allowanceAmount
				account.Allowances[buildAllowanceKey(contractVersion, k)] = allowance
			}

		}
		a.c.Set(address, account, cache.NoExpiration)
		return account
	}
}

func (a *AccountManager) GetBalanceByTokenAddress(address common.Address, token common.Address) (balance, allowance *big.Int, err error) {

	tokenAlias := util.AddressToAlias(token.Hex())
	if tokenAlias == "" {
		err = errors.New("unsupported token address " + token.Hex())
		return
	}

	//todo(xiaolu): 从配置文件中获取
	account := a.GetBalance("v1.2", address.Hex())
	balance = account.Balances[tokenAlias].Balance
	allowance = account.Allowances[tokenAlias].allowance
	return
}

func (a *AccountManager) GetCutoff(contract, address string) (int, error) {
	//todo:stringtoaddress???
	//cutoffTime, err := ethaccessor.GetCutoff("latest", common.StringToAddress(contract), common.StringToAddress(address), "latest")
	cutoffTime, err := ethaccessor.GetCutoff(common.StringToAddress(contract), common.StringToAddress(address), "latest")
	return int(cutoffTime.Int64()), err
}

func (a *AccountManager) HandleTokenTransfer(input eventemitter.EventData) (err error) {
	event := input.(*types.TransferEvent)

	//log.Info("received transfer event...")

	if event.BlockNumber.Cmp(a.newestBlockNumber.BigInt()) < 0 {
		log.Info("the eth network may be forked. flush all cache")
		a.c.Flush()
		a.newestBlockNumber = *types.NewBigPtr(big.NewInt(-1))
	} else {
		tokenAlias := util.AddressToAlias(event.Protocol.Hex())
		errFrom := a.updateBalanceAndAllowance(tokenAlias, event.From.Hex())
		if errFrom != nil {
			return errFrom
		}
		errTo := a.updateBalanceAndAllowance(tokenAlias, event.To.Hex())
		if errTo != nil {
			return errTo
		}
	}
	return nil
}

func (a *AccountManager) HandleApprove(input eventemitter.EventData) (err error) {

	event := input.(*types.ApprovalEvent)
	log.Debugf("received approval event, %s, %s", event.Protocol.Hex(), event.Owner.Hex())
	if event.BlockNumber.Cmp(a.newestBlockNumber.BigInt()) < 0 {
		log.Info("the eth network may be forked. flush all cache")
		a.c.Flush()
		a.newestBlockNumber = *types.NewBigPtr(big.NewInt(-1))
	} else {
		if err = a.updateAllowance(*event); nil != err {
			log.Error(err.Error())
		}
	}
	return
}

func (a *AccountManager) HandleWethDeposit(input eventemitter.EventData) (err error) {
	event := input.(*types.WethDepositMethodEvent)
	if event.BlockNumber.Cmp(a.newestBlockNumber.BigInt()) < 0 {
		log.Info("the eth network may be forked. flush all cache")
		a.c.Flush()
		a.newestBlockNumber = *types.NewBigPtr(big.NewInt(-1))
	} else {
		if err = a.updateWethBalanceByDeposit(*event); nil != err {
			log.Error(err.Error())
		}
	}
	return
}

func (a *AccountManager) HandleWethWithdrawal(input eventemitter.EventData) (err error) {
	event := input.(*types.WethWithdrawalMethodEvent)
	if event.BlockNumber.Cmp(a.newestBlockNumber.BigInt()) < 0 {
		log.Info("the eth network may be forked. flush all cache")
		a.c.Flush()
		a.newestBlockNumber = *types.NewBigPtr(big.NewInt(-1))
	} else {
		if err = a.updateWethBalanceByWithdrawal(*event); nil != err {
			log.Error(err.Error())
		}
	}
	return
}

func (a *AccountManager) GetBalanceFromAccessor(token string, owner string) (*big.Int, error) {
	return ethaccessor.Erc20Balance(util.AllTokens[token].Protocol, common.HexToAddress(owner), "latest")
}

func (a *AccountManager) GetAllowanceFromAccessor(token, owner, spender string) (*big.Int, error) {
	spenderAddress, err := ethaccessor.GetSpenderAddress(common.HexToAddress(util.ContractVersionConfig[spender]))
	if err != nil {
		return big.NewInt(0), errors.New("invalid spender address")
	}
	return ethaccessor.Erc20Allowance(util.AllTokens[token].Protocol, common.HexToAddress(owner), spenderAddress, "latest")
}

func buildAllowanceKey(version, token string) string {
	//return version + "_" + token
	return token
}

func (a *AccountManager) updateBalanceAndAllowance(tokenAlias, address string) error {

	address = strings.ToLower(address)

	if tokenAlias == "" {
		return errors.New("unsupported token type : " + tokenAlias)
	}

	v, ok := a.c.Get(address)
	if ok {
		account := v.(Account)
		balance := Balance{Token: tokenAlias}
		amount, err := a.GetBalanceFromAccessor(tokenAlias, address)
		if err != nil {
			log.Error("get balance failed from accessor")
			return err
		}
		balance.Balance = amount
		account.Balances[tokenAlias] = balance
		allowanceAmount, err := a.GetAllowanceFromAccessor(tokenAlias, address, "v1.0")
		if err != nil {
			log.Error("get allowance failed from accessor")
			return err
		}
		allowance := Allowance{token: tokenAlias, allowance: allowanceAmount}
		account.Allowances[tokenAlias] = allowance
		a.c.Set(address, account, cache.NoExpiration)
	}
	return nil
}

func (a *AccountManager) updateWethBalance(address string) error {
	tokenAlias := "WETH"
	address = strings.ToLower(address)
	v, ok := a.c.Get(address)
	if ok {
		account := v.(Account)
		balance := Balance{Token: tokenAlias}
		amount, err := a.GetBalanceFromAccessor(tokenAlias, address)
		if err != nil {
			log.Error("get balance failed from accessor")
		} else {
			balance.Balance = amount
		}
		account.Balances[tokenAlias] = balance
		a.c.Set(address, account, cache.NoExpiration)
	}
	return nil
}

func (a *AccountManager) updateWethBalanceByDeposit(event types.WethDepositMethodEvent) error {
	return a.updateWethBalance(event.From.Hex())
}

func (a *AccountManager) updateWethBalanceByWithdrawal(event types.WethWithdrawalMethodEvent) error {
	return a.updateWethBalance(event.From.Hex())
}

func (a *AccountManager) updateAllowance(event types.ApprovalEvent) error {
	tokenAlias := util.AddressToAlias(event.Protocol.String())
	spender := event.Spender.String()
	address := strings.ToLower(event.Owner.String())

	// 这里只能根据loopring的合约获取了
	spenderAddress, err := ethaccessor.GetSpenderAddress(common.HexToAddress(util.ContractVersionConfig["v1.0"]))
	if err != nil {
		return errors.New("invalid spender address")
	}

	if strings.ToLower(spenderAddress.Hex()) != strings.ToLower(event.Spender.Hex()) {
		return errors.New("unsupported contract address : " + spender)
	}

	v, ok := a.c.Get(address)
	if ok {
		account := v.(Account)
		allowance := Allowance{
			//contractVersion: spender,
			token:     tokenAlias,
			allowance: event.Value}
		account.Allowances[buildAllowanceKey(spender, tokenAlias)] = allowance
		a.c.Set(address, account, cache.NoExpiration)
	} else {
		log.Debugf("can't get balance  by address : %s ", address)
	}
	return nil
}

func (account *Account) ToJsonObject(contractVersion string) AccountJson {

	var accountJson AccountJson
	accountJson.Address = account.Address
	accountJson.ContractVersion = contractVersion
	accountJson.Tokens = make([]Token, 0)
	for _, v := range account.Balances {
		allowance := account.Allowances[buildAllowanceKey(contractVersion, v.Token)]
		accountJson.Tokens = append(accountJson.Tokens, Token{v.Token, v.Balance.String(), allowance.allowance.String()})
	}
	return accountJson
}

func (a *AccountManager) UnlockedWallet(owner string) (err error) {
	if len(owner) == 0 {
		return errors.New("owner can't be null string")
	}
	return rcache.Set(UnlockCachePreKey+strings.ToLower(owner), RedisCachePlaceHolder, DefaultUnlockTtl)
}

func (a *AccountManager) HasUnlocked(owner string) (exists bool, err error) {
	// todo(fuk): delete after test
	// return true, nil

	if len(owner) == 0 {
		return false, errors.New("owner can't be null string")
	}
	return rcache.Exists(UnlockCachePreKey + strings.ToLower(owner))
}
