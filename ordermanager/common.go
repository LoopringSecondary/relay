package ordermanager

import (
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

func calculateAmountS(state *types.OrderState, req *ethaccessor.BatchErc20Req) {
	cancelOrFilled := new(big.Int).Add(state.DealtAmountS, state.CancelledAmountS)
	available := new(big.Int).Sub(state.RawOrder.AmountS, cancelOrFilled)
	state.AvailableAmountS = getMinAmount(available, req.Allowance.BigInt(), req.Balance.BigInt())
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

func generateErc20Req(state *types.OrderState, spender common.Address) *ethaccessor.BatchErc20Req {
	var batchReq ethaccessor.BatchErc20Req
	batchReq.Spender = spender
	batchReq.Owner = state.RawOrder.Owner
	batchReq.Token = state.RawOrder.TokenS
	batchReq.BlockParameter = "latest"

	return &batchReq
}
