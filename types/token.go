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

package types

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strconv"
)

type Token struct {
	Protocol common.Address `json:"protocol"`
	Symbol   string         `json:"symbol"`
	Source   string         `json:"source"`
	Time     int64          `json:"time"`
	Deny     bool           `json:"deny"`
	Decimals *big.Int       `json:"decimals"`
	IsMarket bool           `json:"isMarket"`
	IcoPrice *big.Rat       `json:"icoPrice"`
}

type CurrencyMarketCap struct {
	Id           string         `json:"id"`
	Name         string         `json:"name"`
	Symbol       string         `json:"symbol"`
	Address      common.Address `json:"address"`
	PriceUsd     *big.Rat       `json:"price_usd"`
	PriceBtc     *big.Rat       `json:"price_btc"`
	PriceCny     *big.Rat       `json:"price_cny"`
	Volume24HCNY *big.Rat       `json:"24h_volume_cny"`
	Volume24HUSD *big.Rat       `json:"24h_volume_usd"`
	LastUpdated  int64          `json:"last_updated"`
	Decimals     *big.Int
}

func (cap *CurrencyMarketCap) UnmarshalJSON(input []byte) error {
	type Cap struct {
		Id           string `json:"id"`
		Name         string `json:"name"`
		Symbol       string `json:"symbol"`
		PriceUsd     string `json:"price_usd"`
		PriceBtc     string `json:"price_btc"`
		PriceCny     string `json:"price_cny"`
		Volume24HCNY string `json:"24h_volume_cny"`
		Volume24HUSD string `json:"24h_volume_usd"`
		LastUpdated  string `json:"last_updated"`
	}
	c := &Cap{}
	if err := json.Unmarshal(input, c); nil != err {
		return err
	} else {
		cap.Id = c.Id
		cap.Symbol = c.Symbol
		cap.Name = c.Name
		if "" == c.PriceUsd {
			c.PriceUsd = "0.0"
		}
		if price, err1 := strconv.ParseFloat(c.PriceUsd, 10); nil != err1 {
			return err1
		} else {
			cap.PriceUsd = new(big.Rat).SetFloat64(price)
		}
		if "" == c.PriceBtc {
			c.PriceBtc = "0.0"
		}
		if price, err1 := strconv.ParseFloat(c.PriceBtc, 10); nil != err1 {
			return err1
		} else {
			cap.PriceBtc = new(big.Rat).SetFloat64(price)
		}
		if "" == c.PriceCny {
			c.PriceCny = "0.0"
		}
		if price, err1 := strconv.ParseFloat(c.PriceCny, 10); nil != err1 {
			return err1
		} else {
			cap.PriceCny = new(big.Rat).SetFloat64(price)
		}
		if "" == c.Volume24HCNY {
			c.Volume24HCNY = "0.0"
		}
		if price, err1 := strconv.ParseFloat(c.Volume24HCNY, 10); nil != err1 {
			return err1
		} else {
			cap.Volume24HCNY = new(big.Rat).SetFloat64(price)
		}
		if "" == c.Volume24HUSD {
			c.Volume24HUSD = "0.0"
		}
		if price, err1 := strconv.ParseFloat(c.Volume24HUSD, 10); nil != err1 {
			return err1
		} else {
			cap.Volume24HUSD = new(big.Rat).SetFloat64(price)
		}
		if "" == c.LastUpdated {
			c.LastUpdated = "0"
		}
		if lastUpdated, err1 := strconv.ParseInt(c.LastUpdated, 0, 0); nil != err1 {
			return err1
		} else {
			cap.LastUpdated = lastUpdated
		}
	}
	return nil
}
