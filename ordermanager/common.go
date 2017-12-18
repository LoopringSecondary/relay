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
	"math/big"
)

var dustOrderValue int64

func newOrderEntity(state *types.OrderState, accessor *ethaccessor.EthNodeAccessor, mc marketcap.MarketCapProvider, blockNumber *big.Int) (*dao.Order, error) {
	blockNumberStr := blockNumberToString(blockNumber)

	state.DealtAmountS = big.NewInt(0)
	state.DealtAmountB = big.NewInt(0)
	state.SplitAmountS = big.NewInt(0)
	state.SplitAmountB = big.NewInt(0)
	state.CancelledAmountB = big.NewInt(0)

	// get order cancelled or filled amount from chain
	if cancelOrFilledAmount, err := accessor.GetCancelledOrFilled(state.RawOrder.Protocol, state.RawOrder.Hash, blockNumberStr); err != nil {
		return nil, fmt.Errorf("order manager,handle gateway order,order %s getCancelledOrFilled error:%s", state.RawOrder.Hash.Hex(), err.Error())
	} else {
		state.CancelledAmountS = cancelOrFilledAmount
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

func settleOrderStatus(state *types.OrderState, mc marketcap.MarketCapProvider) {
	if new(big.Int).Add(state.CancelledAmountS, state.DealtAmountS).Cmp(big.NewInt(0)) <= 0 {
		state.Status = types.ORDER_NEW
	} else {
		finished := isOrderFullFinished(state, mc)
		state.SettleFinishedStatus(finished)
	}
}

func isOrderFullFinished(state *types.OrderState, mc marketcap.MarketCapProvider) bool {
	var valueOfRemainAmount *big.Rat

	if state.RawOrder.BuyNoMoreThanAmountB {
		dealtAndSplitAmountB := new(big.Int).Add(state.DealtAmountB, state.SplitAmountB)
		cancelOrFilledAmountB := new(big.Int).Add(dealtAndSplitAmountB, state.CancelledAmountB)
		remainAmountB := new(big.Int).Sub(state.RawOrder.AmountB, cancelOrFilledAmountB)
		ratRemainAmountB := new(big.Rat).SetInt(remainAmountB)
		valueOfRemainAmount, _ = mc.LegalCurrencyValue(state.RawOrder.TokenB, ratRemainAmountB)
	} else {
		dealtAndSplitAmountS := new(big.Int).Add(state.DealtAmountS, state.SplitAmountS)
		cancelOrFilledAmountS := new(big.Int).Add(dealtAndSplitAmountS, state.CancelledAmountS)
		remainAmountS := new(big.Int).Sub(state.RawOrder.AmountS, cancelOrFilledAmountS)
		ratRemainAmountS := new(big.Rat).SetInt(remainAmountS)
		valueOfRemainAmount, _ = mc.LegalCurrencyValue(state.RawOrder.TokenS, ratRemainAmountS)
	}

	// todo: if valueOfRemainAmount is nil procedure of this may have problem
	minValue := big.NewInt(dustOrderValue)
	if valueOfRemainAmount == nil || valueOfRemainAmount.Cmp(new(big.Rat).SetInt(minValue)) > 0 {
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
