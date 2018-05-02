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

package dao_test

import (
	"github.com/Loopring/relay/test"
	"testing"
)

func TestRdsServiceImpl_FindPendingTxEntity(t *testing.T) {
	db := test.GenerateDaoService()

	txhash := "0x8dd14dc43dbe47247042c57065a1f3a6de7b65a6247f52c28c4350a219f654c8"
	if entity, err := db.FindPendingTxEntity(txhash); err != nil {
		t.Fatalf("no record is error:%s", err.Error())
	} else {
		t.Logf("txhash:%s, from:%s, to:%s, nonce:%d", entity.TxHash, entity.From, entity.To, entity.Nonce)
	}

}
