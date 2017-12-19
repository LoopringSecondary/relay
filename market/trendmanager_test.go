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

package market_test

import (
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/market"
	"testing"
	"fmt"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"time"
)

var (
	tm    *market.TrendManager
)

func prepare() {
	fmt.Println("yyyyyyyyyy")
	globalConfig := test.LoadConfig()
	fmt.Println("xxxxxxxxxxxxxxxskdfjskfj")
	rdsService := dao.NewRdsService(globalConfig.Mysql)
	rdsService.Prepare()
	tm2 := market.NewTrendManager(rdsService)
	tm = &tm2

}

func TestTrendManager_GetTicker(t *testing.T) {
	t.Log("ZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	prepare()
	tm.GetTrends("RDN-WETH")
	fill := &types.OrderFilledEvent{}
	fill.Market = "LRC-WETH"
	fill.Time = big.NewInt(1513319197)
	fill.ContractAddress = common.HexToAddress("0xC01172a87f6cC20E1E3b9aD13a9E715Fbc2D5AA9")
	fill.Owner = common.HexToAddress("0x750aD4351bB728ceC7d639A9511F9D6488f1E259")
	fill.RingIndex = big.NewInt(26)
	fill.Blocknumber = big.NewInt(38811)
	fill.TokenS = common.HexToAddress("0x7599aa3D5B9019cFae7c934f5d42d18891cb3CAf")
	fill.TokenB = common.HexToAddress("0x88699e7FEE2Da0462981a08a15A3B940304CC516")
	fill.SplitS = big.NewInt(0)
	fill.SplitB = big.NewInt(0)
	fill.LrcReward = big.NewInt(0)
	fill.LrcFee = big.NewInt(0)
	fill.AmountS = big.NewInt(11000000000000)
	fill.AmountB = big.NewInt(22000000000000000)
	fill.OrderHash = common.HexToHash("0xc9b1edd3af78055d565731846ca7f43d0fb1985898ac1c77b6a7723f7df5491e")
	fill.NextOrderHash = common.HexToHash("0x739a06fc7db957605a8727b21e273606bbfe1ebbbf7d5c758e8f87ae6a05657f")
	fill.PreOrderHash = common.HexToHash("0x739a06fc7db957605a8727b21e273606bbfe1ebbbf7d5c758e8f87ae6a05657f")
	fill.TxHash = common.HexToHash("0xc91f52ccfc73f592706ac30108dbb5853cf4a380a2ff3407c78f8b8b59599e6c")
	fill.Ringhash = common.HexToHash("0xa3ebd7c6b39aea8beaf6554ca167bf41be69ab78f347cfbc9a7a11e61f40e8be")
	fill.FillIndex = big.NewInt(10)

	eventemitter.Emit(eventemitter.OrderManagerExtractorFill, fill)
	time.Sleep(3 * time.Second)
	fill.AmountS = big.NewInt(10000000000000)
	fill.AmountB = big.NewInt(25000000000000000)
	eventemitter.Emit(eventemitter.OrderManagerExtractorFill, fill)
	time.Sleep(3 * time.Second)
	fmt.Println("xxxxxxxxxxx")
	fmt.Println(tm.GetTrends("RDN-WETH"))

	fmt.Println(tm.GetTicker())
	t.Error("fuckfuckfuck")

}
