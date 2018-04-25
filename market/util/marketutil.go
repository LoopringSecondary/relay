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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/robfig/cron"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
)

const SideSell = "sell"
const SideBuy = "buy"

type TokenPair struct {
	TokenS common.Address
	TokenB common.Address
}

var MarketBaseOrder = map[string]uint8{"BAR": 5, "LRC": 10, "WETH": 20, "DAI": 30}

type TokenStandard uint8

func StringToFloat(token string, amount string) float64 {
	rst, _ := new(big.Rat).SetString(amount)
	ts, _ := AddressToToken(common.HexToAddress(token))
	weiRat := new(big.Rat).SetInt64(ts.Decimals.Int64())
	rst.Quo(rst, weiRat)
	result, _ := rst.Float64()
	return result
}

var (
	SupportTokens  map[string]types.Token // token symbol to entity
	AllTokens      map[string]types.Token
	SupportMarkets map[string]types.Token // token symbol to contract hex address
	AllMarkets     []string
	AllTokenPairs  []TokenPair
	SymbolTokenMap map[common.Address]string
)

func StartRefreshCron(option config.MarketOptions) {
	mktCron := cron.New()
	mktCron.AddFunc("1 0/10 * * * *", func() {
		log.Info("start market util refresh.....")
		SupportTokens, SupportMarkets, AllTokens, AllMarkets, AllTokenPairs, SymbolTokenMap = getTokenAndMarketFromDB(option.TokenFile)
	})
	mktCron.Start()
}

type token struct {
	Protocol string `json:"Protocol"`
	Symbol   string `json:"Symbol"`
	Source   string `json:"Source"`
	Deny     bool   `json:"Deny"`
	Decimals int    `json:"Decimals"`
	IsMarket bool   `json:"IsMarket"`
	IcoPrice string `json:"IcoPrice"`
}

func (t *token) convert() types.Token {
	var dst types.Token

	dst.Protocol = common.HexToAddress(t.Protocol)
	dst.Symbol = strings.ToUpper(t.Symbol)
	dst.Source = t.Source
	dst.Deny = t.Deny
	dst.Decimals = new(big.Int)
	dst.Decimals.SetString("1"+strings.Repeat("0", t.Decimals), 0)
	dst.IsMarket = t.IsMarket
	if "" != t.IcoPrice {
		dst.IcoPrice = new(big.Rat)
		dst.IcoPrice.SetString(t.IcoPrice)
	}

	return dst
}

func getTokenAndMarketFromDB(tokenfile string) (
	supportTokens map[string]types.Token,
	supportMarkets map[string]types.Token,
	allTokens map[string]types.Token,
	allMarkets []string,
	allTokenPairs []TokenPair,
	symbolTokenMap map[common.Address]string) {

	supportTokens = make(map[string]types.Token)
	allTokens = make(map[string]types.Token)
	supportMarkets = make(map[string]types.Token)
	allMarkets = make([]string, 0)
	allTokenPairs = make([]TokenPair, 0)
	symbolTokenMap = make(map[common.Address]string)

	var list []token
	fn, err := os.Open(tokenfile)
	if err != nil {
		log.Fatalf("market util load tokens failed:%s", err.Error())
	}
	bs, err := ioutil.ReadAll(fn)
	if err != nil {
		log.Fatalf("market util read tokens json file failed:%s", err.Error())
	}
	if err := json.Unmarshal(bs, &list); err != nil {
		log.Fatalf("market util unmarshal tokens failed:%s", err.Error())
	}

	for _, v := range list {
		if v.Deny == false {
			t := v.convert()
			if t.IsMarket == true {
				supportMarkets[t.Symbol] = t
			} else {
				supportTokens[t.Symbol] = t
				log.Infof("market util,supported token:%s", t.Symbol)
			}
		}
	}

	// set all tokens
	for k, v := range supportTokens {
		allTokens[k] = v
		symbolTokenMap[v.Protocol] = v.Symbol
	}
	for k, v := range supportMarkets {
		allTokens[k] = v
		symbolTokenMap[v.Protocol] = v.Symbol
	}

	// set all markets
	for k := range allTokens { // lrc,omg
		for kk := range supportMarkets { //eth
			o, ok := MarketBaseOrder[k]
			if ok {
				baseOrder := MarketBaseOrder[kk]
				if o < baseOrder {
					allMarkets = append(allMarkets, k+"-"+kk)
				}
			} else {
				allMarkets = append(allMarkets, k+"-"+kk)
			}
			log.Infof("market util,supported market:%s", k+"-"+kk)
		}
	}

	// set all token pairs
	pairsMap := make(map[string]TokenPair, 0)
	for _, v := range supportMarkets {
		for _, vv := range allTokens {
			if v.Symbol != vv.Symbol {
				pairsMap[v.Symbol+"-"+vv.Symbol] = TokenPair{v.Protocol, vv.Protocol}
				pairsMap[vv.Symbol+"-"+v.Symbol] = TokenPair{vv.Protocol, v.Protocol}
			}
		}
	}

	for _, v := range pairsMap {
		allTokenPairs = append(allTokenPairs, v)
	}

	return
}

