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
	"encoding/json"
	"errors"
	rcache "github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
)

const (
	UnlockedPrefix    = "unlock_"
	BalancePrefix     = "balance_"
	AllowancePrefix   = "allowance_"
	CustomTokenPrefix = "customtoken_"
)

type AccountBase struct {
	Owner        common.Address
	CustomTokens []types.Token
}

type Balance struct {
	LastBlock *types.Big `json:"last_block"`
	Balance   *types.Big `json:"balance"`
}

type Allowance struct {
	LastBlock *types.Big `json:"last_block"`
	Allowance *types.Big `json:"allowance"`
}

type AccountBalances struct {
	AccountBase
	Balances map[common.Address]Balance
}

func balanceCacheKey(owner common.Address) string {
	return BalancePrefix + strings.ToLower(owner.Hex())
}

func unlockCacheKey(owner common.Address) string {
	return UnlockedPrefix + strings.ToLower(owner.Hex())
}

func balanceCacheField(token common.Address) []byte {
	return []byte(strings.ToLower(token.Hex()))
}

func parseCacheField(field []byte) common.Address {
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
	for _, token := range b.CustomTokens {
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

func (accountBalances AccountBalances) save(ttl int64) error {
	data := [][]byte{}
	for token, balance := range accountBalances.Balances {
		//log.Debugf("balance owner:%s, token:%s, amount:", accountBalances.Owner.Hex(), token.Hex(), balance.Balance.BigInt().String())
		if balanceData, err := json.Marshal(balance); nil == err {
			data = append(data, balanceCacheField(token), balanceData)
		} else {
			log.Errorf("accountmanager er:%s", err.Error())
		}
	}
	err := rcache.HMSet(balanceCacheKey(accountBalances.Owner), ttl, data...)
	return err
}

func (accountBalances AccountBalances) applyData(cachedFieldData, balanceData []byte) error {
	if len(balanceData) <= 0 {
		return errors.New("not in cache")
	} else {
		tokenAddress := parseCacheField(cachedFieldData)
		balance := Balance{}
		if err := json.Unmarshal(balanceData, &balance); nil != err {
			log.Errorf("accountmanager, syncFromCache err:%s", err.Error())
			return err
		} else {
			accountBalances.Balances[tokenAddress] = balance
		}
		return nil
	}
}

func (accountBalances AccountBalances) syncFromCache(tokens ...common.Address) error {
	missedTokens := []common.Address{}
	if len(tokens) > 0 {
		tokensBytes := [][]byte{}
		for _, token := range tokens {
			tokensBytes = append(tokensBytes, balanceCacheField(token))
		}
		if balancesData, err := rcache.HMGet(balanceCacheKey(accountBalances.Owner), tokensBytes...); nil != err {
			return err
		} else {
			if len(balancesData) > 0 {
				for idx, data := range balancesData {
					if len(data) > 0 {
						if err := accountBalances.applyData(tokensBytes[idx], data); nil != err {
							missedTokens = append(missedTokens, tokens[idx])
						}
					} else {
						missedTokens = append(missedTokens, tokens[idx])
						return errors.New("this address not in cache")
					}
				}
			} else {
				return errors.New("this address not in cache")
			}
		}
	} else {
		if balancesData, err := rcache.HGetAll(balanceCacheKey(accountBalances.Owner)); nil != err {
			return err
		} else {
			if len(balancesData) > 0 {
				idx := 0
				for idx < len(balancesData) {
					if err := accountBalances.applyData(balancesData[idx], balancesData[idx+1]); nil != err {
						return err
					}
					idx = idx + 2
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
	for _, req := range reqs {
		if nil != req.BalanceErr {
			log.Errorf("get balance failed, owner:%s, token:%s, err:%s", req.Owner.Hex(), req.Token.Hex(), req.BalanceErr.Error())
		} else {
			balance := Balance{}
			balance.Balance = &req.Balance
			//balance.LastBlock =
			accountBalances.Balances[req.Token] = balance
		}
	}
	return nil
}

func (accountBalances AccountBalances) getOrSave(ttl int64, tokens ...common.Address) error {
	if err := accountBalances.syncFromCache(tokens...); nil != err {
		if err := accountBalances.syncFromEthNode(tokens...); nil != err {
			return err
		} else {
			go accountBalances.save(ttl)
		}
	}
	return nil
}

type AccountAllowances struct {
	AccountBase
	Allowances map[common.Address]map[common.Address]Allowance //token -> spender
}

func allowanceCacheKey(owner common.Address) string {
	return AllowancePrefix + strings.ToLower(owner.Hex())
}

func allowanceCacheField(token common.Address, spender common.Address) []byte {
	return []byte(strings.ToLower(token.Hex() + spender.Hex()))
}

func parseAllowanceCacheField(data []byte) (token common.Address, spender common.Address) {
	return common.HexToAddress(string(data[0:42])), common.HexToAddress(string(data[42:]))
}

//todo:tokens
func (accountAllowances *AccountAllowances) batchReqs(tokens, spenders []common.Address) ethaccessor.BatchErc20AllowanceReqs {
	reqs := ethaccessor.BatchErc20AllowanceReqs{}
	for _, v := range util.AllTokens {
		for _, impl := range ethaccessor.ProtocolAddresses() {
			req := &ethaccessor.BatchErc20AllowanceReq{}
			req.BlockParameter = "latest"
			req.Spender = impl.DelegateAddress
			req.Token = v.Protocol
			req.Owner = accountAllowances.Owner
			reqs = append(reqs, req)
		}
	}
	for _, v := range accountAllowances.CustomTokens {
		for _, impl := range ethaccessor.ProtocolAddresses() {
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

func (accountAllowances *AccountAllowances) save(ttl int64) error {
	data := [][]byte{}
	for token, spenderMap := range accountAllowances.Allowances {
		for spender, allowance := range spenderMap {
			if allowanceData, err := json.Marshal(allowance); nil == err {
				data = append(data, allowanceCacheField(token, spender), allowanceData)
			} else {
				log.Errorf("accountmanager allowance.save err:%s", err.Error())
			}
		}
	}
	return rcache.HMSet(allowanceCacheKey(accountAllowances.Owner), ttl, data...)
}

func (accountAllowances *AccountAllowances) applyData(cacheFieldData, allowanceData []byte) error {
	if len(allowanceData) <= 0 {
		return errors.New("invalid allowanceData")
	} else {
		allowance := Allowance{}
		if err := json.Unmarshal(allowanceData, &allowance); nil != err {
			log.Errorf("accountmanager syncFromCache err:%s", err.Error())
			return err
		} else {
			token, spender := parseAllowanceCacheField(cacheFieldData)
			if _, exists := accountAllowances.Allowances[token]; !exists {
				accountAllowances.Allowances[token] = make(map[common.Address]Allowance)
			}
			accountAllowances.Allowances[token][spender] = allowance
		}
	}
	return nil
}

func generateAllowanceCahceFieldList(tokens, spenders []common.Address) [][]byte {
	fields := [][]byte{}
	for _, token := range tokens {
		if !types.IsZeroAddress(token) {
			for _, spender := range spenders {
				if !types.IsZeroAddress(spender) {
					fields = append(fields, allowanceCacheField(token, spender))
				}
			}
		}
	}
	return fields
}

func (accountAllowances *AccountAllowances) syncFromCache(tokens, spenders []common.Address) error {
	fields := generateAllowanceCahceFieldList(tokens, spenders)
	if len(fields) > 0 {
		if allowanceData, err := rcache.HMGet(allowanceCacheKey(accountAllowances.Owner), fields...); nil != err {
			return err
		} else {
			if len(allowanceData) > 0 {
				for idx, data := range allowanceData {
					if len(data) > 0 {
						if err := accountAllowances.applyData(fields[idx], data); nil != err {
							return err
						}
					} else {
						return errors.New("allowance of this address not in cache")
					}
				}
			} else {
				return errors.New("allowance of this address not in cache")
			}
		}
	} else {
		if allowanceData, err := rcache.HGetAll(allowanceCacheKey(accountAllowances.Owner)); nil != err {
			return err
		} else {
			if len(allowanceData) > 0 {
				i := 0
				for i < len(allowanceData) {
					if err := accountAllowances.applyData(allowanceData[i], allowanceData[i+1]); nil != err {
						return err
					}
					i = i + 2
				}
			} else {
				return errors.New("this address not in cache")
			}
		}
	}
	return nil
}

func (accountAllowances *AccountAllowances) syncFromEthNode(tokens, spenders []common.Address) error {
	reqs := accountAllowances.batchReqs(tokens, spenders)
	if err := ethaccessor.BatchCall("latest", []ethaccessor.BatchReq{reqs}); nil != err {
		return err
	}
	for _, req := range reqs {
		if nil != req.AllowanceErr {
			log.Errorf("get balance failed, owner:%s, token:%s, err:%s", req.Owner.Hex(), req.Token.Hex(), req.AllowanceErr.Error())
		} else {
			allowance := Allowance{}
			allowance.Allowance = &req.Allowance
			//balance.LastBlock =
			if _, exists := accountAllowances.Allowances[req.Token]; !exists {
				accountAllowances.Allowances[req.Token] = make(map[common.Address]Allowance)
			}
			accountAllowances.Allowances[req.Token][req.Spender] = allowance
		}
	}

	return nil
}

func (accountAllowances *AccountAllowances) getOrSave(ttl int64, tokens, spenders []common.Address) error {
	if err := accountAllowances.syncFromCache(tokens, spenders); nil != err {
		if err := accountAllowances.syncFromEthNode(tokens, spenders); nil != err {
			return err
		} else {
			go accountAllowances.save(ttl)
		}
	}
	return nil
}

type ChangedOfBlock struct {
	currentBlockNumber *big.Int
	cachedDuration     *big.Int
}

func (b *ChangedOfBlock) saveBalanceKey(owner, token common.Address) error {
	err := rcache.SAdd(b.cacheBalanceKey(), int64(0), b.cacheBalanceField(owner, token))
	if err == nil {
		eventemitter.Emit(eventemitter.BalanceUpdated, types.BalanceUpdateEvent{Owner: owner.Hex()})
	}
	return err
}

func (b *ChangedOfBlock) cacheBalanceKey() string {
	if nil == b.currentBlockNumber {
		log.Error("b.currentBlockNumber is nil")
	}
	return "block_balance_" + b.currentBlockNumber.String()
}

func (b *ChangedOfBlock) cacheBalanceField(owner, token common.Address) []byte {
	return append(owner.Bytes(), token.Bytes()...)
}
func (b *ChangedOfBlock) parseCacheBalanceField(data []byte) (owner, token common.Address) {
	return common.BytesToAddress(data[0:20]), common.BytesToAddress(data[20:])
}

func (b *ChangedOfBlock) cacheAllowanceKey() string {
	if nil == b.currentBlockNumber {
		log.Error("b.currentBlockNumber is nil")
	}
	return "block_allowance_" + b.currentBlockNumber.String()
}

func (b *ChangedOfBlock) cacheAllowanceField(owner, token, spender common.Address) []byte {
	return append(append(owner.Bytes(), token.Bytes()...), spender.Bytes()...)
}

func (b *ChangedOfBlock) parseCacheAllowanceField(data []byte) (owner, token, spender common.Address) {
	return common.BytesToAddress(data[0:20]), common.BytesToAddress(data[20:40]), common.BytesToAddress(data[40:])
}

func (b *ChangedOfBlock) saveAllowanceKey(owner, token, spender common.Address) error {
	err := rcache.SAdd(b.cacheAllowanceKey(), int64(0), b.cacheAllowanceField(owner, token, spender))
	if err == nil {
		eventemitter.Emit(eventemitter.BalanceUpdated, types.BalanceUpdateEvent{Owner: owner.Hex(), DelegateAddress: spender.Hex()})
	}
	return err
}

func removeExpiredBlock(blockNumber, duration *big.Int) error {
	nb := &ChangedOfBlock{}
	nb.currentBlockNumber = new(big.Int)
	nb.currentBlockNumber.Sub(blockNumber, duration)
	log.Debugf("removeExpiredBlock cacheAllowanceKey ")

	if err := rcache.Del(nb.cacheAllowanceKey()); nil != err {
		log.Errorf("removeExpiredBlock cacheAllowanceKey err:%s", err.Error())
	}
	if err := rcache.Del(nb.cacheBalanceKey()); nil != err {
		log.Errorf("removeExpiredBlock cacheBalanceKey err:%s", err.Error())
	}
	return nil
}

func (b *ChangedOfBlock) syncAndSaveBalances() error {
	reqs := b.batchBalanceReqs()
	if err := ethaccessor.BatchCall("latest", []ethaccessor.BatchReq{reqs}); nil != err {
		return err
	}
	accounts := make(map[common.Address]*AccountBalances)
	for _, req := range reqs {
		if nil != req.BalanceErr {
			log.Errorf("get balance failed, owner:%s, token:%s, err:%s", req.Owner.Hex(), req.Token.Hex(), req.BalanceErr.Error())
		} else {
			if _, exists := accounts[req.Owner]; !exists {
				accounts[req.Owner] = &AccountBalances{}
				accounts[req.Owner].Owner = req.Owner
				accounts[req.Owner].Balances = make(map[common.Address]Balance)
			}
			balance := Balance{}
			balance.LastBlock = types.NewBigPtr(b.currentBlockNumber)
			balance.Balance = &req.Balance
			accounts[req.Owner].Balances[req.Token] = balance
		}
	}
	for _, balances := range accounts {
		balances.save(int64(0))
	}

	return nil
}

func (b *ChangedOfBlock) batchBalanceReqs() ethaccessor.BatchBalanceReqs {
	reqs := ethaccessor.BatchBalanceReqs{}
	if balancesData, err := rcache.SMembers(b.cacheBalanceKey()); nil == err && len(balancesData) > 0 {
		for _, data := range balancesData {
			accountAddr, token := b.parseCacheBalanceField(data)
			//log.Debugf("1---batchBalanceReqsbatchBalanceReqsbatchBalanceReqs:%s,%s", accountAddr.Hex(), token.Hex())
			if exists, err := rcache.Exists(balanceCacheKey(accountAddr)); nil == err && exists {
				//log.Debugf("2---batchBalanceReqsbatchBalanceReqsbatchBalanceReqs:%s,%s", accountAddr.Hex(), token.Hex())
				if exists1, err1 := rcache.HExists(balanceCacheKey(accountAddr), balanceCacheField(token)); nil == err1 && exists1 {
					log.Debugf("3---batchBalanceReqsbatchBalanceReqsbatchBalanceReqs:%s,%s", accountAddr.Hex(), token.Hex())
					req := &ethaccessor.BatchBalanceReq{}
					req.Owner = accountAddr
					req.Token = token
					req.BlockParameter = "latest"
					reqs = append(reqs, req)
				}
			}
		}
	}
	return reqs
}

func (b *ChangedOfBlock) batchAllowanceReqs() ethaccessor.BatchErc20AllowanceReqs {
	reqs := ethaccessor.BatchErc20AllowanceReqs{}
	if allowancesData, err := rcache.SMembers(b.cacheAllowanceKey()); nil == err && len(allowancesData) > 0 {
		for _, data := range allowancesData {
			owner, token, spender := b.parseCacheAllowanceField(data)
			//log.Debugf("1---batchAllowanceReqs owner:%s, t:%s, s:%s", owner.Hex(), token.Hex(), spender.Hex())
			if ethaccessor.IsSpenderAddress(spender) {
				if exists, err := rcache.Exists(balanceCacheKey(owner)); nil == err && exists {
					//log.Debugf("2---batchAllowanceReqs owner:%s, t:%s, s:%s", owner.Hex(), token.Hex(), spender.Hex())
					if exists1, err1 := rcache.HExists(allowanceCacheKey(owner), allowanceCacheField(token, spender)); nil == err1 && exists1 {
						log.Debugf("3---batchAllowanceReqs owner:%s, t:%s, s:%s", owner.Hex(), token.Hex(), spender.Hex())
						req := &ethaccessor.BatchErc20AllowanceReq{}
						req.BlockParameter = "latest"
						req.Spender = spender
						req.Token = token
						req.Owner = owner
						reqs = append(reqs, req)
					}
				}
			}

		}
	}
	return reqs
}

func (b *ChangedOfBlock) syncAndSaveAllowances() error {

	reqs := b.batchAllowanceReqs()
	if err := ethaccessor.BatchCall("latest", []ethaccessor.BatchReq{reqs}); nil != err {
		return err
	}
	accountAllowances := make(map[common.Address]*AccountAllowances)
	for _, req := range reqs {
		if nil != req.AllowanceErr {
			log.Errorf("get allowance failed, owner:%s, token:%s, err:%s", req.Owner.Hex(), req.Token.Hex(), req.AllowanceErr.Error())
		} else {
			if _, exists := accountAllowances[req.Owner]; !exists {
				accountAllowances[req.Owner] = &AccountAllowances{}
				accountAllowances[req.Owner].Owner = req.Owner
				accountAllowances[req.Owner].Allowances = make(map[common.Address]map[common.Address]Allowance)
			}
			allowance := Allowance{}
			allowance.LastBlock = types.NewBigPtr(b.currentBlockNumber)
			allowance.Allowance = &req.Allowance
			if _, exists := accountAllowances[req.Owner].Allowances[req.Token]; !exists {
				accountAllowances[req.Owner].Allowances[req.Token] = make(map[common.Address]Allowance)
			}
			accountAllowances[req.Owner].Allowances[req.Token][req.Spender] = allowance
		}
	}
	for _, allowances := range accountAllowances {
		allowances.save(int64(0))
	}

	return nil
}

type AccountManager struct {
	cacheDuration int64

	maxBlockLength uint64
	block          *ChangedOfBlock
}

func NewAccountManager(options config.AccountManagerOptions) AccountManager {
	accountManager := AccountManager{}
	if options.CacheDuration > 0 {
		accountManager.cacheDuration = options.CacheDuration
	} else {
		accountManager.cacheDuration = 3600 * 24 * 100
	}
	accountManager.maxBlockLength = 3000
	b := &ChangedOfBlock{}
	b.cachedDuration = big.NewInt(int64(500))
	accountManager.block = b

	return accountManager
}

func (accountManager *AccountManager) Start() {

	transferWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleTokenTransfer}
	approveWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleApprove}
	wethDepositWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleWethDeposit}
	wethWithdrawalWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleWethWithdrawal}
	blockForkWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleBlockFork}
	blockEndWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleBlockEnd}
	blockNewWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleBlockNew}
	ethTransferWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleEthTransfer}
	cancelOrderWather := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleCancelOrder}
	cutoffAllWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleCutOff}
	cutoffPairAllWatcher := &eventemitter.Watcher{Concurrent: false, Handle: accountManager.handleCutOffPair}

	eventemitter.On(eventemitter.WethDeposit, wethDepositWatcher)
	eventemitter.On(eventemitter.WethWithdrawal, wethWithdrawalWatcher)
	eventemitter.On(eventemitter.Approve, approveWatcher)
	eventemitter.On(eventemitter.Transfer, transferWatcher)
	eventemitter.On(eventemitter.EthTransferEvent, ethTransferWatcher)

	eventemitter.On(eventemitter.CancelOrder, cancelOrderWather)
	eventemitter.On(eventemitter.CutoffAll, cutoffAllWatcher)
	eventemitter.On(eventemitter.CutoffPair, cutoffPairAllWatcher)

	eventemitter.On(eventemitter.Block_End, blockEndWatcher)
	eventemitter.On(eventemitter.Block_New, blockNewWatcher)
	eventemitter.On(eventemitter.ChainForkDetected, blockForkWatcher)

}

