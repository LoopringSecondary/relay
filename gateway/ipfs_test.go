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
	suffix        = "000000000000000000"
	account1      = "0x1b978a1d302335a6f2ebe4b8823b5e17c3c84135"
	account2      = "0xb1018949b241d76a1ab2094f473e9befeabb5ead"
	pwd1          = "201"
	pwd2          = "202"
	TokenAddressA = "0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"
	TokenAddressB = "0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f"
)

func TestSingleOrder(t *testing.T) {
	c := test.LoadConfig()

	// get keystore and unlock account
	acc1 := accounts.Account{Address: common.HexToAddress(account1)}
	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	ks.Unlock(acc1, pwd1)
	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)

	// set order and marshal to json
	impl, _ := c.Common.ProtocolImpls["v_0_1"]

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)

	order := test.CreateOrder(
		common.HexToAddress(TokenAddressA),
		common.HexToAddress(TokenAddressB),
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

func TestRing(t *testing.T) {
	c := test.LoadConfig()

	// get keystore and unlock account
	acc1 := accounts.Account{Address: common.HexToAddress(account1)}
	acc2 := accounts.Account{Address: common.HexToAddress(account2)}
	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	ks.Unlock(acc1, pwd1)
	ks.Unlock(acc2, pwd2)
	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)

	// set order and marshal to json
	impl, _ := c.Common.ProtocolImpls["v_0_1"]

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	order1 := test.CreateOrder(
		common.HexToAddress(TokenAddressA),
		common.HexToAddress(TokenAddressB),
		common.HexToAddress(impl.Address),
		acc1.Address,
		amountS1,
		amountB1,
	)
	bs1, _ := order1.MarshalJSON()

	amountS2, _ := new(big.Int).SetString("20"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("1"+suffix, 0)
	order2 := test.CreateOrder(
		common.HexToAddress(TokenAddressB),
		common.HexToAddress(TokenAddressA),
		common.HexToAddress(impl.Address),
		acc2.Address,
		amountS2,
		amountB2,
	)
	bs2, _ := order2.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewLocalShell()
	pubMessage(sh, string(bs1))
	pubMessage(sh, string(bs2))
}

func pubMessage(sh *shell.Shell, data string) {
	c := test.LoadConfig()
	topic := c.Ipfs.ListenTopics[0]
	err := sh.PubSubPublish(topic, data)
	if err != nil {
		panic(err.Error())
	}
}
