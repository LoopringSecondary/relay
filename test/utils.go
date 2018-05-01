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
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/txmanager"
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
	PrivateKey      crypto.EthPrivateKeyCrypto
}

const (
	Version   = "v1.5.1"
	DebugFile = "debug.toml"
)

var (
	cfg           *config.GlobalConfig
	rds           dao.RdsService
	entity        *TestEntity
	orderAccounts = []accounts.Account{}
	creator       accounts.Account
	protocol      common.Address
	delegate      common.Address
	Path          string
)

func init() {
	Path = strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/" + DebugFile
	cfg = loadConfig()
	rds = GenerateDaoService()
	txmanager.NewTxView(rds)
	cache.NewCache(cfg.Redis)
	util.Initialize(cfg.Market)
	entity = loadTestData()
	ethaccessor.Initialize(cfg.Accessor, cfg.Common, util.WethTokenAddress())
	unlockAccounts()
	protocol = common.HexToAddress(cfg.Common.ProtocolImpl.Address[Version])
	delegate = ethaccessor.ProtocolAddresses()[protocol].DelegateAddress
}

func loadConfig() *config.GlobalConfig {
	c := config.LoadConfig(Path)
	log.Initialize(c.Log)

	return c
}

func LoadConfig() *config.GlobalConfig {
	c := config.LoadConfig(Path)
	log.Initialize(c.Log)

	return c
}

func LoadTestData() *TestEntity {
	return loadTestData()
}

