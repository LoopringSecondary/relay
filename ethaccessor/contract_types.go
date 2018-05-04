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

const (
	EVENT_RING_MINED           = "RingMined"
	EVENT_ORDER_CANCELLED      = "OrderCancelled"
	EVENT_CUTOFF_ALL           = "AllOrdersCancelled"
	EVENT_CUTOFF_PAIR          = "OrdersCancelled"
	EVENT_TOKEN_REGISTERED     = "TokenRegistered"
	EVENT_TOKEN_UNREGISTERED   = "TokenUnregistered"
	EVENT_ADDRESS_AUTHORIZED   = "AddressAuthorized"
	EVENT_ADDRESS_DEAUTHORIZED = "AddressDeauthorized"
	EVENT_TRANSFER             = "Transfer"
	EVENT_APPROVAL             = "Approval"
	EVENT_WETH_DEPOSIT         = "Deposit"
	EVENT_WETH_WITHDRAWAL      = "Withdrawal"
)

const (
	METHOD_UNKNOWN         = ""
	METHOD_SUBMIT_RING     = "submitRing"
	METHOD_CANCEL_ORDER    = "cancelOrder"
	METHOD_CUTOFF_ALL      = "cancelAllOrders"
	METHOD_CUTOFF_PAIR     = "cancelAllOrdersByTradingPair"
	METHOD_WETH_DEPOSIT    = "deposit"
	METHOD_WETH_WITHDRAWAL = "withdraw"
	METHOD_APPROVE         = "approve"
	METHOD_TRANSFER        = "transfer"
)

type TransferEvent struct {
	Sender   common.Address `fieldName:"from" fieldId:"0"`
	Receiver common.Address `fieldName:"to" fieldId:"1"`
	Value    *big.Int       `fieldName:"value" fieldId:"2"`
}

func (e *TransferEvent) ConvertDown() *types.TransferEvent {
	evt := &types.TransferEvent{}
	evt.Sender = e.Sender
	evt.Receiver = e.Receiver
	evt.Amount = e.Value

	return evt
}

type ApprovalEvent struct {
	Owner   common.Address `fieldName:"owner" fieldId:"0"`
	Spender common.Address `fieldName:"spender" fieldId:"1"`
	Value   *big.Int       `fieldName:"value" fieldId:"2"`
}

func (e *ApprovalEvent) ConvertDown() *types.ApprovalEvent {
	evt := &types.ApprovalEvent{}
	evt.Owner = e.Owner
	evt.Spender = e.Spender
	evt.Amount = e.Value

	return evt
}

/// @dev Event to emit if a ring is successfully mined.
/// _amountsList is an array of:
/// [_amountS, _amountB, _lrcReward, _lrcFee, splitS, splitB].
//event RingMined(
//uint            _ringIndex,
//bytes32 indexed _ringHash,
//address         _miner,
//address         _feeRecipient,
//bytes32[]       _orderInfoList
//);
// orderInfoList = new bytes32[](ringSize * 7);
// orderInfoList[q++] = bytes32(state.orderHash);
//orderInfoList[q++] = bytes32(state.owner);
//orderInfoList[q++] = bytes32(state.tokenS);
//orderInfoList[q++] = bytes32(state.fillAmountS);
//orderInfoList[q++] = bytes32(state.lrcReward);
//orderInfoList[q++] = bytes32(
//state.lrcFeeState > 0 ? int(state.lrcFeeState) : -int(state.lrcReward)
//);
//orderInfoList[q++] = bytes32(
//state.splitS > 0 ? int(state.splitS) : -int(state.splitB)
//);

type RingMinedEvent struct {
	RingIndex     *big.Int       `fieldName:"_ringIndex" fieldId:"0"`
	RingHash      common.Hash    `fieldName:"_ringhash" fieldId:"1"`
	Miner         common.Address `fieldName:"_miner" fieldId:"2"`
	FeeRecipient  common.Address `fieldName:"_feeRecipient" fieldId:"3"`
	OrderInfoList [][32]uint8    `fieldName:"_orderInfoList" fieldId:"4"`
}

