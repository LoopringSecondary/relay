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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"os"
	"strings"
	"testing"
)

func loadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/relay.toml"
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}

func TestNewRing(t *testing.T) {

	cfg := loadConfig()

	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)

	acc1 := accounts.Account{Address: common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")}
	acc2 := accounts.Account{Address: common.HexToAddress("0x48ff2269e58a373120ffdbbdee3fbcea854ac30a")}
	ks.Unlock(acc1, "1")
	ks.Unlock(acc2, "1")
	c := crypto.NewKSCrypto(false, ks)
	crypto.Initialize(c)
	s := dao.NewRdsService(cfg.Mysql)
	s.Prepare()

	info := &dao.RingSubmitInfo{}

	info.RingHash = common.HexToHash("0x2c88ebf05254fb82e7ecd10c237036eb4cd0846e1ad8059ca72af40344a9d7d2").Hex()
	info.ProtocolAddress = common.HexToAddress("0xB5FAB0B11776AAD5cE60588C16bd59DCfd61a1c2").Hex()
	info.ProtocolData = "0x9812ad890"

	if err := s.UpdateRingSubmitInfoRegistryTxHash([]common.Hash{common.HexToHash(info.RingHash)}, "0x3c88ebf05254fb82e7ecd10c237036eb4cd0846e1ad8059ca72af40344a9d7d2"); nil != err {
		t.Error(err)
	}
}

func TestGetRing(t *testing.T) {
	cfg := loadConfig()

	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)

	acc1 := accounts.Account{Address: common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")}
	acc2 := accounts.Account{Address: common.HexToAddress("0x48ff2269e58a373120ffdbbdee3fbcea854ac30a")}
	ks.Unlock(acc1, "1")
	ks.Unlock(acc2, "1")
	c := crypto.NewKSCrypto(false, ks)
	crypto.Initialize(c)
	s := dao.NewRdsService(cfg.Mysql)
	s.Prepare()
	ringSubmitInfo, err := s.GetRingForSubmitByHash(common.HexToHash("0x9e75a4fea488f4b765640d1a466ded990477def59f8846e2d7ba070158c7e41b"))
	if nil != err {
		t.Error(err.Error())
	}
	println(ringSubmitInfo.ID)
}