func (a *AccountManager) GetBalanceWithSymbolResult(owner common.Address) (map[string]*big.Int, error) {
	accountBalances := AccountBalances{}
	accountBalances.Owner = owner
	accountBalances.Balances = make(map[common.Address]Balance)

	res := make(map[string]*big.Int)
	//err := accountBalances.getOrSave(common.HexToAddress("0x1fa02762bd046abd30f5bf3513f9347d5e6b4257"), common.HexToAddress("0x"), common.HexToAddress("0x3cbcee9ff904ee0351b0ff2c05e08e860c94a5ea"))
	err := accountBalances.getOrSave(a.cacheDuration)

	if nil == err {
		for tokenAddr, balance := range accountBalances.Balances {
			symbol := ""
			if types.IsZeroAddress(tokenAddr) {
				symbol = "ETH"
			} else {
				symbol = util.AddressToAlias(tokenAddr.Hex())
			}
			res[symbol] = balance.Balance.BigInt()
		}
	}

	return res, err
}

func (a *AccountManager) GetAllowanceWithSymbolResult(owner, spender common.Address) (map[string]*big.Int, error) {
	accountAllowances := &AccountAllowances{}
	accountAllowances.Owner = owner
	accountAllowances.Allowances = make(map[common.Address]map[common.Address]Allowance)

	res := make(map[string]*big.Int)
	err := accountAllowances.getOrSave(a.cacheDuration, []common.Address{}, []common.Address{spender})

	if nil == err {
		for tokenAddr, allowances := range accountAllowances.Allowances {
			symbol := ""
			if types.IsZeroAddress(tokenAddr) {
				symbol = "ETH"
				res[symbol] = big.NewInt(int64(0))
			} else {
				symbol = util.AddressToAlias(tokenAddr.Hex())
				if _, exists := allowances[spender]; !exists || nil == allowances[spender].Allowance {
					res[symbol] = big.NewInt(int64(0))
				} else {
					res[symbol] = allowances[spender].Allowance.BigInt()
				}
			}
		}
	} else {
		log.Errorf("err:%s", err.Error())
	}

	return res, err
}

