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

package test

import (
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/extractor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/Loopring/relay/usermanager"
	"github.com/ethereum/go-ethereum/common"
	"github.com/naoina/toml"
	"math/big"
	"os"
	"strings"
	"time"
)

type AccountEntity struct {
	Address    common.Address
	Passphrase string
}

type TestEntity struct {
	Tokens          []common.Address
	Accounts        []AccountEntity
	Creator         AccountEntity
	KeystoreDir     string
	AllowanceAmount int64
}

var (
	cfg *config.GlobalConfig
	rds dao.RdsService
)

func Initialize() {
	cfg = loadConfig()
	rds = GenerateDaoService()
	util.Initialize(rds, cfg)
}

func Rds() dao.RdsService       { return rds }
func Cfg() *config.GlobalConfig { return cfg }

func GenerateTomlEntity() *TestEntity {
	var (
		data   TestData
		entity TestEntity
	)

	file := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/" + "/test/testdata.toml"
	io, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer io.Close()

	if err := toml.NewDecoder(io).Decode(&data); err != nil {
		panic(err)
	}

	for _, v := range util.SupportTokens {
		entity.Tokens = append(entity.Tokens, v.Protocol)
	}

	for _, v := range data.Accounts {
		var acc AccountEntity
		acc.Address = common.HexToAddress(v.Address)
		acc.Passphrase = v.Passphrase
		entity.Accounts = append(entity.Accounts, acc)
	}
	entity.Creator = AccountEntity{Address: common.HexToAddress(data.Creator.Address), Passphrase: data.Creator.Passphrase}
	entity.KeystoreDir = cfg.Keystore.Keydir
	entity.AllowanceAmount = data.AllowanceAmount

	return &entity
}

func GenerateAccessor() (*ethaccessor.EthNodeAccessor, error) {
	accessor, err := ethaccessor.NewAccessor(cfg.Accessor, cfg.Common, util.WethTokenAddress())
	if nil != err {
		return nil, err
	}
	return accessor, nil
}

func GenerateExtractor() *extractor.ExtractorServiceImpl {
	accessor, err := GenerateAccessor()
	if err != nil {
		panic(err)
	}
	l := extractor.NewExtractorService(cfg.Accessor, cfg.Common, accessor, rds)
	return l
}

func GenerateOrderManager() *ordermanager.OrderManagerImpl {
	mc := GenerateMarketCap()
	um := usermanager.NewUserManager(rds)
	accessor, err := GenerateAccessor()
	if err != nil {
		panic(err)
	}
	ob := ordermanager.NewOrderManager(cfg.OrderManager, &cfg.Common, rds, um, accessor, mc)
	return ob
}

func GenerateDaoService() *dao.RdsServiceImpl {
	return dao.NewRdsService(cfg.Mysql)
}

func GenerateMarketCap() *marketcap.MarketCapProvider {
	return marketcap.NewMarketCapProvider(cfg.Miner)
}

func LoadConfig() *config.GlobalConfig {
	return loadConfig()
}

func CreateOrder(tokenS, tokenB, protocol, owner common.Address, amountS, amountB *big.Int) *types.Order {
	order := &types.Order{}
	order.Protocol = protocol
	order.TokenS = tokenS
	order.TokenB = tokenB
	order.AmountS = amountS
	order.AmountB = amountB
	order.Timestamp = big.NewInt(time.Now().Unix())
	order.Ttl = big.NewInt(8640000)
	order.Salt = big.NewInt(1000)
	order.LrcFee = big.NewInt(120701538919177881)
	order.BuyNoMoreThanAmountB = false
	order.MarginSplitPercentage = 0
	order.Owner = owner
	order.Hash = order.GenerateHash()
	if err := order.GenerateAndSetSignature(owner); nil != err {
		panic(err.Error())
	}
	return order
}

func loadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/debug.toml"
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}
