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

package test

import (
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/extractor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/Loopring/relay/usermanager"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/naoina/toml"
	"math/big"
	"os"
	"strings"
	"time"
)

type AccountEntity struct {
	Address    common.Address
	Passphrase string
}

type TestEntity struct {
	Tokens          map[string]common.Address
	Accounts        []AccountEntity
	Creator         AccountEntity
	KeystoreDir     string
	AllowanceAmount int64
}

const (
	Version   = "v_0_1"
	DebugFile = "relay.toml"
)

var (
	cfg           *config.GlobalConfig
	rds           dao.RdsService
	entity        *TestEntity
	orderAccounts = []accounts.Account{}
	creator       accounts.Account
	accessor      *ethaccessor.EthNodeAccessor
	protocol      common.Address
)

func init() {
	cfg = loadConfig()
	rds = GenerateDaoService()
	util.Initialize(rds, cfg.Common.ProtocolImpl.Address)
	loadTestData()
	unlockAccounts()
	accessor, _ = ethaccessor.NewAccessor(cfg.Accessor, cfg.Common, util.WethTokenAddress())

	protocol = common.HexToAddress(cfg.Common.ProtocolImpl.Address[Version])
}

func loadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/" + DebugFile
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}
func LoadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/" + DebugFile
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}

func loadTestData() {
	entity = new(TestEntity)

	type Account struct {
		Address    string
		Passphrase string
	}

	type TestData struct {
		Accounts        []Account
		Creator         Account
		AllowanceAmount int64
	}

	file := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/test/testdata.toml"

	io, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer io.Close()

	var testData TestData
	if err := toml.NewDecoder(io).Decode(&testData); err != nil {
		log.Fatalf(err.Error())
	}

	entity.Accounts = make([]AccountEntity, 0)
	for _, v := range testData.Accounts {
		var acc AccountEntity
		acc.Address = common.HexToAddress(v.Address)
		acc.Passphrase = v.Passphrase
		entity.Accounts = append(entity.Accounts, acc)
	}

	entity.Tokens = make(map[string]common.Address)
	for symbol, token := range util.AllTokens {
		entity.Tokens[symbol] = token.Protocol
	}

	entity.Creator = AccountEntity{Address: common.HexToAddress(testData.Creator.Address), Passphrase: testData.Creator.Passphrase}
	entity.KeystoreDir = cfg.Keystore.Keydir
	entity.AllowanceAmount = testData.AllowanceAmount
}

func unlockAccounts() {
	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	c := crypto.NewCrypto(false, ks)
	crypto.Initialize(c)
	accessor, _ = ethaccessor.NewAccessor(cfg.Accessor, cfg.Common, util.WethTokenAddress())

	creator = accounts.Account{Address: entity.Creator.Address}
	ks.Unlock(creator, entity.Creator.Passphrase)

	for _, accTmp := range entity.Accounts {
		account := accounts.Account{Address: accTmp.Address}
		orderAccounts = append(orderAccounts, account)
		if err := ks.Unlock(account, accTmp.Passphrase); nil != err {
			log.Fatalf("unlock account:%s error:%s", accTmp.Address.Hex(), err.Error())
		} else {
			log.Debugf("unlocked:%s", accTmp.Address.Hex())
		}
	}
}

func Rds() dao.RdsService       { return rds }
func Cfg() *config.GlobalConfig { return cfg }
func Entity() *TestEntity       { return entity }
func Protocol() common.Address  { return common.HexToAddress(cfg.Common.ProtocolImpl.Address[Version]) }

func GenerateAccessor() (*ethaccessor.EthNodeAccessor, error) {
	accessor, err := ethaccessor.NewAccessor(cfg.Accessor, cfg.Common, util.WethTokenAddress())
	if nil != err {
		return nil, err
	}
	return accessor, nil
}

func GenerateExtractor() *extractor.ExtractorServiceImpl {
	accessor, err := GenerateAccessor()
	if err != nil {
		panic(err)
	}
	l := extractor.NewExtractorService(cfg.Accessor, cfg.Common, accessor, rds)
	return l
}

func GenerateOrderManager() *ordermanager.OrderManagerImpl {
	mc := GenerateMarketCap()
	um := usermanager.NewUserManager(rds)
	accessor, err := GenerateAccessor()
	if err != nil {
		panic(err)
	}
	ob := ordermanager.NewOrderManager(rds, um, accessor, mc)
	return ob
}

func GenerateDaoService() *dao.RdsServiceImpl {
	return dao.NewRdsService(cfg.Mysql)
}

