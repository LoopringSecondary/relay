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
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-ipfs-api"
	"math/big"
	"testing"
)

//var testParams *test.TestParams

const (
	suffix   = "0"
	account1 = "0xf6c399d9b5bba8f91d000107a21d05913bf7e47f" // pwd 101
	account2 = "0xc70f0ff6315d8b1ea39ec5294b9021e999cfe498" // pwd 202
	pwd1     = "101"
	pwd2     = "202"
)

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
	c := test.LoadConfig()

	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	impl, ok := c.Common.ProtocolImpls["v_0_1"]
	if !ok {
		panic("protocol version not exists")
	}

	acc1 := accounts.Account{Address: common.HexToAddress(account1)}
	if err := ks.Unlock(acc1, pwd1); err != nil {
		panic(err.Error())
	}

	return test.CreateOrder(
		common.HexToAddress(test.TokenAddressA),
		common.HexToAddress(test.TokenAddressB),
		common.HexToAddress(impl.Address),
		acc1.Address,
		amountS1,
		amountB1,
	)
}

func setOrder2() *types.Order {
	c := test.LoadConfig()
	protocol := c.Common.ProtocolImpls["v_0_1"].Address
	amountS2, _ := new(big.Int).SetString("20"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("1"+suffix, 0)
	return test.CreateOrder(
		common.HexToAddress(test.TokenAddressB),
		common.HexToAddress(test.TokenAddressA),
		common.HexToAddress(protocol),
		common.HexToAddress(account1),
		amountS2,
		amountB2,
	)
}

func pubMessage(sh *shell.Shell, data string) {
	c := test.LoadConfig()
	topic := c.Ipfs.ListenTopics[0]
	err := sh.PubSubPublish(topic, data)
	if err != nil {
		panic(err.Error())
	}
}

//func TestIsTestDataReady(t *testing.T) {
//	testParams.IsTestDataReady()
//}

func TestPrepareTestData(t *testing.T) {
	sh := shell.NewLocalShell()

	order1, _ := setOrder1().MarshalJSON()
	pubMessage(sh, string(order1))

	//order2, _ := setOrder2().MarshalJSON()
	//pubMessage(sh, string(order2))
}

//func TestCancelOrder(t *testing.T) {
//	ord := setOrder2()
//	addressList := []common.Address{ord.Owner, ord.TokenS, ord.TokenB}
//
//	cancelAmountS := big.NewInt(2)
//	valueList := []*big.Int{ord.AmountS, ord.AmountB, ord.Timestamp, ord.Ttl, ord.Salt, ord.LrcFee, cancelAmountS}
//
//	ret, err := testParams.Imp.CancelOrder.SendTransaction(ord.Owner,
//		addressList,
//		valueList,
//		ord.BuyNoMoreThanAmountB,
//		ord.MarginSplitPercentage,
//		ord.V,
//		ord.R,
//		ord.S)
//	if err != nil {
//		t.Errorf(err.Error())
//	} else {
//		t.Log(ret)
//	}
//}