func loadTestData() *TestEntity {
	e := new(TestEntity)

	type Account struct {
		Address    string
		Passphrase string
	}

	type AuthKey struct {
		Address common.Address
		Privkey string
	}

	type TestData struct {
		Accounts        []Account
		Creator         Account
		AllowanceAmount int64
		Auth            AuthKey
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

	e.Accounts = make([]AccountEntity, 0)
	for _, v := range testData.Accounts {
		var acc AccountEntity
		acc.Address = common.HexToAddress(v.Address)
		acc.Passphrase = v.Passphrase
		e.Accounts = append(e.Accounts, acc)
	}

	e.Tokens = make(map[string]common.Address)
	for symbol, token := range util.AllTokens {
		e.Tokens[symbol] = token.Protocol
	}

	e.Creator = AccountEntity{Address: common.HexToAddress(testData.Creator.Address), Passphrase: testData.Creator.Passphrase}
	e.KeystoreDir = cfg.Keystore.Keydir
	e.AllowanceAmount = testData.AllowanceAmount

	e.PrivateKey, _ = crypto.NewPrivateKeyCrypto(false, testData.Auth.Privkey)
	return e
}

func unlockAccounts() {
	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	c := crypto.NewKSCrypto(false, ks)
	crypto.Initialize(c)

	creator = accounts.Account{Address: entity.Creator.Address}
	if err := ks.Unlock(creator, entity.Creator.Passphrase); err != nil {
		fmt.Printf(err.Error())
	}

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
func Delegate() common.Address  { return ethaccessor.ProtocolAddresses()[protocol].DelegateAddress }

func GenerateUserManager() *usermanager.UserManagerImpl {
	um := usermanager.NewUserManager(&cfg.UserManager, rds)
	return um
}

func GenerateOrderManager() *ordermanager.OrderManagerImpl {
	mc := GenerateMarketCap()
	um := usermanager.NewUserManager(&cfg.UserManager, rds)
	ob := ordermanager.NewOrderManager(&cfg.OrderManager, rds, um, mc)
	return ob
}

func GenerateDaoService() *dao.RdsServiceImpl {
	return dao.NewRdsService(cfg.Mysql)
}

func GenerateMarketCap() *marketcap.CapProvider_CoinMarketCap {
	return marketcap.NewMarketCapProvider(cfg.MarketCap)
}

func GenerateAccountManager() market.AccountManager {
	return market.NewAccountManager(cfg.AccountManager)
}

func CreateOrder(tokenS, tokenB, owner common.Address, amountS, amountB, lrcFee *big.Int) *types.Order {
	var (
		order types.Order
		state types.OrderState
		model dao.Order
	)
	order.Protocol = protocol
	order.DelegateAddress = delegate
	order.TokenS = tokenS
	order.TokenB = tokenB
	order.AmountS = amountS
	order.AmountB = amountB
	order.ValidSince = big.NewInt(time.Now().Unix())
	order.ValidUntil = big.NewInt(time.Now().Unix() + 8640000)
	order.LrcFee = lrcFee
	order.BuyNoMoreThanAmountB = false
	order.MarginSplitPercentage = 0
	order.Owner = owner
	order.PowNonce = 1
	order.AuthPrivateKey = entity.PrivateKey
	order.AuthAddr = order.AuthPrivateKey.Address()
	order.WalletAddress = owner
	order.Hash = order.GenerateHash()
	order.GeneratePrice()
	if err := order.GenerateAndSetSignature(owner); nil != err {
		log.Fatalf(err.Error())
	}

	state.RawOrder = order
	state.DealtAmountS = big.NewInt(0)
	state.DealtAmountB = big.NewInt(0)
	state.SplitAmountS = big.NewInt(0)
	state.SplitAmountB = big.NewInt(0)
	state.CancelledAmountB = big.NewInt(0)
	state.CancelledAmountS = big.NewInt(0)
	state.UpdatedBlock = big.NewInt(0)
	state.RawOrder.Side = util.GetSide(state.RawOrder.TokenS.Hex(), state.RawOrder.TokenB.Hex())
	state.Status = types.ORDER_NEW

	market, err := util.WrapMarketByAddress(state.RawOrder.TokenB.Hex(), state.RawOrder.TokenS.Hex())
	if err != nil {
		log.Fatalf("get market error:%s", err.Error())
	}
	model.Market = market
	model.ConvertDown(&state)

	rds.Add(&model)

	return &order
}

func getCallArg(a *abi.ABI, protocol common.Address, methodName string, args ...interface{}) *ethaccessor.CallArg {
	if callData, err := a.Pack(methodName, args...); nil != err {
		panic(err)
	} else {
		arg := ethaccessor.CallArg{}
		arg.From = protocol
		arg.To = protocol
		arg.Data = common.ToHex(callData)
		return &arg
	}
}

func PrepareTestData() {
	// name registry
	// nameRegistryAbi := ethaccessor.nam

	//delegate registry
	delegateAbi := ethaccessor.DelegateAbi()
	delegateAddress := ethaccessor.ProtocolAddresses()[protocol].DelegateAddress
	var res types.Big
	if err := ethaccessor.Call(&res, getCallArg(delegateAbi, delegateAddress, "isAddressAuthorized", protocol), "latest"); nil != err {
		log.Errorf("err:%s", err.Error())
	} else {
		if res.Int() <= 0 {
			delegateCallMethod := ethaccessor.ContractSendTransactionMethod("latest", delegateAbi, delegateAddress)
			if hash, err := delegateCallMethod(creator.Address, "authorizeAddress", big.NewInt(106762), big.NewInt(21000000000), nil, protocol); nil != err {
				log.Errorf("delegate add version error:%s", err.Error())
			} else {
				log.Infof("delegate add version hash:%s", hash)
			}
		} else {
			log.Infof("delegate had added this version")
		}
	}

	log.Infof("tokenregistry")
	//tokenregistry
	tokenRegisterAbi := ethaccessor.TokenRegistryAbi()
	tokenRegisterAddress := ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress
	for symbol, tokenAddr := range entity.Tokens {
		log.Infof("token:%s addr:%s", symbol, tokenAddr.Hex())
		callMethod := ethaccessor.ContractCallMethod(tokenRegisterAbi, tokenRegisterAddress)
		var res types.Big
		if err := callMethod(&res, "isTokenRegistered", "latest", tokenAddr); nil != err {
			log.Errorf("err:%s", err.Error())
		} else {
			if res.Int() <= 0 {
				registryMethod := ethaccessor.ContractSendTransactionMethod("latest", tokenRegisterAbi, tokenRegisterAddress)
				if hash, err := registryMethod(creator.Address, "registerToken", big.NewInt(106762), big.NewInt(21000000000), nil, tokenAddr, symbol); nil != err {
					log.Errorf("token registry error:%s", err.Error())
				} else {
					log.Infof("token registry hash:%s", hash)
				}
			} else {
				log.Infof("token %s had registered, res:%s", res.BigInt().String())
			}
		}
	}

	//approve
	for _, tokenAddr := range entity.Tokens {
		erc20SendMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.Erc20Abi(), tokenAddr)
		for _, acc := range orderAccounts {
			approval := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000000))
			if hash, err := erc20SendMethod(acc.Address, "approve", big.NewInt(106762), big.NewInt(21000000000), nil, delegateAddress, approval); nil != err {
				log.Errorf("token approve error:%s", err.Error())
			} else {
				log.Infof("token approve hash:%s", hash)
			}
		}
	}
}

