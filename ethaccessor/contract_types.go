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
	From  common.Address `json:"from" alias:"from"`
	To    common.Address `json:"to" alias:"to"`
	Value *big.Int       `json:"value" alias:"value"`
}

func (e *TransferEvent) ConvertDown() *types.TransferEvent {
	evt := &types.TransferEvent{}
	evt.From = types.HexToAddress(e.From.Hex())
	evt.To = types.HexToAddress(e.To.Hex())
	evt.Value = types.NewBigPtr(e.Value)

	return evt
}

type ApprovalEvent struct {
	Owner   common.Address `json:"owner" alias:"owner"`
	Spender common.Address `json:"spender" alias:"spender"`
	Value   *big.Int       `json:"value" alias:"value"`
}

func (e *ApprovalEvent) ConvertDown() *types.ApprovalEvent {
	evt := &types.ApprovalEvent{}
	evt.Owner = types.HexToAddress(e.Owner.Hex())
	evt.Spender = types.HexToAddress(e.Spender.Hex())
	evt.Value = types.NewBigPtr(e.Value)

	return evt
}

//go:generate gencodec -type RingMinedEvent -field-override ringMinedEventMarshaling -out gen_ringminedevent_json.go
type RingMinedEvent struct {
	RingIndex          *big.Int       `json:"ringIndex" alias:"_ringIndex" gencodec:"required"`
	Time               *big.Int       `json:"time" alias:"_time" gencodec:"required"`
	Blocknumber        *big.Int       `json:"blockNumber" alias:"_blocknumber" gencodec:"required"`
	Ringhash           types.Hash     `json:"ringHash" alias:"_ringhash" gencodec:"required"`
	Miner              common.Address `json:"miner" alias:"_miner" gencodec:"required"`
	FeeRecepient       common.Address `json:"feeRecepient" alias:"_feeRecepient" gencodec:"required"`
	IsRinghashReserved bool           `json:"isRinghashReserved" alias:"_isRinghashReserved" gencodec:"required"`
}

type ringMinedEventMarshaling struct {
	RingIndex   *types.Big
	Time        *types.Big
	Blocknumber *types.Big
}

func (e *RingMinedEvent) ConvertDown() *types.RingMinedEvent {
	evt := &types.RingMinedEvent{}
	evt.RingIndex = types.NewBigPtr(e.RingIndex)
	evt.Ringhash = e.Ringhash
	evt.Miner = types.HexToAddress(e.Miner.Hex())
	evt.FeeRecipient = types.HexToAddress(e.FeeRecepient.Hex())
	evt.IsRinghashReserved = e.IsRinghashReserved

	return evt
}

//go:generate gencodec -type OrderFilledEvent -field-override orderFilledEventMarshaling -out gen_orderfilledevent_json.go
type OrderFilledEvent struct {
	RingIndex     *big.Int `json:"ringIndex" alias:"_ringIndex" gencodec:"required"`
	Time          *big.Int `json:"time" alias:"_time" gencodec:"required"`
	Blocknumber   *big.Int `json:"blockNumber" alias:"_blocknumber" gencodec:"required"`
	Ringhash      []byte   `json:"ringHash" alias:"_ringhash" gencodec:"required"`
	PreOrderHash  []byte   `json:"preOrderHash" alias:"_prevOrderHash" gencodec:"required"`
	OrderHash     []byte   `json:"orderHash" alias:"_orderHash" gencodec:"required"`
	NextOrderHash []byte   `json:"nextOrderHash" alias:"_nextOrderHash" gencodec:"required"`
	AmountS       *big.Int `json:"amountS" alias:"_amountS" gencodec:"required"` // 存量数据
	AmountB       *big.Int `json:"amountB" alias:"_amountB" gencodec:"required"` // 存量数据
	LrcReward     *big.Int `json:"lrcReward" alias:"_lrcReward" gencodec:"required"`
	LrcFee        *big.Int `json:"lrcFee" alias:"_lrcFee" gencodec:"required"`
}

type orderFilledEventMarshaling struct {
	RingIndex   *types.Big
	Time        *types.Big
	Blocknumber *types.Big
	AmountS     *types.Big
	AmountB     *types.Big
	LrcReward   *types.Big
	LrcFee      *types.Big
}

func (e *OrderFilledEvent) ConvertDown() *types.OrderFilledEvent {
	evt := &types.OrderFilledEvent{}

	evt.Ringhash = types.BytesToHash(e.Ringhash)
	evt.PreOrderHash = types.BytesToHash(e.PreOrderHash)
	evt.OrderHash = types.BytesToHash(e.OrderHash)
	evt.NextOrderHash = types.BytesToHash(e.NextOrderHash)

	evt.RingIndex = types.NewBigPtr(e.RingIndex)
	evt.AmountS = types.NewBigPtr(e.AmountS)
	evt.AmountB = types.NewBigPtr(e.AmountB)
	evt.LrcReward = types.NewBigPtr(e.LrcReward)
	evt.LrcFee = types.NewBigPtr(e.LrcFee)

	return evt
}

//go:generate gencodec -type OrderCancelledEvent -field-override orderCancelledEventMarshaling -out gen_ordercancelledevent_json.go
type OrderCancelledEvent struct {
	Time            *big.Int `json:"time" alias:"_time" gencodec:"required"`
	Blocknumber     *big.Int `json:"blockNumber" alias:"_blocknumber" gencodec:"required"`
	OrderHash       []byte   `json:"orderHash" alias:"_orderHash" gencodec:"required"`
	AmountCancelled *big.Int `json:"amountCancelled" alias:"_amountCancelled" gencodec:"required"`
}

type orderCancelledEventMarshaling struct {
	Time            *types.Big
	Blocknumber     *types.Big
	AmountCancelled *types.Big
}

func (e *OrderCancelledEvent) ConvertDown() *types.OrderCancelledEvent {
	evt := &types.OrderCancelledEvent{}
	evt.OrderHash = types.BytesToHash(e.OrderHash)
	evt.AmountCancelled = types.NewBigPtr(e.AmountCancelled)

	return evt
}

//go:generate gencodec -type CutoffTimestampChangedEvent -field-override cutoffTimestampChangedEventtMarshaling -out gen_cutofftimestampevent_json.go
type CutoffTimestampChangedEvent struct {
	Time        *big.Int       `json:"time" alias:"_time" gencodec:"required"`
	Blocknumber *big.Int       `json:"blockNumber" alias:"_blocknumber" gencodec:"required"`
	Owner       common.Address `json:"address" alias:"_address" gencodec:"required"`
	Cutoff      *big.Int       `json:"cutoff" alias:"_cutoff" gencodec:"required"`
}

type cutoffTimestampChangedEventtMarshaling struct {
	Time        *types.Big
	Blocknumber *types.Big
	Cutoff      *types.Big
}

func (e *CutoffTimestampChangedEvent) ConvertDown() *types.CutoffEvent {
	evt := &types.CutoffEvent{}
	//todo:
	//evt.Owner = e.ContractAddress
	evt.Cutoff = types.NewBigPtr(e.Cutoff)
	return evt
}

type RinghashSubmitted struct {
	RingHash  []byte         `alias:"_ringhash"`
	RingMiner common.Address `alias:"_ringminer"`
}
