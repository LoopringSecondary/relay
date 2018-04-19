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
package dao_test

import (
	"fmt"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/types"
	"testing"
)

func TestRdsServiceImpl_NewOrder(t *testing.T) {
	o := dao.Order{}
	o.ID = 1
	o.Protocol = "0xC01172a87f6cC20E1E3b9aD13a9E715Fbc2D5AA9"
	o.Owner = "0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A"
	o.PrivateKey = "acfe437a8e0f65124c44647737c0471b8adc9a0763f139df76766f46d6af8e15"
	o.OrderHash = "0xcbb02f5df389993aea21e98e7ade8ae9f34e57eeb639dcd754ba79a0223d51e5"
	o.TokenS = "0x2956356cD2a2bf3202F771F50D3D14A367b48070"
	o.TokenB = "0xEF68e7C694F40c8202821eDF525dE3782458639f"
	o.AmountS = "100000000000000000"
	o.AmountB = "100000000000000000000"
	o.CreateTime = 1520507123
	o.ValidSince = 1520501925
	o.ValidUntil = 1523093925
	o.LrcFee = "200000000000000000000"
	o.BuyNoMoreThanAmountB = true
	o.MarginSplitPercentage = 50
	o.V = 27
	o.R = "0xbbc27e0aa7a3df3942ab7886b78d205d7bf8161abbece04e8d841f0de508522e"
	o.S = "0x2b19076f2fe24b58eedd00f0151d058bd7b1bf5fa38759c15902f03552492042"
	o.Price = 0.001
	o.UpdatedBlock = 0
	o.DealtAmountS = "0"
	o.DealtAmountB = "0"
	o.Market = "LRC-WETH"
	var state types.OrderState
	c := crypto.NewKSCrypto(true, nil)
	crypto.Initialize(c)
	o.ConvertUp(&state)
	fmt.Println(state)

}
