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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
	"encoding/json"
	"github.com/Loopring/relay/market/util"
)

//balance address token {lastBlock:,amount:}
//
//allowance address token {lastBlock:,,amount:}

const (
	BalancePrefix = "balance_"
	AllowancePrefix = "allowance_"
	CustomTokens = "customtoken_"
)

type AccountBase struct {
	Owner common.Address
	CustomTokens []types.Token
}

type Balance struct {
	LastBlock *types.Big `json:"last_block"`
	Balance *types.Big `json:"balance"`
}

type Allowance struct {
	LastBlock *types.Big `json:"last_block"`
	Allowance *types.Big `json:"allowance"`
}

type AccountBalances struct {
	AccountBase
	Balances map[common.Address]Balance
}

type Balances map[common.Address]Balance

func (b AccountBalances) cacheKey() string {
	return BalancePrefix + strings.ToLower(b.Owner.Hex())
}

func (b AccountBalances) cacheField(token common.Address) []byte {
	return []byte(strings.ToLower(token.Hex()))
}

func (b AccountBalances) cacheFieldToAddress(field []byte) common.Address {
	return common.HexToAddress(string(field))
}

//todo:tokens
func (b AccountBalances) batchReqs(tokens ...common.Address) ethaccessor.BatchBalanceReqs {
	reqs := ethaccessor.BatchBalanceReqs{}
	for _, token := range util.AllTokens {
		req := &ethaccessor.BatchBalanceReq{}
		req.BlockParameter = "latest"
		req.Token = token.Protocol
		req.Owner = b.Owner
		reqs = append(reqs, req)
	}
	for _,token := range b.CustomTokens {
		req := &ethaccessor.BatchBalanceReq{}
		req.BlockParameter = "latest"
		req.Token = token.Protocol
		req.Owner = b.Owner
		reqs = append(reqs, req)
	}
	req := &ethaccessor.BatchBalanceReq{}
	req.BlockParameter = "latest"
	req.Token = common.HexToAddress("0x")
	req.Owner = b.Owner
	reqs = append(reqs, req)
	return reqs
}

func (accountBalances AccountBalances) save() error {
	data := [][]byte{}
	for token,balance := range accountBalances.Balances {
		if balanceData,err := json.Marshal(balance);nil == err {
			data = append(data, accountBalances.cacheField(token), balanceData)
		} else {
			log.Errorf("accountmanager er:%s", err.Error())
		}
	}
	return rcache.HMSet(accountBalances.cacheKey(), data...)
}
func (accountBalances AccountBalances) applyData(cachedFieldData, balanceData []byte) error {
	if len(balanceData) <= 0 {
		return errors.New("not in cache")
	} else {
		key := accountBalances.cacheFieldToAddress(cachedFieldData)
		balance := Balance{}
		if err := json.Unmarshal(balanceData, &balance); nil != err {
			log.Errorf("accountmanager, syncFromCache err:%s", err.Error())
			return err
		} else {
			accountBalances.Balances[key] = balance
		}
		return nil
	}
}
func (accountBalances AccountBalances) syncFromCache(tokens ...common.Address) error {
	if len(tokens) > 0 {
		tokensBytes := [][]byte{}
		for _,token := range tokens {
			tokensBytes = append(tokensBytes, accountBalances.cacheField(token))
		}
		if balancesData,err := rcache.HMGet(accountBalances.cacheKey(), tokensBytes...);nil != err {
			return err
		} else {
			if len(balancesData) > 0 {
				for idx,data := range balancesData {
					if err := accountBalances.applyData(tokensBytes[idx], data); nil != err {
						return err
					}
				}
			} else {
				return errors.New("this address not in cache")
			}
		}
	} else {
		if balancesData,err := rcache.HGetAll(accountBalances.cacheKey()); nil != err {
			return err
		} else {
			if len(balancesData) > 0 {
				i := 0
				for i < len(balancesData) {
					accountBalances.applyData(balancesData[i], balancesData[i+1])
					i = i + 2;
				}
			} else {
				return errors.New("this address not in cache")
			}
		}
	}

	return nil
}

