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

package market_test

import (
	"github.com/Loopring/relay/test"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestAccountManager_GetBalance(t *testing.T) {
	accManager := test.GenerateAccountManager()
	balances, err := accManager.GetBalanceWithSymbolResult(common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259"))
	if nil != err {
		t.Fatalf("err:%s", err.Error())
	}
	for k, v := range balances {
		t.Logf("token:%s, balance:%s", k, v.String())
	}
}
func TestAccountManager_UnlockedWallet(t *testing.T) {
	owner := common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259")
	data := append(append(owner.Bytes(), owner.Bytes()...), owner.Bytes()...)
	t.Log(common.BytesToAddress(data[0:20]).Hex(), common.BytesToAddress(data[20:40]).Hex(), common.BytesToAddress(data[40:]).Hex())
	//accManager := test.GenerateAccountManager()
	//
	//if err := accManager.UnlockedWallet("0x750ad4351bb728cec7d639a9511f9d6488f1e259"); nil != err {
	//	t.Errorf("err:%s", err.Error())
	//} else {
	//	t.Log("##ooooo")
	//}
	//balances,err := accManager.GetBalanceWithSymbolResult(common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259"))
	//if nil != err {
	//	t.Fatalf("err:%s", err.Error())
	//}
	//for k,v := range balances {
	//	t.Logf("token:%s, balance:%s", k, v.Balance.String())
	//}
}
func TestAccountManager_GetBAndAllowance(t *testing.T) {
	accManager := test.GenerateAccountManager()
	balance, allowance, err := accManager.GetBalanceAndAllowance(common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259"), common.HexToAddress("0x"), common.HexToAddress("0x"))
	if nil != err {
		t.Fatalf("err:%s", err.Error())
	}
	t.Logf("balance:%s, allowance:%s", balance.String(), allowance.String())
}

func TestAccountManager_GetAllowances(t *testing.T) {
	accManager := test.GenerateAccountManager()
	balances, err := accManager.GetAllowanceWithSymbolResult(common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259"), common.HexToAddress("0x0000000000000000000000004e9d4d3b7db4973c91b23a634eb9f675d0e19f79"))
	if nil != err {
		t.Fatalf("err:%s", err.Error())
	}
	for k, v := range balances {
		t.Logf("token:%s, balance:%s", k, v.String())
	}

}
