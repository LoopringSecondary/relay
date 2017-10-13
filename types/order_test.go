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
	"github.com/Loopring/ringminer/types"
	"testing"
	//ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

//func TestOrder_MarshalJson(t *testing.T) {
//	var ord types.Order
//
//	ord.Protocol = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.TokenS = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.TokenB = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.AmountS = types.IntToBig(20000)
//	ord.AmountB = types.IntToBig(800)
//	ord.Expiration = uint64(time.Now().Unix())
//	ord.Rand = types.IntToBig(int64(3))
//	ord.LrcFee = types.IntToBig(30)
//	ord.SavingSharePercentage = 51
//	ord.V = 8
//	ord.R = types.StringToSign("hhhhhhhh")
//	ord.S = types.StringToSign("fjalskdf")
//
//	data, err := json.Marshal(&ord)
//	if err != nil {
//		t.Log(err.Error())
//	} else {
//		t.Log(string(data))
//	}
//}

//func TestOrderState_MarshalJson(t *testing.T) {
//	var ord types.OrderState
//
//	ord.RawOrder.Protocol = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.RawOrder.TokenS = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.RawOrder.TokenB = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.RawOrder.AmountS = types.IntToBig(20000)
//	ord.RawOrder.AmountB = types.IntToBig(800)
//	ord.RawOrder.Expiration = uint64(time.Now().Unix())
//	ord.RawOrder.Rand = types.IntToBig(int64(3))
//	ord.RawOrder.LrcFee = types.IntToBig(30)
//	ord.RawOrder.SavingSharePercentage = 51
//	ord.RawOrder.V = 8
//	ord.RawOrder.R = types.StringToSign("hhhhhhhh")
//	ord.RawOrder.S = types.StringToSign("fjalskdf")
//
//	ord.RemainedAmountS = types.IntToBig(10000)
//	ord.RemainedAmountB = types.IntToBig(400)
//	ord.Owner = types.StringToAddress("0x3334f5ea0ba39494ce839613fffba74279579268")
//	ord.OrderHash = types.StringToHash("Qme85LtECPhvx4Px5i7s2Ht2dXdHrgXYpqkDsKvxdpFQP4")
//
//	if data, err := ord.MarshalJson(); err != nil {
//		t.Log(err.Error())
//	} else {
//		t.Log(string(data))
//	}
//}
//
//func TestOrderState_UnMarshalJson(t *testing.T) {
//	input := `
//	{
//		"protocol":"0xb794f5ea0ba39494ce839613fffba74279579268",
//		"tokenS":"0xb794f5ea0ba39494ce839613fffba74279579268",
//		"tokenB":"0xb794f5ea0ba39494ce839613fffba74279579268",
//		"amountS":20000,
//		"amountB":800,
//		"rand":3,
//		"expiration":1504259224,
//		"lrcFee":30,
//		"savingShareRate":51,
//		"v":8,"r":"\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000hhhhhhhh",
//		"s":"\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000\u0000fjalskdf",
//		"owner":"0ba39494ce839613fffba74279579268",
//		"hash":"dHrgXYpqkDsKvxdpFQP4",
//		"remainedAmountS":10000,
//		"remainedAmountB":400,
//		"status":0
//	}`
//
//	ord := &types.OrderState{}
//	if err := ord.UnMarshalJson([]byte(input)); err != nil {
//		t.Log(err.Error())
//	} else {
//		t.Log(ord.OrderHash.Str())
//		t.Log(ord.RemainedAmountS)
//		t.Log(ord.RemainedAmountB)
//		t.Log(ord.RawOrder.TokenS.Str())
//		t.Log(ord.RawOrder.TokenB.Str())
//		t.Log(ord.RawOrder.AmountS)
//		t.Log(ord.RawOrder.AmountB)
//		t.Log(ord.RawOrder.LrcFee)
//	}
//}
//
//func TestNewOrderUnMarshal(t *testing.T) {
//
//	//type Address [10]byte
//	type order struct {
//		Protocol              types.Address	`json:"protocol"`
//		Amount                *big.Int  `json:"amount"`
//	}
//
//	str := `{"protocol":"0xb794f5ea0ba39494ce839613fffba74279579268","amount":10000001000000100000010000001000000100000010000001000000}`
//	var res order
//	json.Unmarshal([]byte(str), &res)
//	t.Log("protocol", len(res.Protocol))
//	t.Log("amount", res.Amount)
//
//	//for i:=0;i<8;i++ {
//	//	t.Log(res.Amount.Div(res.Amount, big.NewInt(1000000)))
//	//}
//
//	t.Log(res.Protocol.Str())
//
//	data, err := json.Marshal(&res)
//	if err != nil {
//		t.Log(err.Error())
//	}
//
//	t.Log(string(data))
//}

