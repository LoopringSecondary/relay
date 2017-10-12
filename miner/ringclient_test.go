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

package miner_test

import (
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	ethCryptoLib "github.com/Loopring/ringminer/crypto/eth"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"testing"
	"time"
)

var client *chainclient.Client
var imp *chainclient.LoopringProtocolImpl = &chainclient.LoopringProtocolImpl{}
var implAddress string = "0xfd9ecf92e3684451c2502cf9cdc45962a4febffa"
var registry *chainclient.LoopringRinghashRegistry = &chainclient.LoopringRinghashRegistry{}
var ringClient *miner.RingClient

func init() {
	globalConfig := config.LoadConfig("../config/ringminer.toml")
	log.Initialize(globalConfig.Log)

	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	ethClient := eth.NewChainClient(globalConfig.ChainClient)

	database := db.NewDB(globalConfig.Database)
	ringClient = miner.NewRingClient(database, ethClient.Client)
	//
	miner.Initialize(globalConfig.Miner, ringClient.Chainclient)

	client = ethClient.Client
	client.NewContract(imp, implAddress, chainclient.CurrentImplAbiStr)

	var lrcTokenAddressHex string
	imp.LrcTokenAddress.Call(&lrcTokenAddressHex, "pending")
	lrcTokenAddress := types.HexToAddress(lrcTokenAddressHex)
	lrcToken := &chainclient.Erc20Token{}
	client.NewContract(lrcToken, lrcTokenAddress.Hex(), chainclient.Erc20TokenAbiStr)

	var registryAddressHex string
	imp.RinghashRegistryAddress.Call(&registryAddressHex, "pending")
	registryAddress := types.HexToAddress(registryAddressHex)
	client.NewContract(registry, registryAddress.Hex(), chainclient.CurrentRinghashRegistryAbiStr)
}

func createOrder(tokenS, tokenB types.Address, amountS, amountB *big.Int, pkBytes []byte) *types.Order {
	order := &types.Order{}
	order.Protocol = types.HexToAddress(implAddress)
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
	order.GenerateHash()
	//order.GenerateAndSetSignature(types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"))
	order.GenerateAndSetSignature(pkBytes)
	return order
}

func TestRingClient_NewRing(t *testing.T) {
	order1 := createOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		big.NewInt(100),
		big.NewInt(1000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
	)
	order1.Owner = types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")

	order2 := createOrder(
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		big.NewInt(1000),
		big.NewInt(100),
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
	)
	order2.Owner = types.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A")

	ring := &types.Ring{}
	ring.Orders = make([]*types.FilledOrder, 0)
	fOrder1 := &types.FilledOrder{}
	fOrder1.OrderState = types.OrderState{}
	fOrder1.OrderState.RawOrder = *order1
	fOrder1.RateAmountS = order1.AmountS
	fOrder1.FeeSelection = uint8(0)

	fOrder2 := &types.FilledOrder{}
	fOrder2.OrderState = types.OrderState{}
	fOrder2.OrderState.RawOrder = *order2
	fOrder2.RateAmountS = order2.AmountS
	fOrder2.FeeSelection = uint8(0)

	cTest := &chainclient.Erc20Token{}
	client.NewContract(cTest, "0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511", chainclient.Erc20TokenAbiStr)

	cTest1 := &chainclient.Erc20Token{}
	client.NewContract(cTest1, "0x96124db0972e3522a9b3910578b3f2e1a50159c7", chainclient.Erc20TokenAbiStr)

	miner.LoopringInstance.Tokens[types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511")] = cTest
	miner.LoopringInstance.Tokens[types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7")] = cTest1

	miner.AvailableAmountS(fOrder1)
	t.Log(fOrder1.AvailableAmountS.Int64())
	ring.Orders = append(ring.Orders, fOrder1)
	ring.Orders = append(ring.Orders, fOrder2)
	ring.Hash = ring.GenerateHash()
	ring.ThrowIfTokenAllowanceOrBalanceIsInsuffcient = false

	t.Logf("ring.Hash:%x", ring.Hash)

	ringState := &types.RingState{}
	ringState.RawRing = ring
	//ringClient.NewRing(ringState)
}