func (accountBalances AccountBalances) syncFromEthNode(tokens ...common.Address) error {
	reqs := accountBalances.batchReqs(tokens...)
	if err := ethaccessor.BatchCall("latest", []ethaccessor.BatchReq{reqs}); nil != err {
		return err
	}
	for _,req := range reqs {
		if nil != req.BalanceErr {
			log.Errorf("get balance failed, owner:%s, token:%s", req.Owner.Hex(), req.Token.Hex())
		} else {
			balance := Balance{}
			balance.Balance = &req.Balance
			//balance.LastBlock =
			accountBalances.Balances[req.Token] = balance
		}
	}
	return nil
}

func (accountBalances AccountBalances) getOrSave(tokens ...common.Address) error {
	if err := accountBalances.syncFromCache(tokens...); nil != err {
		if err := accountBalances.syncFromEthNode(tokens...); nil != err {
			return err
		} else {
			go accountBalances.save()
		}
	}
	return nil
}

type AccountAllowances struct {
	AccountBase
	Allowances map[common.Address]map[common.Address]Allowance
}

func (accountAllowances *AccountAllowances) cacheKey() string {
	return AllowancePrefix + strings.ToLower(accountAllowances.Owner.Hex())
}

func (accountAllowances *AccountAllowances) cacheField(token common.Address, spender common.Address) []byte {
	return []byte(strings.ToLower(token.Hex() + spender.Hex()))
}

func (accountAllowances *AccountAllowances) cacheFieldToToken(data []byte) (token common.Address, spender common.Address) {
	return common.BytesToAddress(data[0:43]), common.BytesToAddress(data[42:])
}

//todo:tokens
func (accountAllowances *AccountAllowances) batchReqs(tokens ...common.Address) ethaccessor.BatchErc20AllowanceReqs {
	reqs := ethaccessor.BatchErc20AllowanceReqs{}
	for _, v := range util.AllTokens {
		for _,impl := range ethaccessor.ProtocolAddresses() {
			req := &ethaccessor.BatchErc20AllowanceReq{}
			req.BlockParameter = "latest"
			req.Spender = impl.DelegateAddress
			req.Token = v.Protocol
			req.Owner = accountAllowances.Owner
			reqs = append(reqs, req)
		}
	}
	for _, v := range accountAllowances.CustomTokens {
		for _,impl := range ethaccessor.ProtocolAddresses() {
			req := &ethaccessor.BatchErc20AllowanceReq{}
			req.BlockParameter = "latest"
			req.Spender = impl.DelegateAddress
			req.Token = v.Protocol
			req.Owner = accountAllowances.Owner
			reqs = append(reqs, req)
		}
	}
	return reqs
}

func (accountAllowances *AccountAllowances) save() error {
	data := [][]byte{}
	for token,spenderMap := range accountAllowances.Allowances {
		for spender, allowance := range spenderMap {
			if allowanceData,err := json.Marshal(allowance);nil == err {
				data = append(data, accountAllowances.cacheField(token, spender), allowanceData)
			} else {
				log.Errorf("accountmanager allowance.save err:%s", err.Error())
			}
		}
	}
	return rcache.HMSet(accountAllowances.cacheKey(), data...)
}

func (accountAllowances *AccountAllowances) applyData(cacheFieldData,allowanceData []byte) error {
	if len(allowanceData) <= 0 {
		return errors.New("invalid allowanceData")
	} else {
		allowance := Allowance{}
		if err := json.Unmarshal(allowanceData, &allowance); nil != err {
			log.Errorf("accountmanager syncFromCache err:%s", err.Error())
			return err
		} else {
			token,spender := accountAllowances.cacheFieldToToken(cacheFieldData)
			if _,exists := accountAllowances.Allowances[token]; !exists {
				accountAllowances.Allowances = make(map[common.Address]map[common.Address]Allowance)
			} else if _,exists1 := accountAllowances.Allowances[token][spender]; !exists1 {
				accountAllowances.Allowances[token] = make(map[common.Address]Allowance)
			}
			accountAllowances.Allowances[token][spender] = allowance
		}
	}
	return nil
}

