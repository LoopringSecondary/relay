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
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	gocache "github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"io/ioutil"
	"net/http"
	"qiniupkg.com/x/errors.v7"
	"strconv"
	"strings"
	"time"
)

type ExchangeType string

const (
	binance = "binance"
	okex    = "okex"
	huobi   = "huobi"
)

// only support eth market now
var supportedMarkets = make([]string, 0)

const cachePreKey = "TICKER_EX_"

//TODO (xiaolu)  add more exchanges to this list
var exchanges = map[string]string{
	binance: "https://api.binance.com/api/v1/ticker/24hr?symbol=%s",
	okex:    "https://www.okex.com/api/v1/ticker.do?symbol=%s",
	huobi:   "https://api.huobi.pro/market/detail/merged?symbol=%s",
}

const defaultSyncInterval = 5 // minutes

type Exchange interface {
	updateCache()
}

type TickerField struct {
	key   []byte
	value []byte
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
	exs          []ExchangeImpl
	syncInterval int
	cron         *cron.Cron
	cronJobLock  bool
	localCache   *gocache.Cache
}

func NewExchange(name, tickerUrl string) ExchangeImpl {
	return ExchangeImpl{name, tickerUrl}
}

func (e *ExchangeImpl) updateCache() {
	log.Info("step in update cache method........")

	switch e.name {
	case binance:
		updateCacheByExchange(e.name, GetTickerFromBinance)
	case okex:
		updateCacheByExchange(e.name, GetTickerFromOkex)
	case huobi:
		updateCacheByExchange(e.name, GetTickerFromHuobi)
	}
}

func mockUpdateCache() {
	tickers := make([]Ticker, 0)
	t1 := Ticker{Market: "LRC-WETH", Amount: 0.001}
	t2 := Ticker{Market: "LRC-WETH", Amount: 0.003}
	t3 := Ticker{Market: "FOO-BAR", Amount: 0.002}
	tickers = append(tickers, t1)
	tickers = append(tickers, t2)
	tickers = append(tickers, t3)
	var err error
	if err == nil && len(tickers) > 0 {
		tkFields := make([]TickerField, 0)
		for _, t := range tickers {
			tkField, err := buildTickerField(t.Market, t)
			if err == nil {
				tkFields = append(tkFields, tkField)
			}
			//setCache(binance, t.Market, t)
		}
		setHMCache(binance, tkFields)
		t11, _ := buildTickerField(t1.Market, t1)
		setHMCache(okex, []TickerField{t11})
	}
}

func updateBinanceCache() {
	//updateCacheByExchange("binance", GetTickerFromBinance)
	tickers, err := GetAllTickerFromBinance()
	if err == nil && len(tickers) > 0 {
		tkFields := make([]TickerField, 0)
		for _, t := range tickers {
			tkField, err := buildTickerField(t.Market, t)
			if err == nil {
				tkFields = append(tkFields, tkField)
			}
			//setCache(binance, t.Market, t)
		}
		setHMCache(binance, tkFields)
	}
}
func updateOkexCache() {
	tickers, err := GetAllTickerFromOkex()
	if err == nil && len(tickers) > 0 {
		tkFields := make([]TickerField, 0)
		for _, t := range tickers {
			tkField, err := buildTickerField(t.Market, t)
			if err == nil {
				tkFields = append(tkFields, tkField)
			}
			//setCache(okex, t.Market, t)
		}
		setHMCache(okex, tkFields)
	}
}
func updateHuobiCache() {
	updateCacheByExchange(huobi, GetTickerFromHuobi)
}

func updateCacheByExchange(exchange string, getter func(mkt string) (ticker Ticker, err error)) {

	tkFields := make([]TickerField, 0)
	for _, v := range util.AllMarkets {

		if !stringInSlice(v, supportedMarkets) {
			continue
		}

		vv := strings.ToUpper(v)
		vv = strings.Replace(vv, "WETH", "ETH", 1)
		ticker, err := getter(vv)
		if err != nil {
			//log.Debug("get ticker error " + err.Error())
		} else {
			//setCache(exchange, v, ticker)
			tkField, err := buildTickerField(v, ticker)
			if err == nil {
				tkFields = append(tkFields, tkField)
			}
		}
	}
	setHMCache(exchange, tkFields)
}

func setCache(exchange, market string, ticker Ticker) {
	cacheKey := cachePreKey + exchange
	tickerByte, err := json.Marshal(ticker)
	if err != nil {
		log.Info("marshal ticker json error " + err.Error())
	} else {
		cache.Set(cacheKey, tickerByte, 3600)
	}
}

