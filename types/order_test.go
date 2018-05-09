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

package types_test

import (
	"encoding/json"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/types"
	"math/big"
	"testing"
)

func TestOrder_GeneratePrice(t *testing.T) {
	ord := types.Order{}
	ord.AmountB = big.NewInt(100)
	ord.AmountS = big.NewInt(5)
	ord.GeneratePrice()

	t.Log(ord.Price.String())
}

func TestOrder_GenerateHash(t *testing.T) {
	s := `{"protocol":"0x123456789012340F73A93993E5101362656Af116",
"owner":"0x48ff2269e58a373120ffdbbdee3fbcea854ac30a",
"tokenB":"0xEF68e7C694F40c8202821eDF525dE3782458639f","tokenS":"0x2956356cD2a2bf3202F771F50D3D14A367b48070",
"authAddr":"0x90feb7c492db20afce48e830cc0c6bea1b6721dd",
"authPrivateKey":"acfe437a8e0f65124c44647737c0471b8adc9a0763f139df76766f46d6af8e15",
"amountB":"0x56bc75e2d63100000","amountS":"0x16345785d8a0000",
"lrcFee":"0xad78ebc5ac6200000",
"validSince":"0x5aa104a5",
"validUntil":"0x5ac891a5",
"marginSplitPercentage":50,"buyNoMoreThanAmountB":true,"walletId":"0x1","v":27,
"r":"0xbbc27e0aa7a3df3942ab7886b78d205d7bf8161abbece04e8d841f0de508522e","s":"0x2b19076f2fe24b58eedd00f0151d058bd7b1bf5fa38759c15902f03552492042"}`
	oJson := &types.OrderJsonRequest{}
	if err := json.Unmarshal([]byte(s), oJson); nil != err {
		t.Error(err.Error())
	} else {
		c := crypto.NewKSCrypto(true, nil)
		crypto.Initialize(c)
		o := types.ToOrder(oJson)
		t.Log(o.ValidUntil.String())
		t.Log(o.GenerateHash().Hex())
	}
}
