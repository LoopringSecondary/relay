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
	"encoding/json"
	"fmt"
	"github.com/Loopring/relay/cache"
	"github.com/robfig/cron"
	"io/ioutil"
	"net/http"
	"strings"
)

type ExchangeType string

const (
	Binance ExchangeType = "binance"
	OkEx    ExchangeType = "okex"
	Huobi   ExchangeType = "huobi"
)

const cachePreKey = "TICKER_EX_"

//TODO (xiaolu)  add more exchanges to this list
var exchanges = map[string]string{
	"binance": "https://api.binance.com/api/v1/ticker/24hr?symbol=%s",
	"okex":    "https://www.okex.com/api/v1/ticker.do?symbol=%s",
	"huobi":   "https://api.huobi.pro/market/detail?symbol=%s",
}

const defaultSyncInterval = 5 // minutes

type Exchange interface {
	updateCache()
}

type ExchangeImpl struct {
	name      string
	tickerUrl string
}

type Collector interface {
	getTickers(market string) ([]Ticker, error)
	Start()
}

type CollectorImpl struct {
	exs          []Exchange
	syncInterval int
	cron         *cron.Cron
}

func NewExchange(name, tickerUrl string) *ExchangeImpl {
	return &ExchangeImpl{name, tickerUrl}
}

func (e *ExchangeImpl) updateCache() {
	cache.Set("", make([]byte, 0), 3600)

}

func NewCollector() *CollectorImpl {
	rst := &CollectorImpl{exs: make([]Exchange, 0), syncInterval: defaultSyncInterval, cron: cron.New()}

	for k, v := range exchanges {
		var exchange Exchange = NewExchange(k, v)
		rst.exs = append(rst.exs, exchange)
	}
	return rst
}

func (c *CollectorImpl) Start() {
	// create cron job and exec sync
	for _, e := range c.exs {
		c.cron.AddFunc("@every "+string(c.syncInterval)+"m", e.updateCache)
	}
	c.cron.Start()

}

func (c *CollectorImpl) getTickers(market string) ([]Ticker, error) {

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

type HuobiTicker struct {
	Id        string    `json:"id"`
	Timestamp int64     `json:"ts"`
	Close     float64   `json:"close"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Amount    float64   `json:"amount"`
	Count     int       `json:"count"`
	Vol       float64   `json:"vol"`
	Ask       []float64 `json:"ask"`
	Bid       []float64 `json:"bid"`
}

type BinanceTicker struct {
	Symbol    string  `json:"symbol"`
	Change    float64 `json:"priceChangePercent"`
	Close     float64 `json:"prevClosePrice"`
	Open      float64 `json:"openPrice"`
	High      float64 `json:"highPrice"`
	Low       float64 `json:"lowPrice"`
	LastPrice float64 `json:"lastPrice"`
	Amount    float64 `json:"volume"`
	Vol       float64 `json:"quoteVolume"`
	Ask       float64 `json:"askPrice"`
	Bid       float64 `json:"bidPrice"`
}

type OkexTicker struct {
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	LastPrice float64 `json:"last"`
	Vol       float64 `json:"vol"`
	Ask       float64 `json:"sell"`
	Bid       float64 `json:"buy"`
}

func GetTickerFromHuobi(market string) (ticker Ticker, err error) {

	huobiMarket := strings.Replace(market, "-", "", 1)
	huobiMarket = strings.ToLower(huobiMarket)
	url := fmt.Sprintf(exchanges["huobi"], huobiMarket)
	resp, err := http.Get(url)
	if err != nil {
		return ticker, err
	}
	defer func() {
		if nil != resp && nil != resp.Body {
			resp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return ticker, err
	} else {
		var huobiTicker *HuobiTicker
		if err := json.Unmarshal(body, &huobiTicker); nil != err {
			return ticker, err
		} else {
			ticker = Ticker{}
			ticker.Market = market
			ticker.Amount = huobiTicker.Amount
			ticker.Open = huobiTicker.Open
			ticker.Close = huobiTicker.Close
			ticker.Last = huobiTicker.Bid[0]
			ticker.Change = fmt.Sprintf("%.2f%%", 100*(ticker.Last-ticker.Open)/ticker.Open)
			ticker.Exchange = "huobi"
			ticker.Vol = huobiTicker.Vol
			ticker.High = huobiTicker.High
			ticker.Low = huobiTicker.Low

			return ticker, nil
		}
	}
}

func GetTickerFromBinance(market string) (ticker Ticker, err error) {

	binanceMarket := strings.Replace(market, "-", "", 1)
	binanceMarket = strings.ToUpper(binanceMarket)
	url := fmt.Sprintf(exchanges["binance"], binanceMarket)

	resp, err := http.Get(url)
	if err != nil {
		return ticker, err
	}
	defer func() {
		if nil != resp && nil != resp.Body {
			resp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return ticker, err
	} else {
		var binanceTicker *BinanceTicker
		if err := json.Unmarshal(body, &binanceTicker); nil != err {
			return ticker, err
		} else {
			ticker = Ticker{}
			//ticker.Market = market
			ticker.Amount = binanceTicker.Amount
			ticker.Open = binanceTicker.Open
			ticker.Close = binanceTicker.Close
			ticker.Last = binanceTicker.LastPrice
			ticker.Change = fmt.Sprintf("%.2f%%", binanceTicker.Change)
			ticker.Exchange = "binance"
			ticker.Vol = binanceTicker.Vol
			ticker.High = binanceTicker.High
			ticker.Low = binanceTicker.Low
			return ticker, nil
		}
	}
}

func GetTickerFromOkex(market string) (ticker Ticker, err error) {

	okexMarket := strings.Replace(market, "_", "", 1)
	okexMarket = strings.ToLower(okexMarket)
	url := fmt.Sprintf(exchanges["okex"], okexMarket)

	resp, err := http.Get(url)
	if err != nil {
		return ticker, err
	}
	defer func() {
		if nil != resp && nil != resp.Body {
			resp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return ticker, err
	} else {
		var okexTicker *OkexTicker
		if err := json.Unmarshal(body, &okexTicker); nil != err {
			return ticker, err
		} else {
			ticker = Ticker{}
			//ticker.Market = market
			ticker.Last = okexTicker.LastPrice
			ticker.Change = fmt.Sprintf("%.2f%%", 100*(ticker.Last-ticker.Open)/ticker.Open)
			ticker.Exchange = "okex"
			ticker.Vol = okexTicker.Vol
			ticker.High = okexTicker.High
			ticker.Low = okexTicker.Low
			return ticker, nil
		}
	}
}