func (a *AccountManager) GetBalanceAndAllowance(owner, token, spender common.Address) (balance, allowance *big.Int, err error) {

	accountBalances := &AccountBalances{}
	accountBalances.Owner = owner
	accountBalances.Balances = make(map[common.Address]Balance)
	accountBalances.getOrSave(a.cacheDuration, token)
	balance = accountBalances.Balances[token].Balance.BigInt()

	accountAllowances := &AccountAllowances{}
	accountAllowances.Owner = owner
	accountAllowances.Allowances = make(map[common.Address]map[common.Address]Allowance)
	accountAllowances.getOrSave(a.cacheDuration, []common.Address{token}, []common.Address{spender})
	allowance = accountAllowances.Allowances[token][spender].Allowance.BigInt()

	return
}

func (a *AccountManager) GetCutoff(contract, address string) (int, error) {
	cutoffTime, err := ethaccessor.GetCutoff(common.HexToAddress(contract), common.HexToAddress(address), "latest")
	return int(cutoffTime.Int64()), err
}

func (a *AccountManager) handleTokenTransfer(input eventemitter.EventData) (err error) {
	event := input.(*types.TransferEvent)

	//log.Info("received transfer event...")

	if event == nil || event.Status != types.TX_STATUS_SUCCESS {
		log.Info("received wrong status event, drop it")
		return nil
	}

	//balance
	a.block.saveBalanceKey(event.Sender, event.Protocol)
	a.block.saveBalanceKey(event.From, types.NilAddress)
	a.block.saveBalanceKey(event.Receiver, event.Protocol)

	//allowance
	if spender, err := ethaccessor.GetSpenderAddress(event.To); nil == err {
		log.Debugf("handleTokenTransfer allowance owner:%s", event.Sender.Hex(), event.Protocol.Hex(), spender.Hex())
		a.block.saveAllowanceKey(event.Sender, event.Protocol, spender)
	}

	return nil
}

