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
	txtyp "github.com/Loopring/relay/txmanager/types"
	"github.com/Loopring/relay/types"
	"testing"
)

func TestTransactionViewImpl_GetPendingTransactions(t *testing.T) {
	txmanager.NewTxView(test.Rds())

	owner := "0x43e85E2c882bbcE41C69740Eed4BfFFb45E3f9dd"
	list, err := txmanager.GetPendingTransactions(owner)
	if err != nil {
		t.Fatalf(err.Error())
	}

	for _, v := range list {
		t.Logf("tx:%s, from:%s, to:%s, type:%s, status:%s", v.TxHash.Hex(), v.From.Hex(), v.To.Hex(), v.Type, v.Status)
	}
}

func TestTransactionViewImpl_GetAllTransactionCount(t *testing.T) {
	txmanager.NewTxView(test.Rds())

	owner := "0xb1018949b241D76A1AB2094f473E9bEfeAbB5Ead"
	symbol := "eth"
	status := "pending"
	typ := "all"
	if number, err := txmanager.GetAllTransactionCount(owner, symbol, status, typ); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Logf("owner:%s have %d transactions in %s", owner, number, symbol)
	}
}

func TestTransactionViewImpl_GetAllTransactions(t *testing.T) {
	txmanager.NewTxView(test.Rds())

	owner := "0xb1018949b241D76A1AB2094f473E9bEfeAbB5Ead"
	symbol := "weth"
	status := "all"
	typ := "all"

	txs, err := txmanager.GetAllTransactions(owner, symbol, status, typ, 20, 0)
	if err != nil {
		t.Fatalf(err.Error())
	}
	for k, v := range txs {
		t.Logf("%d >>>>>> txhash:%s, symbol:%s, protocol:%s, from:%s, to:%s, type:%s, status:%s", k, v.TxHash.Hex(), v.Symbol, v.Protocol.Hex(), v.From.Hex(), v.To.Hex(), v.Type, v.Status)
	}
}

func TestTransactionViewerImpl_FillView(t *testing.T) {
	txhash := "0x23e58f736fd83345353a594efc1a4d137f7780f2f9c2a41556ee935af4eeb3a6"
	fillindex := int64(0)

	rds := test.Rds()
	model, err := rds.FindFillEvent(txhash, fillindex)
	if err != nil {
		t.Fatalf(err.Error())
	}
	var event types.OrderFilledEvent
	model.ConvertUp(&event)

	list := txtyp.OrderFilledView(&event)
	t.Log(len(list))
}
