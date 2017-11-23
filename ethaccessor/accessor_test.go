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

package ethaccessor_test

import (
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestNewAccessor(t *testing.T) {
	options := config.ChainClientOptions{}
	options.RawUrl = "http://127.0.0.1:8545"
	accessor := ethaccessor.NewAccessor(options)
	var b types.Big
	if err := accessor.Call(&b, "eth_getBalance", common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259"), "pending"); nil != err {
		println(err.Error())
	}

	t.Log(b.BigInt().String())

	balance, _ := accessor.Erc20Balance(types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"), types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"), "pending")
	t.Log(balance.String())

	reqs := []*ethaccessor.BatchErc20BalanceAndAllowanceReq{&ethaccessor.BatchErc20BalanceAndAllowanceReq{
		Address:        types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
		Token:          types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"),
		BlockParameter: "pending",
	}}
	accessor.BatchErc20BalanceAndAllowance(reqs)

	t.Log("balance", reqs[0].Balance.BigInt().String())
	//
	//allowance := ethaccessor.Erc20Allowance(types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"),types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"), types.HexToAddress("0xd02d3e40cde61c267a3886f5828e03aa4914073d"), "pending")
	//println(allowance.String())
	//abiStr := `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"who","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	//
	//erc20 := ethaccessor.NewAbi(abiStr)
	//
	//balance := erc20.NewMethod("balanceOf", types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"))
	//
	//if err := balance.Call(&b, "pending", types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")); nil != err {
	//	println(err.Error())
	//}
	//println(b.BigInt().String())
}
