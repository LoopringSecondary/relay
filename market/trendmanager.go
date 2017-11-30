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
	"errors"
	"fmt"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"log"
	"sort"
	"sync"
	"time"
)

const (
	OneHour = "1Hr"
	//TwoHour = "2Hr"
	//OneDay = "1Day"
)

type Ticker struct {
	Market   string
	Interval string
	Amount   float64
	Vol      float64
	Open     float64
	Close    float64
	High     float64
	Low      float64
	Last     float64
	Change   string
}

type Cache struct {
	Trends []Trend
	Fills  []dao.FillEvent
}

type Trend struct {
	Interval   string
	Market     string
	Vol        float64
	Amount     float64
	CreateTime int64
	Open       float64
	Close      float64
	High       float64
	Low        float64
	Start      int64
	End        int64
}

type TrendManager struct {
	c             *cache.Cache
	IsTickerReady bool
	IsTrendReady  bool
	cacheReady    bool
	rds           dao.RdsService
	cron          *cron.Cron
}

var once sync.Once
var trendManager TrendManager

const trendKey = "market_ticker"
const tickerKey = "market_ticker_view"

func NewTrendManager(dao dao.RdsService) TrendManager {

	once.Do(func() {
		trendManager = TrendManager{rds: dao, cron: cron.New()}
		trendManager.c = cache.New(cache.NoExpiration, cache.NoExpiration)
		trendManager.initCache()
		fillOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: trendManager.handleOrderFilled}
		eventemitter.On(eventemitter.OrderManagerExtractorFill, fillOrderWatcher)
		//trendManager.startScheduleUpdate()
	})

	return trendManager
}

// ======> init cache steps
// step.1 init all market
// step.2 get all trend record into cache
// step.3 get all order fillFilled into cache
// step.4 calculate 24hr ticker
// step.5 send channel cache ready
// step.6 start schedule update

func (t *TrendManager) initCache() {

	trendMap := make(map[string]Cache)
	tickerMap := make(map[string]Ticker)
	for _, mkt := range util.AllMarkets {
		mktCache := Cache{}
		mktCache.Trends = make([]Trend, 0)
		mktCache.Fills = make([]dao.FillEvent, 0)

		// default 100 records load first time
		trends, err := t.rds.TrendPageQuery(dao.Trend{Market: mkt}, 1, 100)

		if err != nil {
			log.Fatal(err)
		}

		for _, trend := range trends.Data {
			mktCache.Trends = append(mktCache.Trends, ConvertUp(trend.(dao.Trend)))
		}

		tokenS, tokenB := util.UnWrap(mkt)
		now := time.Now()
		firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
		sbFills, err := t.rds.QueryRecentFills(tokenS, tokenB, "", firstSecondThisHour.Unix(), 0)
		if err != nil {
			log.Fatal(err)
		}

		bsFills, err := t.rds.QueryRecentFills(tokenB, tokenS, "", firstSecondThisHour.Unix(), 0)

		if err != nil {
			log.Fatal(err)
		}

		for _, f := range sbFills {
			mktCache.Fills = append(mktCache.Fills, f)
		}
		for _, f := range bsFills {
			mktCache.Fills = append(mktCache.Fills, f)
		}

		trendMap[mkt] = mktCache

		ticker := calculateTicker(mkt, mktCache.Fills, mktCache.Trends, firstSecondThisHour)
		tickerMap[mkt] = ticker
	}
	t.c.Set(trendKey, trendMap, cache.NoExpiration)
	t.c.Set(tickerKey, tickerMap, cache.NoExpiration)

	t.cacheReady = true
	t.startScheduleUpdate()

}

func calculateTicker(market string, fills []dao.FillEvent, trends []Trend, now time.Time) Ticker {

	var result = Ticker{Market: market}

	before24Hour := now.Unix() - 24*60*60

	var (
		high   float64
		low    float64
		vol    float64
		amount float64
	)

	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Start < trends[j].Start
	})

	for _, data := range trends {

		if data.Start > before24Hour {
			continue
		}

		vol += data.Vol
		amount += data.Amount
		if high == 0 || high < data.High {
			high = data.High
		}
		if low == 0 || low < data.Low {
			low = data.Low
		}
	}

	for i, data := range fills {

		if util.IsBuy(data.TokenS) {
			vol += util.ByteToFloat(data.AmountB)
			amount += util.ByteToFloat(data.AmountS)
		} else {
			vol += util.ByteToFloat(data.AmountS)
			amount += util.ByteToFloat(data.AmountB)
		}

		price := util.CalculatePrice(data.AmountS, data.AmountB, data.TokenS, data.TokenB)

		if i == len(fills)-1 {
			result.Last = price
		}

		if high == 0 || high < price {
			high = price
		}
		if low == 0 || low < price {
			low = price
		}
	}

	result.High = high
	result.Low = low

	result.Open = trends[0].Open
	result.Close = trends[len(trends)-1].Close
	result.Change = fmt.Sprintf("%.2f%%", 100*result.Last/result.Open)

	result.Vol = vol
	result.Amount = amount
	return result
}

func (t *TrendManager) startScheduleUpdate() {
	t.cron.AddFunc("10 * * * * *", t.insertTrend)
	t.cron.Start()
}