func (e *RingMinedEvent) ConvertDown() (*types.RingMinedEvent, []*types.OrderFilledEvent, error) {
	length := len(e.OrderInfoList)
	idx := length / 7

	if length%7 != 0 || idx < 2 {
		return nil, nil, errors.New("ringMined event unpack error:orderInfoList length invalid")
	}

	evt := &types.RingMinedEvent{}
	evt.RingIndex = e.RingIndex
	evt.Ringhash = e.RingHash
	evt.Miner = e.Miner
	evt.FeeRecipient = e.FeeRecipient

	var list []*types.OrderFilledEvent
	totalLrcFee := big.NewInt(0)

	firstFill := 0
	lastFill := idx - 1
	for i := 0; i < idx; i++ {
		var (
			fill                        types.OrderFilledEvent
			preOrderHash, nextOrderHash common.Hash
			start                       = i * 7
			tokenB                      common.Address
			amountB                     *big.Int
		)

		if i == firstFill {
			preOrderHash = safeHash(e.OrderInfoList[lastFill*7])
			nextOrderHash = safeHash(e.OrderInfoList[(i+1)*7])
			tokenB = safeAddress(e.OrderInfoList[lastFill*7+2])
			amountB = safeBig(e.OrderInfoList[lastFill*7+3])
		} else if i == lastFill {
			preOrderHash = safeHash(e.OrderInfoList[(i-1)*7])
			nextOrderHash = safeHash(e.OrderInfoList[firstFill*7])
			tokenB = safeAddress(e.OrderInfoList[firstFill*7+2])
			amountB = safeBig(e.OrderInfoList[firstFill*7+3])
		} else {
			preOrderHash = safeHash(e.OrderInfoList[(i-1)*7])
			nextOrderHash = safeHash(e.OrderInfoList[(i+1)*7])
			tokenB = safeAddress(e.OrderInfoList[(i-1)*7+2])
			amountB = safeBig(e.OrderInfoList[(i-1)*7+3])
		}

		fill.Ringhash = e.RingHash
		fill.RingIndex = e.RingIndex
		fill.FillIndex = big.NewInt(int64(i))

		fill.OrderHash = safeHash(e.OrderInfoList[start])
		fill.PreOrderHash = preOrderHash
		fill.NextOrderHash = nextOrderHash

		fill.Owner = safeAddress(e.OrderInfoList[start+1])
		fill.TokenS = safeAddress(e.OrderInfoList[start+2])
		fill.TokenB = tokenB
		fill.AmountS = safeBig(e.OrderInfoList[start+3])
		fill.AmountB = amountB
		fill.LrcReward = safeBig(e.OrderInfoList[start+4])

		// lrcFee or lrcReward, if >= 0 lrcFee, else lrcReward
		if lrcFeeOrReward := safeBig(e.OrderInfoList[start+5]); lrcFeeOrReward.Cmp(big.NewInt(0)) > 0 {
			fill.LrcFee = lrcFeeOrReward
		} else {
			fill.LrcFee = big.NewInt(0)
		}

		// splitS or splitB: if > 0 splitS, else splitB
		if split := safeBig(e.OrderInfoList[start+6]); split.Cmp(big.NewInt(0)) > 0 {
			fill.SplitS = split
			fill.SplitB = big.NewInt(0)
		} else {
			fill.SplitS = big.NewInt(0)
			fill.SplitB = new(big.Int).Mul(split, big.NewInt(-1))
		}

		totalLrcFee = totalLrcFee.Add(totalLrcFee, fill.LrcFee)
		list = append(list, &fill)
	}

	evt.TotalLrcFee = totalLrcFee
	evt.TradeAmount = idx

	return evt, list, nil
}

// sateHash contract bytes32 to common.hash
func safeHash(bytes [32]uint8) common.Hash {
	return common.Hash(bytes)
}