func TestOrder_MarshalJson(t *testing.T) {
	order := &types.Order{}
	order.Protocol = types.HexToAddress("0x211c9fb2c5ad60a31587a4a11b289e37ed3ea520")
	order.TokenS = types.HexToAddress("0xc184dd351f215f689f481c329916bb33d8df8ced")
	order.TokenB = types.HexToAddress("0xc184dd351f215f689f481c329916bb33d8df8ced")
	order.AmountS = big.NewInt(100000)
	order.AmountB = big.NewInt(100000000)

	order.Salt = big.NewInt(10000)
	order.Ttl = big.NewInt(1000000)
	order.LrcFee = big.NewInt(10000000000)
	order.MarginSplitPercentage = 40
	order.BuyNoMoreThanAmountB = false

	order.V = byte(1)
	order.R = big.NewInt(10000000)
	order.S = big.NewInt(200000)

	if bytes, err := order.MarshalJSON(); err != nil {
		t.Error(err)
	} else {
		t.Log(string(bytes))
	}
}

func TestOrder_UnMarshalJson(t *testing.T) {
	input := "{\"protocol\":\"0x4ec94e1007605d70a86279370ec5e4b755295eda\"," +
		"\"tokenS\":\"0xc184dd351f215f689f481c329916bb33d8df8ced\"," +
		"\"tokenB\":\"0xc184dd351f215f689f481c329916bb33d8df8ced\"," +
		"\"amountS\":\"0x0186a0\"," +
		"\"amountB\":\"0x05f5e100\"," +
		"\"rand\":\"0x2710\"," +
		"\"expiration\":\"0x0f4240\"," +
		"\"lrcFee\":\"0x02540be400\"," +
		"\"savingSharePercentage\":30," +
		"\"buyNoMoreThanAmountB\":false," +
		"\"v\":1," +
		"\"r\":\"0x02540be400\"," +
		"\"s\":\"0x02540be400\"" +
		"}"

	//types.Crypto = &eth.EthCrypto{Homestead: false}
	//pkHex := "4f5b916dc82fb59cc57dbdd2fee5b49b2bdfe6ea34534a5d40c4475e9740c66e"
	//pk,_ := ethCrypto.HexToECDSA(pkHex)
	ord := &types.Order{}
	if err := ord.UnmarshalJSON([]byte(input)); err != nil {
		t.Log(err.Error())
	} else {
		//state := ord.Convert()
		//state.GenHash()
		//if sig, err := types.Crypto.Sign(state.OrderHash.Bytes(), common.Hex2Bytes(pkHex)); err != nil {
		//	println(err.Error())
		//} else {
		//	v, r, s := types.Crypto.SigToVRS(sig)
		//	state.RawOrder.V = uint8(v)
		//	state.RawOrder.R = r
		//	state.RawOrder.S = s
		//
		//	println("r:", common.Bytes2Hex(r.Bytes()), " s:", common.Bytes2Hex(s.Bytes()))
		//}
		//addr, _ := state.SignerAddress()
		t.Logf("protocol:%s, tokenS:%s, tokenB:%s, amountS:%d", ord.Protocol.Hex(), ord.TokenS.Hex(), ord.TokenB.Hex(), ord.AmountS.Int64())
	}
}

func TestOrderState_MarshalJson(t *testing.T) {
	//orderState := &types.OrderState{}
	//order := &types.Order{}
	//order.Protocol = types.HexToAddress("0x211c9fb2c5ad60a31587a4a11b289e37ed3ea520")
	//order.TokenS = types.HexToAddress("0xc184dd351f215f689f481c329916bb33d8df8ced")
	//order.TokenB = types.HexToAddress("0xc184dd351f215f689f481c329916bb33d8df8ced")
	//order.AmountS = big.NewInt(100000)
	//order.AmountB = big.NewInt(100000000)
	//
	//order.Rand = big.NewInt(10000)
	//order.Expiration = big.NewInt(1000000)
	//order.LrcFee = big.NewInt(10000000000)
	//order.SavingSharePercentage = 40
	//order.BuyNoMoreThanAmountB = false
	//
	//order.V = byte(1)
	//order.R = big.NewInt(10000000)
	//order.S = big.NewInt(200000)
	//orderState.RawOrder = *order
	//
	//orderState.Owner = types.HexToAddress("0xc184dd351f215f689f481c329916bb33d8df8ced")

	//if bytes,err := orderState();err != nil {
	//	t.Error(err)
	//} else {
	//	t.Log(string(bytes))
	//}
}

func TestOrderState_UnMarshalJson(t *testing.T) {
	//input := "{\"protocol\":\"0x4ec94e1007605d70a86279370ec5e4b755295eda\"," +
	//	"\"tokenS\":\"0xc184dd351f215f689f481c329916bb33d8df8ced\"," +
	//	"\"tokenB\":\"0xc184dd351f215f689f481c329916bb33d8df8ced\"," +
	//	"\"amountS\":\"0x0186a0\"," +
	//	"\"amountB\":\"0x05f5e100\"," +
	//	"\"rand\":\"0x2710\"," +
	//	"\"expiration\":\"0x0f4240\"," +
	//	"\"lrcFee\":\"0x02540be400\"," +
	//	"\"savingSharePercentage\":30," +
	//	"\"buyNoMoreThanAmountB\":false" +
	//	"}"
	//var o types.OrderState
	//if err := o.UnMarshalJson([]byte(input)); err != nil {
	//	t.Log(err.Error())
	//} else {
	//	t.Logf("protocol:%s", o.RawOrder.Protocol.Hex())
	//}
}
