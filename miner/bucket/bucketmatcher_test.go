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

package bucket_test

import (
	"github.com/Loopring/ringminer/chainclient"
	ethClientLib "github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	ethCryptoLib "github.com/Loopring/ringminer/crypto/eth"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/miner/bucket"
	"github.com/Loopring/ringminer/test"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"testing"
	"time"
)

func init() {
	globalConfig := config.LoadConfig("../../config/ringminer.toml")
	globalConfig.Common.Passphrase = []byte("sa")
	log.Initialize(globalConfig.Log)

	ethClient := ethClientLib.NewChainClient(globalConfig.ChainClient, globalConfig.Common.Passphrase)
	database := db.NewDB(globalConfig.Database)
	submitter := miner.NewSubmitter(globalConfig.Miner, globalConfig.Common, database, ethClient.Client)
	matcher := bucket.NewBucketMatcher(submitter, 4)
	loopring := chainclient.NewLoopringInstance(globalConfig.Common, ethClient.Client)
	miner.MinerInstance = &miner.Miner{Loopring: loopring}
	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	matcher.Start()
}

func TestBucket(t *testing.T) {
	contractAddress := types.HexToAddress("0xd02d3e40cde61c267a3886f5828e03aa4914073d")
	order1 := test.CreateOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92512"),
		contractAddress,
		big.NewInt(1000),
		big.NewInt(100000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)
	orderState1 := &types.OrderState{}
	orderState1.RawOrder = *order1
	orderState1.States = []types.VersionData{types.VersionData{Status: types.ORDER_NEW}}

	order2 := test.CreateOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92513"),
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		contractAddress,
		big.NewInt(100000),
		big.NewInt(1000),
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)
	orderState2 := &types.OrderState{}
	orderState2.RawOrder = *order2
	orderState2.States = []types.VersionData{types.VersionData{Status: types.ORDER_NEW}}

	order3 := test.CreateOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92512"),
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92513"),
		contractAddress,
		big.NewInt(10000),
		big.NewInt(10000),
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)
	orderState3 := &types.OrderState{}
	orderState3.RawOrder = *order3
	orderState3.States = []types.VersionData{types.VersionData{Status: types.ORDER_NEW}}

	eventemitter.Emit(eventemitter.MinedOrderState, orderState1)
	eventemitter.Emit(eventemitter.MinedOrderState, orderState2)
	eventemitter.Emit(eventemitter.MinedOrderState, orderState3)
	time.Sleep(20 * time.Second)

}

//volume
func volumeTest(matcher miner.Matcher, nomorethanB bool) {
	//i := 0
	//
	////rate 0.37003947505256
	////price 2 ratePrice 1.2599210498948731647
	////volumeS: {false amountS:20000, savingAmountB:5874}   {true amountS:12599,savingAmountS:7401}
	//order1 := newOrder("token1", "token2", 20000, 10000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order1
	//
	////price 1 ratePrice 0.629960524947436
	////volume: {false amountS:15874, savingAmountB:9324} {true amountS:10000,savingAmountS:5874}
	//order2 := newOrder("token2", "token3", 30000, 30000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order2
	//
	////price 2 ratePrice 1.2599210498948731647
	////volume: {false amountS:25198, savingAmouontB:7401} {true amountS:15874,savingAmountS:9324}
	//order3 := newOrder("token3", "token1", 40000, 20000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order3
}

//choice the ring of max fee
func bestRing(matcher miner.Matcher, nomorethanB bool) {
	//i := 0
	//order1 := newOrder("token1", "token2", 20000, 10000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order1
	//
	//order4 := newOrder("token1", "token2", 80000, 20000, nomorethanB, &i)
	//
	//order2 := newOrder("token2", "token3", 30000, 30000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order2
	//matcher.GetOrderStateChan() <- order4
	//
	//order3 := newOrder("token3", "token1", 40000, 20000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order3

}

//bucket must store all of the related orders and semirings
func bucketOfAllOrders(matcher miner.Matcher, nomorethanB bool) {
	//i := 0
	//order1 := newOrder("token1", "token2", 20000, 10000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order1
	//
	//order4 := newOrder("token1", "token2", 20000, 20000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order4
	//
	//order2 := newOrder("token2", "token3", 30000, 30000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order2
	//
	//order3 := newOrder("token3", "token1", 40000, 20000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order3
}

//
func bucketOfDeleteFilledOrders(matcher miner.Matcher, nomorethanB bool) {
	//i := 0
	//order1 := newOrder("token1", "token2", 20000, 10000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order1
	//
	//order4 := newOrder("token1", "token2", 20000, 20000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order4
	//
	//order2 := newOrder("token2", "token3", 30000, 30000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order2
	//
	//order3 := newOrder("token3", "token1", 40000, 20000, nomorethanB, &i)
	//matcher.GetOrderStateChan() <- order3
}
