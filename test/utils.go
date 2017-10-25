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
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	ethCryptoLib "github.com/Loopring/ringminer/crypto/eth"
	"github.com/Loopring/ringminer/db"
	ethChainListener "github.com/Loopring/ringminer/listener/chain/eth"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/orderbook"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

type TestParams struct {
	Client               *chainclient.Client
	Imp                  *chainclient.LoopringProtocolImpl
	ImplAddress          types.Address
	Registry             *chainclient.LoopringRinghashRegistry
	MinerPrivateKey      []byte
	DelegateAddress      types.Address
	Owner                types.Address
	TokenRegistryAddress types.Address
	Accounts             map[string]string
	TokenAddrs           []string
	Config               *config.GlobalConfig
}

const (
	TokenAddressA = "0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"
	TokenAddressB = "0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f"
)

//const (
//	TokenAddressA = "0x359bbea6ade5155bce1e95918879903d3e93365f"
//	TokenAddressB = "0xc85819398e4043f3d951367d6d97bb3257b862e0"
//)

var (
	testAccounts = map[string]string{
		"0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A": "07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e",
		"0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2": "11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446",
	}

	testTokens = []string{TokenAddressA, TokenAddressB}
)

func CreateOrder(tokenS, tokenB, protocol types.Address, amountS, amountB *big.Int, pkBytes []byte, owner types.Address) *types.Order {
	order := &types.Order{}
	order.Protocol = protocol
	order.TokenS = tokenS
	order.TokenB = tokenB
	order.AmountS = amountS
	order.AmountB = amountB
	order.Timestamp = big.NewInt(time.Now().Unix())
	order.Ttl = big.NewInt(10000)
	order.Salt = big.NewInt(1000)
	order.LrcFee = big.NewInt(100)
	order.BuyNoMoreThanAmountB = false
	order.MarginSplitPercentage = 0
	order.Owner = owner
	order.GenerateAndSetSignature(pkBytes)
	return order
}

func LoadConfigAndGenerateTestParams() *TestParams {
	params := &TestParams{Imp: &chainclient.LoopringProtocolImpl{}, Registry: &chainclient.LoopringRinghashRegistry{}}
	params.Accounts = testAccounts
	params.TokenAddrs = testTokens

	globalConfig := loadConfig()
	params.Config = globalConfig

	params.ImplAddress = types.HexToAddress(globalConfig.Common.LoopringImpAddresses[0])
	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	ethClient := eth.NewChainClient(globalConfig.ChainClient, "sa")
	params.Client = ethClient.Client
	params.Client.NewContract(params.Imp, params.ImplAddress.Hex(), chainclient.ImplAbiStr)

	var lrcTokenAddressHex string
	params.Imp.LrcTokenAddress.Call(&lrcTokenAddressHex, "pending")
	lrcTokenAddress := types.HexToAddress(lrcTokenAddressHex)
	lrcToken := &chainclient.Erc20Token{}
	params.Client.NewContract(lrcToken, lrcTokenAddress.Hex(), chainclient.Erc20TokenAbiStr)

	var registryAddressHex string
	params.Imp.RinghashRegistryAddress.Call(&registryAddressHex, "pending")
	registryAddress := types.HexToAddress(registryAddressHex)
	params.Client.NewContract(params.Registry, registryAddress.Hex(), chainclient.RinghashRegistryAbiStr)

	var delegateAddressHex string
	params.Imp.DelegateAddress.Call(&delegateAddressHex, "pending")
	params.DelegateAddress = types.HexToAddress(delegateAddressHex)
	var tokenRegistryAddressHex string
	params.Imp.TokenRegistryAddress.Call(&tokenRegistryAddressHex, "pending")
	params.TokenRegistryAddress = types.HexToAddress(tokenRegistryAddressHex)

	passphrase := &types.Passphrase{}
	passphrase.SetBytes([]byte(globalConfig.Common.Passphrase))
	var err error
	params.MinerPrivateKey, err = crypto.AesDecrypted(passphrase.Bytes(), types.FromHex(globalConfig.Miner.Miner))
	if nil != err {
		panic(err)
	}

	var implOwners []string
	if err := params.Client.Accounts(&implOwners); nil != err {
		panic(err)
	}
	params.Owner = types.HexToAddress(implOwners[0])
	return params
}

