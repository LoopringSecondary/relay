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
	InitialMarketUtil(rds)
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
	mc := GenerateMarketCap(c)
	um := usermanager.NewUserManager(rds)
	accessor, err := GenerateAccessor(c)
	if err != nil {
		panic(err)
	}
	ob := ordermanager.NewOrderManager(c.OrderManager, &c.Common, rds, um, accessor, mc)
	return ob
}

func GenerateDaoService(c *config.GlobalConfig) *dao.RdsServiceImpl {
	return dao.NewRdsService(c.Mysql)
}

func GenerateMarketCap(c *config.GlobalConfig) *marketcap.MarketCapProvider {
	return marketcap.NewMarketCapProvider(c.Miner)
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

func InitialMarketUtil(rds dao.RdsService) {
	util.SupportTokens = make(map[string]types.Token)
	util.SupportMarkets = make(map[string]types.Token)
	util.AllTokens = make(map[string]types.Token)

	tokens, err := rds.FindUnDeniedTokens()
	if err != nil {
		log.Fatalf("market util cann't find any token!")
	}
	markets, err := rds.FindUnDeniedMarkets()
	if err != nil {
		log.Fatalf("market util cann't find any base market!")
	}

	// set support tokens
	for _, v := range tokens {
		var token types.Token
		v.ConvertUp(&token)
		util.SupportTokens[v.Symbol] = token
		log.Infof("supported token %s->%s", token.Symbol, token.Protocol.Hex())
	}

	// set all tokens
	for k, v := range util.SupportTokens {
		util.AllTokens[k] = v
	}
	for k, v := range util.SupportMarkets {
		util.AllTokens[k] = v
	}

	// set support markets
	for _, v := range markets {
		var token types.Token
		v.ConvertUp(&token)
		util.SupportMarkets[token.Symbol] = token
	}

	// set all markets
	for _, k := range util.SupportTokens { // lrc,omg
		for _, kk := range util.SupportMarkets { //eth
			symbol := k.Symbol + "-" + kk.Symbol
			util.AllMarkets = append(util.AllMarkets, symbol)
			log.Infof("supported market:%s", symbol)
		}
	}

	// set all token pairs
	pairsMap := make(map[string]util.TokenPair, 0)
	for _, v := range util.SupportMarkets {
		for _, vv := range util.SupportTokens {
			pairsMap[v.Symbol+"-"+vv.Symbol] = util.TokenPair{v.Protocol, vv.Protocol}
			pairsMap[vv.Symbol+"-"+v.Symbol] = util.TokenPair{vv.Protocol, v.Protocol}
		}
	}
	for _, v := range pairsMap {
		util.AllTokenPairs = append(util.AllTokenPairs, v)
	}
}

func loadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/relay.toml"
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}
