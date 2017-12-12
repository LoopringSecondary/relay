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
	"github.com/Loopring/relay/market/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"os"
	"strings"

	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/naoina/toml"
)

type Account struct {
	Address    string
	Passphrase string
}

type TestData struct {
	Tokens          []string
	Accounts        []Account
	Creator         Account
	AllowanceAmount int64
}

var testData = &TestData{}

var tokens = []common.Address{}

var orderAccounts = []accounts.Account{}

var creator accounts.Account

var accessor *ethaccessor.EthNodeAccessor

func PrepareTestData() {
	c := loadConfig()
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address["v_0_1"])

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
	for _, tokenAddr := range tokens {
		callMethod := accessor.ContractCallMethod(tokenRegisterAbi, tokenRegisterAddress)
		var res types.Big
		if err := callMethod(&res, "isTokenRegistered", "latest", tokenAddr); nil != err {
			log.Errorf("err:%s", err.Error())
		} else {
			if res.Int() <= 0 {
				registryMethod := accessor.ContractSendTransactionMethod(tokenRegisterAbi, tokenRegisterAddress)
				if hash, err := registryMethod(creator, "registerToken", nil, nil, nil, tokenAddr, "WETH"); nil != err {
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
	for _, tokenAddr := range tokens {
		erc20SendMethod := accessor.ContractSendTransactionMethod(accessor.Erc20Abi, tokenAddr)
		for _, acc := range orderAccounts {
			println("###", acc.Address.Hex())
			if hash, err := erc20SendMethod(acc, "approve", nil, nil, nil, delegateAddress, big.NewInt(testData.AllowanceAmount)); nil != err {
				log.Errorf("token approve error:%s", err.Error())
			} else {
				log.Infof("token approve hash:%s", hash)
			}
		}
	}
}

func AllowanceToLoopring(tokens1 []common.Address, orderAccounts1 []accounts.Account) {
	if nil == tokens1 {
		tokens1 = tokens
	}
	if nil == orderAccounts1 {
		orderAccounts1 = orderAccounts
	}
	for _, tokenAddr := range tokens1 {
		callMethod := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddr)

		for _, account := range orderAccounts1 {
			//balance := &types.Big{}

			var balance types.Big
			if err := callMethod(&balance, "balanceOf", "latest", account.Address); nil != err {
				log.Errorf("err:%s", err.Error())
			} else {
				log.Infof("token: %s, balance %s : %s", tokenAddr.Hex(), account.Address.Hex(), balance.BigInt().String())
			}

			var allowance types.Big
			for _, impl := range accessor.ProtocolAddresses {
				if err := callMethod(&allowance, "allowance", "latest", account.Address, impl.DelegateAddress); nil != err {
					log.Error(err.Error())
				} else {
					log.Infof("token:%s, allowance: %s -> %s %s", tokenAddr.Hex(), account.Address.Hex(), impl.DelegateAddress.Hex(), allowance.BigInt().String())
				}
			}
		}
	}
}

func init() {
	file := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/" + "/test/testdata.toml"

	io, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer io.Close()

	if err := toml.NewDecoder(io).Decode(testData); err != nil {
		panic(err)
	}

	cfg := loadConfig()
	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)

	creator = accounts.Account{Address: common.HexToAddress(testData.Creator.Address)}
	ks.Unlock(creator, testData.Creator.Passphrase)
	for _, accTmp := range testData.Accounts {
		account := accounts.Account{Address: common.HexToAddress(accTmp.Address)}
		orderAccounts = append(orderAccounts, account)
		if err := ks.Unlock(account, accTmp.Passphrase); nil != err {
			println("===", accTmp.Address, err.Error())
		} else {
			println("(((((", accTmp.Address)
		}
	}

	// set supported tokens
	rds := GenerateDaoService(cfg)
	InitialMarketUtil(rds)
	for _, token := range util.AllTokens {
		tokens = append(tokens, token.Protocol)
	}

	c := crypto.NewCrypto(false, ks)
	crypto.Initialize(c)
	accessor, err = ethaccessor.NewAccessor(cfg.Accessor, cfg.Common)

	tokens = []common.Address{common.HexToAddress("0x98C9D14a894d19a38744d41CD016D89Cf9699a51")}
}

//setbalance after deploy token by protocol
func SetTokenBalances() {
	tokens := []string{"LRC", "EOS", "REP", "NEO", "QTUM", "RDN", "RCN", "YOYO", "WETH"}
	addresses := []string{"251f3bd45b06a8b29cb6d171131e192c1254fec1", "0x5ed4698829d8fedac58d75a90dc111416ffd4e2c"}
	dummyTokenAbiStr := `[{"constant":true,"inputs":[],"name":"mintingFinished","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_amount","type":"uint256"}],"name":"mint","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_subtractedValue","type":"uint256"}],"name":"decreaseApproval","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[],"name":"finishMinting","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_addedValue","type":"uint256"}],"name":"increaseApproval","outputs":[{"name":"success","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"_target","type":"address"},{"name":"_value","type":"uint256"}],"name":"setBalance","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"_name","type":"string"},{"name":"_symbol","type":"string"},{"name":"_decimals","type":"uint8"},{"name":"_totalSupply","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"amount","type":"uint256"}],"name":"Mint","type":"event"},{"anonymous":false,"inputs":[],"name":"MintFinished","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"previousOwner","type":"address"},{"indexed":true,"name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	dummyTokenAbi := &abi.ABI{}
	dummyTokenAbi.UnmarshalJSON([]byte(dummyTokenAbiStr))
	//var delegateAddress common.Address
	//for _, impl := range accessor.ProtocolAddresses {
	//	delegateAddress = impl.DelegateAddress
	//}
	//for _,token := range tokens {
	//	erc20Method := accessor.ContractCallMethod(accessor.Erc20Abi, common.HexToAddress(token))
	//
	//	for _,addr := range addresses {
	//		var res types.Big
	//		var allow types.Big
	//		erc20Method(&res, "balanceOf", "latest", common.HexToAddress(addr))
	//		if err := erc20Method(&allow, "allowance", "latest", common.HexToAddress(addr),delegateAddress);nil != err {
	//			println(err.Error())
	//		}
	//		println("token:", token, "addr:", addr, "balance:", res.BigInt().String(), "allowance", allow.BigInt().String())
	//	}
	//}

	sender := accounts.Account{Address: common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259")}

	amount := new(big.Int)
	amount.SetString("10000000000000000000000", 0)
	for _, implAddress := range accessor.ProtocolAddresses {
		callMethod := accessor.ContractCallMethod(accessor.TokenRegistryAbi, implAddress.TokenRegistryAddress)
		for _, token := range tokens {
			var tokenAddressStr string
			if err := callMethod(&tokenAddressStr, "getAddressBySymbol", "latest", token); nil != err {
				println(err.Error())
			}
			tokenAddress := common.HexToAddress(tokenAddressStr)
			sendTransactionMethod := accessor.ContractSendTransactionMethod(dummyTokenAbi, tokenAddress)

			erc20Method := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddress)
			for _, address := range addresses {
				var res types.Big
				if err := erc20Method(&res, "balanceOf", "latest", common.HexToAddress(address)); nil != err {
					println(err.Error())
				}
				if res.BigInt().Cmp(big.NewInt(int64(0))) <= 0 {
					hash, err := sendTransactionMethod(sender, "setBalance", big.NewInt(1000000), big.NewInt(18000000000), nil, common.HexToAddress(address), amount)
					if nil != err {
						println(err.Error())
					}
					println("sendhash:", hash)
					//time.Sleep(20 * time.Second)
				}
				println("token:", token, "tokenAddress:", tokenAddress.Hex(), "useraddress:", address, "balance:", res.BigInt().String())
			}
			println(token, ":", tokenAddress.Hex())
		}
	}
}