func humanNumber(amount *big.Int) string {
	base := new(big.Int).SetInt64(1e18)
	ret := new(big.Rat).SetFrac(amount, base)
	return ret.FloatString(6)
}

func AllowanceToLoopring(tokens1 []common.Address, orderAccounts1 []accounts.Account) {
	if nil == tokens1 {
		for _, v := range entity.Tokens {
			tokens1 = append(tokens1, v)
		}
	}
	if nil == orderAccounts1 {
		for _, v := range orderAccounts {
			orderAccounts1 = append(orderAccounts1, v)
		}
	}

	for _, tokenAddr := range tokens1 {
		for _, account := range orderAccounts1 {
			if balance, err := ethaccessor.Erc20Balance(tokenAddr, account.Address, "latest"); err != nil {
				log.Errorf("err:%s", err.Error())
			} else {
				log.Infof("token:%s, owner:%s, balance:%s", tokenAddr.Hex(), account.Address.Hex(), humanNumber(balance))
			}

			for _, impl := range ethaccessor.ProtocolAddresses() {
				if allowance, err := ethaccessor.Erc20Allowance(tokenAddr, account.Address, impl.DelegateAddress, "latest"); nil != err {
					log.Error(err.Error())
				} else {
					log.Infof("token:%s, owner:%s, spender:%s, allowance:%s", tokenAddr.Hex(), account.Address.Hex(), impl.DelegateAddress.Hex(), humanNumber(allowance))
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

	sender := accounts.Account{Address: entity.Creator.Address}
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2000000))
	//wethAmount, _ := new(big.Int).SetString("79992767978000000000", 0)

	// deposit weth
	//wethToken := entity.Tokens["WETH"]
	//for _, v := range entity.Accounts {
	//	owner := accounts.Account{Address: v.Address}
	//	sendTransactionMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethToken)
	//	hash, err := sendTransactionMethod(owner, "deposit", nil, nil, wethAmount)
	//	if nil != err {
	//		log.Fatalf("call method weth-deposit error:%s", err.Error())
	//	} else {
	//		log.Debugf("weth-deposit txhash:%s", hash)
	//	}
	//}

	// other token set balance
	for symbol, tokenAddress := range entity.Tokens {
		if symbol == "WETH" {
			continue
		}
		sendTransactionMethod := ethaccessor.ContractSendTransactionMethod("latest", dummyTokenAbi, tokenAddress)
		for _, acc := range orderAccounts {
			if balance, err := ethaccessor.Erc20Balance(tokenAddress, acc.Address, "latest"); nil != err {
				fmt.Errorf(err.Error())
			} else if balance.Cmp(big.NewInt(int64(0))) <= 0 {
				hash, err := sendTransactionMethod(sender.Address, "setBalance", big.NewInt(106762), big.NewInt(21000000000), nil, acc.Address, amount)
				if nil != err {
					fmt.Errorf(err.Error())
				}
				fmt.Printf("sendhash:%s", hash)
			} else {
				fmt.Printf("tokenAddress:%s, useraddress:%s, balance:%s", tokenAddress.Hex(), acc.Address.Hex(), balance.String())
			}
		}
	}
}

// 给lrc，rdn等dummy合约支持的代币充值
func SetTokenBalance(account, tokenAddress common.Address, amount *big.Int) {
	dummyTokenAbi := &abi.ABI{}
	dummyTokenAbiStr := `[{"constant":true,"inputs":[],"name":"mintingFinished","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_amount","type":"uint256"}],"name":"mint","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_subtractedValue","type":"uint256"}],"name":"decreaseApproval","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"finishMinting","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_addedValue","type":"uint256"}],"name":"increaseApproval","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_target","type":"address"},{"name":"_value","type":"uint256"}],"name":"setBalance","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"_name","type":"string"},{"name":"_symbol","type":"string"},{"name":"_decimals","type":"uint8"},{"name":"_totalSupply","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"amount","type":"uint256"}],"name":"Mint","type":"event"},{"anonymous":false,"inputs":[],"name":"MintFinished","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	dummyTokenAbi.UnmarshalJSON([]byte(dummyTokenAbiStr))

	sender := accounts.Account{Address: entity.Creator.Address}
	sendTransactionMethod := ethaccessor.ContractSendTransactionMethod("latest", dummyTokenAbi, tokenAddress)

	hash, err := sendTransactionMethod(sender.Address, "setBalance", big.NewInt(1000000), big.NewInt(21000000000), nil, account, amount)
	if nil != err {
		fmt.Errorf(err.Error())
	}
	fmt.Printf("sendhash:%s", hash)
}