func (accountAllowances *AccountAllowances) syncFromCache(tokens ...common.Address) error {
	if len(tokens) > 0 {
		tokensBytes := [][]byte{}
		for _,token := range tokens {
			for _,impl := range ethaccessor.ProtocolAddresses() {
				tokensBytes = append(tokensBytes, accountAllowances.cacheField(token, impl.DelegateAddress))
				println(string(tokensBytes[len(tokensBytes)-1]))
			}
		}
		if allowanceData,err := rcache.HMGet(accountAllowances.cacheKey(), tokensBytes...);nil != err {
			return err
		} else {
			if len(allowanceData) > 0 {
				for idx,data := range allowanceData {
					if err := accountAllowances.applyData(tokensBytes[idx], data); nil != err {
						return err
					}
				}
			} else {
				return errors.New("this address not in cache")
			}
		}
	} else {
		if allowanceData,err := rcache.HGetAll(accountAllowances.cacheKey()); nil != err {
			return err
		} else {
			if len(allowanceData) > 0 {
				i := 0
				for i < len(allowanceData) {
					if err := accountAllowances.applyData(allowanceData[i], allowanceData[i + 1]);nil != err {
						return err
					}
					i = i + 2;
				}
			} else {
				println("####",accountAllowances.cacheKey(),len(allowanceData))
				return errors.New("this address not in cache")
			}
		}
	}
	return nil
}

func (accountAllowances *AccountAllowances) syncFromEthNode(tokens ...common.Address) error {
	reqs := accountAllowances.batchReqs(tokens...)
	if err := ethaccessor.BatchCall("latest", []ethaccessor.BatchReq{reqs}); nil != err {
		return err
	}
	for _,req := range reqs {
		if nil != req.AllowanceErr {
			log.Errorf("get balance failed, owner:%s, token:%s", req.Owner.Hex(), req.Token.Hex())
		} else {
			allowance := Allowance{}
			allowance.Allowance = &req.Allowance
			//balance.LastBlock =
			if _,exists := accountAllowances.Allowances[req.Token]; !exists {
				accountAllowances.Allowances[req.Token] = make(map[common.Address]Allowance)
			}
			accountAllowances.Allowances[req.Token][req.Spender] = allowance
		}
	}

	return nil
}

func (accountAllowances *AccountAllowances) getOrSave(tokens ...common.Address) error {
	if err := accountAllowances.syncFromCache(tokens...); nil != err {
		if err := accountAllowances.syncFromEthNode(tokens...); nil != err {
			return err
		} else {
			go accountAllowances.save()
		}
	}
	return nil
}

//type Account struct {
//	Address    common.Address
//	Balances   Balances
//	Allowances Allowances
//}
//
//func (a Account) AllowanceKey() string {
//	return Allowance + strings.ToLower(a.Address.Hex())
//}
//
//func (a Account) BalanceKey() string {
//	return Balance + strings.ToLower(a.Address.Hex())
//}
//
//func (a Account) SaveBalances() error {
//	data := []byte{}
//	for token,balance := range a.Balances {
//		if balanceData,err := json.Marshal(balance);nil == err {
//			data = append(data, token.Bytes(), balanceData)
//		} else {
//			log.Errorf("accountmanager er:%s", err.Error())
//		}
//	}
//	return rcache.HMSet(a.BalanceKey(), data)
//}
//
//func (a Account) SyncAndSaveBalances() error {
//
//	reqs := a.Balances.batchReqs(a.Address)
//
//	if err := ethaccessor.BatchCall("latest", reqs); nil != err {
//		return err
//	}
//	for _,req := range reqs {
//		if nil != req.BalanceErr {
//			log.Errorf("get balance failed, token:%s", req.Token.Hex())
//		} else {
//			a.Balances[req.Token] = req.Balance
//		}
//	}
//	return nil
//}
//
//func (a Account) SyncAndSaveAllowances() error {
//	reqs := a.Allowances.batchReqs(a.Address)
//
//	if err := ethaccessor.BatchCall("latest", reqs); nil != err {
//		return err
//	}
//	for _,req := range reqs {
//		if nil != req.AllowanceErr {
//			log.Errorf("get allowance failed, token:%s", req.Token.Hex())
//		} else {
//			a.Balances[req.Token] = req.Allowance
//		}
//	}
//	return nil
//}
//
//func (a Account) GetBalancesFromCache() string {
//	return Balance + strings.ToLower(a.Address.Hex())
//}
//
//func (a Account) GetAllowancesFromCache() string {
//	return Balance + strings.ToLower(a.Address.Hex())
//}

type ChangedOfBlock struct {
	currentBlockNumber uint64
	Balances map[common.Address]map[common.Address]bool
	Allowances map[common.Address]map[common.Address]map[common.Address]bool
}