func buildTickerField(market string, ticker Ticker) (tf TickerField, err error) {

	if len(market) == 0 {
		return tf, errors.New("market is nil")
	}

	tickerByte, err := json.Marshal(ticker)
	if err != nil {
		log.Info("marshal ticker json error " + err.Error())
		return tf, err
	} else {
		return TickerField{key: []byte(market), value: tickerByte}, nil
	}
}

func setHMCache(exchange string, tickers []TickerField) {

	data := [][]byte{}

	for _, tk := range tickers {
		data = append(data, tk.key)
		data = append(data, tk.value)
	}

	cacheKey := cachePreKey + exchange
	cache.HMSet(cacheKey, 3600*24*30, data...)
}

func NewCollector(cronJobLock bool) *CollectorImpl {
	rst := &CollectorImpl{exs: make([]ExchangeImpl, 0), syncInterval: defaultSyncInterval, cron: cron.New(), cronJobLock: cronJobLock}
	rst.localCache = gocache.New(5*time.Second, 5*time.Minute)
	for _, v := range util.AllMarkets {
		if strings.HasSuffix(v, "ETH") {
			supportedMarkets = append(supportedMarkets, v)
		}
	}

	for k, v := range exchanges {
		exchange := NewExchange(k, v)
		rst.exs = append(rst.exs, exchange)
	}
	return rst
}

func (c *CollectorImpl) Start() {
	// create cron job and exec sync
	if c.cronJobLock {
		//mockUpdateCache()
		updateBinanceCache()
		updateOkexCache()
		updateHuobiCache()
		c.cron.AddFunc("@every 20s", updateBinanceCache)
		c.cron.AddFunc("@every 5s", updateOkexCache)
		c.cron.AddFunc("@every 5s", updateHuobiCache)
		log.Info("start collect cron jobs......... ")
		c.cron.Start()
	}
}

func (c *CollectorImpl) GetTickers(market string) ([]Ticker, error) {

	result := make([]Ticker, 0)

	if len(market) <= 0 {
		return nil, errors.New("market can't be null")
	}

	for _, e := range c.exs {

		tkByteInLocal, ok := c.localCache.Get(e.name)
		if ok {
			//log.Infof("get ticker from local cache, ex : %s, market : %s", e.name, market)
			localTickers := tkByteInLocal.(map[string]Ticker)
			if _, ok := localTickers[market]; ok {
				result = append(result, localTickers[market])
			}
		} else {

			//log.Infof("get ticker from redis cache, ex : %s, market : %s", e.name, market)
			tickerMap, err := getAllMarketFromRedis(e.name)
			if err != nil {
				continue
			}

			v, ok := tickerMap[market]
			if ok {
				result = append(result, v)
			}

			c.localCache.Set(e.name, tickerMap, 5*time.Second)

			//cacheKey := cachePreKey + e.name + "_" + market
			//byteRst, err := cache.Get(cacheKey)
			//cacheKey := cachePreKey + e.name
			//byteRst, err := cache.HMGet(cacheKey, []byte(market))
			//if err == nil && len(byteRst) == 1 && len(byteRst[0]) > 0 {
			//	var unmarshalRst Ticker
			//	json.Unmarshal(byteRst[0], &unmarshalRst)
			//	result = append(result, unmarshalRst)
			//} else {
			//	//log.Info("get ticker from redis error" + err.Error())
			//}
		}

	}
	return result, nil
}

func getAllMarketFromRedis(exchange string) (tickers map[string]Ticker, err error) {
	keys := make([][]byte, 0)
	for _, m := range supportedMarkets {
		keys = append(keys, []byte(m))
	}

	if len(keys) == 0 {
		return nil, errors.New("no supported market found")
	}

	byteRst, err := cache.HMGet(cachePreKey+exchange, keys...)

	if err != nil {
		return nil, err
	}
	tickers = make(map[string]Ticker)
	if err == nil && len(byteRst) > 1 {
		for _, tb := range byteRst {
			if len(tb) == 0 {
				continue
			}
			var unmarshalRst Ticker
			json.Unmarshal(tb, &unmarshalRst)
			if len(unmarshalRst.Market) > 0 {
				tickers[unmarshalRst.Market] = unmarshalRst
			}
		}
	}
	return
}

type HuobiTicker struct {
	Timestamp int64            `json:"ts"`
	ErrorCode string           `json:"err-code"`
	Status    string           `json:"status"`
	Tick      HuobiInnerTicker `json:"tick"`
}