func (t *TrendManager) insertTrend() {
	// get latest 24 hour trend if not exist generate

	for _, mkt := range util.AllMarkets {
		now := time.Now()
		firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)

		for i := 1; i < 10; i++ {

			start := firstSecondThisHour.Unix() - int64(i*60*60)
			end := firstSecondThisHour.Unix() - int64((i-1)*60*60)

			trends, err := t.rds.TrendQueryByTime(mkt, start, end)
			if err != nil {
				log.Println("query trend err", err)
				return
			}

			tokenS, tokenB := util.UnWrap(mkt)
			if trends == nil || len(trends) == 0 {
				fills, _ := t.rds.QueryRecentFills(tokenS, tokenB, "", start, end)

				toInsert := dao.Trend{
					Interval:   OneHour,
					Market:     mkt,
					CreateTime: time.Now().Unix(),
					Start:      start,
					End:        end}

				var (
					high   float64
					low    float64
					vol    float64
					amount float64
				)

				sort.Slice(fills, func(i, j int) bool {
					return fills[i].CreateTime < fills[j].CreateTime
				})

				for _, data := range fills {

					if util.IsBuy(data.TokenS) {
						vol += util.ByteToFloat(data.AmountB)
						amount += util.ByteToFloat(data.AmountS)
					} else {
						vol += util.ByteToFloat(data.AmountS)
						amount += util.ByteToFloat(data.AmountB)
					}

					price := util.CalculatePrice(data.AmountS, data.AmountB, data.TokenS, data.TokenB)

					if high == 0 || high < price {
						high = price
					}
					if low == 0 || low < price {
						low = price
					}
				}

				toInsert.High = high
				toInsert.Low = low

				openFill := fills[0]
				toInsert.Open = util.CalculatePrice(openFill.AmountS, openFill.AmountB, openFill.TokenS, openFill.TokenB)
				closeFill := fills[len(fills)-1]
				toInsert.Close = util.CalculatePrice(closeFill.AmountS, closeFill.AmountB, closeFill.TokenS, closeFill.TokenB)

				toInsert.Vol = vol
				toInsert.Amount = amount

				if err := t.rds.Add(toInsert); err != nil {
					fmt.Println(err)
				}
			}
		}

	}
}

func (t *TrendManager) GetTrends(market string) (trends []Trend, err error) {

	if t.cacheReady {
		if trendCache, ok := t.c.Get(trendKey); !ok {
			err = errors.New("can't found trends by key : " + trendKey)
		} else {
			trends = trendCache.(map[string][]Trend)[market]
		}
	} else {
		err = errors.New("cache is not ready , please access later")
	}
	return
}

func (t *TrendManager) GetTicker() (tickers []Ticker, err error) {

	if t.cacheReady {
		if tickerInCache, ok := t.c.Get(tickerKey); ok {
			tickerMap := tickerInCache.(map[string]Ticker)
			tickers = make([]Ticker, len(tickerMap))
			for _, v := range tickerMap {
				tickers = append(tickers, v)
			}
		} else {
			err = errors.New("get ticker from cache error, no value found")
		}
	} else {
		err = errors.New("cache is not ready , please access later")
	}
	return
}

func (t *TrendManager) handleOrderFilled(input eventemitter.EventData) (err error) {
	if t.cacheReady {

		event := input.(*types.OrderFilledEvent)
		newFillModel := &dao.FillEvent{}
		if err = newFillModel.ConvertDown(event); err != nil {
			return
		}

		market, wrapErr := util.WrapMarketByAddress(newFillModel.TokenS, newFillModel.TokenB)

		if wrapErr != nil {
			err = wrapErr
			return
		}

		if tickerInCache, ok := t.c.Get(trendKey); ok {
			trendMap := tickerInCache.(map[string]Cache)
			trendMap[market].Fills[len(trendMap[market].Fills)] = *newFillModel
		} else {
			fills := make([]dao.FillEvent, 0)
			fills = append(fills, *newFillModel)
			newCache := Cache{make([]Trend, 0), fills}
			t.c.Set(trendKey, newCache, cache.NoExpiration)
			t.reCalTicker(market)
		}
	} else {
		err = errors.New("cache is not ready , please access later")
	}

	return
}

func (t *TrendManager) reCalTicker(market string) {
	trendInCache, _ := t.c.Get(trendKey)
	mktCache := trendInCache.(map[string]Cache)[market]
	now := time.Now()
	firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
	ticker := calculateTicker(market, mktCache.Fills, mktCache.Trends, firstSecondThisHour)
	tickerInCache, _ := t.c.Get(tickerKey)
	tickerMap := tickerInCache.(map[string]Ticker)
	tickerMap[market] = ticker
}

func ConvertUp(src dao.Trend) Trend {

	return Trend{
		Interval:   src.Interval,
		Market:     src.Market,
		Vol:        src.Vol,
		Amount:     src.Amount,
		CreateTime: src.CreateTime,
		Open:       src.Open,
		Close:      src.Close,
		High:       src.High,
		Low:        src.Low,
		Start:      src.Start,
		End:        src.End,
	}
}
