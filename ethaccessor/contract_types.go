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
	"fmt"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

func NewAbi(abiStr string) (*abi.ABI, error) {
	a := &abi.ABI{}
	err := a.UnmarshalJSON([]byte(abiStr))
	return a, err
}

type TransferEvent struct {
	Sender   common.Address `fieldName:"from" fieldId:"0"`
	Receiver common.Address `fieldName:"to" fieldId:"1"`
	Value    *big.Int       `fieldName:"value" fieldId:"2"`
}

func (e *TransferEvent) ConvertDown() *types.TransferEvent {
	evt := &types.TransferEvent{}
	evt.Sender = e.Sender
	evt.Receiver = e.Receiver
	evt.Value = e.Value

	return evt
}

type ApprovalEvent struct {
	Owner   common.Address `fieldName:"owner" fieldId:"0"`
	Spender common.Address `fieldName:"spender fieldId:"1""`
	Value   *big.Int       `fieldName:"value" fieldId:"2"`
}

func (e *ApprovalEvent) ConvertDown() *types.ApprovalEvent {
	evt := &types.ApprovalEvent{}
	evt.Owner = e.Owner
	evt.Spender = e.Spender
	evt.Value = e.Value

	return evt
}

type RingMinedEvent struct {
	RingIndex     *big.Int       `fieldName:"_ringIndex" fieldId:"0"`
	RingHash      common.Hash    `fieldName:"_ringhash" fieldId:"1"`
	Miner         common.Address `fieldName:"_miner" fieldId:"2"`
	FeeRecipient  common.Address `fieldName:"_feeRecipient" fieldId:"3"`
	OrderHashList [][32]uint8    `fieldName:"_orderHashList" fieldId:"4"`
	AmountsList   [][6]*big.Int  `fieldName:"_amountsList" fieldId:"5"`
}

func (e *RingMinedEvent) ConvertDown() (*types.RingMinedEvent, []*types.OrderFilledEvent, error) {
	length := len(e.OrderHashList)

	if length != len(e.AmountsList) || length < 2 {
		return nil, nil, errors.New("ringMined event unpack error:orderHashList length invalid")
	}

	evt := &types.RingMinedEvent{}
	evt.RingIndex = e.RingIndex
	evt.Ringhash = e.RingHash
	evt.Miner = e.Miner
	evt.FeeRecipient = e.FeeRecipient

	var list []*types.OrderFilledEvent
	lrcFee := big.NewInt(0)
	for i := 0; i < length; i++ {
		var (
			fill                        types.OrderFilledEvent
			preOrderHash, nextOrderHash common.Hash
		)

		if i == 0 {
			preOrderHash = common.Hash(e.OrderHashList[length-1])
			nextOrderHash = common.Hash(e.OrderHashList[1])
		} else if i == length-1 {
			preOrderHash = common.Hash(e.OrderHashList[length-2])
			nextOrderHash = common.Hash(e.OrderHashList[0])
		} else {
			preOrderHash = common.Hash(e.OrderHashList[i-1])
			nextOrderHash = common.Hash(e.OrderHashList[i+1])
		}

		fill.Ringhash = e.RingHash
		fill.PreOrderHash = preOrderHash
		fill.OrderHash = common.Hash(e.OrderHashList[i])
		fill.NextOrderHash = nextOrderHash

		// [_amountS, _amountB, _lrcReward, _lrcFee, splitS, splitB]. amountS&amountB为单次成交量
		fill.RingIndex = e.RingIndex
		fill.AmountS = e.AmountsList[i][0]
		fill.AmountB = e.AmountsList[i][1]
		fill.LrcReward = e.AmountsList[i][2]
		fill.LrcFee = e.AmountsList[i][3]
		fill.SplitS = e.AmountsList[i][4]
		fill.SplitB = e.AmountsList[i][5]
		fill.FillIndex = big.NewInt(int64(i))

		lrcFee = lrcFee.Add(lrcFee, fill.LrcFee)
		list = append(list, &fill)
	}

	evt.TotalLrcFee = lrcFee
	evt.TradeAmount = length

	return evt, list, nil
}

type OrderCancelledEvent struct {
	OrderHash       common.Hash `fieldName:"_orderHash" fieldId:"0"`
	AmountCancelled *big.Int    `fieldName:"_amountCancelled" fieldId:"1"` // amountCancelled为多次取消累加总量，根据orderhash以及amountCancelled可以确定其唯一性
}

func (e *OrderCancelledEvent) ConvertDown() *types.OrderCancelledEvent {
	evt := &types.OrderCancelledEvent{}
	evt.OrderHash = e.OrderHash
	evt.AmountCancelled = e.AmountCancelled

	return evt
}

type CutoffEvent struct {
	Owner  common.Address `fieldName:"_address" fieldId:"0"`
	Cutoff *big.Int       `fieldName:"_cutoff" fieldId:"1"`
}

func (e *CutoffEvent) ConvertDown() *types.CutoffEvent {
	evt := &types.CutoffEvent{}
	evt.Owner = e.Owner
	evt.Cutoff = e.Cutoff

	return evt
}

