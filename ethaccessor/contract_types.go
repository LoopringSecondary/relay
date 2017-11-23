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

package ethaccessor

import (
	"errors"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

func NewAbi(abiStr string) (abi.ABI, error) {
	a := abi.ABI{}
	err := a.UnmarshalJSON([]byte(abiStr))
	return a, err
}

type TransferEvent struct {
	From  common.Address `fieldName:"from"`
	To    common.Address `fieldName:"to"`
	Value *big.Int       `fieldName:"value"`
}

func (e *TransferEvent) ConvertDown() *types.TransferEvent {
	evt := &types.TransferEvent{}
	evt.From = types.HexToAddress(e.From.Hex())
	evt.To = types.HexToAddress(e.To.Hex())
	evt.Value = types.NewBigPtr(e.Value)

	return evt
}

type ApprovalEvent struct {
	Owner   common.Address `fieldName:"owner"`
	Spender common.Address `fieldName:"spender"`
	Value   *big.Int       `fieldName:"value"`
}

func (e *ApprovalEvent) ConvertDown() *types.ApprovalEvent {
	evt := &types.ApprovalEvent{}
	evt.Owner = types.HexToAddress(e.Owner.Hex())
	evt.Spender = types.HexToAddress(e.Spender.Hex())
	evt.Value = types.NewBigPtr(e.Value)

	return evt
}

type RingMinedEvent struct {
	RingIndex          *big.Int      `fieldName:"_ringIndex"`
	RingHash           types.Hash    `fieldName:"_ringhash"`
	Miner              types.Address `fieldName:"_miner"`
	FeeRecipient       types.Address `fieldName:"_feeRecipient"`
	IsRingHashReserved bool          `fieldName:"_isRinghashReserved"`
	OrderHashList      []types.Hash  `fieldName:"_orderHashList"`
	AmountsList        [][4]*big.Int `fieldName:"_amountsList"`
}

func (e *RingMinedEvent) ConvertDown() (*types.RingMinedEvent, []*types.OrderFilledEvent, error) {
	length := len(e.OrderHashList)

	if length != len(e.AmountsList) || length < 2 {
		return nil, nil, errors.New("ringMined event unpack error:orderHashList length invalid")
	}

	evt := &types.RingMinedEvent{}
	evt.RingIndex = types.NewBigPtr(e.RingIndex)
	evt.Ringhash = e.RingHash
	evt.Miner = e.Miner
	evt.FeeRecipient = e.FeeRecipient
	evt.IsRinghashReserved = e.IsRingHashReserved

	var list []*types.OrderFilledEvent
	for i := 0; i < length; i++ {
		var (
			fill                        types.OrderFilledEvent
			preOrderHash, nextOrderHash types.Hash
		)

		if i == 0 {
			preOrderHash = e.OrderHashList[length-1]
			nextOrderHash = e.OrderHashList[1]
		} else if i == length-1 {
			preOrderHash = e.OrderHashList[length-2]
			nextOrderHash = e.OrderHashList[0]
		} else {
			preOrderHash = e.OrderHashList[i-1]
			nextOrderHash = e.OrderHashList[i+1]
		}

		fill.Ringhash = e.RingHash
		fill.PreOrderHash = preOrderHash
		fill.OrderHash = e.OrderHashList[i]
		fill.NextOrderHash = nextOrderHash

		// _amountSList, _amountBList, _lrcRewardList, _lrcFeeList
		fill.RingIndex = types.NewBigPtr(e.RingIndex)
		fill.AmountS = types.NewBigPtr(e.AmountsList[i][0])
		fill.AmountB = types.NewBigPtr(e.AmountsList[i][1])
		fill.LrcReward = types.NewBigPtr(e.AmountsList[i][2])
		fill.LrcFee = types.NewBigPtr(e.AmountsList[i][3])

		list = append(list, &fill)
	}

	return evt, list, nil
}

type OrderCancelledEvent struct {
	OrderHash       types.Hash `fieldName:"_orderHash"`
	AmountCancelled *big.Int   `fieldName:"_amountCancelled"`
}

func (e *OrderCancelledEvent) ConvertDown() *types.OrderCancelledEvent {
	evt := &types.OrderCancelledEvent{}
	evt.OrderHash = e.OrderHash
	evt.AmountCancelled = types.NewBigPtr(e.AmountCancelled)

	return evt
}

type CutoffTimestampChangedEvent struct {
	Owner  types.Address `fieldName:"_address"`
	Cutoff *big.Int      `fieldName:"_cutoff"`
}

func (e *CutoffTimestampChangedEvent) ConvertDown() *types.CutoffEvent {
	evt := &types.CutoffEvent{}
	evt.Owner = e.Owner
	evt.Cutoff = types.NewBigPtr(e.Cutoff)
	return evt
}

type RinghashSubmitted struct {
	RingHash  []byte         `alias:"_ringhash"`
	RingMiner common.Address `alias:"_ringminer"`
}