func Initialize(options config.MarketOptions) {

	SupportTokens = make(map[string]types.Token)
	SupportMarkets = make(map[string]types.Token)
	AllTokens = make(map[string]types.Token)
	SymbolTokenMap = make(map[common.Address]string)

	SupportTokens, SupportMarkets, AllTokens, AllMarkets, AllTokenPairs, SymbolTokenMap = getTokenAndMarketFromDB(options.TokenFile)

	// StartRefreshCron(rds)

	//tokenRegisterWatcher := &eventemitter.Watcher{false, TokenRegister}
	tokenUnRegisterWatcher := &eventemitter.Watcher{false, TokenUnRegister}
	//eventemitter.On(eventemitter.TokenRegistered, tokenRegisterWatcher)
	eventemitter.On(eventemitter.TokenUnRegistered, tokenUnRegisterWatcher)
}

func TokenRegister(input eventemitter.EventData) error {
	evt := input.(*types.TokenRegisterEvent)

	var token types.Token
	token.Protocol = evt.Token
	token.Symbol = strings.ToUpper(evt.Symbol)
	token.Deny = false
	token.IsMarket = false
	token.Time = evt.BlockTime

	// todo: how to get source token.Source = ""
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
	return AllTokens["WETH"].Protocol
}

func WrapMarket(s, b string) (market string, err error) {

	s, b = strings.ToUpper(s), strings.ToUpper(b)

	if IsSupportedMarket(s) && isSupportedToken(b) {
		market = fmt.Sprintf("%s-%s", b, s)
	} else if IsSupportedMarket(b) && isSupportedToken(s) {
		market = fmt.Sprintf("%s-%s", s, b)
	} else if IsSupportedMarket(b) && IsSupportedMarket(s) {
		if MarketBaseOrder[s] < MarketBaseOrder[b] {
			market = fmt.Sprintf("%s-%s", s, b)
		} else {
			market = fmt.Sprintf("%s-%s", b, s)
		}
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

func isSupportedToken(token string) bool {
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

func AddressToToken(t common.Address) (*types.Token, error) {
	for _, v := range AllTokens {
		if v.Protocol == t {
			return &v, nil
		}
	}

	return nil, fmt.Errorf("unsupported token:%s", t.Hex())
}

func CalculatePrice(amountS, amountB string, s, b string) float64 {

	as, _ := new(big.Int).SetString(amountS, 0)
	ab, _ := new(big.Int).SetString(amountB, 0)

	result := new(big.Rat).SetInt64(0)

	tokenS, ok := AllTokens[AddressToAlias(s)]
	if !ok {
		return 0
	}
	tokenB, ok := AllTokens[AddressToAlias(b)]
	if !ok {
		return 0
	}

	if as.Cmp(big.NewInt(0)) == 0 || ab.Cmp(big.NewInt(0)) == 0 {
		return 0
	}

	if GetSide(s, b) == SideBuy {
		result.Quo(new(big.Rat).SetFrac(as, tokenS.Decimals), new(big.Rat).SetFrac(ab, tokenB.Decimals))
	} else {
		result.Quo(new(big.Rat).SetFrac(ab, tokenB.Decimals), new(big.Rat).SetFrac(as, tokenS.Decimals))
	}

	price, _ := result.Float64()
	return price
}

//
//func IsBuy(tokenB string) bool {
//	if IsAddress(tokenB) {
//		tokenB = AddressToAlias(tokenB)
//	}
//
//
//	if _, ok := SupportTokens[tokenB]; !ok {
//		return false
//	}
//	return true
//}

func GetSide(s, b string) string {

	if IsAddress(s) {
		s = AddressToAlias(s)
	}

	if IsAddress(b) {
		b = AddressToAlias(b)
	}

	if IsSupportedMarket(s) && isSupportedToken(b) {
		return SideBuy
	} else if IsSupportedMarket(b) && isSupportedToken(s) {
		return SideSell
	} else if IsSupportedMarket(b) && IsSupportedMarket(s) {
		if MarketBaseOrder[s] < MarketBaseOrder[b] {
			return SideSell
		} else {
			return SideBuy
		}
	}
	return ""
}

func IsAddress(token string) bool {
	return strings.HasPrefix(token, "0x")
}

func GetSymbolWithAddress(address common.Address) (string, error) {
	if symbol, ok := SymbolTokenMap[address]; ok {
		return symbol, nil
	}
	return "", fmt.Errorf("market util, unsupported address:%s", address.Hex())
}
