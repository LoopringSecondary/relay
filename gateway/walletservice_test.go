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

package gateway_test

import (
	"github.com/Loopring/relay/types"
	"math/big"
	"testing"
	"github.com/Loopring/relay/market"
	"fmt"
	"strconv"
	"github.com/Loopring/relay/gateway"
)


func TestWalletServiceImpl_GetPortfolio(t *testing.T) {
	priceQuoteMap := make(map[string]*big.Rat)
	priceQuoteMap["WETH"] = new(big.Rat).SetFloat64(4532.01)
	priceQuoteMap["RDN"] = new(big.Rat).SetFloat64(12.01)
	priceQuoteMap["LRC"] = new(big.Rat).SetFloat64(2.32)
	balances := make(map[string]market.Balance)
	balances["WETH"] = market.Balance{Token:"WETH", Balance:types.HexToBigint("0x2342342384")}
	balances["LRC"] = market.Balance{Token:"WETH", Balance:types.HexToBigint("0x0")}
	balances["RDN"] = market.Balance{Token:"WETH", Balance:types.HexToBigint("0x0")}

	totalAsset := big.NewRat(0, 1)
	for k, v := range balances {
		fmt.Println("start handle asset handler.....")
		asset := priceQuoteMap[k]
		fmt.Println(asset)
		fmt.Println(v.Balance)
		fmt.Println(new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		fmt.Println(totalAsset.Float64())
		fmt.Println(asset)
		totalAsset = totalAsset.Add(totalAsset, asset)
		fmt.Println(totalAsset.Float64())
	}

	for k, v := range balances {
		fmt.Println("start collect asset handler.....")
		portfolio := gateway.Portfolio{Token: k, Amount: types.BigintToHex(v.Balance)}
		asset := priceQuoteMap[k]
		fmt.Println(asset)
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		fmt.Println(asset)
		fmt.Println(v.Balance)
		fmt.Println(totalAsset)
		percentage, _ := asset.Quo(asset, totalAsset).Float64()
		fmt.Println(percentage)
		portfolio.Percentage = strconv.FormatFloat(percentage, 'f', 2, 64)
	}


}