type CutoffPairEvent struct {
	Owner  common.Address `fieldName:"_address" fieldId:"0"`
	Token1 common.Address `fieldName:"_token1" fieldId:"1"`
	Token2 common.Address `fieldName:"_token2" fieldId:"2"`
	Cutoff *big.Int       `fieldName:"_cutoff" fieldId:"3"`
}

func (e *CutoffPairEvent) ConvertDown() *types.CutoffPairEvent {
	evt := &types.CutoffPairEvent{}
	evt.Owner = e.Owner
	evt.Token1 = e.Token1
	evt.Token2 = e.Token2
	evt.Cutoff = e.Cutoff

	return evt
}

type TokenRegisteredEvent struct {
	Token  common.Address `fieldName:"addr" fieldId:"0"`
	Symbol string         `fieldName:"symbol" fieldId:"1"`
}

func (e *TokenRegisteredEvent) ConvertDown() *types.TokenRegisterEvent {
	evt := &types.TokenRegisterEvent{}
	evt.Token = e.Token
	evt.Symbol = e.Symbol

	return evt
}

type TokenUnRegisteredEvent struct {
	Token  common.Address `fieldName:"addr" fieldId:"0"`
	Symbol string         `fieldName:"symbol" fieldId:"1"`
}

func (e *TokenUnRegisteredEvent) ConvertDown() *types.TokenUnRegisterEvent {
	evt := &types.TokenUnRegisterEvent{}
	evt.Token = e.Token
	evt.Symbol = e.Symbol

	return evt
}

type AddressAuthorizedEvent struct {
	ContractAddress common.Address `fieldName:"addr" fieldId:"0"`
	Number          int            `fieldName:"number" fieldId:"1"`
}

func (e *AddressAuthorizedEvent) ConvertDown() *types.AddressAuthorizedEvent {
	evt := &types.AddressAuthorizedEvent{}
	evt.Protocol = e.ContractAddress
	evt.Number = e.Number

	return evt
}

type AddressDeAuthorizedEvent struct {
	ContractAddress common.Address `fieldName:"addr" fieldId:"0"`
	Number          int            `fieldName:"number" fieldId:"1"`
}

func (e *AddressDeAuthorizedEvent) ConvertDown() *types.AddressDeAuthorizedEvent {
	evt := &types.AddressDeAuthorizedEvent{}
	evt.Protocol = e.ContractAddress
	evt.Number = e.Number

	return evt
}

// event  Deposit(address indexed dst, uint wad);
type WethDepositEvent struct {
	DstAddress common.Address `fieldName:"dst" fieldId:"0"` // 充值到哪个地址
	Value      *big.Int       `fieldName:"wad" fieldId:"1"`
}

func (e *WethDepositEvent) ConvertDown() *types.WethDepositEvent {
	evt := &types.WethDepositEvent{}
	evt.Value = e.Value

	return evt
}

// event  Withdrawal(address indexed src, uint wad);
type WethWithdrawalEvent struct {
	SrcAddress common.Address `fieldName:"src" fieldId:"0"`
	Value      *big.Int       `fieldName:"wad" fieldId:"1"`
}

func (e *WethWithdrawalEvent) ConvertDown() *types.WethWithdrawalEvent {
	evt := &types.WethWithdrawalEvent{}
	evt.Value = e.Value

	return evt
}

type SubmitRingMethod struct {
	AddressList        [][3]common.Address `fieldName:"addressList" fieldId:"0"`   // owner,tokenS,tokenB(authAddress)
	UintArgsList       [][7]*big.Int       `fieldName:"uintArgsList" fieldId:"1"`  // amountS, amountB, validSince (second),validUntil (second), lrcFee, rateAmountS, and walletId.
	Uint8ArgsList      [][1]uint8          `fieldName:"uint8ArgsList" fieldId:"2"` // marginSplitPercentageList
	BuyNoMoreThanBList []bool              `fieldName:"buyNoMoreThanAmountBList" fieldId:"3"`
	VList              []uint8             `fieldName:"vList" fieldId:"4"`
	RList              [][32]uint8         `fieldName:"rList" fieldId:"5"`
	SList              [][32]uint8         `fieldName:"sList" fieldId:"6"`
	MinerId            *big.Int            `fieldName:"minerId" fieldId:"7"`
	FeeSelections      uint16              `fieldName:"feeSelections" fieldId:"8"`
	Protocol           common.Address
}