type HuobiInnerTicker struct {
	Close  float64   `json:"close"`
	Open   float64   `json:"open"`
	High   float64   `json:"high"`
	Low    float64   `json:"low"`
	Amount float64   `json:"amount"`
	Count  int       `json:"count"`
	Vol    float64   `json:"vol"`
	Ask    []float64 `json:"ask"`
	Bid    []float64 `json:"bid"`
}

type BinanceTicker struct {
	Symbol    string `json:"symbol"`
	Change    string `json:"priceChangePercent"`
	Close     string `json:"prevClosePrice"`
	Open      string `json:"openPrice"`
	High      string `json:"highPrice"`
	Low       string `json:"lowPrice"`
	LastPrice string `json:"lastPrice"`
	Amount    string `json:"volume"`
	Vol       string `json:"quoteVolume"`
	Ask       string `json:"askPrice"`
	Bid       string `json:"bidPrice"`
}

type OkexTicker struct {
	Date   string          `json:"date"`
	Ticker OkexInnerTicker `json:"ticker"`
}

type OkexInnerTicker struct {
	High      string `json:"high"`
	Low       string `json:"low"`
	LastPrice string `json:"last"`
	Vol       string `json:"vol"`
	Ask       string `json:"sell"`
	Bid       string `json:"buy"`
}

func GetTickerFromHuobi(market string) (ticker Ticker, err error) {

	huobiMarket := strings.Replace(market, "-", "", 1)
	huobiMarket = strings.ToLower(huobiMarket)
	url := fmt.Sprintf(exchanges[huobi], huobiMarket)
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

			if huobiTicker.Status == "error" {
				return ticker, errors.New("get ticker from huobi error" + huobiTicker.ErrorCode)
			}

			ticker = Ticker{}
			innerTicker := huobiTicker.Tick
			ticker.Market = market
			ticker.Amount = innerTicker.Amount
			ticker.Open = innerTicker.Open
			ticker.Close = innerTicker.Close
			if len(innerTicker.Bid) > 0 {
				ticker.Last = innerTicker.Bid[0]
			}
			if ticker.Last-ticker.Open > 0 {
				ticker.Change = fmt.Sprintf("+%.2f%%", 100*(ticker.Last-ticker.Open)/ticker.Open)
			} else {
				ticker.Change = fmt.Sprintf("%.2f%%", 100*(ticker.Last-ticker.Open)/ticker.Open)
			}
			ticker.Exchange = huobi
			ticker.Vol = innerTicker.Vol
			ticker.High = innerTicker.High
			ticker.Low = innerTicker.Low

			return ticker, nil
		}
	}
}

func GetTickerFromBinance(market string) (ticker Ticker, err error) {

	binanceMarket := strings.Replace(market, "-", "", 1)
	binanceMarket = strings.ToUpper(binanceMarket)
	url := fmt.Sprintf(exchanges[binance], binanceMarket)

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
			ticker.Market = market
			ticker.Amount, _ = strconv.ParseFloat(binanceTicker.Amount, 64)
			ticker.Open, _ = strconv.ParseFloat(binanceTicker.Open, 64)
			ticker.Close, _ = strconv.ParseFloat(binanceTicker.Close, 64)
			ticker.Last, _ = strconv.ParseFloat(binanceTicker.LastPrice, 64)
			change, _ := strconv.ParseFloat(binanceTicker.Change, 64)
			if change > 0 {
				ticker.Change = fmt.Sprintf("+%.2f%%", change)
			} else {
				ticker.Change = fmt.Sprintf("%.2f%%", change)
			}
			ticker.Exchange = binance
			ticker.Vol, _ = strconv.ParseFloat(binanceTicker.Vol, 64)
			ticker.High, _ = strconv.ParseFloat(binanceTicker.High, 64)
			ticker.Low, _ = strconv.ParseFloat(binanceTicker.Low, 64)
			return ticker, nil
		}
	}
}

