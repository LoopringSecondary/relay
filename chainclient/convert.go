package chainclient

import "github.com/Loopring/ringminer/types"

// 必须先在orderbook中查到orderState,如果没有则新建一个
func (e *OrderFilledEvent) ConvertDown(r *types.OrderState) {
	r.RawOrder.Hash = types.BytesToHash(e.OrderHash)
	r.RawOrder.Timestamp = e.Time
	// r.RawOrder.AmountB = e.AmountB
	// r.RawOrder.AmountS =
	// r.RemainedAmountB = r.RawOrder.AmountB.Div(e.AmountB)
}

func (e *OrderCancelledEvent) ConvertDown(r *types.OrderState) {

}

func (e *CutoffTimestampChangedEvent) ConvertDown(r *types.OrderState) {

}
