package chainclient

import (
	"github.com/Loopring/ringminer/types"
	"errors"
	"math/big"
)

// 必须先在orderbook中查到orderState,如果没有则新建一个
func (e *OrderFilledEvent) ConvertDown(r *types.OrderState) error {
	rawOrderHashHex := r.RawOrder.Hash.Hex()
	evtOrderHashHex := types.BytesToHash(e.OrderHash).Hex()
	if rawOrderHashHex != evtOrderHashHex {
		return errors.New("raw orderhash hex:"+rawOrderHashHex+"not equal event orderhash hex:"+evtOrderHashHex)
	}

	// orderState更新时间
	r.RawOrder.Timestamp = e.Time

	rawAmountS := types.NewEnlargedInt(r.RawOrder.AmountS)
	rawAmountB := types.NewEnlargedInt(r.RawOrder.AmountB)
	evtAmountS := types.NewEnlargedInt(e.AmountS)
	evtAmountB := types.NewEnlargedInt(e.AmountB)
	remainAmountS := rawAmountS.Sub(rawAmountS, evtAmountS).Value
	remainAmountB := rawAmountB.Sub(rawAmountB, evtAmountB).Value

	if remainAmountS.Cmp(big.NewInt(0)) < 0 {
		return errors.New("orderhash:"+rawOrderHashHex+" remainAmountS " + remainAmountS.String() + "error")
	}
	if remainAmountB.Cmp(big.NewInt(0)) < 0 {
		return errors.New("orderhash:"+rawOrderHashHex+" remainAmountB " + remainAmountB.String() + "error")
	}

	v := types.VersionData{}
	v.Block = e.Blocknumber
	v.RemainedAmountS = remainAmountS
	v.RemainedAmountB = remainAmountB
	v.Status = types.ORDER_PARTIAL
	// todo: judge whether finished

	r.States = append(r.States, v)
	return nil
}

func (e *OrderCancelledEvent) ConvertDown(r *types.OrderState) error {
	rawOrderHashHex := r.RawOrder.Hash.Hex()
	evtOrderHashHex := types.BytesToHash(e.OrderHash).Hex()
	if rawOrderHashHex != evtOrderHashHex {
		return errors.New("raw orderhash hex:"+rawOrderHashHex+"not equal event orderhash hex:"+evtOrderHashHex)
	}

	// orderState更新时间
	r.RawOrder.Timestamp = e.Time

	// todo: calculate remain amount s or b
	//rawAmountS := types.NewEnlargedInt(r.RawOrder.AmountS)
	//rawAmountB := types.NewEnlargedInt(r.RawOrder.AmountB)
	//cancelAmount := types.NewEnlargedInt(e.AmountCancelled)

	v := types.VersionData{}
	v.Status = types.ORDER_CANCEL
	v.Block = e.Blocknumber

	// todo
	//v.RemainedAmountS =
	//v.RemainedAmountB =

	r.States = append(r.States, v)
	return nil
}

func (e *CutoffTimestampChangedEvent) ConvertDown(r *types.OrderState) {

}