func (b ChangedOfBlock) batchBalanceReqs() ethaccessor.BatchBalanceReqs {
	reqs := ethaccessor.BatchBalanceReqs{}
	for owner,tokens := range b.Balances {
		for token,_ := range tokens {
			req := &ethaccessor.BatchBalanceReq{}
			req.BlockParameter = "latest"
			req.Token = token
			req.Owner = owner
			reqs = append(reqs, req)
		}
	}
	return reqs
}

func (b ChangedOfBlock) batchAllowanceReqs() ethaccessor.BatchErc20AllowanceReqs {
	reqs := ethaccessor.BatchErc20AllowanceReqs{}
	for owner,tokens := range b.Allowances {
		for token,spenders := range tokens {
			for spender,_ := range spenders {
				req := &ethaccessor.BatchErc20AllowanceReq{}
				req.BlockParameter = "latest"
				req.Spender = spender
				req.Token = token
				req.Owner = owner
				reqs = append(reqs, req)
			}
		}
	}
	return reqs
}

type AccountManager struct {
	cacheDuration uint64

	maxBlockLength uint64
	block ChangedOfBlock
}


type BalanceJson struct {
	Symbol  string
	Balance *big.Int
}

type AllowanceJson struct {
	//contractVersion string
	Symbol    string
	Allowance *big.Int
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
	accountManager.cacheDuration = 3600 * 24 * 30
	accountManager.maxBlockLength = 3000
	b := ChangedOfBlock{}
	b.Balances = make(map[common.Address]map[common.Address]bool)
	b.Allowances = make(map[common.Address]map[common.Address]map[common.Address]bool)
	accountManager.block = b

	transferWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleTokenTransfer}
	approveWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleApprove}
	wethDepositWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleWethDeposit}
	wethWithdrawalWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleWethWithdrawal}
	blockForkWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleBlockFork}
	blockEndWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleBlockEnd}
	ethTransferWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleEthTransfer}
	eventemitter.On(eventemitter.AccountTransfer, transferWatcher)
	eventemitter.On(eventemitter.AccountApproval, approveWatcher)
	eventemitter.On(eventemitter.ethtransfer, ethTransferWatcher)
	eventemitter.On(eventemitter.block_end, blockEndWatcher)
	eventemitter.On(eventemitter.WethDepositEvent, wethDepositWatcher)
	eventemitter.On(eventemitter.WethWithdrawalEvent, wethWithdrawalWatcher)
	eventemitter.On(eventemitter.ChainForkDetected, blockForkWatcher)

	return accountManager
}

func (a *AccountManager) GetBalance(owner common.Address) (map[string]BalanceJson, error) {
	accountBalances := AccountBalances{}
	accountBalances.Owner = owner
	accountBalances.Balances = make(map[common.Address]Balance)

	res := make(map[string]BalanceJson)
	err := accountBalances.getOrSave(common.HexToAddress("0x1fa02762bd046abd30f5bf3513f9347d5e6b4257"), common.HexToAddress("0x"), common.HexToAddress("0x3cbcee9ff904ee0351b0ff2c05e08e860c94a5ea"))

	if nil == err {
		for tokenAddr,balance := range accountBalances.Balances {
			b := BalanceJson{}
			b.Balance = balance.Balance.BigInt()
			if types.IsZeroAddress(tokenAddr) {
				b.Symbol = "ETH"
			} else {
				b.Symbol = util.AddressToAlias(tokenAddr.Hex())
			}
			res[b.Symbol] = b
		}
	}

	return res, err
}

func (a *AccountManager) GetAllowance(owner,spender common.Address) (map[string]AllowanceJson, error) {
	accountAllowances := &AccountAllowances{}
	accountAllowances.Owner = owner
	accountAllowances.Allowances = make(map[common.Address]map[common.Address]Allowance)

	res := make(map[string]AllowanceJson)
	//err := accountAllowances.getOrSave(common.HexToAddress("0x1fa02762bd046abd30f5bf3513f9347d5e6b4257"), common.HexToAddress("0x"), common.HexToAddress("0x3cbcee9ff904ee0351b0ff2c05e08e860c94a5ea"))
	err := accountAllowances.getOrSave()

	if nil == err {
		for tokenAddr,allowances := range accountAllowances.Allowances {
			b := AllowanceJson{}
			b.Allowance = allowances[spender].Allowance.BigInt()
			if types.IsZeroAddress(tokenAddr) {
				b.Symbol = "ETH"
			} else {
				b.Symbol = util.AddressToAlias(tokenAddr.Hex())
			}
			res[b.Symbol] = b
		}
	}

	return res, err
}

