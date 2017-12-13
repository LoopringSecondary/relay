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
