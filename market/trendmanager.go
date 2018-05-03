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
	"errors"
	"fmt"
	redisCache "github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	gocache "github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
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

	tsOneHour        = 60 * 60
	tsTwoHour        = 2 * tsOneHour
	tsFourHour       = 4 * tsOneHour
	tsOneDay         = 24 * tsOneHour
	tsOneWeek        = 7 * tsOneDay
	localCacheTicker = "LocalCacheTicker"
)

var allInterval = []string{OneHour, TwoHour, FourHour, OneDay, OneWeek}

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
	Buy       float64 `json:"buy"`
	Sell      float64 `json:"sell"`
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
	cacheReady  bool
	proofReady  bool
	rds         dao.RdsService
	cron        *cron.Cron
	cronJobLock bool
	localCache  *gocache.Cache
}

var once sync.Once
var trendManager TrendManager

const trendKeyPre = "market_trend_"
const tickerKey = "lpr_ticker_view_"

func NewTrendManager(dao dao.RdsService, cronJobLock bool) TrendManager {

	once.Do(func() {
		trendManager = TrendManager{rds: dao, cron: cron.New(), cronJobLock: cronJobLock}
		trendManager.localCache = gocache.New(5*time.Second, 5*time.Minute)
		trendManager.LoadCache()
		if cronJobLock {
			trendManager.startScheduleUpdate()
		}
		fillOrderWatcher := &eventemitter.Watcher{Concurrent: false, Handle: trendManager.HandleOrderFilled}
		eventemitter.On(eventemitter.OrderFilled, fillOrderWatcher)

	})

	return trendManager
}

