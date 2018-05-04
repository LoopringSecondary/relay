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

package txmanager

import (
	"github.com/Loopring/relay/cache"
	"github.com/ethereum/go-ethereum/common"
)

const (
	FillOwnerPrefix = "txmanager_submitring_"
	FillOwnerTtl    = 600 // todo 临时数据,只存储10分钟,系统性宕机后无法重启后丢失?
)

func SetFillOwner(txhash common.Hash, owner common.Address) error {
	key := generateFillOwnerKey(txhash)
	field := []byte(owner.Hex())
	return cache.SAdd(key, FillOwnerTtl, field)
}

func ExistFillOwner(txhash common.Hash, owner common.Address) (bool, error) {
	key := generateFillOwnerKey(txhash)
	field := []byte(owner.Hex())
	return cache.SIsMember(key, field)
}

func generateFillOwnerKey(txhash common.Hash) string {
	return FillOwnerPrefix + txhash.Hex()
}
