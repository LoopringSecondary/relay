package market

import (
	"sync"
	"github.com/patrickmn/go-cache"
	"github.com/Loopring/ringminer/dao"
	"strings"
	"errors"
	"sort"
	"time"
)

type Ticker struct {
	Market	              string
	AmountS               float64
	AmountB               float64
	Open                  float64
	Close                 float64
	High                  float64
	Low                   float64
	Last 				  float64
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
	From       int64
	To         int64
}

type TrendManager struct {
	c *cache.Cache
	IsTickerReady bool
	IsTrendReady bool
	EnableCache bool
	rds dao.RdsServiceImpl
}

var once sync.Once
var trendManager TrendManager
var tickerReadyChan chan bool
var trendReadyChan chan bool

const TICKER_KEY = "market_ticker"
const TREND_KEY_PRE = "trend_"
//TODO(xiaolu) move this to config
var supportTokens = []string{"lrc", "coss", "rdn"}
const MARKET_BASE_TOKEN = "weth"

func NewTrendManager(enableCache bool) TrendManager {

	once.Do(func () {
		trendManager = TrendManager{EnableCache:enableCache}

		if enableCache {
			trendManager.c = cache.New(cache.NoExpiration, cache.NoExpiration)
			trendManager.RefreshCache()
		}
	})

	return trendManager
}

func (t *TrendManager) RefreshCache() {
	//timeTicker := time.NewTicker()

}

// now every hour invoke this
func GenerateTrend(interval string) error {

	return nil
}


func(t *TrendManager) GetTicker() (tickers [] Ticker, err error) {

	if t.EnableCache {
		tickerInCache, ok := t.c.Get(TICKER_KEY)
		if ok {
			tickers = tickerInCache.([]Ticker)
			return
		} else {
			tickers, err = t.GetAllFromDB()
			if err == nil {
				return
			}
			t.c.Set(TICKER_KEY, tickers, cache.NoExpiration)
		}
	}
	return
}

func (t *TrendManager) GetAllFromDB() (tickers [] Ticker, err error) {

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

func(t *TrendManager) GetTickerFromDB(market string) (ticker Ticker, err error) {


	trends, err := t.rds.TrendPageQuery(dao.Trend{Market:market}, 1, 24)

	if err != nil {
		return
	}

	var result = Ticker{Market:market}

	var (
		high float64
		low float64
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
		return trends.Data[i].(dao.Trend).From < trends.Data[j].(dao.Trend).From
	})

	result.Open = trends.Data[0].(dao.Trend).Open
	result.Close = trends.Data[len(trends.Data) - 1].(dao.Trend).Close

	return result, err
}

// when order filled event comming, invoke this
func UpdateTicker(market string, ticker Ticker) {

}

func GetTrends(market string) ([]Trend, error) {

	return nil, nil
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
func byteToFloat(amount [] byte) float64 {
	return 0.0
}
