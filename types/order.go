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

package types

import (
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/log"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type OrderStatus uint8

const (
	ORDER_UNKNOWN OrderStatus = iota
	ORDER_NEW
	ORDER_PARTIAL
	ORDER_FINISHED
	ORDER_CANCEL
	ORDER_CUTOFF
)

//订单原始信息
/**
1、是否整体成交
2、指定成交对象，对方单的hash
3、分润比例 是否需要设置
4、成交方向 待定
5、过期时间，使用块数
*/

//go:generate gencodec -type Order -field-override orderMarshaling -out gen_order_json.go
type Order struct {
	Protocol              common.Address `json:"protocol" gencodec:"required"` // 智能合约地址
	TokenS                common.Address `json:"tokenS" gencodec:"required"`   // 卖出erc20代币智能合约地址
	TokenB                common.Address `json:"tokenB" gencodec:"required"`   // 买入erc20代币智能合约地址
	AmountS               *big.Int       `json:"amountS" gencodec:"required"`  // 卖出erc20代币数量上限
	AmountB               *big.Int       `json:"amountB" gencodec:"required"`  // 买入erc20代币数量上限
	Timestamp             *big.Int       `json:"timestamp" gencodec:"required"`
	Ttl                   *big.Int       `json:"ttl" gencodec:"required"` // 订单过期时间
	Salt                  *big.Int       `json:"salt" gencodec:"required"`
	LrcFee                *big.Int       `json:"lrcFee" ` // 交易总费用,部分成交的费用按该次撮合实际卖出代币额与比例计算
	BuyNoMoreThanAmountB  bool           `json:"buyNoMoreThanAmountB" gencodec:"required"`
	MarginSplitPercentage uint8          `json:"marginSplitPercentage" gencodec:"required"` // 不为0时支付给交易所的分润比例，否则视为100%
	V                     uint8          `json:"v" gencodec:"required"`
	R                     Bytes32        `json:"r" gencodec:"required"`
	S                     Bytes32        `json:"s" gencodec:"required"`
	Price                 *big.Rat       `json:"price"`
	Owner                 common.Address `json:"owner"`
	Hash                  common.Hash    `json:"hash"`
}

//go:generate gencodec -type OrderJsonRequest -field-override orderJsonRequestMarshaling -out gen_order_request_json.go
type OrderJsonRequest struct {
	Protocol              common.Address `json:"protocol" gencodec:"required"` // 智能合约地址
	TokenS                common.Address `json:"tokenS" gencodec:"required"`   // 卖出erc20代币智能合约地址
	TokenB                common.Address `json:"tokenB" gencodec:"required"`   // 买入erc20代币智能合约地址
	AmountS               *big.Int       `json:"amountS" gencodec:"required"`  // 卖出erc20代币数量上限
	AmountB               *big.Int       `json:"amountB" gencodec:"required"`  // 买入erc20代币数量上限
	Timestamp             int64          `json:"timestamp" gencodec:"required"`
	Ttl                   int64          `json:"ttl" gencodec:"required"` // 订单过期时间
	Salt                  int64          `json:"salt" gencodec:"required"`
	LrcFee                *big.Int       `json:"lrcFee" ` // 交易总费用,部分成交的费用按该次撮合实际卖出代币额与比例计算
	BuyNoMoreThanAmountB  bool           `json:"buyNoMoreThanAmountB" gencodec:"required"`
	MarginSplitPercentage uint8          `json:"marginSplitPercentage" gencodec:"required"` // 不为0时支付给交易所的分润比例，否则视为100%
	V                     uint8          `json:"v" gencodec:"required"`
	R                     Bytes32        `json:"r" gencodec:"required"`
	S                     Bytes32        `json:"s" gencodec:"required"`
	Price                 *big.Rat       `json:"price"`
	Owner                 common.Address `json:"owner"`
	Hash                  common.Hash    `json:"hash"`
}

type orderMarshaling struct {
	AmountS   *Big
	AmountB   *Big
	Timestamp *Big
	Ttl       *Big
	Salt      *Big
	LrcFee    *Big
}

type orderJsonRequestMarshaling struct {
	AmountS *Big
	AmountB *Big
	LrcFee  *Big
}

func (o *Order) GenerateHash() common.Hash {
	h := &common.Hash{}

	buyNoMoreThanAmountB := byte(0)
	if o.BuyNoMoreThanAmountB {
		buyNoMoreThanAmountB = byte(1)
	}

	hashBytes := crypto.GenerateHash(
		o.Protocol.Bytes(),
		o.Owner.Bytes(),
		o.TokenS.Bytes(),
		o.TokenB.Bytes(),
		common.LeftPadBytes(o.AmountS.Bytes(), 32),
		common.LeftPadBytes(o.AmountB.Bytes(), 32),
		common.LeftPadBytes(o.Timestamp.Bytes(), 32),
		common.LeftPadBytes(o.Ttl.Bytes(), 32),
		common.LeftPadBytes(o.Salt.Bytes(), 32),
		common.LeftPadBytes(o.LrcFee.Bytes(), 32),
		[]byte{buyNoMoreThanAmountB},
		[]byte{byte(o.MarginSplitPercentage)},
	)
	h.SetBytes(hashBytes)

	return *h
}

func (o *Order) GenerateAndSetSignature(singerAddr common.Address) error {
	if IsZeroHash(o.Hash) {
		o.Hash = o.GenerateHash()
	}

	if sig, err := crypto.Sign(o.Hash.Bytes(), singerAddr); nil != err {
		return err
	} else {
		v, r, s := crypto.SigToVRS(sig)
		o.V = uint8(v)
		o.R = BytesToBytes32(r)
		o.S = BytesToBytes32(s)
		return nil
	}
}

func (o *Order) ValidateSignatureValues() bool {
	return crypto.ValidateSignatureValues(byte(o.V), o.R.Bytes(), o.S.Bytes())
}

func (o *Order) SignerAddress() (common.Address, error) {
	address := &common.Address{}
	if IsZeroHash(o.Hash) {
		o.Hash = o.GenerateHash()
	}

	sig, _ := crypto.VRSToSig(o.V, o.R.Bytes(), o.S.Bytes())

	if addressBytes, err := crypto.SigToAddress(o.Hash.Bytes(), sig); nil != err {
		log.Errorf("type,order signer address error:%s", err.Error())
		return *address, err
	} else {
		address.SetBytes(addressBytes)
		return *address, nil
	}
}

func (o *Order) GeneratePrice() {
	o.Price = new(big.Rat).SetFrac(o.AmountS, o.AmountB)
}

// 根据big.Rat价格计算big.int remainAmount
// buyNoMoreThanAmountB == true  已知remainAmountB计算remainAmountS
// buyNoMoreThanAmountB == false 已知remainAmountS计算remainAmountB
func (ord *OrderState) CalculateRemainAmount() {
	const RATE = 1.0e18

	price, _ := ord.RawOrder.Price.Float64()
	price = price * RATE
	bigPrice := big.NewInt(int64(price))
	bigRate := big.NewInt(RATE)

	if ord.RawOrder.BuyNoMoreThanAmountB == true {
		beenRateAmountB := new(big.Int).Mul(ord.DealtAmountB, bigPrice)
		ord.DealtAmountS = new(big.Int).Div(beenRateAmountB, bigRate)
	} else {
		beenRateAmountS := new(big.Int).Mul(ord.DealtAmountS, bigRate)
		ord.DealtAmountB = new(big.Int).Div(beenRateAmountS, bigPrice)
	}
}

//RateAmountS、FeeSelection 需要提交到contract
type FilledOrder struct {
	OrderState       OrderState `json:"orderState" gencodec:"required"`
	FeeSelection     uint8      `json:"feeSelection"`     //0 -> lrc
	RateAmountS      *big.Rat   `json:"rateAmountS"`      //提交需要
	AvailableAmountS *big.Rat   `json:"availableAmountS"` //需要，也是用于计算fee
	AvailableAmountB *big.Rat   //需要，也是用于计算fee
	FillAmountS      *big.Rat   `json:"fillAmountS"`
	FillAmountB      *big.Rat   `json:"fillAmountB"` //计算需要
	LrcReward        *big.Rat   `json:"lrcReward"`
	LrcFee           *big.Rat   `json:"lrcFee"`
	FeeS             *big.Rat   `json:"feeS"`
	//FeeB             *EnlargedInt
	LegalFee *big.Rat `json:"legalFee"` //法币计算的fee

	SPrice *big.Rat `json:"SPrice"`
	BPrice *big.Rat `json:"BPrice"`

	AvailableLrcBalance    *big.Rat
	AvailableTokenSBalance *big.Rat
}

func (filledOrder *FilledOrder) SetAvailableAmount() {
	filledOrder.AvailableAmountS, filledOrder.AvailableAmountB = filledOrder.OrderState.RemainedAmount()
	sellPrice := new(big.Rat).SetFrac(filledOrder.OrderState.RawOrder.AmountS, filledOrder.OrderState.RawOrder.AmountB)
	availableBalance := new(big.Rat).Set(filledOrder.AvailableTokenSBalance)
	if availableBalance.Cmp(filledOrder.AvailableAmountS) < 0 {
		filledOrder.AvailableAmountS = availableBalance
		filledOrder.AvailableAmountB.Mul(filledOrder.AvailableAmountS, new(big.Rat).Inv(sellPrice))
	}
	if filledOrder.OrderState.RawOrder.BuyNoMoreThanAmountB {
		filledOrder.AvailableAmountS.Mul(filledOrder.AvailableAmountB, sellPrice)
	} else {
		filledOrder.AvailableAmountB.Mul(filledOrder.AvailableAmountS, new(big.Rat).Inv(sellPrice))
	}
}

// 从[]byte解析时使用json.Unmarshal
type OrderState struct {
	RawOrder         Order       `json:"rawOrder"`
	UpdatedBlock     *big.Int    `json:"updatedBlock"`
	DealtAmountS     *big.Int    `json:"dealtAmountS"`
	DealtAmountB     *big.Int    `json:"dealtAmountB"`
	CancelledAmountS *big.Int    `json:"cancelledAmountS"`
	CancelledAmountB *big.Int    `json:"cancelledAmountB"`
	AvailableAmountS *big.Int    `json:"availableAmountS"`
	Status           OrderStatus `json:"status"`
	BroadcastTime    int         `json:"broadcastTime"`
}

type OrderDelayList struct {
	OrderHash    []common.Hash
	DelayedCount int
}

// 根据是否完全成交确定订单状态
func (ord *OrderState) SettleFinishedStatus(isFullFinished bool) {
	if isFullFinished {
		ord.Status = ORDER_FINISHED
	} else {
		ord.Status = ORDER_PARTIAL
	}
}

func (orderState *OrderState) RemainedAmount() (remainedAmountS *big.Rat, remainedAmountB *big.Rat) {
	remainedAmountS = new(big.Rat)
	remainedAmountB = new(big.Rat)
	if orderState.RawOrder.BuyNoMoreThanAmountB {
		reducedAmountB := new(big.Rat)
		reducedAmountB.Add(reducedAmountB, new(big.Rat).SetInt(orderState.DealtAmountB)).
			Add(reducedAmountB, new(big.Rat).SetInt(orderState.CancelledAmountB))
		sellPrice := new(big.Rat).SetFrac(orderState.RawOrder.AmountS, orderState.RawOrder.AmountB)
		remainedAmountB.Sub(new(big.Rat).SetInt(orderState.RawOrder.AmountB), reducedAmountB)
		remainedAmountS.Mul(remainedAmountB, sellPrice)
	} else {
		reducedAmountS := new(big.Rat)
		reducedAmountS.Add(reducedAmountS, new(big.Rat).SetInt(orderState.DealtAmountS)).
			Add(reducedAmountS, new(big.Rat).SetInt(orderState.CancelledAmountS))
		buyPrice := new(big.Rat).SetFrac(orderState.RawOrder.AmountB, orderState.RawOrder.AmountS)
		remainedAmountS.Sub(new(big.Rat).SetInt(orderState.RawOrder.AmountS), reducedAmountS)
		remainedAmountB.Mul(remainedAmountS, buyPrice)
	}

	return remainedAmountS, remainedAmountB
}

func ToOrder(request *OrderJsonRequest) *Order {
	order := &Order{}
	order.Protocol = request.Protocol
	order.TokenS = request.TokenS
	order.TokenB = request.TokenB
	order.AmountS = request.AmountS
	order.AmountB = request.AmountB
	order.Timestamp = big.NewInt(request.Timestamp)
	order.Ttl = big.NewInt(request.Ttl)
	order.Salt = big.NewInt(request.Salt)
	order.LrcFee = request.LrcFee
	order.BuyNoMoreThanAmountB = request.BuyNoMoreThanAmountB
	order.MarginSplitPercentage = request.MarginSplitPercentage
	order.V = request.V
	order.R = request.R
	order.S = request.S
	order.Owner = request.Owner
	return order
}
