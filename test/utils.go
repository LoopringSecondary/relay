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
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"os"
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
}

var testAccounts = map[string]string{
	"0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A": "07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e",
	"0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2": "11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446",
}

var testTokens = []string{"0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e", "0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f"}

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

	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/ringminer/config/ringminer.toml"
	globalConfig := config.LoadConfig(path)
	log.Initialize(globalConfig.Log, globalConfig.LogDir)

	params.ImplAddress = types.HexToAddress(globalConfig.Common.LoopringImpAddresses[0])
	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	ethClient := eth.NewChainClient(globalConfig.ChainClient)
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
	passphrase.SetBytes([]byte(globalConfig.Miner.Passphrase))
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

func (testParams *TestParams) TestPrepareData() {

	accounts := []string{}
	for k, _ := range testParams.Accounts {
		accounts = append(accounts, k)
	}

	//delegate registry
	delegateContract := &chainclient.TransferDelegate{}
	testParams.Client.NewContract(delegateContract, testParams.DelegateAddress.Hex(), chainclient.TransferDelegateAbiStr)
	delegateContract.AddVersion.SendTransaction(testParams.Owner, common.HexToAddress(testParams.ImplAddress.Hex()))

	//token registry
	tokenRegistry := &chainclient.TokenRegistry{}
	testParams.Client.NewContract(tokenRegistry, testParams.TokenRegistryAddress.Hex(), chainclient.TokenRegistryAbiStr)
	for _, tokenAddr := range testParams.TokenAddrs {
		tokenRegistry.RegisterToken.SendTransaction(testParams.Owner, common.HexToAddress(tokenAddr))
	}
	testParams.approveToLoopring(accounts, testParams.TokenAddrs, big.NewInt(30000000))
}

func (testParams *TestParams) approveToLoopring(accounts []string, tokenAddrs []string, amount *big.Int) {
	token := &chainclient.Erc20Token{}

	for _, tokenAddr := range tokenAddrs {
		testParams.Client.NewContract(token, tokenAddr, chainclient.Erc20TokenAbiStr)

		for _, account := range accounts {
			//balance := &types.Big{}
			//
			//token.BalanceOf.Call(balance, "pending", common.HexToAddress(account))
			//t.Log(balance.BigInt().String())
			//token.Allowance.Call(balance, "pending", common.HexToAddress(account), common.HexToAddress(implAddress))
			//
			//t.Log(balance.BigInt().String())

			if txHash, err := token.Approve.SendTransaction(types.HexToAddress(account), testParams.DelegateAddress, amount); nil != err {
				println(err.Error())
			} else {
				println(txHash)
			}
		}

	}
}
