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

package marketcap_test

import (
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func TestStart(t *testing.T) {
	cfg := config.LoadConfig("/Users/yuhongyu/Desktop/service/go/src/github.com/Loopring/relay/config/relay.toml")

	log.Initialize(cfg.Log)
	m := make(map[string]string)
	util.Initialize(cfg.Market, m)
	provider := marketcap.NewMarketCapProvider(cfg.MarketCap)
	provider.Start()
	a := new(big.Rat).SetInt64(int64(1000000000000000000))
	s, _ := provider.LegalCurrencyValue(common.HexToAddress("0x5ca9a71b1d01849c0a95490cc00559717fcf0d1d"), a)
	t.Log(s.String())
	for _, token := range util.AllTokens {
		p1, _ := provider.GetMarketCap(token.Protocol)
		p2, _ := provider.GetMarketCapByCurrency(token.Protocol, "USD")

		t.Logf("second round token:%s, p1:%s, p2:%s", token.Symbol, p1.String(), p2.String())
	}
	//
	//time.Sleep(3 * time.Minute)
	//for _, token := range util.AllTokens {
	//	p1, _ := provider.GetMarketCap(token.Protocol)
	//	p2, _ := provider.GetMarketCapByCurrency(token.Protocol, "USD")
	//
	//	t.Logf("first round token:%s, p1:%s, p2:%s", token.Symbol, p1.String(), p2.String())
	//}
}
