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
	"errors"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/dao"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
	"math/big"
)

type forkProcessor struct {
	dao       dao.RdsService
	contracts map[types.Address]*chainclient.LoopringProtocolImpl
	tokens    map[types.Address]*chainclient.Erc20Token
}

func newForkProcess(rds dao.RdsService) *forkProcessor {
	processor := &forkProcessor{}
	processor.dao = rds
	processor.contracts = make(map[types.Address]*chainclient.LoopringProtocolImpl)
	processor.tokens = make(map[types.Address]*chainclient.Erc20Token)

	for _, impl := range miner.MinerInstance.Loopring.LoopringImpls {
		processor.contracts[impl.Address] = impl
	}

	for _, token := range miner.MinerInstance.Loopring.Tokens {
		processor.tokens[token.Address] = token
	}

	return processor
}

func (p *forkProcessor) fork(event *chainclient.ForkedEvent) error {
	from := event.ForkBlock.Int64()
	to := event.DetectedBlock.Int64()

	orderList, err := p.dao.GetOrdersWithBlockNumberRange(from, to)
	if err != nil {
		return err
	}

	forkBlockNumber := big.NewInt(from)
	for _, v := range orderList {
		state := &types.OrderState{}
		if err := v.ConvertUp(state); err != nil {
			log.Errorf("order manager fork error:%s", err.Error())
			continue
		}

		// todo(fuk):get contract cancelOrFilledMap remainAmount and approval token amount,compare and get min
		if state.RawOrder.BuyNoMoreThanAmountB == true {
			remain, allowance, balance, err := p.getAmounts(state, state.RawOrder.TokenB, forkBlockNumber)
			if err != nil {
				log.Debugf("order manager fork error %s", err.Error())
				continue
			}
			state.RemainedAmountB = getMinAmount(remain, allowance, balance)
		} else {
			remain, allowance, balance, err := p.getAmounts(state, state.RawOrder.TokenS, forkBlockNumber)
			if err != nil {
				log.Debugf("order manager fork error %s", err.Error())
				continue
			}
			state.RemainedAmountS = getMinAmount(remain, allowance, balance)
		}

		state.CalculateRemainAmount()
		state.BlockNumber = forkBlockNumber

		newOrderModel := dao.Order{}
		if err := newOrderModel.ConvertDown(state); err != nil {
			log.Debugf("order manager fork error:%s", err.Error())
			continue
		}

		if err := p.dao.Update(newOrderModel); err != nil {
			log.Debugf("order manager fork erorr:%s", err.Error())
			continue
		}
	}
	// todo find order in contract
	return nil
}

// getAmounts return remain,allowance,balance
func (p *forkProcessor) getAmounts(state *types.OrderState, tokenAddress types.Address, blockNumber *big.Int) (*big.Int, *big.Int, *big.Int, error) {
	var remain, allowance, balance *big.Int

	contractAddress := state.RawOrder.Protocol
	blockNumStr := blockNumber.String()

	impl, ok := p.contracts[contractAddress]
	if !ok {
		return nil, nil, nil, errors.New("order manager fork error:contract address doesn't exist")
	}

	token, ok := p.tokens[tokenAddress]
	if !ok {
		return nil, nil, nil, errors.New("order manager fork get token error")
	}

	if err := impl.GetCancelledOrFilled.Call(&remain, blockNumStr, state.RawOrder.Hash); err != nil {
		return nil, nil, nil, err
	}

	// owner & spender
	if err := token.Allowance.Call(&allowance, blockNumStr, tokenAddress, contractAddress); err != nil {
		return nil, nil, nil, err
	}
	if err := token.BalanceOf.Call(&balance, blockNumStr, tokenAddress); err != nil {
		return nil, nil, nil, err
	}

	return remain, allowance, balance, nil
}

func getMinAmount(a1, a2, a3 *big.Int) *big.Int {
	min := a1

	if min.Cmp(a2) > 0 {
		min = a2
	}
	if min.Cmp(a3) > 0 {
		min = a3
	}

	return min
}
