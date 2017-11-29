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

package ordermanager_test

import (
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/test"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestOrderManagerImpl_MinerOrders(t *testing.T) {
	c := test.LoadConfig()
	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)

	om := test.LoadConfigAndGenerateOrderManager()
	tokenS := common.HexToAddress(test.TokenAddressA)
	tokenB := common.HexToAddress(test.TokenAddressB)

	states := om.MinerOrders(tokenS, tokenB, []common.Hash{})
	for k, v := range states {
		t.Logf("list number %d, order.hash %s", k, v.RawOrder.Hash.Hex())
		t.Logf("list number %d, order.tokenS %s", k, v.RawOrder.TokenS.Hex())
	}
}

func TestOrderManagerImpl_GetOrderByHash(t *testing.T) {
	c := test.LoadConfig()
	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)

	om := test.LoadConfigAndGenerateOrderManager()
	states, _ := om.GetOrderByHash(common.HexToHash("0xaaa99b5c64fe1f6ae594994d1f6c252dc49c2d0db6bb185df99f5ffa8de64fdb"))

	t.Logf("order.hash %s", states.RawOrder.Hash.Hex())
	t.Logf("order.tokenS %s", states.RawOrder.TokenS.Hex())
}