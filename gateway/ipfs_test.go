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

package gateway_test

import (
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/test"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-ipfs-api"
	"math/big"
	"testing"
)

const (
	suffix   = "0"
	account1 = "0xf6c399d9b5bba8f91d000107a21d05913bf7e47f" // pwd 101
	pwd1     = "101"
)

func TestSingleOrder(t *testing.T) {
	c := test.LoadConfig()

	// get keystore and unlock account
	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)
	acc1 := accounts.Account{Address: common.HexToAddress(account1)}
	if err := ks.Unlock(acc1, pwd1); err != nil {
		panic(err.Error())
	}

	// set order and marshal to json
	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	impl, ok := c.Common.ProtocolImpls["v_0_1"]
	if !ok {
		panic("protocol version not exists")
	}
	order := test.CreateOrder(
		common.HexToAddress(test.TokenAddressA),
		common.HexToAddress(test.TokenAddressB),
		common.HexToAddress(impl.Address),
		acc1.Address,
		amountS1,
		amountB1,
	)
	bs, _ := order.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewLocalShell()
	pubMessage(sh, string(bs))
}

func TestMinerOrders(t *testing.T) {

}

func pubMessage(sh *shell.Shell, data string) {
	c := test.LoadConfig()
	topic := c.Ipfs.ListenTopics[0]
	err := sh.PubSubPublish(topic, data)
	if err != nil {
		panic(err.Error())
	}
}
