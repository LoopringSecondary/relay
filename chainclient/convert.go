package chainclient

import (
	"errors"
	"github.com/Loopring/ringminer/types"
	"math/big"
)

func (e *OrderCancelledEvent) ConvertDown(r *types.OrderState) error {
	rawOrderHashHex := r.RawOrder.Hash.Hex()
	evtOrderHashHex := types.BytesToHash(e.OrderHash).Hex()
	if rawOrderHashHex != evtOrderHashHex {
		return errors.New("raw orderhash hex:" + rawOrderHashHex + "not equal event orderhash hex:" + evtOrderHashHex)
	}

	// orderState更新时间
	r.RawOrder.Timestamp = e.Time

	latestVd, err := r.LatestVersion()
	if err != nil {
		return err
	}

	v := types.VersionData{}
	v.Status = types.ORDER_CANCEL
	v.Block = e.Blocknumber

	if r.RawOrder.BuyNoMoreThanAmountB {
		remainAmountB := new(big.Int).Sub(latestVd.RemainedAmountB, e.AmountCancelled)
		if remainAmountB.Cmp(big.NewInt(0)) < 0 {
			return errors.New("order:" + rawOrderHashHex + " cancel amountB->" + e.AmountCancelled.String() + " error")
		}
		v.RemainedAmountB = remainAmountB
	} else {
		remainAmountS := new(big.Int).Sub(latestVd.RemainedAmountS, e.AmountCancelled)
		if remainAmountS.Cmp(big.NewInt(0)) < 0 {
			return errors.New("order:" + rawOrderHashHex + " cancel amountS->" + e.AmountCancelled.String() + " error")
		}
		v.RemainedAmountS = remainAmountS
	}

	r.AddVersion(v)

	return nil
}

func (e *CutoffTimestampChangedEvent) ConvertDown(r *types.OrderState) {

}