// safeAbsBig return abs value and isNeg
func safeBig(bytes [32]uint8) *big.Int {
	num := new(big.Int).SetBytes(bytes[:])
	if bytes[0] > uint8(128) {
		num.Xor(types.MaxUint256, num)
		num.Not(num)
	}
	return num
}

//// safeBig contract bytes32 to *big.int
//func safeBig(bytes [32]uint8) *big.Int {
//	var newbytes []byte = bytes[0:]
//	return new(big.Int).SetBytes(newbytes)
//}

// safeAddress contract bytes32 to common.address
func safeAddress(bytes [32]uint8) common.Address {
	var newbytes []byte = bytes[0:]
	return common.BytesToAddress(newbytes)
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
	evt.Amount = e.Value

	return evt
}

// event  Withdrawal(address indexed src, uint wad);
type WethWithdrawalEvent struct {
	SrcAddress common.Address `fieldName:"src" fieldId:"0"`
	Value      *big.Int       `fieldName:"wad" fieldId:"1"`
}

func (e *WethWithdrawalEvent) ConvertDown() *types.WethWithdrawalEvent {
	evt := &types.WethWithdrawalEvent{}
	evt.Amount = e.Value

	return evt
}

type SubmitRingMethod struct {
	AddressList        [][4]common.Address `fieldName:"addressList" fieldId:"0"`   // owner,tokenS,tokenB(authAddress)
	UintArgsList       [][6]*big.Int       `fieldName:"uintArgsList" fieldId:"1"`  // amountS, amountB, validSince (second),validUntil (second), lrcFee, rateAmountS.
	Uint8ArgsList      [][1]uint8          `fieldName:"uint8ArgsList" fieldId:"2"` // marginSplitPercentageList
	BuyNoMoreThanBList []bool              `fieldName:"buyNoMoreThanAmountBList" fieldId:"3"`
	VList              []uint8             `fieldName:"vList" fieldId:"4"`
	RList              [][32]uint8         `fieldName:"rList" fieldId:"5"`
	SList              [][32]uint8         `fieldName:"sList" fieldId:"6"`
	Miner              common.Address      `fieldName:"miner" fieldId:"7"`
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
		order.WalletAddress = m.AddressList[i][2]
		order.AuthAddr = m.AddressList[i][3]

		order.AmountS = m.UintArgsList[i][0]
		order.AmountB = m.UintArgsList[i][1]
		order.ValidSince = m.UintArgsList[i][2]
		order.ValidUntil = m.UintArgsList[i][3]
		order.LrcFee = m.UintArgsList[i][4]
		// order.rateAmountS

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
	AddressList    [5]common.Address `fieldName:"addresses" fieldId:"0"`   //  owner, tokenS, tokenB, authAddr
	OrderValues    [6]*big.Int       `fieldName:"orderValues" fieldId:"1"` //  amountS, amountB, validSince (second), validUntil (second), lrcFee, and cancelAmount
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
	order.WalletAddress = m.AddressList[3]
	order.AuthAddr = m.AddressList[4]

	order.AmountS = m.OrderValues[0]
	order.AmountB = m.OrderValues[1]
	order.ValidSince = m.OrderValues[2]
	order.ValidUntil = m.OrderValues[3]
	order.LrcFee = m.OrderValues[4]
	cancelAmount := m.OrderValues[5]

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
	evt.Amount = e.Value

	return evt
}

type ApproveMethod struct {
	Spender common.Address `fieldName:"spender" fieldId:"0"`
	Value   *big.Int       `fieldName:"value" fieldId:"1"`
}

func (e *ApproveMethod) ConvertDown() *types.ApprovalEvent {
	evt := &types.ApprovalEvent{}
	evt.Spender = e.Spender
	evt.Amount = e.Value

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
	evt.Amount = e.Value

	return evt
}

type ProtocolAddress struct {
	Version         string
	ContractAddress common.Address

	LrcTokenAddress common.Address

	TokenRegistryAddress common.Address

	//NameRegistryAddress common.Address

	DelegateAddress common.Address
}
