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

package gateway_test

import (
	//"github.com/Loopring/relay/types"
	//"math/big"
	"testing"
	//"github.com/Loopring/relay/market"
	//"fmt"
	//"github.com/Loopring/relay/gateway"
	//"github.com/Loopring/relay/crypto"
	"reflect"
	//"github.com/libp2p/go-libp2p-interface-conn"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Loopring/relay/gateway"
)

type AB struct {
	s string
}

type ABRes1 struct {
	A string
	B int
}

type ABRes2 struct {
	C string
	D int
}

type ABReq1 struct {
	A string
	B int
}

type ABReq2 struct {
	C string
}

func (ab *AB) ABTest1(query ABReq1) (res1 ABRes1, err error) {
	return ABRes1{A: "AA", B: 11}, nil
}

func (ab *AB) ABTest2() (res2 ABRes2, err error) {
	fmt.Println("step in abtest 2.....")
	return ABRes2{C: "CC", D: 11}, nil
}

func handleWithT(ab *AB, query interface{}, methodName string, ctx string) {

	results := make([]reflect.Value, 0)
	var err error

	//reflect.ValueOf(query).Elem().
	if query == nil {
		results = reflect.ValueOf(ab).MethodByName(methodName).Call(nil)
	} else {
		queryType := reflect.TypeOf(query)
		queryClone := reflect.New(queryType)
		err = json.Unmarshal([]byte(ctx), queryClone.Interface())
		if err != nil {
			fmt.Println("unmarshal error " + err.Error())
		}
		params := make([]reflect.Value, 1)
		params[0] = queryClone.Elem()
		results = reflect.ValueOf(ab).MethodByName(methodName).Call(params)
	}

	res := results[0]
	if results[1].Interface() == nil {
		err = nil
	} else {
		err = results[1].Interface().(error)
	}
	if err != nil {
		fmt.Println("invoke error .voke error .voke error .voke error ." + err.Error())
	} else {
		fmt.Println(res)
		b, _ := json.Marshal(res.Interface())
		fmt.Println(b)
	}
}

func SetField(obj interface{}, name string, value interface{}) error {
	structValue := reflect.ValueOf(obj)
	structFieldValue := structValue.FieldByName(name)

	if !structFieldValue.IsValid() {
		return fmt.Errorf("No such field: %s in obj", name)
	}

	if !structFieldValue.CanSet() {
		return fmt.Errorf("Cannot set %s field value", name)
	}

	structFieldType := structFieldValue.Type()
	val := reflect.ValueOf(value)
	if structFieldType != val.Type() {
		return errors.New("Provided value type didn't match obj field type")
	}

	structFieldValue.Set(val)
	return nil
}

