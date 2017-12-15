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
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/test"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-ipfs-api"
	"math/big"
	"testing"
)

const (
	suffix       = "00"
	TOKEN_SYMBOL = "LRC"
	WETH         = "WETH"
)

func TestSingleOrder(t *testing.T) {
	c := test.Cfg()
	entity := test.GenerateTomlEntity()

	// get keystore and unlock account
	tokenAddressA := util.AllTokens[TOKEN_SYMBOL].Protocol
	tokenAddressB := util.AllTokens[WETH].Protocol
	testAcc := entity.Accounts[0]

	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	account := accounts.Account{Address: testAcc.Address}
	ks.Unlock(account, testAcc.Passphrase)
	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)

	order := test.CreateOrder(
		tokenAddressA,
		tokenAddressB,
		protocol,
		account.Address,
		amountS1,
		amountB1,
	)
	bs, _ := order.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewLocalShell()
	pubMessage(sh, string(bs))
}

func TestRing(t *testing.T) {
	c := test.Cfg()
	entity := test.GenerateTomlEntity()

	tokenAddressA := util.SupportTokens[TOKEN_SYMBOL].Protocol
	tokenAddressB := util.SupportMarkets[WETH].Protocol

	testAcc1 := entity.Accounts[0]
	testAcc2 := entity.Accounts[1]
	password1 := entity.Accounts[0].Passphrase
	password2 := entity.Accounts[1].Passphrase

	// get keystore and unlock account
	ks := keystore.NewKeyStore(entity.KeystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)

	acc1 := accounts.Account{Address: testAcc1.Address}
	acc2 := accounts.Account{Address: testAcc2.Address}

	ks.Unlock(acc1, password1)
	ks.Unlock(acc2, password2)

	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	order1 := test.CreateOrder(
		tokenAddressA,
		tokenAddressB,
		protocol,
		acc1.Address,
		amountS1,
		amountB1,
	)
	bs1, _ := order1.MarshalJSON()

	amountS2, _ := new(big.Int).SetString("20"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("1"+suffix, 0)
	order2 := test.CreateOrder(
		tokenAddressB,
		tokenAddressA,
		protocol,
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

func TestPrepareProtocol(t *testing.T) {
	test.PrepareTestData()
}

func TestPrepareAccount(t *testing.T) {
	test.SetTokenBalances()
}

func TestAllowance(t *testing.T) {
	test.AllowanceToLoopring(nil, nil)
}

func pubMessage(sh *shell.Shell, data string) {
	c := test.LoadConfig()
	topic := c.Ipfs.BroadcastTopics[0]
	err := sh.PubSubPublish(topic, data)
	if err != nil {
		panic(err.Error())
	}
}
