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
	"testing"
	"github.com/Loopring/ringminer/types"
	"time"
	"encoding/json"
	"math/big"
)

func TestOrder_MarshalJson(t *testing.T) {
	var ord types.Order

	ord.Protocol = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
	ord.TokenS = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
	ord.TokenB = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
	ord.AmountS = types.IntToBig(20000)
	ord.AmountB = types.IntToBig(800)
	ord.Expiration = uint64(time.Now().Unix())
	ord.Rand = types.IntToBig(int64(3))
	ord.LrcFee = types.IntToBig(30)
	ord.SavingSharePercentage = 51
	ord.V = 8
	ord.R = types.StringToSign("hhhhhhhh")
	ord.S = types.StringToSign("fjalskdf")

	data, err := json.Marshal(&ord)
	if err != nil {
		t.Log(err.Error())
	} else {
		t.Log(string(data))
	}
}

func TestOrder_UnMarshalJson(t *testing.T) {
	input := `
	{"protocol":"0xb794f5ea0ba39494ce839613fffba74279579268","tokenS":"0xb794f5ea0ba39494ce839613fffba74279579268","tokenB":"0xb794f5ea0ba39494ce839613fffba74279579268","amountS":20000,"amountB":800,"rand":3,"expiration":1504259224,"lrcFee":30,"savingShareRate":51,"v":8,"r":"\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000hhhhhhhh","s":"\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000fjalskdf"}
	`
	ord := &types.Order{}
	if err := ord.UnMarshalJson([]byte(input)); err != nil {
		t.Log(err.Error())
	} else {
		t.Log(ord.TokenS.Str())
		t.Log(ord.TokenB.Str())
		t.Log(ord.AmountS)
		t.Log(ord.AmountB)
		t.Log(ord.LrcFee)
	}
}

func TestOrderState_MarshalJson(t *testing.T) {
	var ord types.OrderState

	ord.RawOrder.Protocol = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
	ord.RawOrder.TokenS = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
	ord.RawOrder.TokenB = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
	ord.RawOrder.AmountS = types.IntToBig(20000)
	ord.RawOrder.AmountB = types.IntToBig(800)
	ord.RawOrder.Expiration = uint64(time.Now().Unix())
	ord.RawOrder.Rand = types.IntToBig(int64(3))
	ord.RawOrder.LrcFee = types.IntToBig(30)
	ord.RawOrder.SavingSharePercentage = 51
	ord.RawOrder.V = 8
	ord.RawOrder.R = types.StringToSign("hhhhhhhh")
	ord.RawOrder.S = types.StringToSign("fjalskdf")

	ord.RemainedAmountS = types.IntToBig(10000)
	ord.RemainedAmountB = types.IntToBig(400)
	ord.Owner = types.StringToAddress("0x3334f5ea0ba39494ce839613fffba74279579268")
	ord.OrderHash = types.StringToHash("Qme85LtECPhvx4Px5i7s2Ht2dXdHrgXYpqkDsKvxdpFQP4")

	if data, err := ord.MarshalJson(); err != nil {
		t.Log(err.Error())
	} else {
		t.Log(string(data))
	}
}

func TestOrderState_UnMarshalJson(t *testing.T) {
	input := `
	{
		"protocol":"0xb794f5ea0ba39494ce839613fffba74279579268",
		"tokenS":"0xb794f5ea0ba39494ce839613fffba74279579268",
		"tokenB":"0xb794f5ea0ba39494ce839613fffba74279579268",
		"amountS":20000,
		"amountB":800,
		"rand":3,
		"expiration":1504259224,
		"lrcFee":30,
		"savingShareRate":51,
		"v":8,"r":"\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000hhhhhhhh",
		"s":"\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000fjalskdf",
		"owner":"0ba39494ce839613fffba74279579268",
		"hash":"dHrgXYpqkDsKvxdpFQP4",
		"remainedAmountS":10000,
		"remainedAmountB":400,
		"status":0
	}`

	ord := &types.OrderState{}
	if err := ord.UnMarshalJson([]byte(input)); err != nil {
		t.Log(err.Error())
	} else {
		t.Log(ord.OrderHash.Str())
		t.Log(ord.RemainedAmountS)
		t.Log(ord.RemainedAmountB)
		t.Log(ord.RawOrder.TokenS.Str())
		t.Log(ord.RawOrder.TokenB.Str())
		t.Log(ord.RawOrder.AmountS)
		t.Log(ord.RawOrder.AmountB)
		t.Log(ord.RawOrder.LrcFee)
	}
}

func TestNewOrderUnMarshal(t *testing.T) {

	type Address [10]byte
	type order struct {
		Protocol              Address	`json:"protocol"`
		Amount                *big.Int  `json:"amount"`
	}

	str := `{"protocol":"aaaaabbbbb","amount":10000001000000100000010000001000000100000010000001000000}`
	var res order
	json.Unmarshal([]byte(str), &res)
	t.Log("protocol", len(res.Protocol))
	t.Log("amount", res.Amount)

	for i:=0;i<8;i++ {
		t.Log(res.Amount.Div(res.Amount, big.NewInt(1000000)))
	}
}