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
	"encoding/json"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/test"
	"github.com/Loopring/ringminer/types"
	"github.com/ipfs/go-ipfs-api"
	"math/big"
	"testing"
)

var testParams *test.TestParams

const suffix = "0"

func init() {
	testParams = test.LoadConfigAndGenerateTestParams()
	log.Infof("contract address:%s", testParams.ImplAddress.Hex())
	log.Infof("delegate address:%s", testParams.DelegateAddress.Hex())
	log.Infof("register address:%s", testParams.TokenRegistryAddress.Hex())
}

func TestPrepareAccount(t *testing.T) {
	testParams.PrepareTestData()
}

func TestCheckAllowance(t *testing.T) {
	testParams.IsTestDataReady()
}

func TestOrdersOfRing(t *testing.T) {
	sh := shell.NewLocalShell()

	//scheme 1:MarginSplitPercentage = 0

	order1 := setOrder1()
	data1, _ := json.Marshal(order1)
	pubMessage(sh, string(data1))

	order2 := setOrder2()
	data2, _ := json.Marshal(order2)
	pubMessage(sh, string(data2))
}

func setOrder1() *types.Order {
	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	return test.CreateOrder(
		types.HexToAddress(test.TokenAddressA),
		types.HexToAddress(test.TokenAddressB),
		testParams.ImplAddress,
		amountS1,
		amountB1,
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)
}

func setOrder2() *types.Order {
	amountS2, _ := new(big.Int).SetString("20"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("1"+suffix, 0)
	return test.CreateOrder(
		types.HexToAddress(test.TokenAddressB),
		types.HexToAddress(test.TokenAddressA),
		testParams.ImplAddress,
		amountS2,
		amountB2,
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
		types.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A"),
	)
}

func pubMessage(sh *shell.Shell, data string) {
	topic := testParams.Config.Ipfs.ListenTopics[0]
	err := sh.PubSubPublish(topic, data)
	if err != nil {
		panic(err.Error())
	}
}

func TestIsTestDataReady(t *testing.T) {
	testParams.IsTestDataReady()
}

func TestPrepareTestData(t *testing.T) {
	sh := shell.NewLocalShell()

	order1 := `{"protocol":"0x29d4178372d890e3127d35c3f49ee5ee215d6fe8","tokenS":"0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f","tokenB":"0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e","amountS":"0xc8","amountB":"0xa","timestamp":"0x59ef0cc8","ttl":"0x2710","salt":"0x3e8","lrcFee":"0x64","buyNoMoreThanAmountB":false,"marginSplitPercentage":0,"v":27,"r":"0xecdfe5d96346e1a4fffce7a63fe0c8ff6111b13c3c387a296cdc6d9a10599fb0","s":"0x18640bbb9ccc6b667a05abcd349531b58211084b33fbb73270f1eb1861d6559a","owner":"0x48ff2269e58a373120ffdbbdee3fbcea854ac30a","hash":"0x9b7857b006236a148e70e8b07adf6347610a7d1beb88328810528d98f20496e8"}`
	order2 := `{"protocol":"0x29d4178372d890e3127d35c3f49ee5ee215d6fe8","tokenS":"0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e","tokenB":"0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f","amountS":"0xa","amountB":"0x64","timestamp":"0x59ef0cc8","ttl":"0x2710","salt":"0x3e8","lrcFee":"0x64","buyNoMoreThanAmountB":false,"marginSplitPercentage":0,"v":27,"r":"0xe4c79971b1949223b185101e3bd890f5b5d236d0f6c067bb1e3f36fa3784e79c","s":"0x0512ac6bd868bb92a5e4270ba3cfd3856d21ed5092d4b23b54f5b854fb0777df","owner":"0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2","hash":"0xdf2d02d7533d4c2faea71588df3e41f8199853cd97ea5eeb29ee0a5c643dd953"}`

	pubMessage(sh, order1)
	pubMessage(sh, order2)

}

func TestCancelOrder(t *testing.T) {
	ord := setOrder2()
	addressList := []types.Address{ord.Owner, ord.TokenS, ord.TokenB}

	cancelAmountS := big.NewInt(2)
	valueList := []*big.Int{ord.AmountS, ord.AmountB, ord.Timestamp, ord.Ttl, ord.Salt, ord.LrcFee, cancelAmountS}

	ret, err := testParams.Imp.CancelOrder.SendTransaction(ord.Owner,
		addressList,
		valueList,
		ord.BuyNoMoreThanAmountB,
		ord.MarginSplitPercentage,
		ord.V,
		ord.R,
		ord.S)
	if err != nil {
		t.Errorf(err.Error())
	} else {
		t.Log(ret)
	}
}
