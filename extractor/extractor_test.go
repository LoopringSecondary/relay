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

package extractor_test

import (
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/extractor"
	"github.com/Loopring/relay/test"
	"testing"
)

func TestExtractorServiceImpl_UnlockWallet(t *testing.T) {
	account := test.Entity().Accounts[1].Address.Hex()
	manager := test.GenerateAccountManager()
	manager.UnlockedWallet(account)
}

func TestExtractorServiceImpl_PendingTransaction(t *testing.T) {

	var tx ethaccessor.Transaction
	if err := ethaccessor.GetTransactionByHash(&tx, "0x4c56e92c88be781fd013ff666684e13690543492f920b16b0ea10e4892d3186f", "latest"); err != nil {
		t.Fatalf(err.Error())
	} else {
		eventemitter.Emit(eventemitter.PendingTransaction, &tx)
	}

	accmanager := test.GenerateAccountManager()
	processor := extractor.NewExtractorService(test.Cfg().Extractor, test.Rds(), &accmanager)
	processor.ProcessPendingTransaction(&tx)
}