func TestWalletServiceImpl_GetPortfolio(t *testing.T) {
	//priceQuoteMap := make(map[string]*big.Rat)
	//priceQuoteMap["WETH"] = new(big.Rat).SetFloat64(4532.01)
	//priceQuoteMap["RDN"] = new(big.Rat).SetFloat64(12.01)
	//priceQuoteMap["LRC"] = new(big.Rat).SetFloat64(2.32)
	//balances := make(map[string]market.Balance)
	//balances["WETH"] = market.Balance{Token:"WETH", Balance:types.HexToBigint("0x22")}
	//balances["LRC"] = market.Balance{Token:"LRC", Balance:types.HexToBigint("0x1")}
	//balances["RDN"] = market.Balance{Token:"RDN", Balance:types.HexToBigint("0x23")}
	//
	//totalAsset := big.NewRat(0, 1)
	//for k, v := range balances {
	//	asset := new(big.Rat).Set(priceQuoteMap[k])
	//	asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
	//	totalAsset = totalAsset.Add(totalAsset, asset)
	//}
	//
	//fmt.Println("total asset is .........")
	//fmt.Println(totalAsset.Float64())
	//fmt.Println("xxxxxxxxxxxx")
	//
	//for k, v := range balances {
	//	portfolio := gateway.Portfolio{Token: k, Amount: types.BigintToHex(v.Balance)}
	//	asset := new(big.Rat).Set(priceQuoteMap[k])
	//	fmt.Println(asset.Float64())
	//	asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
	//	fmt.Println(asset.Float64())
	//	percentage, _ := asset.Quo(asset, totalAsset).Float64()
	//	fmt.Println("percentage .......")
	//	fmt.Println(percentage)
	//	portfolio.Percentage = fmt.Sprintf("%.4f%%", 100*percentage)
	//	fmt.Println(portfolio.Percentage)
	//}
	//
	//s, _ := crypto.NewPrivateKeyCrypto(false, "0x7d0a1121fb170361b6483d922d72258e6d4da9aa65234ac7ba0c9c833e6adc71")
	//fmt.Println(s.Address().Hex())

	txNotify := gateway.TxNotify{}
	txNotify.Hash = "0x0ceff977a5335b34eb6a5e16f29c7ba815d30d20e061f3e0a0b1dcbacfeed9b4"
	txNotify.From = "0x71c079107b5af8619d54537a93dbf16e5aab4900"
	txNotify.To = "0xf5b3b365fa319342e89a3da71ba393e12d9f63c3"
	txNotify.Nonce = "0x2b"
	txNotify.Gas = "0x4e3b29200"
	txNotify.GasPrice = "0x15f90"
	txNotify.Value = "0x10"
	txNotify.V = "0x25"
	txNotify.R = "0x94b92f2068f8e41b8c2987d8f184345a016505addbda340261feebaedab38dc4"
	txNotify.S = "0x6e8029ad29eeb0a1368b352de54dc452cef1bf1efe36b2327ccb06ae5c29b7ba"
	txNotify.Input = "0xa9059cbb0000000000000000000000008311804426a24495bd4306daf5f595a443a52e32000000000000000000000000000000000000000000000000000000174876e800"
	gateway.NotifyTransactionSubmitted(txNotify)

	//ethTx := &ethTypes.Transaction{}
	//a := "0xf8aa248504e3b2920083015f9094f5b3b365fa319342e89a3da71ba393e12d9f63c380b844a9059cbb0000000000000000000000008311804426a24495bd4306daf5f595a443a52e32000000000000000000000000000000000000000000000000000000174876e80025a094b92f2068f8e41b8c2987d8f184345a016505addbda340261feebaedab38dc4a06e8029ad29eeb0a1368b352de54dc452cef1bf1efe36b2327ccb06ae5c29b7ba"
	//if strings.HasPrefix(a, "0x") {
	//	a = strings.TrimLeft(a, "0x")
	//	fmt.Println(a)
	//}
	//fmt.Println(common.StringToHash(a).Str())
	//fmt.Println(common.StringToHash(a).String())
	//fmt.Println(common.StringToHash(a).Hex())
	//rawTx, err := hex.DecodeString("f8aa248504e3b2920083015f9094f5b3b365fa319342e89a3da71ba393e12d9f63c380b844a9059cbb0000000000000000000000008311804426a24495bd4306daf5f595a443a52e32000000000000000000000000000000000000000000000000000000174876e80025a094b92f2068f8e41b8c2987d8f184345a016505addbda340261feebaedab38dc4a06e8029ad29eeb0a1368b352de54dc452cef1bf1efe36b2327ccb06ae5c29b7ba")
	//fmt.Println(rawTx)
	//fmt.Println(err)
	//err = rlp.DecodeBytes(rawTx, ethTx)
	//if err != nil {
	//	fmt.Println(err)
	//} else {
	//	fmt.Println(ethTx)
	//	tx := &ethaccessor.Transaction{}
	//	tx.Hash = ethTx.Hash().Hex()
	//	tx.Input = hexutil.Encode(ethTx.Data())
	//	tx.Gas = *types.NewBigPtr(ethTx.Gas())
	//	tx.GasPrice = *types.NewBigPtr(ethTx.GasPrice())
	//	tx.Nonce = *types.NewBigWithInt(int(ethTx.Nonce()))
	//	tx.BlockNumber = *types.NewBigWithInt(0)
	//	tx.BlockHash = ""
	//
	//	fmt.Println(tx.Hash)
	//	fmt.Println(tx.Input)
	//	fmt.Println(tx.Gas)
	//	fmt.Println(tx.GasPrice)
	//	fmt.Println(tx.Nonce)
	//	fmt.Println(tx.BlockHash)
	//	fmt.Println(tx.BlockNumber)
	//
	//	//eventemitter.Emit(eventemitter.PendingTransaction, tx)
	//	fmt.Println(tx)
	//}
	//
	//fmt.Println(fmt.Sprintf("+%.2f%%", -2.3334))
	//fmt.Println(fmt.Sprintf("%.2f%%", -2.3334))
	//fmt.Println(fmt.Sprintf("+%.2f%%", 2.3334))

	//ab := AB{"tttt"}
	//abrq := ABReq1{A :"ttttttt", B: 10}
	//abrqJson, _ :=  json.Marshal(abrq)

	//handleWithT(&ab, abrq, "ABTest1", string(abrqJson[:]))
	//handleWithT(&ab, nil, "ABTest2", string(abrqJson[:]))

}
