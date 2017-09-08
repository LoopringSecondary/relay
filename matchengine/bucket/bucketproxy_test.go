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
	"math/big"
	"strconv"
	"github.com/Loopring/ringminer/types"
	"testing"
	"github.com/Loopring/ringminer/matchengine/bucket"
)

func newOrder(outToken string, inToken string, outAmount, inAmount int64, buyFirstEnough bool, idx int) *types.OrderState {
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
	order.LrcFee = big.NewInt(1000)
	order.SavingSharePercentage = 30
	h := &types.Hash{}
	h.SetBytes([]byte(strconv.Itoa(idx)))
	orderState.RawOrder = *order
	orderState.OrderHash = *h
	return orderState
}

func TestBucketProxy(t *testing.T) {
	proxy := bucket.NewBucketProxy()
	go proxy.Start()

	order1 := newOrder("token1", "token2", 20000, 30000, true, 1)

	proxy.NewOrder(order1)

	order2 := newOrder("token2", "token3", 40000, 30000, true,  2)
	proxy.NewOrder(order2)

	order3 := newOrder("token3", "token1", 40000, 20000, true,  3)
	proxy.NewOrder(order3)
}