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

package marketcap

import (
	"encoding/json"
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"github.com/ethereum/go-ethereum/common"
	"io/ioutil"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type LegalCurrency int

func StringToLegalCurrency(currency string) LegalCurrency {
	currency = strings.ToUpper(currency)
	switch currency {
	default:
		return CNY
	case "CNY":
		return CNY
	case "USD":
		return USD
	case "BTC":
		return BTC
	}
}

const (
	CNY LegalCurrency = iota
	USD
	EUR
	BTC
)

type MarketCapProvider struct {
	baseUrl       string
	currenciesMap map[common.Address]*CurrencyMarketCap
	currency      LegalCurrency
}

type CurrencyMarketCap struct {
	Id           string         `json:"id"`
	Name         string         `json:"name"`
	Symbol       string         `json:"symbol"`
	Address      common.Address `json:"address"`
	PriceUsd     float64        `json:"price_usd"`
	PriceBtc     float64        `json:"price_btc"`
	PriceCny     float64        `json:"price_cny"`
	Volume24HCNY float64        `json:"24h_volume_cny"`
	Volume24HUSD float64        `json:"24h_volume_usd"`
}

func (cap *CurrencyMarketCap) UnmarshalJSON(input []byte) error {
	type Cap struct {
		PriceUsd     string `json:"price_usd"`
		PriceBtc     string `json:"price_btc"`
		PriceCny     string `json:"price_cny"`
		Volume24HCNY string `json:"24h_volume_cny"`
		Volume24HUSD string `json:"24h_volume_usd"`
	}
	var c *Cap = &Cap{}
	if err := json.Unmarshal(input, c); nil != err {
		return err
	} else {
		var err1 error
		if cap.PriceUsd, err1 = strconv.ParseFloat(c.PriceUsd, 10); nil != err1 {
			return err1
		}
		if cap.PriceBtc, err1 = strconv.ParseFloat(c.PriceBtc, 10); nil != err1 {
			return err1
		}
		if cap.PriceCny, err1 = strconv.ParseFloat(c.PriceCny, 10); nil != err1 {
			return err1
		}
		if cap.Volume24HCNY, err1 = strconv.ParseFloat(c.Volume24HCNY, 10); nil != err1 {
			return err1
		}
		if cap.Volume24HUSD, err1 = strconv.ParseFloat(c.Volume24HUSD, 10); nil != err1 {
			return err1
		}
	}
	return nil
}

func (p *MarketCapProvider) GetEthCap() *big.Rat {
	//todo:
	return new(big.Rat).SetInt64(int64(1))
}

func (p *MarketCapProvider) GetMarketCap(tokenAddress common.Address) *big.Rat {
	if c, ok := p.currenciesMap[tokenAddress]; ok {
		v := new(big.Rat)
		switch p.currency {
		case CNY:
			v = v.SetFloat64(c.PriceCny)
		case USD:
			v = v.SetFloat64(c.PriceUsd)
		case BTC:
			v = v.SetFloat64(c.PriceBtc)
		}
		return v
	} else {
		return new(big.Rat).SetInt64(int64(1))
	}
}

func (p *MarketCapProvider) Stop() {
	//todo:
}

func (p *MarketCapProvider) Start() {
	go func() {
		for {
			for _, c := range p.currenciesMap {
				select {
				case <-time.After(7 * time.Second):
					url := fmt.Sprintf(p.baseUrl, c.Name)
					resp, err := http.Get(url)
					if err != nil {
						log.Errorf("can't get new currency cap, err:%s", err.Error())
					}
					defer resp.Body.Close()

					body, err := ioutil.ReadAll(resp.Body)
					if nil != err {
						log.Errorf("err:%s", err.Error())
					} else {
						var caps []*CurrencyMarketCap
						err := json.Unmarshal([]byte(body), &caps)
						if nil != err {
							log.Errorf("err:%s", err.Error())
						} else {
							c = caps[0]
						}
					}
				}
			}
		}
	}()
}

func NewMarketCapProvider(options config.MinerOptions) *MarketCapProvider {
	provider := &MarketCapProvider{}
	provider.baseUrl = options.RateProvider.BaseUrl
	provider.currency = StringToLegalCurrency(options.RateProvider.Currency)
	provider.currenciesMap = make(map[common.Address]*CurrencyMarketCap)

	for addr, name := range options.RateProvider.CurrenciesMap {
		c := &CurrencyMarketCap{}
		c.Address = common.HexToAddress(addr)
		c.Id = name
		c.Name = name
		provider.currenciesMap[c.Address] = c
	}
	return provider
}