func GenerateMarketCap() *marketcap.CapProvider_CoinMarketCap {
	return marketcap.NewMarketCapProvider(cfg.MarketCap)
}

func CreateOrder(tokenS, tokenB, protocol, owner common.Address, amountS, amountB, lrcFee *big.Int) *types.Order {
	order := &types.Order{}
	order.Protocol = protocol
	order.TokenS = tokenS
	order.TokenB = tokenB
	order.AmountS = amountS
	order.AmountB = amountB
	order.Timestamp = big.NewInt(time.Now().Unix())
	order.Ttl = big.NewInt(8640000)
	order.Salt = big.NewInt(1000)
	order.LrcFee = lrcFee
	order.BuyNoMoreThanAmountB = false
	order.MarginSplitPercentage = 0
	order.Owner = owner
	order.Hash = order.GenerateHash()
	if err := order.GenerateAndSetSignature(owner); nil != err {
		log.Fatalf(err.Error())
	}
	return order
}

func PrepareTestData() {
	//delegate registry
	delegateAbi := accessor.DelegateAbi
	delegateAddress := accessor.ProtocolAddresses[protocol].DelegateAddress
	callMethod := accessor.ContractCallMethod(delegateAbi, delegateAddress)
	var res types.Big
	if err := callMethod(&res, "isAddressAuthorized", "latest", protocol); nil != err {
		log.Errorf("err:%s", err.Error())
	} else {
		if res.Int() <= 0 {
			delegateCallMethod := accessor.ContractSendTransactionMethod(delegateAbi, delegateAddress)
			if hash, err := delegateCallMethod(creator, "authorizeAddress", nil, nil, nil, protocol); nil != err {
				log.Errorf("delegate add version error:%s", err.Error())
			} else {
				log.Infof("delegate add version hash:%s", hash)
			}
		} else {
			log.Infof("delegate had added this version")
		}
	}

	//tokenregistry
	tokenRegisterAbi := accessor.TokenRegistryAbi
	tokenRegisterAddress := accessor.ProtocolAddresses[protocol].TokenRegistryAddress
	for symbol, tokenAddr := range entity.Tokens {
		callMethod := accessor.ContractCallMethod(tokenRegisterAbi, tokenRegisterAddress)
		var res types.Big
		if err := callMethod(&res, "isTokenRegistered", "latest", tokenAddr); nil != err {
			log.Errorf("err:%s", err.Error())
		} else {
			if res.Int() <= 0 {
				registryMethod := accessor.ContractSendTransactionMethod(tokenRegisterAbi, tokenRegisterAddress)
				if hash, err := registryMethod(creator, "registerToken", nil, nil, nil, tokenAddr, symbol); nil != err {
					log.Errorf("token registry error:%s", err.Error())
				} else {
					log.Infof("token registry hash:%s", hash)
				}
			} else {
				log.Infof("token had registered")
			}
		}
	}

	//approve
	for _, tokenAddr := range entity.Tokens {
		erc20SendMethod := accessor.ContractSendTransactionMethod(accessor.Erc20Abi, tokenAddr)
		for _, acc := range orderAccounts {
			if hash, err := erc20SendMethod(acc, "approve", big.NewInt(106762), big.NewInt(21000000000), nil, delegateAddress, big.NewInt(int64(1000000000000000000))); nil != err {
				log.Errorf("token approve error:%s", err.Error())
			} else {
				log.Infof("token approve hash:%s", hash)
			}
		}
	}
}

func AllowanceToLoopring(tokens1 []common.Address, orderAccounts1 []accounts.Account) {
	if nil == tokens1 {
		for _, v := range entity.Tokens {
			tokens1 = append(tokens1, v)
		}
	}
	if nil == orderAccounts1 {
		orderAccounts1 = orderAccounts
	}

	for _, tokenAddr := range tokens1 {
		callMethod := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddr)
		for _, account := range orderAccounts1 {
			var balance types.Big
			if err := callMethod(&balance, "balanceOf", "latest", account.Address); nil != err {
				log.Errorf("err:%s", err.Error())
			} else {
				log.Infof("token:%s, owner:%s, balance:%s", tokenAddr.Hex(), account.Address.Hex(), balance.BigInt().String())
			}

			var allowance types.Big
			for _, impl := range accessor.ProtocolAddresses {
				if err := callMethod(&allowance, "allowance", "latest", account.Address, impl.DelegateAddress); nil != err {
					log.Error(err.Error())
				} else {
					log.Infof("token:%s, owner:%s, spender:%s, allowance:%s", tokenAddr.Hex(), account.Address.Hex(), impl.DelegateAddress.Hex(), allowance.BigInt().String())
				}
			}
		}
	}
}

