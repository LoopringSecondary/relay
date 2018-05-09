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

package ordermanager_test

import (
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestOrderManagerImpl_MinerOrders(t *testing.T) {
	//entity := test.Entity()

	om := test.GenerateOrderManager()
	//protocol := test.Protocol()
	tokenS := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	tokenB := common.HexToAddress("0xEF68e7C694F40c8202821eDF525dE3782458639f")

	states := om.MinerOrders(common.HexToAddress("0x7b126ab811f278f288bf1d62d47334351dA20d1d"), tokenS, tokenB, 10, 0, 200000000, &types.OrderDelayList{})
	for _, v := range states {
		t.Logf("owner:%s, hash:%s", v.RawOrder.Owner.Hex(), v.RawOrder.Hash.Hex())
		//t.Logf("list number %d, order.hash %s", k, v.RawOrder.Hash.Hex())
		//t.Logf("list number %d, order.tokenS %s", k, v.RawOrder.TokenS.Hex())
		//t.Logf("list number %d, order.price %s", k, v.RawOrder.Price.String())
	}
}

func TestOrderManagerImpl_GetOrderByHash(t *testing.T) {
	om := test.GenerateOrderManager()
	states, _ := om.GetOrderByHash(common.HexToHash("0xaaa99b5c64fe1f6ae594994d1f6c252dc49c2d0db6bb185df99f5ffa8de64fdb"))

	t.Logf("order.hash %s", states.RawOrder.Hash.Hex())
	t.Logf("order.tokenS %s", states.RawOrder.TokenS.Hex())
}

func TestOrderManagerImpl_GetOrderBook(t *testing.T) {
	om := test.GenerateOrderManager()
	protocol := common.HexToAddress("0x03E0F73A93993E5101362656Af1162eD80FB54F2")
	tokenS := common.HexToAddress("0x2956356cD2a2bf3202F771F50D3D14A367b48070")
	tokenB := common.HexToAddress("0x86Fa049857E0209aa7D9e616F7eb3b3B78ECfdb0")
	list, err := om.GetOrderBook(protocol, tokenS, tokenB, 100)
	if err != nil {
		panic(err)
	}

	for _, v := range list {
		t.Logf("orderhash", v.RawOrder.Hash.Hex())
	}
}

func TestOrderManagerImpl_GetOrders(t *testing.T) {
	om := test.GenerateOrderManager()

	query := map[string]interface{}{"order_hash": "0xf5b657335c4044e11170be3b35cda21b0819e396da0b7d258422f7203887aaf3"}
	status := []types.OrderStatus{}
	pageRes, err := om.GetOrders(query, status, 0, 20)
	if err != nil {
		t.Fatalf(err.Error())
	}
	for _, v := range pageRes.Data {
		state := v.(types.OrderState)
		t.Logf("dealtAmounts:%s, dealtAmountB:%s, cancelAmountS:%s, cancelAmountB:%s, status:%d",
			state.DealtAmountS.String(),
			state.DealtAmountB.String(),
			state.CancelledAmountS.String(),
			state.CancelledAmountB.String(),
			state.Status)
	}
}
