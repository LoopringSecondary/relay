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
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/crypto"
)


func TestWalletServiceImpl_GetPortfolio(t *testing.T) {
	priceQuoteMap := make(map[string]*big.Rat)
	priceQuoteMap["WETH"] = new(big.Rat).SetFloat64(4532.01)
	priceQuoteMap["RDN"] = new(big.Rat).SetFloat64(12.01)
	priceQuoteMap["LRC"] = new(big.Rat).SetFloat64(2.32)
	balances := make(map[string]market.Balance)
	balances["WETH"] = market.Balance{Token:"WETH", Balance:types.HexToBigint("0x22")}
	balances["LRC"] = market.Balance{Token:"LRC", Balance:types.HexToBigint("0x1")}
	balances["RDN"] = market.Balance{Token:"RDN", Balance:types.HexToBigint("0x23")}

	totalAsset := big.NewRat(0, 1)
	for k, v := range balances {
		asset := new(big.Rat).Set(priceQuoteMap[k])
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		totalAsset = totalAsset.Add(totalAsset, asset)
	}

	fmt.Println("total asset is .........")
	fmt.Println(totalAsset.Float64())
	fmt.Println("xxxxxxxxxxxx")

	for k, v := range balances {
		portfolio := gateway.Portfolio{Token: k, Amount: types.BigintToHex(v.Balance)}
		asset := new(big.Rat).Set(priceQuoteMap[k])
		fmt.Println(asset.Float64())
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		fmt.Println(asset.Float64())
		percentage, _ := asset.Quo(asset, totalAsset).Float64()
		fmt.Println("percentage .......")
		fmt.Println(percentage)
		portfolio.Percentage = fmt.Sprintf("%.4f%%", 100*percentage)
		fmt.Println(portfolio.Percentage)
	}

	s, _ := crypto.NewPrivateKeyCrypto(false, "0x7d0a1121fb170361b6483d922d72258e6d4da9aa65234ac7ba0c9c833e6adc71")
	fmt.Println(s.Address().Hex())


}