// should add protocol, miner, feeRecipient
func (m *SubmitRingMethod) ConvertDown() ([]*types.Order, error) {
	var list []*types.Order
	length := len(m.AddressList)
	vsrLength := 2*length + 1

	if length != len(m.UintArgsList) || length != len(m.Uint8ArgsList) || vsrLength != len(m.VList) || vsrLength != len(m.SList) || vsrLength != len(m.RList) || length < 2 {
		return nil, fmt.Errorf("ringMined method unpack error:orders length invalid")
	}

	for i := 0; i < length; i++ {
		var order types.Order

		order.Protocol = m.Protocol
		order.Owner = m.AddressList[i][0]
		order.TokenS = m.AddressList[i][1]
		if i == length-1 {
			order.TokenB = m.AddressList[0][1]
		} else {
			order.TokenB = m.AddressList[i+1][1]
		}
		order.AuthAddr = m.AddressList[i][2]

		order.AmountS = m.UintArgsList[i][0]
		order.AmountB = m.UintArgsList[i][1]
		order.ValidSince = m.UintArgsList[i][2]
		order.ValidUntil = m.UintArgsList[i][3]
		order.LrcFee = m.UintArgsList[i][4]
		// order.rateAmountS
		order.WalletId = m.UintArgsList[i][6]

		order.MarginSplitPercentage = m.Uint8ArgsList[i][0]

		order.BuyNoMoreThanAmountB = m.BuyNoMoreThanBList[i]

		order.V = m.VList[i]
		order.R = m.RList[i]
		order.S = m.SList[i]

		list = append(list, &order)
	}

	return list, nil
}

type CancelOrderMethod struct {
	AddressList    [4]common.Address `fieldName:"addresses" fieldId:"0"`   //  owner, tokenS, tokenB, authAddr
	OrderValues    [7]*big.Int       `fieldName:"orderValues" fieldId:"1"` //  amountS, amountB, validSince (second), validUntil (second), lrcFee, walletId, and cancelAmount
	BuyNoMoreThanB bool              `fieldName:"buyNoMoreThanAmountB" fieldId:"2"`
	MarginSplit    uint8             `fieldName:"marginSplitPercentage" fieldId:"3"`
	V              uint8             `fieldName:"v" fieldId:"4"`
	R              [32]byte          `fieldName:"r" fieldId:"5"`
	S              [32]byte          `fieldName:"s" fieldId:"6"`
}

// todo(fuk): modify internal cancelOrderMethod and implement related functions
func (m *CancelOrderMethod) ConvertDown() (*types.Order, *big.Int, error) {
	var order types.Order

	order.Owner = m.AddressList[0]
	order.TokenS = m.AddressList[1]
	order.TokenB = m.AddressList[2]
	order.AuthAddr = m.AddressList[3]

	order.AmountS = m.OrderValues[0]
	order.AmountB = m.OrderValues[1]
	order.ValidSince = m.OrderValues[2]
	order.ValidUntil = m.OrderValues[3]
	order.LrcFee = m.OrderValues[4]
	order.WalletId = m.OrderValues[5]
	cancelAmount := m.OrderValues[6]

	order.BuyNoMoreThanAmountB = bool(m.BuyNoMoreThanB)
	order.MarginSplitPercentage = m.MarginSplit

	order.V = m.V
	order.S = m.S
	order.R = m.R

	return &order, cancelAmount, nil
}

type CutoffMethod struct {
	Cutoff *big.Int `fieldName:"cutoff" fieldId:"0"`
}

func (method *CutoffMethod) ConvertDown() *types.CutoffEvent {
	evt := &types.CutoffEvent{}
	evt.Cutoff = method.Cutoff

	return evt
}

type CutoffPairMethod struct {
	Token1 common.Address `fieldName:"token1" fieldId:"0"`
	Token2 common.Address `fieldName:"token2" fieldId:"1"`
	Cutoff *big.Int       `fieldName:"cutoff" fieldId:"2"`
}

func (method *CutoffPairMethod) ConvertDown() *types.CutoffPairEvent {
	evt := &types.CutoffPairEvent{}
	evt.Cutoff = method.Cutoff
	evt.Token1 = method.Token1
	evt.Token2 = method.Token2

	return evt
}

type WethWithdrawalMethod struct {
	Value *big.Int `fieldName:"wad" fieldId:"0"`
}

func (e *WethWithdrawalMethod) ConvertDown() *types.WethWithdrawalEvent {
	evt := &types.WethWithdrawalEvent{}
	evt.Value = e.Value

	return evt
}

type ApproveMethod struct {
	Spender common.Address `fieldName:"spender" fieldId:"0"`
	Value   *big.Int       `fieldName:"value" fieldId:"1"`
}

func (e *ApproveMethod) ConvertDown() *types.ApprovalEvent {
	evt := &types.ApprovalEvent{}
	evt.Spender = e.Spender
	evt.Value = e.Value

	return evt
}

// function transfer(address to, uint256 value) public returns (bool);
type TransferMethod struct {
	Receiver common.Address `fieldName:"to" fieldId:"0"`
	Value    *big.Int       `fieldName:"value" fieldId:"1"`
}

func (e *TransferMethod) ConvertDown() *types.TransferEvent {
	evt := &types.TransferEvent{}
	evt.Receiver = e.Receiver
	evt.Value = e.Value

	return evt
}

type ProtocolAddress struct {
	Version         string
	ContractAddress common.Address

	LrcTokenAddress common.Address

	TokenRegistryAddress common.Address

	NameRegistryAddress common.Address

	DelegateAddress common.Address
}
