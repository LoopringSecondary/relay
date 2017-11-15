package market

import "sync"

type Ticker struct {
	Market  string
	AmountS float64
	AmountB float64
	Open    float64
	Close   float64
	High    float64
	Low     float64
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
	TickerCache   map[string]Ticker
	TrendCache    map[string][]Trend
	IsTickerReady bool
	IsTrendReady  bool
	EnableCache   bool
}

var once sync.Once
var trendManager TrendManager
var tickerReadyChan chan bool
var trendReadyChan chan bool

func NewTrendManager(enableCache bool) TrendManager {
	once.Do(func() {
		trendManager = TrendManager{EnableCache: enableCache}
		trendManager.TickerCache = make(map[string]Ticker)
		trendManager.TrendCache = make(map[string][]Trend)

		if enableCache {

		}
	})

	return trendManager
}

var tickerCache map[string]Ticker

// now every hour invoke this
func GenerateTrend(interval string) error {

	return nil
}

func GetTicker(market string) ([]Ticker, error) {

	return nil, nil
}

// when order filled event comming, invoke this
func UpdateTicker(market string, ticker Ticker) {
	tickerCache[market] = ticker
}

func GetTrends(market string) ([]Trend, error) {

	return nil, nil
}
