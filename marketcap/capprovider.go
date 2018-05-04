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
	"errors"
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"io/ioutil"
	"math/big"
	"net/http"
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

type MarketCapProvider interface {
	Start()
	Stop()

	LegalCurrencyValue(tokenAddress common.Address, amount *big.Rat) (*big.Rat, error)
	LegalCurrencyValueOfEth(amount *big.Rat) (*big.Rat, error)
	LegalCurrencyValueByCurrency(tokenAddress common.Address, amount *big.Rat, currencyStr string) (*big.Rat, error)
	GetMarketCap(tokenAddress common.Address) (*big.Rat, error)
	GetEthCap() (*big.Rat, error)
	GetMarketCapByCurrency(tokenAddress common.Address, currencyStr string) (*big.Rat, error)
}

type CapProvider_LocalCap struct {
	trendManager    *market.TrendManager
	tokenMarketCaps map[common.Address]*types.CurrencyMarketCap
	stopChan        chan bool
	stopFuncs       []func()
}

//todo:
func NewLocalCap() *CapProvider_LocalCap {
	localCap := &CapProvider_LocalCap{}
	localCap.stopChan = make(chan bool)
	localCap.stopFuncs = make([]func(), 0)
	localCap.tokenMarketCaps = make(map[common.Address]*types.CurrencyMarketCap)
	return localCap
}

func (cap *CapProvider_LocalCap) Start() {
	for _, marketStr := range util.AllMarkets {
		tokenAddress, _ := util.UnWrapToAddress(marketStr)
		token, _ := util.AddressToToken(tokenAddress)
		c := &types.CurrencyMarketCap{}
		c.Address = token.Protocol
		c.Id = token.Source
		c.Name = token.Symbol
		c.Symbol = token.Symbol
		c.Decimals = new(big.Int).Set(token.Decimals)
		cap.tokenMarketCaps[tokenAddress] = c
	}
	//if stopFunc,err := eventemitter.NewSerialWatcher(eventemitter.RingMined, cap.listenRingMinedEvent); nil != err {
	//	log.Debugf("err:%s", err.Error())
	//} else {
	//	cap.stopFuncs = append(cap.stopFuncs, stopFunc)
	//}
}

func (cap *CapProvider_LocalCap) Stop() {
	for _, stopFunc := range cap.stopFuncs {
		stopFunc()
	}
	cap.stopChan <- true
}

//func (cap *CapProvider_LocalCap) LegalCurrencyValue(tokenAddress common.Address, amount *big.Rat) (*big.Rat, error) {
//
//}
//
//func (cap *CapProvider_LocalCap) LegalCurrencyValueOfEth(amount *big.Rat) (*big.Rat, error) {
//
//}
//
//func (cap *CapProvider_LocalCap) LegalCurrencyValueByCurrency(tokenAddress common.Address, amount *big.Rat, currencyStr string) (*big.Rat, error) {
//
//}
//
//func (cap *CapProvider_LocalCap) GetMarketCap(tokenAddress common.Address) (*big.Rat, error) {
//
//}
//
//func (cap *CapProvider_LocalCap) GetEthCap() (*big.Rat, error) {
//
//}
//
//func (cap *CapProvider_LocalCap) GetMarketCapByCurrency(tokenAddress common.Address, currencyStr string) (*big.Rat, error) {
//
//}

type MixMarketCap struct {
	coinMarketProvider *CapProvider_CoinMarketCap
	localCap           *CapProvider_LocalCap
}

func (cap *MixMarketCap) Start() {
	cap.coinMarketProvider.Start()
	cap.localCap.Start()
}

func (cap *MixMarketCap) Stop() {
	cap.coinMarketProvider.Stop()
	cap.localCap.Stop()
}

func (cap *MixMarketCap) selectCap(tokenAddress common.Address) MarketCapProvider {
	return cap.coinMarketProvider
	//if _,exists := cap.coinMarketProvider.tokenMarketCaps[tokenAddress]; exists || types.IsZeroAddress(tokenAddress) {
	//	return cap.coinMarketProvider
	//} else {
	//	return cap.localCap
	//}
}

func (cap *MixMarketCap) LegalCurrencyValue(tokenAddress common.Address, amount *big.Rat) (*big.Rat, error) {
	return cap.selectCap(tokenAddress).LegalCurrencyValue(tokenAddress, amount)
}