func GetAllTickerFromBinance() (tickers []Ticker, err error) {

	resp, err := http.Get("https://api.binance.com/api/v1/ticker/24hr")
	if err != nil {
		return tickers, err
	}
	defer func() {
		if nil != resp && nil != resp.Body {
			resp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return tickers, err
	} else {
		var binanceTickers []BinanceTicker
		if err := json.Unmarshal(body, &binanceTickers); nil != err {
			return tickers, err
		} else {

			if len(binanceTickers) == 0 {
				return tickers, errors.New("fetch ticker from binance failed")
			}

			tickers = make([]Ticker, 0)
			for _, binanceTicker := range binanceTickers {

				if !strings.HasSuffix(binanceTicker.Symbol, "ETH") {
					// only handle eth market tickers
					continue
				}

				ticker := Ticker{}
				//ticker.Market = market
				marketPre := binanceTicker.Symbol[0 : len(binanceTicker.Symbol)-len("ETH")]

				ticker.Market = marketPre + "-" + "WETH"
				ticker.Amount, _ = strconv.ParseFloat(binanceTicker.Amount, 64)
				ticker.Open, _ = strconv.ParseFloat(binanceTicker.Open, 64)
				ticker.Close, _ = strconv.ParseFloat(binanceTicker.Close, 64)
				ticker.Last, _ = strconv.ParseFloat(binanceTicker.LastPrice, 64)
				change, _ := strconv.ParseFloat(binanceTicker.Change, 64)
				if change > 0 {
					ticker.Change = fmt.Sprintf("+%.2f%%", change)
				} else {
					ticker.Change = fmt.Sprintf("%.2f%%", change)
				}
				ticker.Exchange = binance
				ticker.Vol, _ = strconv.ParseFloat(binanceTicker.Vol, 64)
				ticker.High, _ = strconv.ParseFloat(binanceTicker.High, 64)
				ticker.Low, _ = strconv.ParseFloat(binanceTicker.Low, 64)
				tickers = append(tickers, ticker)
			}
			return tickers, nil
		}
	}
}

func GetTickerFromOkex(market string) (ticker Ticker, err error) {

	okexMarket := strings.Replace(market, "-", "_", 1)
	okexMarket = strings.ToLower(okexMarket)
	url := fmt.Sprintf(exchanges[okex], okexMarket)

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
		var okexOutTicker *OkexTicker
		if err := json.Unmarshal(body, &okexOutTicker); nil != err {
			return ticker, err
		} else {
			ticker = Ticker{}
			okexTicker := okexOutTicker.Ticker
			ticker.Market = market
			ticker.Last, _ = strconv.ParseFloat(okexTicker.LastPrice, 64)
			if 100*(ticker.Last-ticker.Open)/ticker.Open > 0 {
				ticker.Change = fmt.Sprintf("+%.2f%%", 100*(ticker.Last-ticker.Open)/ticker.Open)
			} else {
				ticker.Change = fmt.Sprintf("%.2f%%", 100*(ticker.Last-ticker.Open)/ticker.Open)
			}
			ticker.Exchange = okex
			ticker.Amount, _ = strconv.ParseFloat(okexTicker.Vol, 64)
			ticker.Vol = ticker.Amount * ticker.Last
			ticker.High, _ = strconv.ParseFloat(okexTicker.High, 64)
			ticker.Low, _ = strconv.ParseFloat(okexTicker.Low, 64)
			return ticker, nil
		}
	}
}

type OkexFullTicker struct {
	Code int              `json:"code"`
	Data []OkexTickerElem `json:"data"`
	Msg  string           `json:"msg"`
}

type OkexTickerElem struct {
	Buy    string `json:"buy"`
	Last   string `json:"last"`
	Vol    string `json:"volume"`
	Symbol string `json:"symbol"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Change string `json:"changePercentage"`
}

func GetAllTickerFromOkex() (tickers []Ticker, err error) {
	tickers = make([]Ticker, 0)
	resp, err := http.Get("https://www.okex.com/v2/markets/tickers")
	if err != nil {
		return tickers, err
	}
	defer func() {
		if nil != resp && nil != resp.Body {
			resp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return tickers, err
	} else {
		var okexOutTicker *OkexFullTicker
		if err := json.Unmarshal(body, &okexOutTicker); nil != err {
			return tickers, err
		} else {

			for _, v := range okexOutTicker.Data {
				ticker := Ticker{}
				okexMarket := strings.Replace(v.Symbol, "_", "-", 1)
				okexMarket = strings.ToUpper(okexMarket)
				okexMarket = strings.Replace(okexMarket, "ETH", "WETH", 1)
				if stringInSlice(okexMarket, supportedMarkets) {
					ticker.Market = okexMarket
					ticker.Last, _ = strconv.ParseFloat(v.Last, 64)
					ticker.Change = v.Change
					ticker.Exchange = okex
					ticker.Amount, _ = strconv.ParseFloat(v.Vol, 64)
					ticker.Vol = ticker.Amount * ticker.Last
					ticker.High, _ = strconv.ParseFloat(v.High, 64)
					ticker.Low, _ = strconv.ParseFloat(v.Low, 64)
					tickers = append(tickers, ticker)
				}
			}

			return tickers, nil
		}
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
