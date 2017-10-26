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

package orderbook_test

import (
	"github.com/Loopring/ringminer/test"
	"github.com/Loopring/ringminer/types"
	"testing"
)

func TestOrderBook_GetOrder(t *testing.T) {
	ob := test.LoadConfigAndGenerateOrderBook()
	orders := []string{
		"0x26de2b6fffebcb5e88e930cc8e03d7f991add424c02c9678ac019c310be0600b",
		"0x5ace4d91fded46d318b3741689ba73282626bc87fcf8be75d770bb729af74710",
	}

	for _, orderhash := range orders {
		st, err := ob.GetOrder(types.HexToHash(orderhash))
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("protocol:%s", st.RawOrder.Protocol.Hex())
		t.Logf("owner:%s", st.RawOrder.Owner.Hex())
		t.Logf("orderhash:%s", st.RawOrder.Hash.Hex())
		t.Logf("buyNoMoreThanB:%d", st.RawOrder.BuyNoMoreThanAmountB)
		t.Logf("tokenS:%s", st.RawOrder.TokenS.Hex())
		t.Logf("tokenB:%s", st.RawOrder.TokenB.Hex())
		t.Logf("amountS:%s", st.RawOrder.AmountS.String())
		t.Logf("amountB:%s", st.RawOrder.AmountB.String())

		if _, err := st.LatestVersion(); err != nil {
			t.Fatal(err)
		}

		for k, v := range st.States {
			t.Logf("version %d status %d", k, v.Status)
			t.Logf("version %d block %s", k, v.Block.String())
			t.Logf("version %d remainS %s", k, v.RemainedAmountS.String())
			t.Logf("version %d remainB %s", k, v.RemainedAmountB.String())
		}
		t.Log("++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++")
	}
}