func (cap *MixMarketCap) LegalCurrencyValueOfEth(amount *big.Rat) (*big.Rat, error) {
	return cap.selectCap(types.NilAddress).LegalCurrencyValueOfEth(amount)
}

func (cap *MixMarketCap) LegalCurrencyValueByCurrency(tokenAddress common.Address, amount *big.Rat, currencyStr string) (*big.Rat, error) {
	return cap.selectCap(tokenAddress).LegalCurrencyValueByCurrency(tokenAddress, amount, currencyStr)
}

func (cap *MixMarketCap) GetMarketCap(tokenAddress common.Address) (*big.Rat, error) {
	return cap.selectCap(tokenAddress).GetMarketCap(tokenAddress)
}

func (cap *MixMarketCap) GetEthCap() (*big.Rat, error) {
	return cap.selectCap(types.NilAddress).GetEthCap()
}

func (cap *MixMarketCap) GetMarketCapByCurrency(tokenAddress common.Address, currencyStr string) (*big.Rat, error) {
	return cap.selectCap(tokenAddress).GetMarketCapByCurrency(tokenAddress, currencyStr)
}

type CapProvider_CoinMarketCap struct {
	baseUrl         string
	tokenMarketCaps map[common.Address]*types.CurrencyMarketCap
	idToAddress     map[string]common.Address
	currency        string
	duration        int
	stopChan        chan bool
}

func (p *CapProvider_CoinMarketCap) LegalCurrencyValue(tokenAddress common.Address, amount *big.Rat) (*big.Rat, error) {
	return p.LegalCurrencyValueByCurrency(tokenAddress, amount, p.currency)
}

func (p *CapProvider_CoinMarketCap) LegalCurrencyValueOfEth(amount *big.Rat) (*big.Rat, error) {
	tokenAddress := util.AllTokens["WETH"].Protocol
	return p.LegalCurrencyValueByCurrency(tokenAddress, amount, p.currency)
}

func (p *CapProvider_CoinMarketCap) LegalCurrencyValueByCurrency(tokenAddress common.Address, amount *big.Rat, currencyStr string) (*big.Rat, error) {
	if c, exists := p.tokenMarketCaps[tokenAddress]; !exists {
		return nil, errors.New("not found tokenCap:" + tokenAddress.Hex())
	} else {
		v := new(big.Rat).SetInt(c.Decimals)
		v.Quo(amount, v)
		price, _ := p.GetMarketCapByCurrency(tokenAddress, currencyStr)
		//log.Debugf("LegalCurrencyValueByCurrency token:%s,decimals:%s, amount:%s, currency:%s, price:%s", tokenAddress.Hex(), c.Decimals.String(), amount.FloatString(2), currencyStr, price.FloatString(2) )
		v.Mul(price, v)
		return v, nil
	}
}

func (p *CapProvider_CoinMarketCap) GetMarketCap(tokenAddress common.Address) (*big.Rat, error) {
	return p.GetMarketCapByCurrency(tokenAddress, p.currency)
}

func (p *CapProvider_CoinMarketCap) GetEthCap() (*big.Rat, error) {
	return p.GetMarketCapByCurrency(util.AllTokens["WETH"].Protocol, p.currency)
}

func (p *CapProvider_CoinMarketCap) GetMarketCapByCurrency(tokenAddress common.Address, currencyStr string) (*big.Rat, error) {
	currency := StringToLegalCurrency(currencyStr)
	if c, exists := p.tokenMarketCaps[tokenAddress]; exists {
		var v *big.Rat
		switch currency {
		case CNY:
			v = c.PriceCny
		case USD:
			v = c.PriceUsd
		case BTC:
			v = c.PriceBtc
		}
		if "VITE" == c.Symbol || "ARP" == c.Symbol {
			wethCap, _ := p.GetMarketCapByCurrency(util.AllTokens["WETH"].Protocol, currencyStr)
			v = wethCap.Mul(wethCap, util.AllTokens[c.Symbol].IcoPrice)
		}
		if v == nil {
			return nil, errors.New("tokenCap is nil")
		} else {
			return new(big.Rat).Set(v), nil
		}
	} else {
		err := errors.New("not found tokenCap:" + tokenAddress.Hex())
		res := new(big.Rat).SetInt64(int64(1))
		if nil != err {
			log.Errorf("get MarketCap of token:%s, occurs error:%s. the value will be default value:%s", tokenAddress.Hex(), err.Error(), res.String())
		}
		return res, err
	}
}

