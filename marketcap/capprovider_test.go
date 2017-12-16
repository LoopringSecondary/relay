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
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/test"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	//cfg := test.LoadConfig()
	//provider := marketcap.NewMarketCapProvider(cfg.MarketCap)
	//provider.Start()
	//
	//for _, token := range util.AllTokens {
	//	p1, _ := provider.GetMarketCap(token.Protocol)
	//	p2, _ := provider.GetMarketCapByCurrency(token.Protocol, "USD")
	//
	//	t.Logf("second round token:%s, p1:%s, p2:%s", token.Symbol, p1.String(), p2.String())
	//}
	//
	//time.Sleep(3 * time.Minute)
	//for _, token := range util.AllTokens {
	//	p1, _ := provider.GetMarketCap(token.Protocol)
	//	p2, _ := provider.GetMarketCapByCurrency(token.Protocol, "USD")
	//
	//	t.Logf("first round token:%s, p1:%s, p2:%s", token.Symbol, p1.String(), p2.String())
	//}
}