func (testParams *TestParams) PrepareTestData() {

	var err error
	var hash string
	accounts := []string{}
	for k, _ := range testParams.Accounts {
		accounts = append(accounts, k)
	}

	//delegate registry
	delegateContract := &chainclient.TransferDelegate{}
	testParams.Client.NewContract(delegateContract, testParams.DelegateAddress.Hex(), chainclient.TransferDelegateAbiStr)

	hash, err = delegateContract.AddVersion.SendTransaction(testParams.Owner, common.HexToAddress(testParams.ImplAddress.Hex()))
	if nil != err {
		log.Errorf("delegate add version error:%s", err.Error())
	} else {
		log.Infof("delegate add version hash:%s", hash)
	}
	//
	//tokenregistry
	tokenRegistry := &chainclient.TokenRegistry{}
	testParams.Client.NewContract(tokenRegistry, testParams.TokenRegistryAddress.Hex(), chainclient.TokenRegistryAbiStr)
	for idx, tokenAddr := range testParams.TokenAddrs {
		hash, err = tokenRegistry.RegisterToken.SendTransaction(testParams.Owner, common.HexToAddress(tokenAddr), "token"+strconv.Itoa(idx))
		if nil != err {
			log.Errorf("register token error:%s", err.Error())
		} else {
			log.Infof("register token hash:%s", hash)
		}
	}
	testParams.approveToLoopring(accounts, testParams.TokenAddrs, big.NewInt(30000000))
}

func (testParams *TestParams) IsTestDataReady() {

	accounts := []string{}
	for k, _ := range testParams.Accounts {
		accounts = append(accounts, k)
	}

	testParams.allowanceToLoopring(accounts, testParams.TokenAddrs)
}

func (testParams *TestParams) allowanceToLoopring(accounts []string, tokenAddrs []string) {
	token := &chainclient.Erc20Token{}
	for _, tokenAddr := range tokenAddrs {
		testParams.Client.NewContract(token, tokenAddr, chainclient.Erc20TokenAbiStr)
		for _, account := range accounts {
			balance := &types.Big{}
			if err := token.BalanceOf.Call(balance, "latest", common.HexToAddress(account)); nil != err {
				log.Error(err.Error())
			} else {
				log.Infof("balance %s : %s", account, balance.BigInt().String())
			}
			if err := token.Allowance.Call(balance, "latest", common.HexToAddress(account), testParams.DelegateAddress); nil != err {
				log.Error(err.Error())
			} else {
				log.Infof("allowance: %s -> %s %s", account, testParams.DelegateAddress.Hex(), balance.BigInt().String())
			}
		}
	}
}

func (testParams *TestParams) approveToLoopring(accounts []string, tokenAddrs []string, amount *big.Int) {
	token := &chainclient.Erc20Token{}
	for _, tokenAddr := range tokenAddrs {
		testParams.Client.NewContract(token, tokenAddr, chainclient.Erc20TokenAbiStr)
		for _, account := range accounts {
			if txHash, err := token.Approve.SendTransaction(types.HexToAddress(account), testParams.DelegateAddress, amount); nil != err {
				log.Error(err.Error())
			} else {
				log.Info(txHash)
			}
		}

	}
}

func (testParams *TestParams) CheckAllowance(tokenAddress, account string) {
	var result types.Big
	token := &chainclient.Erc20Token{}
	testParams.Client.NewContract(token, tokenAddress, chainclient.Erc20TokenAbiStr)
	if err := token.Allowance.Call(&result, "pending", types.HexToAddress(account), testParams.DelegateAddress); err != nil {
		panic(err)
	} else {
		println(result.BigInt().String())
	}
}

func loadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/ringminer/config/ringminer.toml"
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}

func LoadConfigAndGenerateSimpleEthListener() *ethChainListener.EthClientListener {
	c := loadConfig()
	db := db.NewDB(c.Database)
	l := ethChainListener.NewListener(c.ChainClient, c.Common, nil, nil, nil, db)
	return l
}

func LoadConfigAndGenerateOrderBook() *orderbook.OrderBook {
	c := loadConfig()
	db := db.NewDB(c.Database)
	ob := orderbook.NewOrderBook(c.Orderbook, c.Common, db, nil)
	return ob
}