func (a *AccountManager) handleApprove(input eventemitter.EventData) error {
	event := input.(*types.ApprovalEvent)
	log.Debugf("received approval event, %s, %s", event.Protocol.Hex(), event.Owner.Hex())
	if event == nil || event.Status != types.TX_STATUS_SUCCESS {
		log.Info("received wrong status event, drop it")
		return nil
	}

	a.block.saveAllowanceKey(event.Owner, event.Protocol, event.Spender)

	a.block.saveBalanceKey(event.Owner, types.NilAddress)

	return nil
}

func (a *AccountManager) handleWethDeposit(input eventemitter.EventData) (err error) {
	event := input.(*types.WethDepositEvent)
	if event == nil || event.Status != types.TX_STATUS_SUCCESS {
		log.Info("received wrong status event, drop it")
		return nil
	}
	a.block.saveBalanceKey(event.Dst, event.Protocol)
	a.block.saveBalanceKey(event.From, types.NilAddress)
	return
}

func (a *AccountManager) handleWethWithdrawal(input eventemitter.EventData) (err error) {
	event := input.(*types.WethWithdrawalEvent)
	if event == nil || event.Status != types.TX_STATUS_SUCCESS {
		log.Info("received wrong status event, drop it")
		return nil
	}

	a.block.saveBalanceKey(event.Src, event.Protocol)
	a.block.saveBalanceKey(event.From, types.NilAddress)

	return
}

