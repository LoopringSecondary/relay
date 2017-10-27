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
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/miner/bucket"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"strconv"
	"testing"
	"time"
)

func newOrder(outToken string, inToken string, outAmount, inAmount int64, buyFirstEnough bool, idx *int) *types.OrderState {
	*idx++

	orderState := &types.OrderState{}
	order := &types.Order{}

	outAddress := &types.Address{}
	outAddress.SetBytes([]byte(outToken))
	inAddress := &types.Address{}
	inAddress.SetBytes([]byte(inToken))

	order.TokenS = *outAddress
	order.TokenB = *inAddress
	order.AmountS = big.NewInt(outAmount)
	order.AmountB = big.NewInt(inAmount)
	order.BuyNoMoreThanAmountB = buyFirstEnough
	order.LrcFee = big.NewInt(10)
	order.MarginSplitPercentage = 30
	h := &types.Hash{}
	h.SetBytes([]byte(strconv.Itoa(*idx)))
	orderState.RawOrder = *order
	orderState.RawOrder.Hash = *h
	orderState.Status = types.ORDER_NEW

	return orderState
}

func TestBucket_GenerateRing(t *testing.T) {
	log.Initialize()
	ringClient := miner.NewRingSubmitClient()
	//ringClient.Start()
	c := &config.MinerOptions{}
	proxy := bucket.NewBucketProxy(*c, ringClient)
	debugRingChan := make(chan *types.RingState)
	go proxy.Start(debugRingChan)
	go listentRingStored(debugRingChan)

	volumeTest(proxy, true)

	//bestRing(proxy, false)

	time.Sleep(100000000)
}

//volume
func volumeTest(proxy miner.Proxy, nomorethanB bool) {
	i := 0

	//rate 0.37003947505256
	//price 2 ratePrice 1.2599210498948731647
	//volumeS: {false amountS:20000, savingAmountB:5874}   {true amountS:12599,savingAmountS:7401}
	order1 := newOrder("token1", "token2", 20000, 10000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order1

	//price 1 ratePrice 0.629960524947436
	//volume: {false amountS:15874, savingAmountB:9324} {true amountS:10000,savingAmountS:5874}
	order2 := newOrder("token2", "token3", 30000, 30000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order2

	//price 2 ratePrice 1.2599210498948731647
	//volume: {false amountS:25198, savingAmouontB:7401} {true amountS:15874,savingAmountS:9324}
	order3 := newOrder("token3", "token1", 40000, 20000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order3
}

//choice the ring of max fee
func bestRing(proxy miner.Proxy, nomorethanB bool) {
	i := 0
	order1 := newOrder("token1", "token2", 20000, 10000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order1

	order4 := newOrder("token1", "token2", 80000, 20000, nomorethanB, &i)

	order2 := newOrder("token2", "token3", 30000, 30000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order2
	proxy.GetOrderStateChan() <- order4

	order3 := newOrder("token3", "token1", 40000, 20000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order3

}

//bucket must store all of the related orders and semirings
func bucketOfAllOrders(proxy miner.Proxy, nomorethanB bool) {
	i := 0
	order1 := newOrder("token1", "token2", 20000, 10000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order1

	order4 := newOrder("token1", "token2", 20000, 20000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order4

	order2 := newOrder("token2", "token3", 30000, 30000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order2

	order3 := newOrder("token3", "token1", 40000, 20000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order3
}

//
func bucketOfDeleteFilledOrders(proxy miner.Proxy, nomorethanB bool) {
	i := 0
	order1 := newOrder("token1", "token2", 20000, 10000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order1

	order4 := newOrder("token1", "token2", 20000, 20000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order4

	order2 := newOrder("token2", "token3", 30000, 30000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order2

	order3 := newOrder("token3", "token1", 40000, 20000, nomorethanB, &i)
	proxy.GetOrderStateChan() <- order3
}

func listentRingStored(debugRingChan chan *types.RingState) {
	for {
		select {
		case orderRing := <-debugRingChan:
			s := ""
			for _, o := range orderRing.RawRing.Orders {
				s = s + " -> " + " {outtoken:" + string(o.OrderState.RawOrder.TokenS.Bytes()) +
					", fillamountS:" + o.FillAmountS.RealValue().String() +
					", intoken:" + string(o.OrderState.RawOrder.TokenB.Bytes()) +
					", idx:" + o.OrderState.RawOrder.Hash.Str() +
					"}"
			}
			log.Infof("ringChan receive:%s ring is %s", string(orderRing.Hash.Bytes()), s)
		}
	}
}
