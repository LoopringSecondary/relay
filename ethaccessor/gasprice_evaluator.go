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

package ethaccessor

import (
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"math/big"
	"sort"
)

type GasPriceEvaluator struct {
	Blocks []*BlockWithTxAndReceipt

	gasPrice *big.Int
	stopChan chan bool
}

func (e *GasPriceEvaluator) GasPrice(minGasPrice, maxGasPrice *big.Int) *big.Int {
	gasPrice := new(big.Int)
	if nil != e.gasPrice {
		if nil != maxGasPrice && maxGasPrice.Cmp(e.gasPrice) < 0 {
			gasPrice.Set(maxGasPrice)
		} else if nil != minGasPrice && minGasPrice.Cmp(e.gasPrice) > 0 {
			gasPrice.Set(minGasPrice)
		} else {
			gasPrice.Set(e.gasPrice)
		}
	} else {
		gasPrice.Set(maxGasPrice)
	}
	return gasPrice
}

func (e *GasPriceEvaluator) start() {
	var blockNumber types.Big
	if err := BlockNumber(&blockNumber); nil == err {
		go func() {
			number := new(big.Int).Set(blockNumber.BigInt())
			number.Sub(number, big.NewInt(30))
			iterator := NewBlockIterator(number, nil, true, uint64(0))
			for {
				select {
				case <-e.stopChan:
					return
				default:
					blockInterface, err := iterator.Next()
					if nil == err {
						blockWithTxAndReceipt := blockInterface.(*BlockWithTxAndReceipt)
						log.Debugf("gasPriceEvaluator, blockNumber:%s, gasPrice:%s", blockWithTxAndReceipt.Number.BigInt().String(), e.gasPrice.String())
						e.Blocks = append(e.Blocks, blockWithTxAndReceipt)
						if len(e.Blocks) > 30 {
							e.Blocks = e.Blocks[1:]
						}
						var prices gasPrices = []*big.Int{}
						for _, block := range e.Blocks {
							for _, tx := range block.Transactions {
								prices = append(prices, tx.GasPrice.BigInt())
							}
						}
						e.gasPrice = prices.bestGasPrice()
					}
				}
			}
		}()

	}
}

func (e *GasPriceEvaluator) stop() {
	e.stopChan <- true
}

type gasPrices []*big.Int

func (prices gasPrices) Len() int {
	return len(prices)
}

func (prices gasPrices) Swap(i, j int) {
	prices[i], prices[j] = prices[j], prices[i]
}

func (prices gasPrices) Less(i, j int) bool {
	return prices[i].Cmp(prices[j]) > 0
}

func (prices gasPrices) bestGasPrice() *big.Int {
	sort.Sort(prices)
	startIdx := 0
	endIdx := (len(prices) / 3) * 2

	averagePrice := big.NewInt(0)
	for _, price := range prices[startIdx:endIdx] {
		averagePrice.Add(averagePrice, price)
	}
	averagePrice.Div(averagePrice, big.NewInt(int64(endIdx-startIdx+1)))

	if averagePrice.Cmp(big.NewInt(int64(0))) <= 0 {
		averagePrice = big.NewInt(int64(1000000000))
	}
	return averagePrice
}