func (a *AccountManager) handleBlockEnd(input eventemitter.EventData) error {
	event := input.(*types.BlockEvent)
	log.Debugf("handleBlockEndhandleBlockEndhandleBlockEnd:%s", event.BlockNumber.String())

	a.block.syncAndSaveBalances()
	a.block.syncAndSaveAllowances()

	removeExpiredBlock(a.block.currentBlockNumber, a.block.cachedDuration)

	return nil
}

func (a *AccountManager) handleBlockNew(input eventemitter.EventData) error {
	event := input.(*types.BlockEvent)
	log.Debugf("handleBlockNewhandleBlockNewhandleBlockNewhandleBlockNew:%s", event.BlockNumber.String())
	a.block.currentBlockNumber = new(big.Int).Set(event.BlockNumber)
	return nil
}

func (a *AccountManager) handleEthTransfer(input eventemitter.EventData) error {
	event := input.(*types.TransferEvent)
	a.block.saveBalanceKey(event.From, types.NilAddress)
	a.block.saveBalanceKey(event.To, types.NilAddress)
	return nil
}

func (a *AccountManager) handleCancelOrder(input eventemitter.EventData) error {
	event := input.(*types.OrderCancelledEvent)
	a.block.saveBalanceKey(event.From, types.NilAddress)
	return nil
}

