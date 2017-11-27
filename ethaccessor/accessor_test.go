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
	"testing"
)

func TestNewAccessor(t *testing.T) {
	cfg := config.LoadConfig("/Users/yuhongyu/Desktop/service/go/src/github.com/Loopring/relay/config/relay.toml")
	accessor, err := ethaccessor.NewAccessor(cfg.Accessor, cfg.Common, nil)
	if nil != err {
		println(err.Error())
	}
	var b types.Big
	if err := accessor.Call(&b, "eth_getBalance", types.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259"), "pending"); nil != err {
		t.Error(err.Error())
	}

	t.Log(b.BigInt().String())

	balance, _ := accessor.Erc20Balance(types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"), types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"), "pending")
	t.Log(balance.String())

	reqs := []*ethaccessor.BatchErc20Req{&ethaccessor.BatchErc20Req{
		Address:        types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
		Token:          types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"),
		BlockParameter: "pending",
	},
		&ethaccessor.BatchErc20Req{
			Address:        types.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A"),
			Token:          types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"),
			BlockParameter: "pending",
		}}
	accessor.BatchErc20BalanceAndAllowance(reqs)

	t.Log("balance", reqs[0].Balance.BigInt().String())
	t.Log("balance", reqs[1].Balance.BigInt().String())
}
