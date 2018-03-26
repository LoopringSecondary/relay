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
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	OneHour  = "1Hr"
	TwoHour  = "2Hr"
	FourHour = "4Hr"
	OneDay   = "1Day"
	OneWeek  = "1Week"
	//OneMonth = "1Month"
	//OneYear = "1Year"

	//TwoHour = "2Hr"
	//OneDay = "1Day"

	tsOneHour  = 60 * 60
	tsTwoHour  = 2 * tsOneHour
	tsFourHour = 4 * tsOneHour
	tsOneDay   = 24 * tsOneHour
	tsOneWeek  = 7 * tsOneDay
)

var allInterval = []string{OneHour, TwoHour, FourHour, OneDay, OneWeek}

//var allInterval = []string{OneHour, TwoHour}

type Ticker struct {
	Market    string  `json:"market"`
	Exchange  string  `json:"exchange"`
	Intervals string  `json:"interval"`
	Amount    float64 `json:"amount"`
	Vol       float64 `json:"vol"`
	Open      float64 `json:"open"`
	Close     float64 `json:"close"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Last      float64 `json:"last"`
	Buy       string  `json:"buy"`
	Sell      string  `json:"sell"`
	Change    string  `json:"change"`
}

type Cache struct {
	Trends []Trend
	Fills  []dao.FillEvent
}

type Trend struct {
	Intervals  string  `json:"intervals"`
	Market     string  `json:"market"`
	Vol        float64 `json:"vol"`
	Amount     float64 `json:"amount"`
	CreateTime int64   `json:"createTime"`
	Open       float64 `json:"open"`
	Close      float64 `json:"close"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	Start      int64   `json:"start"`
	End        int64   `json:"end"`
}

type TrendManager struct {
	c          *cache.Cache
	cacheReady bool
	rds        dao.RdsService
	cron       *cron.Cron
}

var once sync.Once
var trendManager TrendManager

const trendKeyPre = "market_trend_"
const tickerKey = "market_ticker_view"

func NewTrendManager(dao dao.RdsService) TrendManager {

	once.Do(func() {
		trendManager = TrendManager{rds: dao, cron: cron.New()}
		trendManager.c = cache.New(cache.NoExpiration, cache.NoExpiration)
		trendManager.ProofRead()
		trendManager.LoadCache()
		trendManager.startScheduleUpdate()
		fillOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: trendManager.handleOrderFilled}
		eventemitter.On(eventemitter.OrderManagerExtractorFill, fillOrderWatcher)

	})

	return trendManager
}

func (t *TrendManager) ProofRead() {
	checkPoint, err := t.rds.QueryCheckPointByType(dao.TrendUpdateType)
	if err != nil {
		log.Fatal("trend manager check point get failed, " + err.Error())
		return
	}

	for _, mkt := range util.AllMarkets {
		copyOfMkt := mkt
		go func(market string) {
			for _, interval := range allInterval {
				err := t.proofByInterval(market, interval, checkPoint.CheckPoint)
				if err != nil {
					log.Fatalf("proof by interval error occurs, %s, %s, %d ", err.Error(), interval, checkPoint.CheckPoint)
				}
			}
		}(copyOfMkt)
	}

	toUpdateCheckPoint := &dao.CheckPoint{}
	toUpdateCheckPoint.ID = checkPoint.ID
	toUpdateCheckPoint.CheckPoint = time.Now().Unix()
	toUpdateCheckPoint.BusinessType = checkPoint.BusinessType
	toUpdateCheckPoint.CreateTime = checkPoint.CreateTime
	toUpdateCheckPoint.ModifyTime = time.Now().Unix()
	err = t.rds.Save(toUpdateCheckPoint)
	if err != nil {
		log.Fatal("check point update error, " + err.Error())
	}
}

