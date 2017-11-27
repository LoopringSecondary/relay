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
	"github.com/Loopring/relay/chainclient"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type forkProcessor struct {
	dao      dao.RdsService
	accessor *ethaccessor.EthNodeAccessor
}

func newForkProcess(rds dao.RdsService, accessor *ethaccessor.EthNodeAccessor) *forkProcessor {
	processor := &forkProcessor{}
	processor.dao = rds

	return processor
}

func (p *forkProcessor) fork(event *ethaccessor.ForkedEvent) error {
	from := event.ForkBlock.Int64()
	to := event.DetectedBlock.Int64()

	if err := p.dao.RollBackRingMined(from, to); err != nil {
		log.Errorf("order manager fork error:%s", err.Error())
	}
	if err := p.dao.RollBackFill(from, to); err != nil {
		log.Errorf("order manager fork error:%s", err.Error())
	}
	if err := p.dao.RollBackCancel(from, to); err != nil {
		log.Errorf("order manager fork error:%s", err.Error())
	}
	if err := p.dao.RollBackCutoff(from, to); err != nil {
		log.Errorf("order manager fork error:%s", err.Error())
	}

	orderList, err := p.dao.GetOrdersWithBlockNumberRange(from, to)
	if err != nil {
		return err
	}

	forkBlockNumber := big.NewInt(from)
	forkBlockNumStr := forkBlockNumber.String()
	for _, v := range orderList {
		state := &types.OrderState{}
		if err := v.ConvertUp(state); err != nil {
			log.Errorf("order manager fork error:%s", err.Error())
			continue
		}

		// todo(fuk):get contract cancelOrFilledMap remainAmount and approval token amount,compare and get min
		if state.RawOrder.BuyNoMoreThanAmountB == true {
			remain, err := p.accessor.GetCancelledOrFilled(state.RawOrder.Protocol, state.RawOrder.Hash, forkBlockNumStr)
			if err != nil {
				log.Debugf("order manager fork error:%s", err.Error())
				continue
			}
			state.RemainedAmountB = remain // getMinAmount(remain, allowance, balance)
		} else {
			remain, err := p.accessor.GetCancelledOrFilled(state.RawOrder.Protocol, state.RawOrder.Hash, forkBlockNumStr)
			if err != nil {
				log.Debugf("order manager fork error:%s", err.Error())
				continue
			}
			batchReq := ethaccessor.BatchErc20Req{}
			batchReq.Spender, err = p.accessor.GetSenderAddress(state.RawOrder.Protocol)
			if err != nil {
				log.Debugf("order manager fork error:%s", err.Error())
				continue
			}
			batchReq.Owner = state.RawOrder.Owner
			batchReq.Token = state.RawOrder.TokenS
			batchReq.BlockParameter = forkBlockNumStr

			p.accessor.BatchErc20BalanceAndAllowance([]*ethaccessor.BatchErc20Req{&batchReq})
			if err != nil || batchReq.AllowanceErr != nil || batchReq.BalanceErr != nil {
				log.Debugf("order manager fork error:%s", err.Error())
				continue
			}

			state.RemainedAmountS = getMinAmount(remain, batchReq.Allowance.BigInt(), batchReq.Balance.BigInt())
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