//func (a *AccountManager) GetBalanceAndAllowance(contractVersion, address string) (account Account, err error) {
//	if len(contractVersion) == 0 {
//		return account, errors.New("contract version must be applied")
//	}
//
//	address = strings.ToLower(address)
//	accountInCache, ok := a.c.Get(address)
//	if ok {
//		account := accountInCache.(Account)
//		return account, err
//	} else {
//		account := Account{Address: address, Balances: make(map[string]Balance), Allowances: make(map[string]Allowance), Lock: sync.Mutex{}}
//		reqs := []*ethaccessor.BatchErc20Req{}
//
//		spenderAddress, err := ethaccessor.GetSpenderAddress(common.HexToAddress(util.ContractVersionConfig[contractVersion]))
//		if nil != err {
//			return account, errors.New("invalid spender address")
//		}
//		for k, v := range util.AllTokens {
//			req := &ethaccessor.BatchErc20Req{}
//			req.BlockParameter = "latest"
//			req.Symbol = k
//			req.Owner = common.HexToAddress(address)
//
//			req.Spender = spenderAddress
//			req.Token = v.Protocol
//			reqs = append(reqs, req)
//
//			//balance := Balance{Token: k}
//			//
//			//amount, err := a.GetBalanceFromAccessor(v.Symbol, address)
//			//if err != nil {
//			//	log.Infof("get balance failed, token:%s", v.Symbol)
//			//} else {
//			//	balance.Balance = amount
//			//	account.Balances[k] = balance
//			//}
//			//
//			//allowance := Allowance{
//			//	//contractVersion: contractVersion,
//			//	token: k}
//			//
//			//allowanceAmount, err := a.GetAllowanceFromAccessor(v.Symbol, address, contractVersion)
//			//if err != nil {
//			//	log.Errorf("get allowance failed, token:%s, address:%s, spender:%s", v.Symbol, address, contractVersion)
//			//} else {
//			//	allowance.allowance = allowanceAmount
//			//	account.Allowances[buildAllowanceKey(contractVersion, k)] = allowance
//			//}
//		}
//		if err := ethaccessor.BatchErc20BalanceAndAllowance("latest", reqs); nil != err {
//			return account, err
//		}
//		for _,req := range reqs {
//			balance := Balance{Symbol: req.Symbol}
//			if nil != req.BalanceErr {
//				log.Errorf("get balance failed, token:%s", req.Symbol)
//			} else {
//				balance.Balance = req.Balance.BigInt()
//				account.Balances[req.Symbol] = balance
//			}
//			allowance := Allowance{ Symbol: req.Symbol }
//			if nil != req.AllowanceErr {
//				log.Errorf("get allowance failed, token:%s, address:%s, spender:%s", req.Symbol, address, contractVersion)
//			} else {
//				allowance.Allowance = req.Allowance.BigInt()
//				account.Allowances[buildAllowanceKey(contractVersion, req.Symbol)] = allowance
//			}
//		}
//
//		a.c.Set(address, account, cache.NoExpiration)
//		return account, nil
//	}
//}

func (a *AccountManager) GetBalanceByTokenAddress(address common.Address, token common.Address) (balance, allowance *big.Int, err error) {

	tokenAlias := util.AddressToAlias(token.Hex())
	if tokenAlias == "" {
		err = errors.New("unsupported token address " + token.Hex())
		return
	}

	//todo(xiaolu): 从配置文件中获取
	//account, _ := a.GetBalanceAndAllowance(a.defaultContractVersion, address.Hex())
	//balance = account.Balances[tokenAlias].Balance
	//allowance = account.Allowances[tokenAlias].allowance
	return
}

func (a *AccountManager) GetCutoff(contract, address string) (int, error) {
	//todo:stringtoaddress???
	//cutoffTime, err := ethaccessor.GetCutoff("latest", common.StringToAddress(contract), common.StringToAddress(address), "latest")
	cutoffTime, err := ethaccessor.GetCutoff(common.StringToAddress(contract), common.StringToAddress(address), "latest")
	return int(cutoffTime.Int64()), err
}