func (t *TrendManager) ProofRead() {
	log.Info(">>>>>>>>>>>>> start proof read cron job")
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

	now := time.Now()
	//for ;

	starts := make([]int64, 0)
	tsInterval := getTsInterval(interval)

	firstStart := (checkPoint/tsInterval)*tsInterval + 1
	firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 1, 0, now.Location())

	for firstStart < firstSecondThisHour.Unix() {
		starts = append(starts, firstStart)
		firstStart = firstStart + tsInterval
	}

	trends, err := t.rds.TrendQueryForProof(mkt, interval, checkPoint)
	if err != nil {
		log.Info(err.Error())
		return err
	}

	trendMap := make(map[int64]dao.Trend)
	for _, v := range trends {
		trendMap[v.Start] = v
	}

	for _, start := range starts {
		if interval == OneHour {
			err = t.insertMinIntervalTrend(OneHour, start, mkt)
		} else {
			err = t.insertByTrendV2(interval, start, mkt)
		}
		if err != nil {
			return err
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
	log.Info("start refresh cache by interval " + interval)

	//trendMap := make(map[string]Cache)
	for _, mkt := range util.AllMarkets {
		mktCache := Cache{}
		mktCache.Trends = make([]Trend, 0)

		// default 100 records load first time
		trends, err := t.rds.TrendQueryLatest(dao.Trend{Market: mkt, Intervals: interval}, 1, 100)

		if err != nil {
			log.Info(err.Error())
			return
		}

		for _, trend := range trends {
			mktCache.Trends = append(mktCache.Trends, ConvertUp(trend))
		}
		//trendMap[mkt] = mktCache
		setTrendCache(interval, mkt, mktCache, 0)
	}
	//t.c.Set(trendKeyPre+interval, trendMap, cache.NoExpiration)
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

	log.Info("start refresh 1hr cache......")

	//trendMap := make(map[string]Cache)
	tickerMap := make(map[string]Ticker)
	for _, mkt := range util.AllMarkets {
		mktCache := Cache{}
		mktCache.Trends = make([]Trend, 0)
		mktCache.Fills = make([]dao.FillEvent, 0)

		// default 100 records load first time
		trends, err := t.rds.TrendQueryLatest(dao.Trend{Market: mkt, Intervals: OneHour}, 1, 100)

		if err != nil {
			log.Info(err.Error())
			return
		}

		for _, trend := range trends {
			mktCache.Trends = append(mktCache.Trends, ConvertUp(trend))
		}

		now := time.Now()
		firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 1, 0, now.Location())
		fills, err := t.rds.QueryRecentFills(mkt, "", firstSecondThisHour.Unix(), 0)
		if err != nil {
			log.Info(err.Error())
			return
		}

		for _, f := range fills {
			mktCache.Fills = append(mktCache.Fills, f)
		}

		//trendMap[mkt] = mktCache

		ticker := calculateTicker(mkt, fills, mktCache.Trends, firstSecondThisHour)

		tickerMap[mkt] = ticker

		setTrendCache(OneHour, mkt, mktCache, 0)
		setLprTickerCache(tickerMap, 0)
	}

	//t.c.Set(trendKeyPre+strings.ToLower(OneHour), trendMap, cache.NoExpiration)
	//t.c.Set(tickerKey, tickerMap, cache.NoExpiration)

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

	copyOfTrends := make([]Trend, 0)

	for _, t := range trends {
		copyOfTrends = append(copyOfTrends, t)
	}

	sort.Slice(copyOfTrends, func(i, j int) bool {
		return copyOfTrends[i].Start < copyOfTrends[j].Start
	})

	for _, data := range copyOfTrends {

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

		if data.Side == "" {
			data.Side = util.GetSide(data.TokenS, data.TokenB)
		}

		if data.Side == util.SideBuy {
			vol += util.StringToFloat(data.TokenS, data.AmountS)
			amount += util.StringToFloat(data.TokenB, data.AmountB)
		} else {
			vol += util.StringToFloat(data.TokenB, data.AmountB)
			amount += util.StringToFloat(data.TokenS, data.AmountS)
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
	t.cron.AddFunc("0 30 1 * * *", t.ProofRead)
	t.cron.Start()
}

func (t *TrendManager) insertTrendByInterval(interval string) error {
	if !isTimeToInsert(interval) {
		log.Info("no need to insert trend by interval " + interval)
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
	start := end.Unix() - getTsInterval(interval) + 1
	//multiple := tsInterval / tsOneHour

	for _, mkt := range util.AllMarkets {

		trends, err := t.rds.TrendQueryByInterval(OneHour, mkt, start, end.Unix())

		if err != nil {
			return err
		}

		toInsert := &dao.Trend{}

		var (
			vol    float64 = 0
			amount float64 = 0
			high   float64 = 0
			low    float64 = 0
			open   float64 = 0
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

			if open == 0 {
				open = t.Open
			}
		}

		if len(trends) == 0 {
			toInsert.Open = 0
			toInsert.Close = 0
		} else {
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
		toInsert.CreateTime = now.Unix()
		toInsert.UpdateTime = now.Unix()
		toInsert.Open = open

		if err := t.rds.Add(toInsert); err != nil {
			log.Info(err.Error())
			return err
		}
	}

	return nil
}

func (t *TrendManager) insertMinIntervalTrend(interval string, start int64, mkt string) (err error) {
	end := start + getTsInterval(interval) - 1
	lastTrends, _ := t.rds.TrendQueryByTime(interval, mkt, start-getTsInterval(interval), end-getTsInterval(interval))
	if len(lastTrends) > 1 {
		log.Info("found more than one last trend!")
		return errors.New("found more than one last trend")
	} else if len(lastTrends) == 0 {
		log.Info("not found last trend!")
	}

	fills, fillsErr := t.rds.QueryRecentFills(mkt, "", start, end)

	if fillsErr != nil {
		return fillsErr
	}

	now := time.Now().Unix()
	toInsert := &dao.Trend{
		Intervals:  OneHour,
		Market:     mkt,
		CreateTime: now,
		UpdateTime: now,
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

		if data.Side == "" {
			data.Side = util.GetSide(data.TokenS, data.TokenB)
		}

		if data.Side == util.SideBuy {
			vol += util.StringToFloat(data.TokenS, data.AmountS)
			amount += util.StringToFloat(data.TokenB, data.AmountB)
		} else {
			vol += util.StringToFloat(data.TokenB, data.AmountB)
			amount += util.StringToFloat(data.TokenS, data.AmountS)
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

	trends, _ := t.rds.TrendQueryByTime(interval, mkt, start, end)
	if len(trends) > 0 {
		log.Info("insert min interval trend, current interval trend exsit")
		toInsert.ID = trends[0].ID
		toInsert.CreateTime = trends[0].CreateTime
	}

	if err := t.rds.Save(toInsert); err != nil {
		log.Info(err.Error())
		return err
	}
	return nil
}

func (t *TrendManager) insertByTrendV2(interval string, start int64, mkt string) error {

	end := start + getTsInterval(interval) - 1
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
		open   float64 = 0
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

		if open == 0 && t.Open > 0 {
			open = t.Open
		}
	}

	if len(trends) == 0 {
		toInsert.Open = 0
		toInsert.Close = 0
	} else {
		toInsert.Close = trends[len(trends)-1].Close
	}
	toInsert.Vol = vol
	toInsert.Amount = amount
	toInsert.Open = open
	toInsert.High = high
	toInsert.Low = low
	toInsert.Start = start
	toInsert.End = end
	toInsert.Market = mkt
	toInsert.Intervals = interval
	toInsert.CreateTime = time.Now().Unix()
	toInsert.UpdateTime = toInsert.CreateTime

	exists, _ := t.rds.TrendQueryByTime(interval, mkt, start, end)
	if len(exists) > 0 {
		log.Info("insert by trend, current interval trend exsit")
		toInsert.ID = exists[0].ID
		toInsert.CreateTime = exists[0].CreateTime
	}

	if err := t.rds.Save(toInsert); err != nil {
		log.Info(err.Error())
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

	log.Info("start insert trend cron job")

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
					log.Info("schedule update, current interval trend exsit")
					continue
				}

				lastTrends, _ := t.rds.TrendQueryByTime(OneHour, tmpMkt, start-int64(60*60), end-int64(60*60))
				if len(lastTrends) > 1 {
					log.Info("found more than one last trend!")
					continue
				} else if len(lastTrends) == 0 {
					log.Info("not found last trend!")
				}

				if trends == nil || len(trends) == 0 {
					fills, fillsErr := t.rds.QueryRecentFills(tmpMkt, "", start, end)

					if fillsErr != nil {
						continue
					}

					now := time.Now().Unix()
					toInsert := &dao.Trend{
						Intervals:  OneHour,
						Market:     tmpMkt,
						CreateTime: now,
						UpdateTime: now,
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
						low = 0
					} else {
						toInsert.Open = lastTrends[0].Close
						toInsert.Close = lastTrends[0].Close
						toInsert.High = lastTrends[0].Close
						low = lastTrends[0].Close
					}

					sort.Slice(fills, func(i, j int) bool {
						return fills[i].CreateTime < fills[j].CreateTime
					})

					for _, data := range fills {

						if data.Side == "" {
							data.Side = util.GetSide(data.TokenS, data.TokenB)
						}

						if data.Side == util.SideBuy {
							vol += util.StringToFloat(data.TokenS, data.AmountS)
							amount += util.StringToFloat(data.TokenB, data.AmountB)
						} else {
							vol += util.StringToFloat(data.TokenB, data.AmountB)
							amount += util.StringToFloat(data.TokenS, data.AmountS)
						}

						price := util.CalculatePrice(data.AmountS, data.AmountB, data.TokenS, data.TokenB)

						if open == 0 && price != 0 {
							open = price
						}

						if toInsert.High == 0 || toInsert.High < price {
							toInsert.High = price
						}
						if (low == 0 && price > 0) || low > price {
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
						log.Info(err.Error())
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
	t.LoadCache()
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
		lastTrend := trends[0]
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

		if data.Side == "" {
			data.Side = util.GetSide(data.TokenS, data.TokenB)
		}

		if data.Side == util.SideBuy {
			vol += util.StringToFloat(data.TokenS, data.AmountS)
			amount += util.StringToFloat(data.TokenB, data.AmountB)
		} else {
			vol += util.StringToFloat(data.TokenB, data.AmountB)
			amount += util.StringToFloat(data.TokenS, data.AmountS)
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

	if t.cacheReady {
		if trendCache, err := redisCache.Get(buildTrendKey(interval, market)); err == nil {
			var tc Cache
			json.Unmarshal(trendCache, &tc)
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
		//if trendCache, ok := t.c.Get(trendKeyPre + interval); !ok {
		//	err = errors.New("can't found trends by key : " + interval)
		//} else {
		//	tc := trendCache.(map[string]Cache)[market]
		//	trends = make([]Trend, 0)
		//	if strings.ToLower(interval) == "1hr" {
		//		trendInFills, aggErr := t.aggregate(tc.Fills, tc.Trends)
		//		if aggErr == nil {
		//			trends = append(trends, trendInFills)
		//		}
		//	}
		//	for _, t := range tc.Trends {
		//		trends = append(trends, t)
		//	}
		//
		//}
	} else {
		err = errors.New("cache is not ready , please access later")
	}
	return
}

func (t *TrendManager) GetTicker() (tickers []Ticker, err error) {

	//log.Info("GetTicker Method Invoked")

	if t.cacheReady {

		//log.Info("[TICKER]ticker key used in GetTicker")
		if tickerCache, err := redisCache.Get(tickerKey); err == nil {
			var tickerMap map[string]Ticker
			json.Unmarshal(tickerCache, &tickerMap)
			tickers = make([]Ticker, 0)
			for _, v := range tickerMap {
				v.Amount, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Amount), 64)
				v.Vol, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Vol), 64)
				v.Buy, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Last), 64)
				v.Sell, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Last), 64)
				v.Open, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Open), 64)
				v.Close, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Close), 64)
				v.Last, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Last), 64)
				v.High, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.High), 64)
				v.Low, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Low), 64)

				if v.Change == "+0.00%" || v.Change == "-0.00%" {
					v.Change = "0.00%"
				}

				tickers = append(tickers, v)
			}

		} else {
			err = errors.New("get ticker from cache error, no value found")
		}

		//if tickerInCache, ok := t.c.Get(tickerKey); ok {
		//	tickerMap := tickerInCache.(map[string]Ticker)
		//	tickers = make([]Ticker, 0)
		//	for _, v := range tickerMap {
		//		v.Buy = strconv.FormatFloat(v.Last, 'f', -1, 64)
		//		v.Sell = strconv.FormatFloat(v.Last, 'f', -1, 64)
		//		tickers = append(tickers, v)
		//	}
		//} else {
		//	err = errors.New("get ticker from cache error, no value found")
		//}
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
		//log.Info("[TICKER]ticker key used in GetTickerByMarket")

		localCacheValue, ok := t.localCache.Get(localCacheTicker)
		if ok {
			//log.Info("[TICKER] get cache from local " + mkt)
			tickerMap := localCacheValue.(map[string]Ticker)
			for k, v := range tickerMap {
				if k == mkt {
					v.Buy, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Last), 64)
					v.Sell, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Last), 64)
					return v, err
				}
			}
		} else if tickerCache, err := redisCache.Get(tickerKey); err == nil {
			//log.Info("[TICKER] get cache from redis " + mkt)
			var tickerMap map[string]Ticker
			json.Unmarshal(tickerCache, &tickerMap)
			t.localCache.Set(localCacheTicker, tickerMap, 5*time.Second)
			for k, v := range tickerMap {
				if k == mkt {
					v.Buy, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Last), 64)
					v.Sell, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", v.Last), 64)
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

func (t *TrendManager) HandleOrderFilled(input eventemitter.EventData) (err error) {

	log.Info("HandleOrderFilled invoked")

	if t.cacheReady {

		event := input.(*types.OrderFilledEvent)
		if event.Status != types.TX_STATUS_SUCCESS {
			return
		}

		newFillModel := &dao.FillEvent{}
		if err = newFillModel.ConvertDown(event); err != nil {
			return
		}

		if newFillModel.Side == "" {
			newFillModel.Side = util.GetSide(newFillModel.TokenS, newFillModel.TokenB)
		}

		if newFillModel.Side == util.SideBuy {
			log.Debug("only calculate sell fill for ticker when ring length is 2")
			return
		}

		market, wrapErr := util.WrapMarketByAddress(newFillModel.TokenS, newFillModel.TokenB)

		if wrapErr != nil {
			err = wrapErr
			return
		}

		if trendInCache, err := redisCache.Get(buildTrendKey(OneHour, market)); err == nil {
			var tc Cache
			json.Unmarshal(trendInCache, &tc)
			tc.Fills = append(tc.Fills, *newFillModel)
			setTrendCache(OneHour, market, tc, 0)
			//t.c.Set(trendKeyPre+strings.ToLower(OneHour), trendMap, cache.NoExpiration)
			t.reCalTicker(market)
		} else {
			fills := make([]dao.FillEvent, 0)
			fills = append(fills, *newFillModel)
			newCache := Cache{make([]Trend, 0), fills}
			setTrendCache(OneHour, market, newCache, 0)
			//t.c.Set(trendKeyPre+strings.ToLower(OneHour), newCache, cache.NoExpiration)
			t.reCalTicker(market)
		}
	} else {
		err = errors.New("cache is not ready , please access later")
	}

	return
}

func (t *TrendManager) reCalTicker(market string) {
	//trendInCache, _ := t.c.Get(trendKeyPre + strings.ToLower(OneHour))
	trendInCache, _ := redisCache.Get(buildTrendKey(OneHour, market))
	var mktCache Cache
	json.Unmarshal(trendInCache, &mktCache)
	now := time.Now()
	firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 1, 0, now.Location())
	ticker := calculateTicker(market, mktCache.Fills, mktCache.Trends, firstSecondThisHour)
	//tickerInCache, _ := t.c.Get(tickerKey)
	//log.Info("[TICKER]ticker key used in reCalTicker")
	tickerInCache, _ := redisCache.Get(tickerKey)
	var tickerMap map[string]Ticker
	json.Unmarshal(tickerInCache, &tickerMap)
	tickerMap[market] = ticker
	setLprTickerCache(tickerMap, 0)
	//t.c.Set(tickerKey, tickerMap, cache.NoExpiration)
}

func setTrendCache(interval, market string, mktCache Cache, ttl int64) {
	cacheKey := buildTrendKey(interval, market)
	tickerByte, err := json.Marshal(mktCache)
	if err != nil {
		log.Info("marshal ticker json error " + err.Error())
	} else {
		eventemitter.Emit(eventemitter.TrendUpdated, market)
		redisCache.Set(cacheKey, tickerByte, ttl)
	}
}

func setLprTickerCache(tickers map[string]Ticker, ttl int64) {
	tickerByte, err := json.Marshal(tickers)
	if err != nil {
		log.Info("marshal ticker json error " + err.Error())
	} else {
		eventemitter.Emit(eventemitter.LoopringTickerUpdated, nil)
		//log.Info("[TICKER]ticker key set in setLprTickerCache")
		redisCache.Set(tickerKey, tickerByte, ttl)
	}
}

func buildTrendKey(interval, mkt string) string {
	return trendKeyPre + strings.ToUpper(interval) + "_" + strings.ToUpper(mkt)
}