func (t *TrendManager) proofByInterval(mkt string, interval string, checkPoint int64) error {

	now := time.Now().Unix()
	//for ;

	starts := make([]int64, 0)
	tsInterval := getTsInterval(interval)

	firstStart := (checkPoint/tsInterval)*tsInterval + 1

	for firstStart < now {
		starts = append(starts, firstStart)
		firstStart = firstStart + tsInterval
	}

	trends, err := t.rds.TrendQueryForProof(mkt, interval, checkPoint)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	trendMap := make(map[int64]dao.Trend)
	for _, v := range trends {
		trendMap[v.Start] = v
	}

	for _, start := range starts {
		_, ok := trendMap[start]
		if !ok {
			if interval == OneHour {
				err = t.insertMinIntervalTrend(OneHour, start, mkt)
			} else {
				err = t.insertByTrendV2(interval, start, mkt)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ======> init cache steps
// step.1 init all market
// step.2 get all trend record into cache
// step.3 get all order fillFilled into cache
// step.4 calculate 24hr ticker
// step.5 send channel cache ready
// step.6 start schedule update

func (t *TrendManager) refreshCacheByInterval(interval string) {
	interval = strings.ToLower(interval)
	log.Println("start refresh cache by interval " + interval)

	trendMap := make(map[string]Cache)
	for _, mkt := range util.AllMarkets {
		mktCache := Cache{}
		mktCache.Trends = make([]Trend, 0)

		// default 100 records load first time
		trends, err := t.rds.TrendQueryLatest(dao.Trend{Market: mkt, Intervals: interval}, 1, 100)

		if err != nil {
			log.Println(err)
			return
		}

		for _, trend := range trends {
			mktCache.Trends = append(mktCache.Trends, ConvertUp(trend))
		}
		trendMap[mkt] = mktCache
	}
	t.c.Set(trendKeyPre+interval, trendMap, cache.NoExpiration)
}

func (t *TrendManager) LoadCache() {
	t.refreshMinIntervalCache()
	intervals := append(allInterval[:0], allInterval[1:]...)
	for _, i := range intervals {
		t.refreshCacheByInterval(i)
	}
	t.cacheReady = true
}

func (t *TrendManager) refreshMinIntervalCache() {

	log.Println("start refresh 1hr cache......")

	trendMap := make(map[string]Cache)
	tickerMap := make(map[string]Ticker)
	for _, mkt := range util.AllMarkets {
		mktCache := Cache{}
		mktCache.Trends = make([]Trend, 0)
		mktCache.Fills = make([]dao.FillEvent, 0)

		// default 100 records load first time
		trends, err := t.rds.TrendQueryLatest(dao.Trend{Market: mkt, Intervals: OneHour}, 1, 100)

		if err != nil {
			log.Println(err)
			return
		}

		for _, trend := range trends {
			mktCache.Trends = append(mktCache.Trends, ConvertUp(trend))
		}

		now := time.Now()
		firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 1, 0, now.Location())
		fills, err := t.rds.QueryRecentFills(mkt, "", firstSecondThisHour.Unix(), 0)
		if err != nil {
			log.Println(err)
			return
		}

		for _, f := range fills {
			mktCache.Fills = append(mktCache.Fills, f)
		}

		trendMap[mkt] = mktCache

		fmt.Println("add trend data ")

		ticker := calculateTicker(mkt, fills, mktCache.Trends, firstSecondThisHour)
		tickerMap[mkt] = ticker
	}

	t.c.Set(trendKeyPre+strings.ToLower(OneHour), trendMap, cache.NoExpiration)
	t.c.Set(tickerKey, tickerMap, cache.NoExpiration)

}

func calculateTicker(market string, fills []dao.FillEvent, trends []Trend, now time.Time) Ticker {

	var result = Ticker{Market: market}

	if len(fills) == 0 && len(trends) == 0 {
		return result
	}

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

		if data.Start < before24Hour {
			continue
		}

		vol += data.Vol
		amount += data.Amount

		if result.Open == 0 && data.Open != 0 {
			result.Open = data.Open
		}

		if high == 0 || high < data.High {
			high = data.High
		}
		if low == 0 || (low > data.Low && data.Low != 0) {
			low = data.Low
		}

		if data.Close != 0 {
			result.Last = data.Close
			result.Close = data.Close
		}
	}

	for _, data := range fills {

		if util.IsBuy(data.TokenS) {
			vol += util.StringToFloat(data.AmountB)
			amount += util.StringToFloat(data.AmountS)
		} else {
			vol += util.StringToFloat(data.AmountS)
			amount += util.StringToFloat(data.AmountB)
		}

		price := util.CalculatePrice(data.AmountS, data.AmountB, data.TokenS, data.TokenB)

		if result.Open == 0 && price != 0 {
			result.Open = price
		}

		if price != 0 {
			result.Last = price
			result.Close = price
		}

		if high == 0 || high < price {
			high = price
		}
		if low == 0 || (low > price && price != 0) {
			low = price
		}
	}

	result.High = high
	result.Low = low

	if result.Open > 0 && result.Last > 0 {
		result.Change = fmt.Sprintf("%.2f%%", 100*(result.Last-result.Open)/result.Open)
	}

	result.Vol = vol
	result.Amount = amount
	return result
}

func (t *TrendManager) startScheduleUpdate() {
	t.cron.AddFunc("10 1 * * * *", t.ScheduleUpdate)
	t.cron.Start()
}

func (t *TrendManager) insertTrendByInterval(interval string) error {
	if !isTimeToInsert(interval) {
		log.Println("no need to insert trend by interval " + interval)
		return nil
	}

	if interval == OneHour {
		//t.InsertTrend()
		return nil
	} else {
		return t.insertByTrend(interval)
	}
}

func (t *TrendManager) insertByTrend(interval string) error {

	now := time.Now()
	end := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	tsInterval := getTsInterval(interval) + 1
	start := end.Unix() - tsInterval
	//multiple := tsInterval / tsOneHour

	for _, mkt := range util.AllMarkets {

		trends, err := t.rds.TrendQueryByTime(OneHour, mkt, start, end.Unix())

		if err != nil {
			return err
		}

		toInsert := &dao.Trend{}

		var (
			vol    float64 = 0
			amount float64 = 0
			high   float64 = 0
			low    float64 = 0
		)

		for _, t := range trends {
			vol += t.Vol
			amount += t.Amount
			if low == 0 || low > t.Low {
				low = t.Low
			}
			if high == 0 || high < t.High {
				high = t.High
			}
		}

		if len(trends) == 0 {
			toInsert.Open = 0
			toInsert.Close = 0
		} else {
			toInsert.Open = trends[0].Open
			toInsert.Close = trends[len(trends)-1].Close
		}
		toInsert.Vol = vol
		toInsert.Amount = amount
		toInsert.High = high
		toInsert.Low = low
		toInsert.Start = start
		toInsert.End = end.Unix()
		toInsert.Market = mkt
		toInsert.Intervals = interval

		if err := t.rds.Add(toInsert); err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

func (t *TrendManager) insertMinIntervalTrend(interval string, start int64, mkt string) (err error) {

	end := start + getTsInterval(interval) - 1

	trends, _ := t.rds.TrendQueryByTime(interval, mkt, start, end)
	if len(trends) > 0 {
		log.Println("current interval trend exsit")
		return
	}

	lastTrends, _ := t.rds.TrendQueryByTime(interval, mkt, start-getTsInterval(interval), end-getTsInterval(interval))
	if len(lastTrends) > 1 {
		log.Println("found more than one last trend!")
		return errors.New("found more than one last trend")
	} else if len(lastTrends) == 0 {
		log.Println("not found last trend!")
	}

	if trends == nil || len(trends) == 0 {
		fills, fillsErr := t.rds.QueryRecentFills(mkt, "", start, end)

		if fillsErr != nil {
			return fillsErr
		}

		toInsert := &dao.Trend{
			Intervals:  OneHour,
			Market:     mkt,
			CreateTime: time.Now().Unix(),
			Start:      start,
			End:        end}

		var (
			vol    float64
			amount float64
			//open   float64
			//low    float64
		)

		if len(lastTrends) == 0 {
			toInsert.Open = 0
			toInsert.Close = 0
			toInsert.High = 0
			toInsert.Low = 0
		} else {
			toInsert.Open = lastTrends[0].Close
			toInsert.Close = lastTrends[0].Close
			toInsert.High = lastTrends[0].Close
			toInsert.Low = lastTrends[0].Close
		}

		sort.Slice(fills, func(i, j int) bool {
			return fills[i].CreateTime < fills[j].CreateTime
		})

		for _, data := range fills {

			if util.IsBuy(data.TokenS) {
				vol += util.StringToFloat(data.AmountB)
				amount += util.StringToFloat(data.AmountS)
			} else {
				vol += util.StringToFloat(data.AmountS)
				amount += util.StringToFloat(data.AmountB)
			}

			price := util.CalculatePrice(data.AmountS, data.AmountB, data.TokenS, data.TokenB)

			if toInsert.Open == 0 && price != 0 {
				toInsert.Open = price
			}

			if toInsert.High == 0 || toInsert.High < price {
				toInsert.High = price
			}
			if toInsert.Low == 0 || (toInsert.Low > price && price > 0) {
				toInsert.Low = price
			}
			if price > 0 {
				toInsert.Close = price
			}
		}

		toInsert.Vol = vol
		toInsert.Amount = amount

		if err := t.rds.Add(toInsert); err != nil {
			log.Println(err.Error())
			return err
		}
	}
	return nil
}

func (t *TrendManager) insertByTrendV2(interval string, start int64, mkt string) error {

	end := start + getTsInterval(interval) - 1

	exists, _ := t.rds.TrendQueryByTime(interval, mkt, start, end)
	if len(exists) > 0 {
		log.Println("current interval trend exsit")
		return nil
	}

	trends, err := t.rds.TrendQueryByInterval(OneHour, mkt, start, end)

	if err != nil {
		return err
	}

	toInsert := &dao.Trend{}

	var (
		vol    float64 = 0
		amount float64 = 0
		high   float64 = 0
		low    float64 = 0
	)

	for _, t := range trends {
		vol += t.Vol
		amount += t.Amount
		if low == 0 || low > t.Low {
			low = t.Low
		}
		if high == 0 || high < t.High {
			high = t.High
		}
	}

	if len(trends) == 0 {
		toInsert.Open = 0
		toInsert.Close = 0
	} else {
		toInsert.Open = trends[0].Open
		toInsert.Close = trends[len(trends)-1].Close
	}
	toInsert.Vol = vol
	toInsert.Amount = amount
	toInsert.High = high
	toInsert.Low = low
	toInsert.Start = start
	toInsert.End = end
	toInsert.Market = mkt
	toInsert.Intervals = interval
	toInsert.CreateTime = time.Now().Unix()

	if err := t.rds.Add(toInsert); err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func getTsInterval(interval string) int64 {
	switch interval {
	case OneHour:
		return tsOneHour
	case TwoHour:
		return tsTwoHour
	case FourHour:
		return tsFourHour
	case OneDay:
		return tsOneDay
	case OneWeek:
		return tsOneWeek
	default:
		return 0
	}
}

func isTimeToInsert(interval string) bool {
	return time.Now().Unix()%getTsInterval(interval) < tsOneHour
}

func (t *TrendManager) ScheduleUpdate() {
	// get latest 24 hour trend if not exist generate

	fmt.Println("start insert trend cron job")

	var wg sync.WaitGroup

	for _, mkt := range util.AllMarkets {
		now := time.Now()
		firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 1, 0, now.Location())

		wg.Add(1)
		go func(tmpMkt string) {
			for i := 24; i >= 1; i-- {

				start := firstSecondThisHour.Unix() - int64(i*60*60)
				end := firstSecondThisHour.Unix() - int64((i-1)*60*60) - 1

				trends, _ := t.rds.TrendQueryByTime(OneHour, tmpMkt, start, end)
				if len(trends) > 0 {
					log.Println("current interval trend exsit")
					wg.Done()
					return
				}

				lastTrends, _ := t.rds.TrendQueryByTime(OneHour, tmpMkt, start-int64(60*60), end-int64(60*60))
				if len(lastTrends) > 1 {
					log.Println("found more than one last trend!")
					wg.Done()
					return
				} else if len(lastTrends) == 0 {
					log.Println("not found last trend!")
				}

				if trends == nil || len(trends) == 0 {
					fills, fillsErr := t.rds.QueryRecentFills(tmpMkt, "", start, end)

					if fillsErr != nil {
						continue
					}

					toInsert := &dao.Trend{
						Intervals:  OneHour,
						Market:     tmpMkt,
						CreateTime: time.Now().Unix(),
						Start:      start,
						End:        end}

					var (
						vol    float64
						amount float64
						open   float64
						low    float64
					)

					if len(lastTrends) == 0 {
						toInsert.Open = 0
						toInsert.Close = 0
						toInsert.High = 0
						toInsert.Low = 0
					} else {
						toInsert.Open = lastTrends[0].Close
						toInsert.Close = lastTrends[0].Close
						toInsert.High = lastTrends[0].Close
						toInsert.Low = lastTrends[0].Close
					}

					sort.Slice(fills, func(i, j int) bool {
						return fills[i].CreateTime < fills[j].CreateTime
					})

					for _, data := range fills {

						if util.IsBuy(data.TokenS) {
							vol += util.StringToFloat(data.AmountB)
							amount += util.StringToFloat(data.AmountS)
						} else {
							vol += util.StringToFloat(data.AmountS)
							amount += util.StringToFloat(data.AmountB)
						}

						price := util.CalculatePrice(data.AmountS, data.AmountB, data.TokenS, data.TokenB)

						if open == 0 && price != 0 {
							open = price
						}

						if toInsert.High == 0 || toInsert.High < price {
							toInsert.High = price
						}
						if low == 0 || low > price {
							low = price
						}
						toInsert.Close = price
					}

					if toInsert.Open != 0 && open != 0 {
						toInsert.Open = open
					}

					toInsert.Low = low
					toInsert.Vol = vol
					toInsert.Amount = amount

					if err := t.rds.Add(toInsert); err != nil {
						fmt.Println(err)
					}
				}
			}
			wg.Done()
		}(mkt)
	}
	wg.Wait()
	var wgInterval sync.WaitGroup
	intervals := append(allInterval[:0], allInterval[1:]...)
	for _, i := range intervals {
		wgInterval.Add(1)
		go t.insertTrendByInterval(i)
		wgInterval.Done()
	}
	wgInterval.Wait()
	//t.wupdateCache()
}

func (t *TrendManager) aggregate(fills []dao.FillEvent, trends []Trend) (trend Trend, err error) {

	if len(fills) == 0 && len(trends) == 0 {
		err = errors.New("fills can't be nil")
		return
	}

	now := time.Now()
	firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 1, 0, now.Location())
	lastSecondThisHour := firstSecondThisHour.Unix() + 60*60 - 1

	if len(fills) == 0 {
		lastTrend := trends[len(trends)-1]
		return Trend{
			Intervals:  OneHour,
			Market:     lastTrend.Market,
			CreateTime: time.Now().Unix(),
			Start:      firstSecondThisHour.Unix(),
			End:        lastSecondThisHour,
			High:       lastTrend.High,
			Low:        lastTrend.Low,
			Vol:        0,
			Amount:     0,
			Open:       lastTrend.Open,
			Close:      lastTrend.Close,
		}, nil
	}

	trend = Trend{
		Intervals:  OneHour,
		Market:     fills[0].Market,
		CreateTime: time.Now().Unix(),
		Start:      firstSecondThisHour.Unix(),
		End:        lastSecondThisHour,
	}

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
			vol += util.StringToFloat(data.AmountB)
			amount += util.StringToFloat(data.AmountS)
		} else {
			vol += util.StringToFloat(data.AmountS)
			amount += util.StringToFloat(data.AmountB)
		}

		price := util.CalculatePrice(data.AmountS, data.AmountB, data.TokenS, data.TokenB)

		if high == 0 || high < price {
			high = price
		}
		if low == 0 || low > price {
			low = price
		}
	}

	trend.High = high
	trend.Low = low

	openFill := fills[0]
	trend.Open = util.CalculatePrice(openFill.AmountS, openFill.AmountB, openFill.TokenS, openFill.TokenB)
	closeFill := fills[len(fills)-1]
	trend.Close = util.CalculatePrice(closeFill.AmountS, closeFill.AmountB, closeFill.TokenS, closeFill.TokenB)

	trend.Vol = vol
	trend.Amount = amount

	return
}

func (t *TrendManager) GetTrends(market, interval string) (trends []Trend, err error) {

	interval = strings.ToLower(interval)
	market = strings.ToUpper(market)

	if t.cacheReady {
		fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>" + trendKeyPre + interval)
		if trendCache, ok := t.c.Get(trendKeyPre + interval); !ok {
			err = errors.New("can't found trends by key : " + interval)
		} else {
			tc := trendCache.(map[string]Cache)[market]
			trends = make([]Trend, 0)
			if strings.ToLower(interval) == "1hr" {
				trendInFills, aggErr := t.aggregate(tc.Fills, tc.Trends)
				if aggErr == nil {
					trends = append(trends, trendInFills)
				}
			}
			for _, t := range tc.Trends {
				trends = append(trends, t)
			}

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
			tickers = make([]Ticker, 0)
			for _, v := range tickerMap {
				v.Buy = strconv.FormatFloat(v.Last, 'f', -1, 64)
				v.Sell = strconv.FormatFloat(v.Last, 'f', -1, 64)
				tickers = append(tickers, v)
			}
		} else {
			err = errors.New("get ticker from cache error, no value found")
		}
	} else {
		err = errors.New("cache is not ready , please access later")
	}

	if len(tickers) == 0 {
		return tickers, nil
	}

	tickerMap := make(map[string]Ticker)
	markets := make([]string, 0)
	for _, t := range tickers {
		tickerMap[t.Market] = t
		markets = append(markets, t.Market)
	}

	sort.Strings(markets)
	result := make([]Ticker, 0)

	for _, m := range markets {
		result = append(result, tickerMap[m])
	}
	return result, nil
}

func (t *TrendManager) GetTickerByMarket(mkt string) (ticker Ticker, err error) {

	if t.cacheReady {
		if tickerInCache, ok := t.c.Get(tickerKey); ok {
			tickerMap := tickerInCache.(map[string]Ticker)
			for k, v := range tickerMap {
				if k == mkt {
					v.Buy = strconv.FormatFloat(v.Last, 'f', -1, 64)
					v.Sell = strconv.FormatFloat(v.Last, 'f', -1, 64)
					return v, err
				}
			}
		} else {
			err = errors.New("get ticker from cache error, no value found")
		}
	} else {
		err = errors.New("cache is not ready , please access later")
	}
	return
}

func ConvertUp(src dao.Trend) Trend {

	return Trend{
		Intervals:  src.Intervals,
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

		if tickerInCache, ok := t.c.Get(trendKeyPre + strings.ToLower(OneHour)); ok {
			trendMap := tickerInCache.(map[string]Cache)
			tc := trendMap[market]
			tc.Fills = append(tc.Fills, *newFillModel)
			trendMap[market] = tc
			t.c.Set(trendKeyPre+strings.ToLower(OneHour), trendMap, cache.NoExpiration)
			t.reCalTicker(market)
		} else {
			fills := make([]dao.FillEvent, 0)
			fills = append(fills, *newFillModel)
			newCache := Cache{make([]Trend, 0), fills}
			t.c.Set(trendKeyPre+strings.ToLower(OneHour), newCache, cache.NoExpiration)
			t.reCalTicker(market)
		}
	} else {
		err = errors.New("cache is not ready , please access later")
	}

	return
}

func (t *TrendManager) reCalTicker(market string) {
	trendInCache, _ := t.c.Get(trendKeyPre + strings.ToLower(OneHour))
	mktCache := trendInCache.(map[string]Cache)[market]
	now := time.Now()
	firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 1, 0, now.Location())
	ticker := calculateTicker(market, mktCache.Fills, mktCache.Trends, firstSecondThisHour)
	tickerInCache, _ := t.c.Get(tickerKey)
	tickerMap := tickerInCache.(map[string]Ticker)
	tickerMap[market] = ticker
	t.c.Set(tickerKey, tickerMap, cache.NoExpiration)
}