func (a *AccountManager) handleTokenTransfer(input eventemitter.EventData) (err error) {
	event := input.(*types.TransferEvent)

	//log.Info("received transfer event...")

	if event == nil || event.Status != types.TX_STATUS_SUCCESS {
		log.Info("received wrong status event, drop it")
		return nil
	}

	if _,exists := a.block.Balances[event.Sender]; !exists {
		a.block.Balances[event.Sender] = make(map[common.Address]bool)
	}
	a.block.Balances[event.Sender][event.Protocol] = true

	if _,exists := a.block.Balances[event.Receiver]; !exists {
		a.block.Balances[event.Receiver] = make(map[common.Address]bool)
	}
	a.block.Balances[event.Receiver][event.Protocol] = true

	return nil
}

func (a *AccountManager) handleApprove(input eventemitter.EventData) (error) {
	event := input.(*types.ApprovalEvent)
	log.Debugf("received approval event, %s, %s", event.Protocol.Hex(), event.Owner.Hex())
	if event == nil || event.Status != types.TX_STATUS_SUCCESS {
		log.Info("received wrong status event, drop it")
		return nil
	}

	if _,exists := a.block.Allowances[event.Owner]; !exists {
		a.block.Allowances[event.Owner] = make(map[common.Address]map[common.Address]bool)
	}
	if _,exists := a.block.Allowances[event.Owner][event.Protocol]; !exists {
		a.block.Allowances[event.Owner][event.Protocol] = make(map[common.Address]bool)
	}
	a.block.Allowances[event.Owner][event.Protocol][event.Spender] = true
	return nil
}

func (a *AccountManager) handleWethDeposit(input eventemitter.EventData) (err error) {
	event := input.(*types.WethDepositMethodEvent)
	if event == nil || event.Status != types.TX_STATUS_SUCCESS {
		log.Info("received wrong status event, drop it")
		return nil
	}

	if _,exists := a.block.Balances[event.Dst];!exists {
		a.block.Balances[event.Dst] = make(map[common.Address]bool)
	}
	a.block.Balances[event.Dst][event.Protocol] = true

	//if err = a.updateWethBalanceByDeposit(*event); nil != err {
	//	log.Error(err.Error())
	//}
	return
}

func (a *AccountManager) handleWethWithdrawal(input eventemitter.EventData) (err error) {
	event := input.(*types.WethWithdrawalMethodEvent)
	if event == nil || event.Status != types.TX_STATUS_SUCCESS {
		log.Info("received wrong status event, drop it")
		return nil
	}

	if _,exists := a.block.Balances[event.Src]; !exists {
		a.block.Balances[event.Src] = make(map[common.Address]bool)
	}
	a.block.Balances[event.Src][event.Protocol] = true

	//if err = a.updateWethBalanceByWithdrawal(*event); nil != err {
	//	log.Error(err.Error())
	//}
	return
}

func (a *AccountManager) GetBalanceFromAccessor(token string, owner string) (*big.Int, error) {
	rst, err := ethaccessor.Erc20Balance(util.AllTokens[token].Protocol, common.HexToAddress(owner), "latest")
	return rst, err

}

func (a *AccountManager) GetAllowanceFromAccessor(token, owner, spender string) (*big.Int, error) {
	spenderAddress, err := ethaccessor.GetSpenderAddress(common.HexToAddress(util.ContractVersionConfig[spender]))
	if err != nil {
		return big.NewInt(0), errors.New("invalid spender address")
	}
	rst, err := ethaccessor.Erc20Allowance(util.AllTokens[token].Protocol, common.HexToAddress(owner), spenderAddress, "latest")
	return rst, err
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

	//v, ok := a.c.Get(address)
	//if ok {
	//	account := v.(Account)
	//	balance := Balance{Symbol: tokenAlias}
	//	amount, err := a.GetBalanceFromAccessor(tokenAlias, address)
	//	if err != nil {
	//		log.Error("get balance failed from accessor")
	//		return err
	//	}
	//	balance.Balance = amount
	//	account.Balances[tokenAlias] = balance
	//	allowanceAmount, err := a.GetAllowanceFromAccessor(tokenAlias, address, a.defaultContractVersion)
	//	if err != nil {
	//		log.Error("get allowance failed from accessor")
	//		return err
	//	}
	//	allowance := Allowance{Symbol: tokenAlias, Allowance: allowanceAmount}
	//	account.Allowances[tokenAlias] = allowance
	//	a.c.Set(address, account, cache.NoExpiration)
	//}
	return nil
}

