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

package util

import (
	"errors"
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
)

const WeiToEther = 1e18

type TokenPair struct {
	TokenS common.Address
	TokenB common.Address
}

type TokenStandard uint8

const (
	TOKEN_STANDARD_ERC20 TokenStandard = iota
	TOKEN_STANDARD_ERC223
)

func StringToFloat(amount string) float64 {
	rst, _ := new(big.Int).SetString(amount, 0)
	return float64(rst.Int64()) / WeiToEther
}

func FloatToByte(amount float64) []byte {
	rst, _ := big.NewInt(int64(amount * WeiToEther)).MarshalText()
	return rst
}

var (
	SupportTokens  map[string]types.Token // token symbol to entity
	AllTokens      map[string]types.Token
	SupportMarkets map[string]types.Token // token symbol to contract hex address
	AllMarkets     []string
	AllTokenPairs  []TokenPair
)

var ContractVersionConfig = map[string]string{}

func Initialize(rds dao.RdsService, conf *config.GlobalConfig) {

	SupportTokens = make(map[string]types.Token)
	SupportMarkets = make(map[string]types.Token)
	AllTokens = make(map[string]types.Token)

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
		SupportTokens[token.Symbol] = token
		log.Infof("market util,supported token %s->%s", token.Symbol, token.Protocol.Hex())
	}

	// set support markets
	for _, v := range markets {
		var token types.Token
		v.ConvertUp(&token)
		SupportMarkets[token.Symbol] = token
	}

	// set all tokens
	for k, v := range SupportTokens {
		AllTokens[k] = v
	}
	for k, v := range SupportMarkets {
		AllTokens[k] = v
	}

	// set all markets
	for _, k := range SupportTokens { // lrc,omg
		for _, kk := range SupportMarkets { //eth
			symbol := k.Symbol + "-" + kk.Symbol
			AllMarkets = append(AllMarkets, symbol)
			log.Infof("market util,supported market:%s", symbol)
		}
	}

	// set all token pairs
	pairsMap := make(map[string]TokenPair, 0)
	for _, v := range SupportMarkets {
		for _, vv := range SupportTokens {
			pairsMap[v.Symbol+"-"+vv.Symbol] = TokenPair{v.Protocol, vv.Protocol}
			pairsMap[vv.Symbol+"-"+v.Symbol] = TokenPair{vv.Protocol, v.Protocol}
		}
	}
	for _, v := range pairsMap {
		AllTokenPairs = append(AllTokenPairs, v)
	}

	for i := 0; i < len(conf.Contract.Versions); i++ {
		ContractVersionConfig[conf.Contract.Versions[i]] = conf.Contract.Addresses[i]
	}

	tokenRegisterWatcher := &eventemitter.Watcher{false, TokenRegister}
	tokenUnRegisterWatcher := &eventemitter.Watcher{false, TokenUnRegister}
	eventemitter.On(eventemitter.TokenRegistered, tokenRegisterWatcher)
	eventemitter.On(eventemitter.TokenUnRegistered, tokenUnRegisterWatcher)
}

func TokenRegister(input eventemitter.EventData) error {
	evt := input.(*types.TokenRegisterEvent)

	var token types.Token
	token.Protocol = evt.Token
	token.Symbol = strings.ToUpper(evt.Symbol)
	token.Deny = false
	token.IsMarket = false
	token.Time = evt.Time.Int64()

	// todo: how to get source
	token.Source = ""
	SupportTokens[token.Symbol] = token
	AllTokens[token.Symbol] = token

	pairsMap := make(map[string]TokenPair, 0)
	for _, v := range SupportMarkets {
		pairsMap[v.Symbol+"-"+token.Symbol] = TokenPair{v.Protocol, token.Protocol}
		pairsMap[token.Symbol+"-"+v.Symbol] = TokenPair{token.Protocol, v.Protocol}
	}
	for _, v := range pairsMap {
		AllTokenPairs = append(AllTokenPairs, v)
	}
	return nil
}

func TokenUnRegister(input eventemitter.EventData) error {
	evt := input.(*types.TokenUnRegisterEvent)

	delete(SupportTokens, strings.ToUpper(evt.Symbol))
	delete(AllTokens, strings.ToUpper(evt.Symbol))

	var list []TokenPair
	for _, v := range AllTokenPairs {
		if v.TokenS == evt.Token || v.TokenB == evt.Token {
			continue
		}
		list = append(list, v)
	}
	AllTokenPairs = list

	return nil
}

func WethTokenAddress() common.Address {
	return SupportMarkets["weth"].Protocol
}

func WrapMarket(s, b string) (market string, err error) {

	s, b = strings.ToUpper(s), strings.ToUpper(b)

	if IsSupportedMarket(s) && IsSupportedToken(b) {
		market = fmt.Sprintf("%s-%s", b, s)
	} else if IsSupportedMarket(b) && IsSupportedToken(s) {
		market = fmt.Sprintf("%s-%s", s, b)
	} else {
		err = errors.New(fmt.Sprintf("not supported market type : %s-%s", s, b))
	}
	return
}

func WrapMarketByAddress(s, b string) (market string, err error) {
	return WrapMarket(AddressToAlias(s), AddressToAlias(b))
}

func UnWrap(market string) (s, b string) {
	mkt := strings.Split(strings.TrimSpace(market), "-")
	if len(mkt) != 2 {
		return "", ""
	}

	s, b = strings.ToUpper(mkt[0]), strings.ToUpper(mkt[1])
	return
}

func UnWrapToAddress(market string) (s, b common.Address) {
	sa, sb := UnWrap(market)
	return common.StringToAddress(sa), common.StringToAddress(sb)
}

func IsSupportedMarket(market string) bool {
	_, ok := SupportMarkets[strings.ToUpper(market)]
	return ok
}

func IsSupportedToken(token string) bool {
	_, ok := SupportTokens[strings.ToUpper(token)]
	return ok
}

func AliasToAddress(t string) common.Address {
	return AllTokens[t].Protocol
}

func AddressToAlias(t string) string {
	for k, v := range AllTokens {
		if strings.ToUpper(t) == strings.ToUpper(v.Protocol.Hex()) {
			return k
		}
	}
	return ""
}

func CalculatePrice(amountS, amountB string, s, b string) float64 {

	as := StringToFloat(amountS)
	ab := StringToFloat(amountB)

	if as == 0 || ab == 0 {
		return 0
	}

	if IsBuy(s) {
		return ab / as
	}

	return as / ab

}

func IsBuy(s string) bool {
	if IsAddress(s) {
		s = AddressToAlias(s)
	}
	if _, ok := SupportTokens[s]; !ok {
		return false
	}
	return true
}

func IsAddress(token string) bool {
	return strings.HasPrefix(token, "0x")
}

func getContractVersion(address string) string {
	for k, v := range ContractVersionConfig {
		if strings.ToLower(v) == strings.ToLower(address) {
			return k
		}
	}
	return ""
}

func IsSupportedContract(address string) bool {
	return getContractVersion(address) != ""
}

func GetWethAddress() common.Address {
	return AllTokens["WETH"].Protocol
}
