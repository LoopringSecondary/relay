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
	"fmt"
	"github.com/Loopring/relay/dao"
	//"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/test"
	//"github.com/Loopring/relay/types"
	//"github.com/ethereum/go-ethereum/common"
	//"math/big"
	"testing"
	//"time"
	"time"
	//"github.com/ethereum/go-ethereum/common"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/Loopring/relay/usermanager"
	"math/big"
)

var (
	tm *market.TrendManager
	bf *gateway.BaseFilter
)

func prepare() {
	//fmt.Println("yyyyyyyyyy")
	globalConfig := test.LoadConfig()
	//fmt.Println(globalConfig)
	fmt.Println("11111111111======================================")
	rdsService := dao.NewRdsService(globalConfig.Mysql)
	userManager := usermanager.NewUserManager(&globalConfig.UserManager, rdsService)
	marketCapProvider := marketcap.NewMarketCapProvider(globalConfig.MarketCap)
	orderManager := ordermanager.NewOrderManager(&globalConfig.OrderManager, rdsService, userManager, marketCapProvider)
	gateway.Initialize(&globalConfig.GatewayFilters, &globalConfig.Gateway, &globalConfig.Ipfs, orderManager, marketCapProvider)
	baseFilter := &gateway.BaseFilter{
		MinLrcFee:             big.NewInt(globalConfig.GatewayFilters.BaseFilter.MinLrcFee),
		MaxPrice:              big.NewInt(globalConfig.GatewayFilters.BaseFilter.MaxPrice),
		MinSplitPercentage:    globalConfig.GatewayFilters.BaseFilter.MinSplitPercentage,
		MaxSplitPercentage:    globalConfig.GatewayFilters.BaseFilter.MaxSplitPercentage,
		MinTokeSAmount:        make(map[string]*big.Int),
		MinTokenSUsdAmount:    globalConfig.GatewayFilters.BaseFilter.MinTokenSUsdAmount,
		MaxValidSinceInterval: globalConfig.GatewayFilters.BaseFilter.MaxValidSinceInterval,
	}
	for k, v := range globalConfig.GatewayFilters.BaseFilter.MinTokeSAmount {
		minAmount := big.NewInt(0)
		amount, succ := minAmount.SetString(v, 10)
		if succ {
			baseFilter.MinTokeSAmount[k] = amount
		}
	}

	fmt.Println("ksdfjlsdjflksdjfkjdsklfjklsdjf")
	fmt.Println(baseFilter)
	bf = baseFilter

}

func TestTrendManager_GetTicker(t *testing.T) {

	//fmt.Println("ZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	prepare()

	order := types.Order{}
	order.AmountS = big.NewInt(1000000)
	order.LrcFee = big.NewInt(500000000000000000)
	order.TokenS = util.AllTokens["RDN"].Protocol
	order.TokenB = util.AllTokens["WETH"].Protocol
	amountS := big.NewInt(0)
	amountS.SetString("3800000000000000000", 10)
	order.AmountS = amountS
	order.AmountB = big.NewInt(100000000000000)
	now := time.Now().Unix() + 3558
	order.ValidUntil = big.NewInt(time.Now().Unix() + 3)
	order.ValidSince = big.NewInt(now)
	order.GeneratePrice()
	order.MarginSplitPercentage = 20

	fmt.Println(bf.Filter(&order))

	//fmt.Println(tm.GetTrends("RDN-WETH", "1Hr"))
	//fmt.Println(tm.GetTrends("LRC-WETH", "1Hr"))
	//fmt.Println(tm.GetTrends("LRC-WETH", "2Hr"))
	//fill := &types.OrderFilledEvent{}
	//fill.Market = "LRC-WETH"
	//fill.TxInfo.Protocol = common.HexToAddress("0xC01172a87f6cC20E1E3b9aD13a9E715Fbc2D5AA9")
	//fill.Owner = common.HexToAddress("0x750aD4351bB728ceC7d639A9511F9D6488f1E259")
	//fill.RingIndex = big.NewInt(26)
	//fill.TxInfo.BlockNumber = big.NewInt(38811)
	//fill.TokenS = common.HexToAddress("0xfe5aFA7BfF3394359af2D277aCc9f00065CdBe2f")
	//fill.TokenB = common.HexToAddress("0x88699e7FEE2Da0462981a08a15A3B940304CC516")
	//fill.SplitS = big.NewInt(0)
	//fill.SplitB = big.NewInt(0)
	//fill.LrcReward = big.NewInt(0)
	//fill.LrcFee = big.NewInt(0)
	//fill.AmountS = big.NewInt(11000000000000)
	//fill.AmountB = big.NewInt(22000000000000000)
	//fill.OrderHash = common.HexToHash("0xc9b1edd3af78055d565731846ca7f43d0fb1985898ac1c77b6a7723f7df5491e")
	//fill.NextOrderHash = common.HexToHash("0x739a06fc7db957605a8727b21e273606bbfe1ebbbf7d5c758e8f87ae6a05657f")
	//fill.PreOrderHash = common.HexToHash("0x739a06fc7db957605a8727b21e273606bbfe1ebbbf7d5c758e8f87ae6a05657f")
	//fill.TxHash = common.HexToHash("0xc91f52ccfc73f592706ac30108dbb5853cf4a380a2ff3407c78f8b8b59599e6c")
	//fill.Ringhash = common.HexToHash("0xa3ebd7c6b39aea8beaf6554ca167bf41be69ab78f347cfbc9a7a11e61f40e8be")
	//fill.FillIndex = big.NewInt(10)
	//
	//eventemitter.Emit(eventemitter.OrderManagerExtractorFill, fill)
	//time.Sleep(3 * time.Second)
	//fill.AmountS = big.NewInt(10000000000000)
	//fill.AmountB = big.NewInt(25000000000000000)
	//eventemitter.Emit(eventemitter.OrderManagerExtractorFill, fill)
	//time.Sleep(3 * time.Second)
	////tm.InsertTrend()
	//fmt.Println("xxxxxxxxxxx")
	//tds, _ := tm.GetTrends("LRC-WETH", "2Hr")
	//fmt.Println(tds)
	//fmt.Println(tm.GetTicker())

	fmt.Println("------------------------------------------> check finished")
	time.Sleep(60 * 60 * time.Second)

}