func (a *AccountManager) updateWethBalance(address string) error {
	//tokenAlias := "WETH"
	address = strings.ToLower(address)
	//v, ok := a.c.Get(address)
	//if ok {
	//	account := v.(Account)
	//	balance := Balance{Symbol: tokenAlias}
	//	amount, err := a.GetBalanceFromAccessor(tokenAlias, address)
	//	if err != nil {
	//		log.Error("get balance failed from accessor")
	//	} else {
	//		balance.Balance = amount
	//	}
	//	account.Balances[tokenAlias] = balance
	//	a.c.Set(address, account, cache.NoExpiration)
	//}
	return nil
}

func (a *AccountManager) handleBlockEnd(input eventemitter.EventData) error {
	return nil
}

func (a *AccountManager) handleEthTransfer(input eventemitter.EventData) error {
	return nil
}

func (a *AccountManager) updateWethBalanceByWithdrawal(event types.WethWithdrawalMethodEvent) error {
	return a.updateWethBalance(event.From.Hex())
}

func (a *AccountManager) updateAllowance(event types.ApprovalEvent) error {
	//tokenAlias := util.AddressToAlias(event.Protocol.String())
	//spender := event.Spender.String()
	//address := strings.ToLower(event.Owner.String())
	//
	//// 这里只能根据loopring的合约获取了
	//spenderAddress, err := ethaccessor.GetSpenderAddress(common.HexToAddress(util.ContractVersionConfig[a.defaultContractVersion]))
	//if err != nil {
	//	return errors.New("invalid spender address")
	//}
	//
	//if strings.ToLower(spenderAddress.Hex()) != strings.ToLower(event.Spender.Hex()) {
	//	return errors.New("unsupported contract address : " + spender)
	//}
	//
	//v, ok := a.c.Get(address)
	//if ok {
	//	account := v.(Account)
	//	allowance := Allowance{
	//		//contractVersion: spender,
	//		Symbol:     tokenAlias,
	//		Allowance: event.Value}
	//	account.Allowances[buildAllowanceKey(spender, tokenAlias)] = allowance
	//	a.c.Set(address, account, cache.NoExpiration)
	//} else {
	//	log.Debugf("can't get balance  by address : %s ", address)
	//}
	return nil
}

//func (account *Account) ToJsonObject(contractVersion string, ethBalance Balance) AccountJson {
//
//	var accountJson AccountJson
//	accountJson.Address = account.Address
//	accountJson.ContractVersion = contractVersion
//	accountJson.Tokens = make([]Token, 0)
//	for _, v := range account.Balances {
//		allowance := account.Allowances[buildAllowanceKey(contractVersion, v.Symbol)]
//		accountJson.Tokens = append(accountJson.Tokens, Token{v.Symbol, v.Balance.String(), allowance.Allowance.String()})
//	}
//	accountJson.Tokens = append(accountJson.Tokens, Token{ethBalance.Symbol, ethBalance.Balance.String(), "0"})
//	return accountJson
//}

func (a *AccountManager) UnlockedWallet(owner string) (err error) {
	if !common.IsHexAddress(owner) {
		return errors.New("owner isn't a valid hex-address")
	}

	accountBalances := AccountBalances{}
	accountBalances.Owner = common.HexToAddress(owner)
	return accountBalances.getOrSave()
}

func (a *AccountManager) HasUnlocked(owner string) (exists bool, err error) {
	if !common.IsHexAddress(owner) {
		return false,errors.New("owner isn't a valid hex-address")
	}
	accountBalances := AccountBalances{}
	accountBalances.Owner = common.HexToAddress(owner)

	return rcache.Exists(accountBalances.cacheKey())
}

func (a *AccountManager) handleBlockFork(event eventemitter.EventData) (err error) {
	log.Info("the eth network may be forked. flush all cache")
	//a.c.Flush()
	return nil
}
