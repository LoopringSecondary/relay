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
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

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

func generateErc20Req(state *types.OrderState, spender common.Address) *ethaccessor.BatchErc20Req {
	var batchReq ethaccessor.BatchErc20Req
	batchReq.Spender = spender
	batchReq.Owner = state.RawOrder.Owner
	batchReq.Token = state.RawOrder.TokenS
	batchReq.BlockParameter = "latest"

	return &batchReq
}
