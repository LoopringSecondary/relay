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
	"fmt"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

var dustOrderValue int64

func newOrderEntity(state *types.OrderState, mc marketcap.MarketCapProvider, blockNumber *big.Int) (*dao.Order, error) {
	blockNumberStr := blockNumberToString(blockNumber)

	state.DealtAmountS = big.NewInt(0)
	state.DealtAmountB = big.NewInt(0)
	state.SplitAmountS = big.NewInt(0)
	state.SplitAmountB = big.NewInt(0)
	state.CancelledAmountB = big.NewInt(0)
	state.CancelledAmountS = big.NewInt(0)

	cancelAmount, dealtAmount, getAmountErr := getCancelledAndDealtAmount(state.RawOrder.Protocol, state.RawOrder.Hash, blockNumberStr)
	if getAmountErr != nil {
		return nil, getAmountErr
	}

	if state.RawOrder.BuyNoMoreThanAmountB {
		state.DealtAmountB = dealtAmount
		state.CancelledAmountB = cancelAmount
	} else {
		state.DealtAmountS = dealtAmount
		state.CancelledAmountS = cancelAmount
	}

	// check order finished status
	settleOrderStatus(state, mc)

	if blockNumber == nil {
		state.UpdatedBlock = big.NewInt(0)
	} else {
		state.UpdatedBlock = blockNumber
	}

	model := &dao.Order{}
	var err error
	model.Market, err = util.WrapMarketByAddress(state.RawOrder.TokenB.Hex(), state.RawOrder.TokenS.Hex())
	if err != nil {
		return nil, fmt.Errorf("order manager,newOrderEntity error:%s", err.Error())
	}
	model.ConvertDown(state)

	return model, nil
}

// 写入订单状态
func settleOrderStatus(state *types.OrderState, mc marketcap.MarketCapProvider) {
	zero := big.NewInt(0)
	finishAmountS := big.NewInt(0).Add(state.CancelledAmountS, state.DealtAmountS)
	totalAmountS := big.NewInt(0).Add(finishAmountS, state.SplitAmountS)
	finishAmountB := big.NewInt(0).Add(state.CancelledAmountB, state.DealtAmountB)
	totalAmountB := big.NewInt(0).Add(finishAmountB, state.SplitAmountB)
	totalAmount := big.NewInt(0).Add(totalAmountS, totalAmountB)
	if totalAmount.Cmp(zero) <= 0 {
		state.Status = types.ORDER_NEW
	} else {
		if isOrderFullFinished(state, mc) {
			state.Status = types.ORDER_FINISHED
		} else {
			state.Status = types.ORDER_PARTIAL
		}
	}
}

func isOrderFullFinished(state *types.OrderState, mc marketcap.MarketCapProvider) bool {
	remainedAmountS, _ := state.RemainedAmount()
	remainedValue, _ := mc.LegalCurrencyValue(state.RawOrder.TokenS, remainedAmountS)

	return isValueDusted(remainedValue)
}

// 订单需求量减去撮合量剩余价值是否为灰尘量，如果是则为finish，如果不是则为cancelled
func isOrderCancelled(state *types.OrderState, mc marketcap.MarketCapProvider) bool {
	if state.Status != types.ORDER_CANCEL && state.Status != types.ORDER_FINISHED {
		return false
	}

	totalAmountS, _ := state.DealtAndSplitAmount()
	totalValue, _ := mc.LegalCurrencyValue(state.RawOrder.TokenS, totalAmountS)

	return !isValueDusted(totalValue)
}

// todo: if valueOfRemainAmount is nil procedure of this may have problem
func isValueDusted(value *big.Rat) bool {
	minValue := big.NewInt(dustOrderValue)
	if value == nil || value.Cmp(new(big.Rat).SetInt(minValue)) > 0 {
		return false
	}

	return true
}

func blockNumberToString(blockNumber *big.Int) string {
	var blockNumberStr string
	if blockNumber == nil {
		blockNumberStr = "latest"
	} else {
		blockNumberStr = types.BigintToHex(blockNumber)
	}

	return blockNumberStr
}

func getCancelledAndDealtAmount(protocol common.Address, orderhash common.Hash, blockNumberStr string) (*big.Int, *big.Int, error) {
	var (
		cancelled, cancelOrFilled, dealt *big.Int
		err                              error
	)

	// get order cancelled amount from chain
	if cancelled, err = ethaccessor.GetCancelled(protocol, orderhash, blockNumberStr); err != nil {
		return nil, nil, fmt.Errorf("order manager,handle gateway order,order %s getCancelled error:%s", orderhash.Hex(), err.Error())
	}

	// get order cancelledOrFilled amount from chain
	if cancelOrFilled, err = ethaccessor.GetCancelledOrFilled(protocol, orderhash, blockNumberStr); err != nil {
		return nil, nil, fmt.Errorf("order manager,handle gateway order,order %s getCancelledOrFilled error:%s", orderhash.Hex(), err.Error())
	}

	if cancelOrFilled.Cmp(cancelled) < 0 {
		return nil, nil, fmt.Errorf("order manager,handle gateway order,order %s cancelOrFilledAmount:%s < cancelledAmount:%s", orderhash.Hex(), cancelOrFilled.String(), cancelled.String())
	}

	dealt = big.NewInt(0).Sub(cancelOrFilled, cancelled)

	return cancelled, dealt, nil
}
