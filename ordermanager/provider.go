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

package ordermanager

import (
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"math/big"
	"sync"
	"time"
)

type minerOrdersProvider struct {
	dao                dao.RdsService
	ticker             *time.Ticker
	tick               time.Duration
	currentBlockNumber *types.Big
	blockNumberPeriod  *types.Big
	mtx                sync.Mutex
	quit               chan struct{}
}

func newMinerOrdersProvider(clearTick time.Duration, blockNumberPeriod *types.Big) *minerOrdersProvider {
	provider := &minerOrdersProvider{}
	provider.tick = clearTick
	provider.blockNumberPeriod = blockNumberPeriod

	return provider
}

func (p *minerOrdersProvider) start() {
	p.quit = make(chan struct{})
	p.ticker = time.NewTicker(p.tick)

	for {
		select {
		case <-p.ticker.C:
			p.clearOrders()
		}
	}
}

func (p *minerOrdersProvider) stop() {
	p.ticker.Stop()
	close(p.quit)
}

func (p *minerOrdersProvider) setBlockNumber(num *types.Big) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.currentBlockNumber = num
}

func (p *minerOrdersProvider) markOrders(orderhashs []types.Hash) error {
	var orderhashstrs []string

	for _, v := range orderhashs {
		orderhashstrs = append(orderhashstrs, v.Hex())
	}

	return p.dao.MarkMinerOrders(orderhashstrs, p.currentBlockNumber.Int64())
}

func (p *minerOrdersProvider) clearOrders() error {
	des := new(big.Int).Sub(p.currentBlockNumber.BigInt(), p.blockNumberPeriod.BigInt())
	if des.Cmp(big.NewInt(1)) < 0 {
		return nil
	}

	return p.dao.ClearMinerOrdersMark(des.Int64())
}

func (p *minerOrdersProvider) getOrders(tokenS, tokenB types.Address, orderhashs []types.Hash) []types.OrderState {
	var list []types.OrderState

	filterStatus := []uint8{types.ORDER_FINISHED.Value(), types.ORDER_CUTOFF.Value(), types.ORDER_CANCEL.Value()}

	models, err := p.dao.GetOrdersForMiner(tokenS.Hex(), tokenB.Hex(), filterStatus)
	if len(models) == 0 || err != nil {
		return list
	}

	for _, v := range models {
		state := types.OrderState{}
		if err := v.ConvertUp(&state); err != nil {
			log.Errorf("provide miner orders error")
		}
		list = append(list, state)
	}

	return list
}
