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

package market

import (
	"github.com/robfig/cron"
	"github.com/Loopring/relay/cache"
)

type ExchangeType string

const (
	Binance ExchangeType = "binance"
	OkEx ExchangeType = "okex"
	Huobi ExchangeType = "huobi"
)

const cachePreKey = "TICKER_EX_"

//TODO (xiaolu)  add more exchanges to this list
var exchanges = map[string]string {
	"binance" : "https://api.binance.com/api/v1/ticker/24hr?symbol=",
	"okex" : "https://www.okex.com/api/v1/ticker.do?symbol=",
	"huobi" : "https://api.huobi.pro/market/detail/merged?symbol=",
}

const defaultSyncInterval = 5 // minutes

type Exchange interface {
	getTickerUrl() string
	tickerMapper() Ticker
	marketMapper() string
	updateCache()
}

type ExchangeImpl struct {
	name string
	tickerUrl string
}

type Collector interface {
	getTickers(market string) ([]Ticker, error)
	Start()
}

type CollectorImpl struct {
	exs []Exchange
	syncInterval int
	cron *cron.Cron
}

func NewExchange(name, tickerUrl string) *ExchangeImpl {
	return &ExchangeImpl{name, tickerUrl}
}

func (e *ExchangeImpl) getTickerUrl() string {
	return ""
}

func (e *ExchangeImpl) tickerMapper() Ticker {
	return Ticker{}
}

func (e *ExchangeImpl) marketMapper() string {
	return ""
}

func (e *ExchangeImpl) updateCache(){
	cache.Set("", make([]byte, 0), 3600)

}

func NewCollector() *CollectorImpl {
	rst := &CollectorImpl{exs:make([]Exchange, 0), syncInterval: defaultSyncInterval, cron : cron.New()}

	for k, v := range exchanges {
		var exchange  Exchange = NewExchange(k, v)
		rst.exs = append(rst.exs, exchange)
	}
	return rst
}

func (c *CollectorImpl) Start() {
	// create cron job and exec sync
	for _, e := range c.exs {
		c.cron.AddFunc("@every " + string(c.syncInterval) + "m", e.updateCache)
	}
	c.cron.Start()

}

func (c *CollectorImpl) getTickers(market string) ([] Ticker, error) {

	cache.Get(cachePreKey + market + "_")
	return nil, nil
}

//func (c *CollectorImpl) getTickerFromRemote(exchange string) ([] Ticker , error) {
//	return nil, nil
//}
//
//func (c *CollectorImpl) getTickerByMarket(exchange string , market string) (Ticker, error) {
//
//}





