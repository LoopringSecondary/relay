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
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

func newOrderEntity(state *types.OrderState, accessor *ethaccessor.EthNodeAccessor, mc *marketcap.MarketCapProvider, blockNumber *big.Int) (*dao.Order, error) {
	blockNumberStr := blockNumberToString(blockNumber)

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

	state.DealtAmountS = big.NewInt(0)
	state.DealtAmountB = big.NewInt(0)
	state.CancelledAmountB = big.NewInt(0)

	model := &dao.Order{}
	var err error
	model.Market, err = util.WrapMarketByAddress(state.RawOrder.TokenB.Hex(), state.RawOrder.TokenS.Hex())
	if err != nil {
		return nil, fmt.Errorf("order manager,newOrderEntity error:%s", err.Error())
	}
	model.ConvertDown(state)

	return model, nil
}

func settleOrderStatus(state *types.OrderState, mc *marketcap.MarketCapProvider) {
	if state.CancelledAmountS.Cmp(big.NewInt(0)) == 0 {
		state.Status = types.ORDER_NEW
	} else {
		finished := isOrderFullFinished(state, mc)
		state.SettleFinishedStatus(finished)
	}
}

func isOrderFullFinished(state *types.OrderState, mc *marketcap.MarketCapProvider) bool {
	var valueOfRemainAmount *big.Rat

	if state.RawOrder.BuyNoMoreThanAmountB {
		cancelOrFilledAmountB := new(big.Int).Add(state.DealtAmountB, state.CancelledAmountB)
		remainAmountB := new(big.Int).Sub(state.RawOrder.AmountB, cancelOrFilledAmountB)
		ratRemainAmountB := new(big.Rat).SetInt(remainAmountB)
		price := mc.GetMarketCap(state.RawOrder.TokenB)
		valueOfRemainAmount = new(big.Rat).Mul(price, ratRemainAmountB)
	} else {
		cancelOrFilledAmountS := new(big.Int).Add(state.DealtAmountS, state.CancelledAmountS)
		remainAmountS := new(big.Int).Sub(state.RawOrder.AmountS, cancelOrFilledAmountS)
		ratRemainAmountS := new(big.Rat).SetInt(remainAmountS)
		price := mc.GetMarketCap(state.RawOrder.TokenS)
		valueOfRemainAmount = new(big.Rat).Mul(price, ratRemainAmountS)
	}

	// todo: get compare number from config
	if valueOfRemainAmount.Cmp(big.NewRat(1, 1)) > 0 {
		return false
	}

	return true
}

func isFundInsufficient(state *types.OrderState, mc *marketcap.MarketCapProvider) bool {
	price := mc.GetMarketCap(state.RawOrder.TokenS)
	value := new(big.Rat).Mul(price, state.AvailableAmountS)

	// todo: get from config
	if value.Cmp(new(big.Rat).SetInt64(1)) > 0 {
		return false
	}

	return true
}

func calculateAmountS(state *types.OrderState, req *ethaccessor.BatchErc20Req) {
	var available, cancelOrFilledRatS *big.Rat

	balance := new(big.Rat).SetInt(req.Balance.BigInt())
	allowance := new(big.Rat).SetInt(req.Allowance.BigInt())
	amountRatS := new(big.Rat).SetInt(state.RawOrder.AmountS)

	if state.RawOrder.BuyNoMoreThanAmountB {
		cancelOrFilledB := new(big.Int).Add(state.DealtAmountB, state.CancelledAmountB)
		cancelOrFilledRatB := new(big.Rat).SetInt(cancelOrFilledB)
		cancelOrFilledRatS = new(big.Rat).Mul(state.RawOrder.Price, cancelOrFilledRatB)
	} else {
		cancelOrFilledS := new(big.Int).Add(state.DealtAmountS, state.CancelledAmountS)
		cancelOrFilledRatS = new(big.Rat).SetInt(cancelOrFilledS)
	}

	if cancelOrFilledRatS.Cmp(amountRatS) >= 0 {
		available = new(big.Rat).SetInt64(0)
	} else {
		available = new(big.Rat).Sub(amountRatS, cancelOrFilledRatS)
	}

	state.AvailableAmountS = getMinAmount(available, balance, allowance)
}

func getMinAmount(a1, a2, a3 *big.Rat) *big.Rat {
	min := a1

	if min.Cmp(a2) > 0 {
		min = a2
	}
	if min.Cmp(a3) > 0 {
		min = a3
	}

	return min
}

func generateErc20Req(state *types.OrderState, spender common.Address, blockNumber *big.Int) *ethaccessor.BatchErc20Req {
	var batchReq ethaccessor.BatchErc20Req
	batchReq.Spender = spender
	batchReq.Owner = state.RawOrder.Owner
	batchReq.Token = state.RawOrder.TokenS
	if blockNumber == nil {
		batchReq.BlockParameter = "latest"
	} else {
		batchReq.BlockParameter = types.BigintToHex(blockNumber)
	}

	return &batchReq
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
