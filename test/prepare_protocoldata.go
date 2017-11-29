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
	"strconv"
)

var tokens = []common.Address{}

var orderAccounts = []common.Address{}

func PrepareTestData() {
	cfg := loadConfig()
	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	//todo:
	owner := accounts.Account{Address: common.HexToAddress("")}
	ks.Unlock(owner, "")
	c := crypto.NewCrypto(false, ks)
	crypto.Initialize(c)
	accessor, _ := ethaccessor.NewAccessor(cfg.Accessor, cfg.Common, ks)

	for protocolAddr, impl := range accessor.ProtocolImpls {
		//delegate registry
		delegateCallMethod := accessor.ContractSendTransactionMethod(impl.DelegateAbi, impl.DelegateAddress)
		if hash, err := delegateCallMethod(owner, "addVersion", nil, nil, protocolAddr); nil != err {
			log.Errorf("delegate add version error:%s", err.Error())
		} else {
			log.Infof("delegate add version hash:%s", hash)
		}

		//tokenregistry
		for idx, tokenAddr := range tokens {
			registryMethod := accessor.ContractSendTransactionMethod(impl.RinghashRegistryAbi, impl.RinghashRegistryAddress)
			if hash, err := registryMethod(owner, "registryToken", nil, nil, tokenAddr, "token"+strconv.Itoa(idx)); nil != err {
				log.Errorf("token registry error:%s", err.Error())
			} else {
				log.Infof("token registry hash:%s", hash)
			}
		}

		//approve
		for _, tokenAddr := range tokens {
			erc20SendMethod := accessor.ContractSendTransactionMethod(accessor.Erc20Abi, tokenAddr)
			for _, account := range orderAccounts {
				if hash, err := erc20SendMethod(owner, "approve", nil, nil, account, tokenAddr); nil != err {
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
