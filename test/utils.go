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

func GenerateTomlEntity(cfg *config.GlobalConfig) *TestEntity {
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

	rds := GenerateDaoService(cfg)
	util.Initialize(rds)
	for _, v := range util.SupportTokens {
		entity.Tokens = append(entity.Tokens, common.HexToAddress(v))
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

func GenerateAccessor(c *config.GlobalConfig) (*ethaccessor.EthNodeAccessor, error) {
	accessor, err := ethaccessor.NewAccessor(c.Accessor, c.Common)
	if nil != err {
		return nil, err
	}
	return accessor, nil
}

func GenerateExtractor(c *config.GlobalConfig) *extractor.ExtractorServiceImpl {
	rds := GenerateDaoService(c)
	accessor, err := GenerateAccessor(c)
	if err != nil {
		panic(err)
	}
	l := extractor.NewExtractorService(c.Accessor, c.Common, accessor, rds)
	return l
}

func GenerateOrderManager(c *config.GlobalConfig) *ordermanager.OrderManagerImpl {
	rds := GenerateDaoService(c)
	um := usermanager.NewUserManager(rds)
	accessor, err := GenerateAccessor(c)
	if err != nil {
		panic(err)
	}
	ob := ordermanager.NewOrderManager(c.OrderManager, &c.Common, rds, um, accessor)
	return ob
}

func GenerateDaoService(c *config.GlobalConfig) *dao.RdsServiceImpl {
	return dao.NewRdsService(c.Mysql)
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
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/relay.toml"
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}
