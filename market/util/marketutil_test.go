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

package util_test

import (
	"fmt"
	log2 "github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-ethereum/log"
	"math/big"
	"testing"
)

func TestCalculatePrice(t *testing.T) {
	util.SupportTokens = make(map[string]types.Token)
	util.AllTokens = make(map[string]types.Token)
	funToken := types.Token{Protocol: common.HexToAddress("0x419D0d8BdD9aF5e606Ae2232ed285Aff190E711b"), Decimals: big.NewInt(1e8)}
	wethToken := types.Token{Protocol: common.HexToAddress("0x2956356cD2a2bf3202F771F50D3D14A367b48070"), Decimals: big.NewInt(1e18)}
	util.SupportTokens["FUN"] = funToken
	util.AllTokens["FUN"] = funToken
	util.AllTokens["WETH"] = wethToken
	price := util.CalculatePrice("10000000000", "7000000000000000", "0x419D0d8BdD9aF5e606Ae2232ed285Aff190E711b", "0x2956356cD2a2bf3202F771F50D3D14A367b48070")
	fmt.Println(price)
	fmt.Println(price == 0.00007)
	if price != 0.00007 {
		log.Error("not right")
	} else {
		log.Error("xxxxx")
	}
	log2.Fatal("ksfjlsdjfklj")
}
