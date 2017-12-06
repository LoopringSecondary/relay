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
	"math/big"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
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
			if hash, err := delegateCallMethod(creator, "authorizeAddress", nil, nil, protocol); nil != err {
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
				if hash, err := registryMethod(creator, "registerToken", nil, nil, tokenAddr, strconv.Itoa(rand.Intn(100000))); nil != err {
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
			if hash, err := erc20SendMethod(acc, "approve", nil, nil, delegateAddress, big.NewInt(testData.AllowanceAmount)); nil != err {
				log.Errorf("token approve error:%s", err.Error())
			} else {
				log.Infof("token approve hash:%s", hash)
			}
		}
	}
}

func AllowanceToLoopring() {
	for _, tokenAddr := range tokens {
		callMethod := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddr)

		for _, account := range orderAccounts {
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
		ks.Unlock(account, accTmp.Passphrase)
	}

	// set supported tokens
	rds := GenerateDaoService(cfg)
	InitialMarketUtil(rds)
	for _, token := range util.SupportTokens {
		tokens = append(tokens, token.Protocol)
	}

	c := crypto.NewCrypto(false, ks)
	crypto.Initialize(c)
	accessor, _ = ethaccessor.NewAccessor(cfg.Accessor, cfg.Common)

}
