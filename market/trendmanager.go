package market

import (
	"errors"
	"github.com/Loopring/relay/dao"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"log"
	"sort"
	"strings"
	"sync"
	"time"
)

type Ticker struct {
	Market  string
	AmountS float64
	AmountB float64
	Open    float64
	Close   float64
	High    float64
	Low     float64
	Last    float64
}

type MarketCache struct {
	trends []Trend
	fills  []dao.FillEvent
}

type Trend struct {
	Interval   string
	Market     string
	AmountS    []byte
	AmountB    []byte
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
var tickerReadyChan chan bool
var trendReadyChan chan bool

const TREND_KEY = "market_ticker"
const TICKER_KEY = "market_ticker_view"

//TODO(xiaolu) move this to config
var supportTokens = []string{"lrc", "coss", "rdn"}
var supportMarkets = []string{"lrc-eth"}

func NewTrendManager(dao dao.RdsService) TrendManager {

	once.Do(func() {
		trendManager = TrendManager{rds: dao, cron: cron.New()}
		trendManager.c = cache.New(cache.NoExpiration, cache.NoExpiration)
		trendManager.initCache()
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

	trendMap := make(map[string]MarketCache)
	tickerMap := make(map[string]Ticker)
	for _, mkt := range supportMarkets {
		mktCache := MarketCache{}
		mktCache.trends = make([]Trend, 0)
		mktCache.fills = make([]dao.FillEvent, 0)

		// default 100 records load first time
		trends, err := t.rds.TrendPageQuery(dao.Trend{Market: mkt}, 1, 100)

		if err != nil {
			log.Fatal(err)
		}

		for _, trend := range trends.Data {
			mktCache.trends = append(mktCache.trends, transferFromDO(trend.(dao.Trend)))
		}

		tokenS, tokenB, _ := Unpack(mkt)
		now := time.Now()
		firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
		fills, err := t.rds.QueryRecentFills(tokenS, tokenB, firstSecondThisHour.Unix(), 0)

		if err != nil {
			log.Fatal(err)
		}

		mktCache.fills = fills
		trendMap[mkt] = mktCache

		ticker := calculateTicker(mkt, fills, mktCache.trends, firstSecondThisHour)
		tickerMap[mkt] = ticker
	}
	t.c.Set(TREND_KEY, trendMap, cache.NoExpiration)
	t.c.Set(TICKER_KEY, tickerMap, cache.NoExpiration)

	t.cacheReady = true
	t.startScheduleUpdate()

}

func calculateTicker(market string, fills []dao.FillEvent, trends []Trend, now time.Time) Ticker {

	var result = Ticker{Market: market}

	before24Hour := now.Unix() - 24*60*60

	var (
		high    float64
		low     float64
		amountS float64
		amountB float64
	)

	sort.Slice(trends, func(i, j int) bool {
		return trends[i].Start < trends[j].Start
	})

	for _, data := range trends {

		if data.Start > before24Hour {
			continue
		}

		amountS += byteToFloat(data.AmountS)
		amountB += byteToFloat(data.AmountB)
		if high == 0 || high < data.High {
			high = data.High
		}
		if low == 0 || low < data.Low {
			low = data.Low
		}
	}

	for i, data := range fills {

		amountS += byteToFloat(data.AmountS)
		amountB += byteToFloat(data.AmountB)

		//TODO(xiaolu) how to cal price?
		var price float64 = 0

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

	result.AmountS = amountS
	result.AmountB = amountB
	return result
}

func (t *TrendManager) startScheduleUpdate() {
	t.cron.AddFunc("10 * * * * *", t.insertTrend)
	t.cron.Start()
}

func (t *TrendManager) insertTrend() {
	// get latest 24 hour trend if not exist generate

	for _, mkt := range supportMarkets {
		now := time.Now()
		firstSecondThisHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.UTC)
		before24Hour := firstSecondThisHour.Unix() - 24*60*60
		trends, err := t.rds.TrendQueryByTime(mkt, before24Hour, firstSecondThisHour.Unix())

	}
}

func (t *TrendManager) GetTrends(market string) (trends []Trend, err error) {

	if t.cacheReady {
		if trendCache, ok := t.c.Get(TICKER_KEY); !ok {
			err = errors.New("can't found trends by key : " + TICKER_KEY)
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
		if tickerInCache, ok := t.c.Get(TICKER_KEY); ok {
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

//func(t *TrendManager) GetTicker() (tickers [] Ticker, err error) {
//
//	if t.cacheReady {
//		tickerInCache, ok := t.c.Get(TICKER_VIEW_KEY)
//		if ok {
//			tickers = tickerInCache.([]Ticker)
//			return
//		} else {
//			tickers, err = t.GetAllFromDB()
//			if err == nil {
//				return
//			}
//			t.c.Set(TICKER_KEY, tickers, cache.NoExpiration)
//		}
//	} else {
//		err = errors.New("cache is not ready , please access later")
//	}
//	return
//}

func (t *TrendManager) GenerateTrend() {
}

func (t *TrendManager) GetAllFromDB() (tickers []Ticker, err error) {

	tickers = make([]Ticker, len(supportTokens))
	for i, mkt := range supportTokens {
		tk, err := t.GetTickerFromDB(mkt)
		if err != nil {
			return
		}
		tickers[i] = tk
	}
	return
}

func (t *TrendManager) GetTickerFromDB(market string) (ticker Ticker, err error) {

	trends, err := t.rds.TrendPageQuery(dao.Trend{Market: market}, 1, 24)

	if err != nil {
		return
	}

	var result = Ticker{Market: market}

	var (
		high    float64
		low     float64
		amountS float64
		amountB float64
	)

	for _, v := range trends.Data {
		data := v.(dao.Trend)
		amountS += byteToFloat(data.AmountS)
		amountB += byteToFloat(data.AmountB)
		if high == 0 || high < data.High {
			high = data.High
		}
		if low == 0 || low < data.Low {
			low = data.Low
		}
	}

	result.High = high
	result.Low = low

	sort.Slice(trends.Data, func(i, j int) bool {
		return trends.Data[i].(dao.Trend).Start < trends.Data[j].(dao.Trend).Start
	})

	result.Open = trends.Data[0].(dao.Trend).Open
	result.Close = trends.Data[len(trends.Data)-1].(dao.Trend).Close

	return result, err
}

// when order filled event comming, invoke this
func UpdateTicker(market string, ticker Ticker) {

}

func Unpack(market string) (tokenS, tokenB string, err error) {
	mkts := strings.Split(strings.TrimSpace(market), "-")
	if len(mkts) != 2 {
		err = errors.New("unsupported market type")
		return
	}

	tokenS, tokenB = mkts[0], mkts[1]
	return
}

//TODO(xiaolu) replace this method after
func byteToFloat(amount []byte) float64 {
	return 0.0
}

//TODO(xiaolu) finish this later
func transferFromDO(trendFromDB dao.Trend) Trend {
	rst := Trend{}
	return rst
}
