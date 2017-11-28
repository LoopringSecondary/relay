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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
	"time"
)

type minerOrdersProvider struct {
	commonOpts         *config.CommonOptions
	rds                dao.RdsService
	ticker             *time.Ticker
	tick               int
	lastBlockNumber    *types.Big
	currentBlockNumber *types.Big
	blockNumberPeriod  *types.Big
	mtx                sync.Mutex
	quit               chan struct{}
}

func newMinerOrdersProvider(clearTick, blockNumberPeriod int, commonOpts *config.CommonOptions, rds dao.RdsService) *minerOrdersProvider {
	provider := &minerOrdersProvider{}
	provider.tick = clearTick
	provider.blockNumberPeriod = types.NewBigWithInt(blockNumberPeriod)
	provider.commonOpts = commonOpts
	provider.rds = rds
	provider.currentBlockNumber = types.NewBigPtr(provider.commonOpts.DefaultBlockNumber)

	if entity, err := provider.rds.FindLatestBlock(); err == nil {
		var block types.Block
		if err := entity.ConvertDown(&block); err != nil {
			log.Fatalf("ordermanager: orders provider,error%s", err.Error())
		}
		provider.currentBlockNumber = types.NewBigPtr(block.BlockNumber)
	}
	provider.lastBlockNumber = provider.currentBlockNumber

	return provider
}

func (p *minerOrdersProvider) start() {
	p.quit = make(chan struct{})
	// todo : get ticker time from config
	p.ticker = time.NewTicker(10 * time.Second)

	log.Debugf("ordermanager provider ticker period %d", p.tick)
	go func() {
		for {
			select {
			case <-p.ticker.C:
				p.unMarkOrders()
			}
		}
	}()
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

func (p *minerOrdersProvider) markOrders(orderhashs []common.Hash) error {
	var orderhashstrs []string

	for _, v := range orderhashs {
		orderhashstrs = append(orderhashstrs, v.Hex())
	}

	return p.rds.MarkMinerOrders(orderhashstrs, p.currentBlockNumber.Int64())
}

func (p *minerOrdersProvider) unMarkOrders() error {
	cmp := big.NewInt(0).Add(p.blockNumberPeriod.BigInt(), p.lastBlockNumber.BigInt())
	des := new(big.Int).Sub(p.currentBlockNumber.BigInt(), cmp)
	if des.Cmp(big.NewInt(1)) < 0 {
		return nil
	}

	log.Debugf("current block number:%s,last block number period:%s,period block number", p.currentBlockNumber.BigInt().String(), p.lastBlockNumber.BigInt().String(), p.blockNumberPeriod.BigInt().String())

	return p.rds.UnMarkMinerOrders(des.Int64())
}

func (p *minerOrdersProvider) getOrders(tokenS, tokenB common.Address, orderhashs []common.Hash) []types.OrderState {
	var list []types.OrderState

	filterStatus := []uint8{types.ORDER_FINISHED.Value(), types.ORDER_CUTOFF.Value(), types.ORDER_CANCEL.Value()}

	models, err := p.rds.GetOrdersForMiner(tokenS.Hex(), tokenB.Hex(), filterStatus)
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

	p.lastBlockNumber = p.currentBlockNumber

	return list
}
