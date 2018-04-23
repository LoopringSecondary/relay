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

package txmanager_test

import (
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/txmanager"
	"testing"
)

func newTxView() txmanager.TransactionView {
	return txmanager.NewTxView(test.Rds())
}

func TestTransactionViewImpl_GetPendingTransactions(t *testing.T) {
	view := newTxView()
	owner := "0xb1018949b241D76A1AB2094f473E9bEfeAbB5Ead"
	list, err := view.GetPendingTransactions(owner)
	if err != nil {
		t.Fatalf(err.Error())
	}

	for _, v := range list {
		t.Logf("tx:%s, from:%s, to:%s, type:%s, status:%s", v.TxHash.Hex(), v.From.Hex(), v.To.Hex(), v.Type, v.Status)
	}
}

func TestTransactionViewImpl_GetMinedTransactionCount(t *testing.T) {
	view := newTxView()
	owner := "0xb1018949b241D76A1AB2094f473E9bEfeAbB5Ead"
	symbol := "ETH"
	if number, err := view.GetMinedTransactionCount(owner, symbol); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Logf("owner:%s have %d transactions in %s", owner, number, symbol)
	}
}

func TestTransactionViewImpl_GetMinedTransactions(t *testing.T) {
	view := newTxView()
	owner := "0xb1018949b241D76A1AB2094f473E9bEfeAbB5Ead"
	symbol := "eth"

	txs, err := view.GetMinedTransactions(owner, symbol, 2, 6)
	if err != nil {
		t.Fatalf(err.Error())
	}
	for k, v := range txs {
		t.Logf("%d >>>>>> txhash:%s, symbol:%s, from:%s, to:%s, type:%s", k, v.TxHash.Hex(), v.Symbol, v.From.Hex(), v.To.Hex(), v.Type)
	}
}
