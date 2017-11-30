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
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestRdsServiceImpl_NewOrder(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()

	ord := &dao.Order{}

	suffix := "100002000030000418"
	amountB, _ := new(big.Int).SetString("20000000"+suffix, 0)
	amountS, _ := new(big.Int).SetString("1"+suffix, 0)
	fee, _ := new(big.Int).SetString("466778", 0)
	price, _ := new(big.Rat).SetFrac(amountB, amountS).Float64()

	ord.Protocol = common.HexToAddress("0xdff9092fc8b0ea74509b9ef5d0b74f7c80876218").Hex()
	ord.OrderHash = common.HexToHash("0x4753513505617586b115b82a0131f5a5da4325063e3f912a49b1aed7ceb80f26").Hex()
	ord.Owner = common.HexToAddress("0xdff9092fc8b0ea74509b9ef5d0b74f7c80876219").Hex()
	ord.TokenB = common.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e").Hex()
	ord.TokenS = common.HexToAddress("0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f").Hex()
	ord.AmountB, _ = amountB.MarshalText()
	ord.AmountS, _ = amountS.MarshalText()
	ord.LrcFee, _ = fee.MarshalText()
	ord.Price = real(complex(price, float64(0.01)))
	ord.MarginSplitPercentage = 32
	ord.BuyNoMoreThanAmountB = false
	ord.Ttl = 10000000
	ord.Salt = 800
	ord.V = 127
	ord.S = "11"
	ord.R = "22"

	if err := s.Add(ord); err != nil {
		t.Fatal(err)
	}
}

func TestRdsServiceImpl_GetOrderByHash(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()
	order, err := s.GetOrderByHash(common.HexToHash("0x7ee8521eabd792fb539975718c0e2433dba0fc3683d8c7d22a7ab1784e1ad383"))

	if err != nil {
		t.Fatal(err)
	}

	t.Log(order.TokenS)
}

func TestRdsServiceImpl_GetOrdersForMiner(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()

	filters := []types.OrderStatus{types.ORDER_CUTOFF, types.ORDER_FINISHED}
	list, err := s.GetOrdersForMiner(test.TokenAddressA, test.TokenAddressB, 10, filters)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("length of list", len(list))

	for _, v := range list {
		t.Log(v.OrderHash)
	}
}

func TestRdsServiceImpl_GetOrdersByHash(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()
	hashList := []string{
		"0xb617960a443c1f351e2e7a90b5005953744c411ee99fae34f3c2f509f8c1a1f5",
		"0xdb14bf9c71b6b026127f72b1efdd270c8940a6300fc6a2ce411eda492ded1c7c",
		"0xce99ebf9f517ba2a6e87925f578994d762b8d581c02492beac50c92a91854968",
	}
	ordMap, err := s.GetOrdersByHash(hashList)
	if err != nil {
		t.Fatalf(err.Error())
	}

	for _, v := range ordMap {
		t.Logf("order owner:%s", v.Owner)
	}
}
