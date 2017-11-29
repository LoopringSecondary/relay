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
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/naoina/toml"
	"math/big"
	"os"
	"strconv"
	"strings"
)

type Account struct {
	Address string
	Passphrase string
}

type TestData struct {
	Tokens []string
	Accounts []Account
	Creator Account
	KeystoreDir string
	AllowanceAmount int64
}

func PrepareTestData() {
	dir := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/"
	file := dir + "/test/testdata.toml"

	io, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer io.Close()

	d := &TestData{}
	if err := toml.NewDecoder(io).Decode(d); err != nil {
		panic(err)
	}

	tokens := []common.Address{}
	orderAccounts := []accounts.Account{}

	cfg := loadConfig()
	println(d.KeystoreDir)
	ks := keystore.NewKeyStore(d.KeystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)

	println(d.Creator.Address)
	creator := accounts.Account{Address: common.HexToAddress(d.Creator.Address)}
	ks.Unlock(creator, d.Creator.Passphrase)
	for _, accTmp := range d.Accounts {
		println(accTmp.Address)
		account := accounts.Account{Address: common.HexToAddress(accTmp.Address)}
		orderAccounts = append(orderAccounts, account)
		ks.Unlock(account, accTmp.Passphrase)
	}

	for _,tokenTmp := range d.Tokens {
		println(tokenTmp)

		tokens = append(tokens, common.HexToAddress(tokenTmp))
	}

	c := crypto.NewCrypto(false, ks)
	crypto.Initialize(c)
	accessor, _ := ethaccessor.NewAccessor(cfg.Accessor, cfg.Common, ks)

	for protocolAddr, impl := range accessor.ProtocolImpls {
		//delegate registry
		delegateCallMethod := accessor.ContractSendTransactionMethod(impl.DelegateAbi, impl.DelegateAddress)
		if hash, err := delegateCallMethod(creator, "authorizeAddress", nil, nil, protocolAddr); nil != err {
			log.Errorf("delegate add version error:%s", err.Error())
		} else {
			log.Infof("delegate add version hash:%s", hash)
		}
		//tokenregistry
		for idx, tokenAddr := range tokens {
			registryMethod := accessor.ContractSendTransactionMethod(impl.TokenRegistryAbi, impl.TokenRegistryAddress)
			if hash, err := registryMethod(creator, "registerToken", nil, nil, tokenAddr, "token"+strconv.Itoa(idx)); nil != err {
				log.Errorf("token registry error:%s", err.Error())
			} else {
				log.Infof("token registry hash:%s", hash)
			}
		}

		//approve
		for _, tokenAddr := range tokens {
			erc20SendMethod := accessor.ContractSendTransactionMethod(accessor.Erc20Abi, tokenAddr)
			for _, acc := range orderAccounts {
				if hash, err := erc20SendMethod(acc, "approve", nil, nil, impl.DelegateAddress, big.NewInt(d.AllowanceAmount)); nil != err {
					log.Errorf("token approve error:%s", err.Error())
				} else {
					log.Infof("token approve hash:%s", hash)

				}
			}

		}
	}
}

//
//func (testParams *TestParams) IsTestDataReady() {
//
//	accounts := []string{}
//	for k, _ := range testParams.Accounts {
//		accounts = append(accounts, k)
//	}
//
//	testParams.allowanceToLoopring(accounts, testParams.TokenAddrs)
//}

//func allowanceToLoopring(accounts []string, tokenAddrs []string) {
//	token := &chainclient.Erc20Token{}
//	for _, tokenAddr := range tokenAddrs {
//		testParams.Client.NewContract(token, tokenAddr, chainclient.Erc20TokenAbiStr)
//		for _, account := range accounts {
//			balance := &types.Big{}
//			if err := token.BalanceOf.Call(balance, "latest", common.HexToAddress(account)); nil != err {
//				log.Error(err.Error())
//			} else {
//				log.Infof("token: %s, balance %s : %s", tokenAddr, account, balance.BigInt().String())
//			}
//			if err := token.Allowance.Call(balance, "latest", common.HexToAddress(account), testParams.DelegateAddress); nil != err {
//				log.Error(err.Error())
//			} else {
//				log.Infof("token:%s, allowance: %s -> %s %s", tokenAddr, account, testParams.DelegateAddress.Hex(), balance.BigInt().String())
//			}
//		}
//	}
//}