func (a *AccountManager) handleCutOff(input eventemitter.EventData) error {
	event := input.(*types.CutoffEvent)
	a.block.saveBalanceKey(event.From, types.NilAddress)
	return nil
}

func (a *AccountManager) handleCutOffPair(input eventemitter.EventData) error {
	event := input.(*types.CutoffPairEvent)
	a.block.saveBalanceKey(event.From, types.NilAddress)
	return nil
}

func (a *AccountManager) UnlockedWallet(owner string) (err error) {
	if !common.IsHexAddress(owner) {
		return errors.New("owner isn't a valid hex-address")
	}

	//accountBalances := AccountBalances{}
	//accountBalances.Owner = common.HexToAddress(owner)
	//accountBalances.Balances = make(map[common.Address]Balance)
	//err = accountBalances.getOrSave(a.cacheDuration)
	rcache.Set(unlockCacheKey(common.HexToAddress(owner)), []byte("true"), a.cacheDuration)
	return
}

func (a *AccountManager) HasUnlocked(owner string) (exists bool, err error) {
	if !common.IsHexAddress(owner) {
		return false, errors.New("owner isn't a valid hex-address")
	}
	return rcache.Exists(unlockCacheKey(common.HexToAddress(owner)))
}

func (a *AccountManager) handleBlockFork(input eventemitter.EventData) (err error) {
	event := input.(*types.ForkedEvent)
	log.Infof("the eth network may be forked. flush all cache, detectedBlock:%s", event.DetectedBlock.String())

	i := new(big.Int).Set(event.DetectedBlock)
	for i.Cmp(event.ForkBlock) >= 0 {
		changedOfBlock := &ChangedOfBlock{}
		changedOfBlock.currentBlockNumber = i
		changedOfBlock.syncAndSaveBalances()
		changedOfBlock.syncAndSaveAllowances()
		i.Sub(i, big.NewInt(int64(1)))
	}

	return nil
}
