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
	util.Initialize(cfg.Market)
	provider := marketcap.NewMarketCapProvider(cfg.MarketCap)
	provider.Start()
	for _, token := range util.AllTokens {
		p1, _ := provider.GetMarketCap(token.Protocol)
		p2, _ := provider.GetMarketCapByCurrency(token.Protocol, "USD")
		t.Logf("second round token:%s, p1:%s, p2:%s", token.Symbol, p1.FloatString(2), p2.FloatString(2))
	}

	a := new(big.Rat)
	a.SetString("284186332622238284")
	f, err := provider.LegalCurrencyValueByCurrency(common.HexToAddress("0xBeB6fdF4ef6CEb975157be43cBE0047B248a8922"), a, "USD")
	if nil != err {
		t.Errorf(err.Error())
	} else {
		println(f.FloatString(2))
	}
	f1, err := provider.LegalCurrencyValueByCurrency(common.HexToAddress("0xEF68e7C694F40c8202821eDF525dE3782458639f"), a, "USD")
	if nil != err {
		t.Errorf(err.Error())
	} else {
		println(f1.FloatString(2))
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
