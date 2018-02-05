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

package gateway

import (
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"qiniupkg.com/x/errors.v7"
	"time"
	"github.com/Loopring/relay/market/util"
	"math/big"
	"strconv"
	"github.com/Loopring/relay/types"
)

const DefaultContractVersion = "v1.1"
const DefaultCapCurrency = "CNY"

type Portfolio struct {
	Token  string
	Amount string
	Percentage string
}

type WalletService interface {
	GetTicker() (res []market.Ticker, err error)
	TestPing(input int) (res string, err error)
	GetPortfolio(owner string) (res []Portfolio, err error)
	GetPriceQuote(currency string) (result PriceQuote, err error)
}

type WalletServiceImpl struct {
	trendManager   market.TrendManager
	orderManager   ordermanager.OrderManager
	accountManager market.AccountManager
	marketCap      marketcap.MarketCapProvider
}

func (w *WalletServiceImpl) TestPing(input int) (resp []byte, err error) {

	var res string
	if input > 0 {
		res = "input is bigger than zero " + time.Now().String()
	} else if input == 0 {
		res = "input is equal zero " + time.Now().String()
	} else if input < 0 {
		res = "input is smaller than zero " + time.Now().String()
	}
	resp = []byte("{'abc' : '" + res + "'}")
	return
}

func (w *WalletServiceImpl) GetPortfolio(owner string) (res []Portfolio, err error) {
	if len(owner) == 0 {
		return nil, errors.New("owner can't be nil")
	}

	account := w.accountManager.GetBalance(DefaultContractVersion, owner)
	balances := account.Balances
	if len(balances) == 0 {
		return
	}

	priceQuote, err := w.GetPriceQuote(DefaultCapCurrency)
	if err != nil {
		return
	}

	priceQuoteMap := make(map[string]*big.Rat)
	for _, pq := range priceQuote.Tokens {
		priceQuoteMap[pq.Token] = new(big.Rat).SetFloat64(pq.Price)
	}

	var totalAsset *big.Rat
	for k, v := range balances {
		asset := priceQuoteMap[k]
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		totalAsset = totalAsset.Add(totalAsset, asset)
	}


	res = make([]Portfolio, 0)

	for k, v := range balances {
		portfolio := Portfolio{Token:k, Amount:types.BigintToHex(v.Balance)}
		asset := priceQuoteMap[k]
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		percentage, _ := asset.Quo(asset, totalAsset).Float64()
		portfolio.Percentage = strconv.FormatFloat(percentage, 'f', 2, 64)
		res = append(res, portfolio)
	}

	return
}

func (w *WalletServiceImpl) GetPriceQuote(currency string) (result PriceQuote, err error) {

	rst := PriceQuote{currency, make([]TokenPrice, 0)}
	for k, v := range util.AllTokens {
		price, _ := w.marketCap.GetMarketCapByCurrency(v.Protocol, currency)
		floatPrice, _ := price.Float64()
		rst.Tokens = append(rst.Tokens, TokenPrice{k, floatPrice})
	}

	return rst, nil
}

