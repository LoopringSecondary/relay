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
)

type Portfolio struct {
	Token  string
	Amount string
}

type WalletService interface {
	GetTicker() (res []market.Ticker, err error)
	TestPing(input int) (res string, err error)
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

	return
}
