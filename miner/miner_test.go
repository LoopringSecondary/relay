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

package miner_test

import (
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/miner/timing_matcher"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/Loopring/relay/usermanager"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

func loadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/relay.toml"
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}

func TestMatch(t *testing.T) {
	cfg := loadConfig()

	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)

	acc1 := accounts.Account{Address: common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")}
	acc2 := accounts.Account{Address: common.HexToAddress("0x48ff2269e58a373120ffdbbdee3fbcea854ac30a")}
	ks.Unlock(acc1, "1")
	ks.Unlock(acc2, "1")
	c := crypto.NewCrypto(false, ks)
	crypto.Initialize(c)
	rdsService := dao.NewRdsService(cfg.Mysql)
	userManager := usermanager.NewUserManager(rdsService)
	accessor, _ := ethaccessor.NewAccessor(cfg.Accessor, cfg.Common)
	om := ordermanager.NewOrderManager(cfg.OrderManager, &cfg.Common, rdsService, userManager, accessor, nil)

	marketCapProvider := marketcap.NewMarketCapProvider(cfg.Miner)
	submitter := miner.NewSubmitter(cfg.Miner, accessor, rdsService, marketCapProvider)
	evaluator := miner.NewEvaluator(marketCapProvider, int64(1000000000000000), accessor)
	matcher := timing_matcher.NewTimingMatcher(submitter, evaluator, om)

	m := miner.NewMiner(submitter, matcher, evaluator, accessor, marketCapProvider)
	m.Start()
	time.Sleep(1 * time.Minute)
}

func createOrder(tokenS, tokenB, protocol common.Address, amountS, amountB *big.Int, owner common.Address) *types.Order {
	order := &types.Order{}
	order.Protocol = protocol
	order.TokenS = tokenS
	order.TokenB = tokenB
	order.AmountS = amountS
	order.AmountB = amountB
	order.Timestamp = big.NewInt(time.Now().Unix())
	order.Ttl = big.NewInt(10000)
	order.Salt = big.NewInt(1000)
	order.LrcFee = big.NewInt(1000)
	order.BuyNoMoreThanAmountB = false
	order.MarginSplitPercentage = 0
	order.Owner = owner
	if err := order.GenerateAndSetSignature(owner); nil != err {
		println(err.Error())
	}
	return order
}

func TestPrepareTestData(t *testing.T) {
	test.PrepareTestData()
}

func TestAllowance(t *testing.T) {
	test.AllowanceToLoopring(nil, nil)
	//b := new(big.Int)
	//b.SetString("18428729675200069633", 0)
	//println(common.Bytes2Hex(b.Bytes()))

}

//18428729675200069633
//9223372036854775807
