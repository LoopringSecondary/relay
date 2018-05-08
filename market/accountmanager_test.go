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
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/test"
	"github.com/ethereum/go-ethereum/common"
	"testing"
	"time"
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
	//owner := common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259")
	//data := append(append(owner.Bytes(), owner.Bytes()...), owner.Bytes()...)
	//t.Log(common.BytesToAddress(data[0:20]).Hex(), common.BytesToAddress(data[20:40]).Hex(), common.BytesToAddress(data[40:]).Hex())
	accManager := test.GenerateAccountManager()

	//if err := accManager.UnlockedWallet(common.HexToAddress("0x8311804426A24495bD4306DAf5f595A443a52E32").Hex()); nil != err {
	//	t.Errorf("err:%s", err.Error())
	//} else {
	//	t.Log("##ooooo")
	//}
	//balance,allowance,err := accManager.GetBalanceAndAllowance(common.HexToAddress("0x80679A2c82aB82F1E73e14c4beC4ba1992F9F25A"), common.HexToAddress("0xb5f64747127be058Ee7239b363269FC8cF3F4A87"), common.HexToAddress("0x5567ee920f7E62274284985D793344351A00142B"))
	//if nil != err {
	//	t.Error(err.Error())
	//} else {
	//	t.Log(balance.String(), allowance.String())
	//}
	balances, err := accManager.GetBalanceWithSymbolResult(common.HexToAddress("0x23bD9CAfe75610C3185b85BC59f760f400bd89b5"))
	if nil != err {
		t.Fatalf("err:%s", err.Error())
	}
	for k, v := range balances {
		t.Logf("token:%s, balance:%s", k, v.String())
	}
	accManager.GetAllowanceWithSymbolResult(common.HexToAddress("0x23bD9CAfe75610C3185b85BC59f760f400bd89b5"), common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"))
	time.Sleep(1 * time.Second)
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

func TestChangedOfBlock_TestAllowance(t *testing.T) {
	test.LoadConfig()
	b := &market.ChangedOfBlock{}
	b.TestAllowance()
	time.Sleep(20 * time.Second)
}
