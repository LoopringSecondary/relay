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

package eth_test

import (
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"strings"
	"testing"
)

// 字符串转地址必须使用hexToAddress
func TestAddress(t *testing.T) {
	account := "0x56d9620237fff8a6c0f98ec6829c137477887ec4"
	t.Log(account)
	t.Log(common.HexToAddress(account).String())
}

func TestUnpack(t *testing.T) {
	const definition = `[
	{"constant":false,"inputs":[
		{"name":"hash","type":"bytes32"},
		{"name":"accountS","type":"address"},
		{"name":"accountB","type":"address"},
		{"name":"amountS","type":"uint256"},
		{"name":"amountB","type":"uint256"}],"name":"submitTransfer","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[
		{"name":"condition","type":"bool"},
		{"name":"message","type":"string"}],"name":"check","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":true,"inputs":[
		{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":false,"inputs":[
		{"name":"_id","type":"bytes32"},
		{"name":"_owner","type":"address"},
		{"name":"_amount","type":"uint256"}],
		"name":"submitDeposit","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"anonymous":false,"inputs":[
		{"indexed":false,"name":"hash","type":"bytes32"},
		{"indexed":false,"name":"account","type":"address"},
		{"indexed":false,"name":"amount","type":"uint256"},
		{"indexed":false,"name":"ok","type":"bool"}],"name":"DepositFilled","type":"event"},
	{"anonymous":false,"inputs":[
		{"indexed":false,"name":"hash","type":"bytes32"},
		{"indexed":false,"name":"accountS","type":"address"},
		{"indexed":false,"name":"accountB","type":"address"},
		{"indexed":false,"name":"amountS","type":"uint256"},
		{"indexed":false,"name":"amountB","type":"uint256"},
		{"indexed":false,"name":"ok","type":"bool"}],"name":"OrderFilled","type":"event"},
	{"anonymous":false,"inputs":[
		{"indexed":false,"name":"message","type":"string"}],"name":"Exception","type":"event"}]
`

	tabi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		t.Fatal(err)
	}

	type DepositEvent struct {
		Hash    []byte
		Account common.Address
		Amount  *big.Int
		Ok      bool
	}

	event := DepositEvent{}
	name := "DepositFilled"
	str := "0x5ad6fe3e08ffa01bb1db674ac8e66c47511e364a4500115dd2feb33dad972d7e00000000000000000000000056d9620237fff8a6c0f98ec6829c137477887ec4000000000000000000000000000000000000000000000000000000000bebc2010000000000000000000000000000000000000000000000000000000000000001"

	data := hexutil.MustDecode(str)
	abievent, ok := tabi.Events[name]
	if !ok {
		t.Error("event do not exist")
	}

	if err := eth.Unpack(abievent, &event, data, []string{}); err != nil {
		panic(err)
	}

	t.Log(common.BytesToHash(event.Hash).Hex())
	t.Log(event.Account.Hex())
	t.Log(event.Amount)
	t.Log(event.Ok)
}
