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
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
)

const weiToEther = 1e18

type TokenPair struct {
	TokenS common.Address
	TokenB common.Address
}

func ByteToFloat(amount []byte) float64 {
	var rst big.Int
	rst.UnmarshalText(amount)
	return float64(rst.Int64()) / weiToEther
}

func FloatToByte(amount float64) []byte {
	rst, _ := big.NewInt(int64(amount * weiToEther)).MarshalText()
	return rst
}

var SupportTokens = map[string]string{
	"lrc":  "0xskdfjdkfj",
	"coss": "0xskdjfskdfj",
}

var SupportMarket = map[string]string{
	"weth": "0xsldkfjsdkfj",
}

var ContractVersionConfig = map[string]string{
	"v1.0": "0x39kdjfskdfjsdfj",
	"v1.2": "0x39kdjfskdfjsdfj",
}

var AllTokens = func() map[string]string {
	all := make(map[string]string)
	for k, v := range SupportMarket {
		all[k] = v
	}
	for k, v := range SupportTokens {
		all[k] = v
	}
	return all
}()

var AllMarkets = AllMarket()

var AllTokenPairs = func() []TokenPair {
	pairsMap := make(map[string]TokenPair, 0)
	for _, v := range SupportMarket {
		for _, vv := range SupportTokens {
			pairsMap[v+"-"+vv] = TokenPair{common.StringToAddress(v), common.StringToAddress(vv)}
			pairsMap[vv+"-"+v] = TokenPair{common.StringToAddress(vv), common.StringToAddress(v)}
		}
	}
	pairs := make([]TokenPair, 0)
	for _, v := range pairsMap {
		pairs = append(pairs, v)
	}

	return pairs
}()

func WrapMarket(s, b string) (market string, err error) {

	s, b = strings.ToLower(s), strings.ToLower(b)

	if SupportMarket[s] != "" && SupportTokens[b] != "" {
		market = fmt.Sprintf("%s-%s", b, s)
	} else if SupportMarket[b] != "" && SupportTokens[s] != "" {
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

	s, b = strings.ToLower(mkt[0]), strings.ToLower(mkt[1])
	return
}

func UnWrapToAddress(market string) (s, b common.Address) {
	sa, sb := UnWrap(market)
	return common.StringToAddress(sa), common.StringToAddress(sb)
}

func IsSupportedToken(token string) bool {
	return SupportTokens[token] != ""
}

func AliasToAddress(t string) string {
	return AllTokens[t]
}

func AddressToAlias(t string) string {
	for k, v := range AllTokens {
		if t == v {
			return k
		}
	}
	return ""
}

func AllMarket() []string {
	mkts := make([]string, 0)
	for k := range SupportTokens {
		for kk := range SupportMarket {
			mkts = append(mkts, k+"-"+kk)
		}
	}
	return mkts
}

func CalculatePrice(amountS, amountB []byte, s, b string) float64 {

	as := ByteToFloat(amountS)
	ab := ByteToFloat(amountB)

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
	if SupportTokens[s] != "" {
		return false
	}
	return true
}

func IsAddress(token string) bool {
	return strings.HasPrefix(token, "0x")
}

func getContractVersion(address string) string {
	for k, v := range ContractVersionConfig {
		if v == address {
			return k
		}
	}
	return ""
}

func IsSupportedContract(address string) bool {
	return getContractVersion(address) != ""
}