//setbalance after deploy token by protocol
//不能设置weth
func SetTokenBalances() {
	dummyTokenAbiStr := `[{"constant":true,"inputs":[],"name":"mintingFinished","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_amount","type":"uint256"}],"name":"mint","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_subtractedValue","type":"uint256"}],"name":"decreaseApproval","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"finishMinting","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_addedValue","type":"uint256"}],"name":"increaseApproval","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_target","type":"address"},{"name":"_value","type":"uint256"}],"name":"setBalance","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"_name","type":"string"},{"name":"_symbol","type":"string"},{"name":"_decimals","type":"uint8"},{"name":"_totalSupply","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"amount","type":"uint256"}],"name":"Mint","type":"event"},{"anonymous":false,"inputs":[],"name":"MintFinished","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	dummyTokenAbi := &abi.ABI{}
	dummyTokenAbi.UnmarshalJSON([]byte(dummyTokenAbiStr))

	sender := accounts.Account{Address: common.HexToAddress(cfg.Miner.Miner)}
	amount, _ := new(big.Int).SetString("10000000000000000000000", 0)
	wethAmount, _ := new(big.Int).SetString("19973507197999000000", 0)

	// deposit weth
	wethToken := entity.Tokens["WETH"]
	for _, v := range entity.Accounts {
		owner := accounts.Account{Address: v.Address}
		sendTransactionMethod := accessor.ContractSendTransactionMethod(accessor.WethAbi, wethToken)
		hash, err := sendTransactionMethod(owner, "deposit", big.NewInt(1000000), big.NewInt(21000000000), wethAmount)
		if nil != err {
			log.Fatalf("call method weth-deposit error:%s", err.Error())
		} else {
			log.Debugf("weth-deposit txhash:%s", hash)
		}
	}

	// other token set balance
	for symbol, tokenAddress := range entity.Tokens {
		if symbol == "WETH" {
			continue
		}
		sendTransactionMethod := accessor.ContractSendTransactionMethod(dummyTokenAbi, tokenAddress)
		erc20Method := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddress)

		for _, acc := range orderAccounts {
			var res types.Big
			if err := erc20Method(&res, "balanceOf", "latest", acc.Address); nil != err {
				fmt.Errorf(err.Error())
			}
			if res.BigInt().Cmp(big.NewInt(int64(0))) <= 0 {
				hash, err := sendTransactionMethod(sender, "setBalance", big.NewInt(1000000), big.NewInt(21000000000), nil, acc.Address, amount)
				if nil != err {
					fmt.Errorf(err.Error())
				}
				fmt.Printf("sendhash:%s", hash)
			}
			fmt.Printf("tokenAddress:%s, useraddress:%s, balance:%s", tokenAddress.Hex(), acc.Address.Hex(), res.BigInt().String())
		}
		fmt.Printf(":", tokenAddress.Hex())
	}
}

// 给lrc，rdn等dummy合约支持的代币充值
func SetTokenBalance(symbol string, account common.Address, amount *big.Int) {
	dummyTokenAbiStr := `[{"constant":true,"inputs":[],"name":"mintingFinished","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_amount","type":"uint256"}],"name":"mint","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_subtractedValue","type":"uint256"}],"name":"decreaseApproval","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"finishMinting","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_addedValue","type":"uint256"}],"name":"increaseApproval","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_target","type":"address"},{"name":"_value","type":"uint256"}],"name":"setBalance","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"_name","type":"string"},{"name":"_symbol","type":"string"},{"name":"_decimals","type":"uint8"},{"name":"_totalSupply","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"amount","type":"uint256"}],"name":"Mint","type":"event"},{"anonymous":false,"inputs":[],"name":"MintFinished","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	dummyTokenAbi := &abi.ABI{}
	dummyTokenAbi.UnmarshalJSON([]byte(dummyTokenAbiStr))

	sender := accounts.Account{Address: common.HexToAddress(cfg.Miner.Miner)}
	tokenAddress := util.AllTokens[symbol].Protocol
	sendTransactionMethod := accessor.ContractSendTransactionMethod(dummyTokenAbi, tokenAddress)

	hash, err := sendTransactionMethod(sender, "setBalance", big.NewInt(1000000), big.NewInt(21000000000), nil, account, amount)
	if nil != err {
		fmt.Errorf(err.Error())
	}
	fmt.Printf("sendhash:%s", hash)
}
