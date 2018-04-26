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
	"github.com/Loopring/relay/txmanager"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestExtractorServiceImpl_UnlockWallet(t *testing.T) {
	account := common.HexToAddress("0x71C079107B5af8619D54537A93dbF16e5aab4900") //test.Entity().Accounts[1].Address
	manager := test.GenerateAccountManager()
	manager.UnlockedWallet(account.Hex())
}

// test save transaction
func TestExtractorServiceImpl_ProcessPendingTransaction(t *testing.T) {

	var tx ethaccessor.Transaction
	if err := ethaccessor.GetTransactionByHash(&tx, "0xf5d32f54d61e00dbece2a0ae851b356a76bf2d0ae178f8ffe13bc2bf4cc84b6d", "latest"); err != nil {
		t.Fatalf(err.Error())
	} else {
		eventemitter.Emit(eventemitter.PendingTransaction, &tx)
	}

	accmanager := test.GenerateAccountManager()
	tm := txmanager.NewTxManager(test.Rds(), &accmanager)
	tm.Start()
	processor := extractor.NewExtractorService(test.Cfg().Extractor, test.Rds(), &accmanager)
	processor.ProcessPendingTransaction(&tx)
}

// test save transaction
func TestExtractorServiceImpl_ProcessMinedTransaction(t *testing.T) {
	txhash := "0x26383249d29e13c4c5f73505775813829875d0b0bf496f2af2867548e2bf8108"

	tx := &ethaccessor.Transaction{}
	receipt := &ethaccessor.TransactionReceipt{}
	if err := ethaccessor.GetTransactionByHash(tx, txhash, "latest"); err != nil {
		t.Fatalf(err.Error())
	}
	if err := ethaccessor.GetTransactionReceipt(receipt, txhash, "latest"); err != nil {
		t.Fatalf(err.Error())
	}

	accmanager := test.GenerateAccountManager()
	tm := txmanager.NewTxManager(test.Rds(), &accmanager)
	tm.Start()
	processor := extractor.NewExtractorService(test.Cfg().Extractor, test.Rds(), &accmanager)
	processor.ProcessMinedTransaction(tx, receipt, big.NewInt(100))
}