func (p *CapProvider_CoinMarketCap) Stop() {
	p.stopChan <- true
}

func (p *CapProvider_CoinMarketCap) Start() {
	go func() {
		for {
			select {
			case <-time.After(time.Duration(p.duration) * time.Minute):
				log.Infof("marketCap sycing...")
				if err := p.syncMarketCap(); nil != err {
					log.Errorf("can't sync marketcap, time:%d", time.Now().Unix())
				}
			case stopped := <-p.stopChan:
				if stopped {
					return
				}
			}
		}
	}()
}

func (p *CapProvider_CoinMarketCap) syncMarketCap() error {
	url := fmt.Sprintf(p.baseUrl, p.currency)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if nil != resp && nil != resp.Body {
			resp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if nil != err {
		return err
	} else {
		var caps []*types.CurrencyMarketCap
		if err := json.Unmarshal(body, &caps); nil != err {
			return err
		} else {
			syncedTokens := make(map[common.Address]bool)
			for _, tokenCap := range caps {
				if tokenAddress, exists := p.idToAddress[strings.ToUpper(tokenCap.Id)]; exists {
					p.tokenMarketCaps[tokenAddress].PriceUsd = tokenCap.PriceUsd
					p.tokenMarketCaps[tokenAddress].PriceBtc = tokenCap.PriceBtc
					p.tokenMarketCaps[tokenAddress].PriceCny = tokenCap.PriceCny
					p.tokenMarketCaps[tokenAddress].Volume24HCNY = tokenCap.Volume24HCNY
					p.tokenMarketCaps[tokenAddress].Volume24HUSD = tokenCap.Volume24HUSD
					p.tokenMarketCaps[tokenAddress].LastUpdated = tokenCap.LastUpdated
					log.Debugf("token:%s, priceUsd:%s", tokenAddress.Hex(), tokenCap.PriceUsd.FloatString(2))
					syncedTokens[p.tokenMarketCaps[tokenAddress].Address] = true
				}
			}
			for _, tokenCap := range p.tokenMarketCaps {
				if _, exists := syncedTokens[tokenCap.Address]; !exists && "VITE" != tokenCap.Symbol && "ARP" != tokenCap.Symbol {
					//todo:
					log.Errorf("token:%s, id:%s, can't sync marketcap at time:%d, it't last updated time:%d", tokenCap.Symbol, tokenCap.Id, time.Now().Unix(), tokenCap.LastUpdated)
				}
			}
		}
	}
	return nil
}

func NewMarketCapProvider(options config.MarketCapOptions) *CapProvider_CoinMarketCap {
	provider := &CapProvider_CoinMarketCap{}
	provider.baseUrl = options.BaseUrl
	provider.currency = options.Currency
	provider.tokenMarketCaps = make(map[common.Address]*types.CurrencyMarketCap)
	provider.idToAddress = make(map[string]common.Address)
	provider.duration = options.Duration
	if provider.duration <= 0 {
		//default 5 min
		provider.duration = 5
	}
	for _, v := range util.AllTokens {
		if "ARP" == v.Symbol || "VITE" == v.Symbol {
			c := &types.CurrencyMarketCap{}
			c.Address = v.Protocol
			c.Id = v.Source
			c.Name = v.Symbol
			c.Symbol = v.Symbol
			c.Decimals = new(big.Int).Set(v.Decimals)
			provider.tokenMarketCaps[c.Address] = c
		} else {
			c := &types.CurrencyMarketCap{}
			c.Address = v.Protocol
			c.Id = v.Source
			c.Name = v.Symbol
			c.Symbol = v.Symbol
			c.Decimals = new(big.Int).Set(v.Decimals)
			provider.tokenMarketCaps[c.Address] = c
			provider.idToAddress[strings.ToUpper(c.Id)] = c.Address
		}
	}

	if err := provider.syncMarketCap(); nil != err {
		log.Fatalf("can't sync marketcap with error:%s", err.Error())
	}

	return provider
}
