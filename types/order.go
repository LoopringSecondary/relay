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
	"errors"
	"github.com/Loopring/ringminer/crypto"
	"github.com/Loopring/ringminer/log"
	"math/big"
)

type OrderStatus uint8

const (
	ORDER_UNKNOWN OrderStatus = iota
	ORDER_NEW
	ORDER_PENDING
	ORDER_FINISHED
	ORDER_CANCEL
	ORDER_REJECT
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
	Protocol              Address  `json:"protocol" gencodec:"required"` // 智能合约地址
	TokenS                Address  `json:"tokenS" gencodec:"required"`   // 卖出erc20代币智能合约地址
	TokenB                Address  `json:"tokenB" gencodec:"required"`   // 买入erc20代币智能合约地址
	AmountS               *big.Int `json:"amountS" gencodec:"required"`  // 卖出erc20代币数量上限
	AmountB               *big.Int `json:"amountB" gencodec:"required"`  // 买入erc20代币数量上限
	Timestamp             *big.Int `json:"timestamp" gencodec:"required"`
	Ttl                   *big.Int `json:"ttl" gencodec:"required"` // 订单过期时间
	Salt                  *big.Int `json:"salt" gencodec:"required"`
	LrcFee                *big.Int `json:"lrcFee" ` // 交易总费用,部分成交的费用按该次撮合实际卖出代币额与比例计算
	BuyNoMoreThanAmountB  bool     `json:"buyNoMoreThanAmountB" gencodec:"required"`
	MarginSplitPercentage uint8    `json:"marginSplitPercentage" gencodec:"required"` // 不为0时支付给交易所的分润比例，否则视为100%
	V                     uint8    `json:"v" gencodec:"required"`
	R                     Sign     `json:"r" gencodec:"required"`
	S                     Sign     `json:"s" gencodec:"required"`
	Price                 *big.Rat `json:"rate"`
	Owner                 Address  `json:"owner"`
	Hash                  Hash     `json:"hash"`
}

type orderMarshaling struct {
	AmountS   *Big
	AmountB   *Big
	Timestamp *Big
	Ttl       *Big
	Salt      *Big
	LrcFee    *Big
}

func (o *Order) GenerateHash() Hash {
	h := &Hash{}

	buyNoMoreThanAmountB := byte(0)
	if o.BuyNoMoreThanAmountB {
		buyNoMoreThanAmountB = byte(1)
	}

	hashBytes := crypto.CryptoInstance.GenerateHash(
		o.Protocol.Bytes(),
		o.Owner.Bytes(),
		o.TokenS.Bytes(),
		o.TokenB.Bytes(),
		LeftPadBytes(o.AmountS.Bytes(), 32),
		LeftPadBytes(o.AmountB.Bytes(), 32),
		LeftPadBytes(o.Timestamp.Bytes(), 32),
		LeftPadBytes(o.Ttl.Bytes(), 32),
		LeftPadBytes(o.Salt.Bytes(), 32),
		LeftPadBytes(o.LrcFee.Bytes(), 32),
		[]byte{buyNoMoreThanAmountB},
		[]byte{byte(o.MarginSplitPercentage)},
	)
	h.SetBytes(hashBytes)

	return *h
}

func (o *Order) GenerateAndSetSignature(pkBytes []byte) error {
	if o.Hash.IsZero() {
		o.Hash = o.GenerateHash()
	}

	if sig, err := crypto.CryptoInstance.Sign(o.Hash.Bytes(), pkBytes); nil != err {
		return err
	} else {
		v, r, s := crypto.CryptoInstance.SigToVRS(sig)
		o.V = uint8(v)
		o.R = BytesToSign(r)
		o.S = BytesToSign(s)
		return nil
	}
}

func (o *Order) ValidateSignatureValues() bool {
	return crypto.CryptoInstance.ValidateSignatureValues(byte(o.V), o.R.Bytes(), o.S.Bytes())
}

func (o *Order) SignerAddress() (Address, error) {
	address := &Address{}
	if o.Hash.IsZero() {
		o.Hash = o.GenerateHash()
	}

	sig, _ := crypto.CryptoInstance.VRSToSig(o.V, o.R.Bytes(), o.S.Bytes())
	log.Debugf("orderstate.hash:%s", o.Hash.Hex())

	if addressBytes, err := crypto.CryptoInstance.SigToAddress(o.Hash.Bytes(), sig); nil != err {
		log.Errorf("error:%s", err.Error())
		return *address, err
	} else {
		address.SetBytes(addressBytes)
		return *address, nil
	}
}

func (o *Order) GeneratePrice() {
	o.Price = new(big.Rat).SetFrac(o.AmountB, o.AmountS)
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

	//FullFilled	bool	//this order is fullfilled
}

//todo: impl it
func (o *FilledOrder) IsFullFilled() bool {
	return true
}

// 从[]byte解析时使用json.Unmarshal
type OrderState struct {
	RawOrder Order         `json:"rawOrder"`
	States   []VersionData `json:"states"`
}

//go:generate gencodec -type VersionData -field-override versionDataMarshaling -out gen_versiondata_json.go
type VersionData struct {
	RemainedAmountS *big.Int    `json:"remainedAmountS" gencodec:"required"`
	RemainedAmountB *big.Int    `json:"remainedAmountB" gencodec:"required"`
	Block           *big.Int    `json:"block"`
	Status          OrderStatus `json:"status"`
}

type versionDataMarshaling struct {
	RemainedAmountS *Big
	RemainedAmountB *Big
	Block           *Big
}

func (ord *OrderState) LatestVersion() (VersionData, error) {
	length := len(ord.States)
	if length < 1 {
		return VersionData{}, errors.New("no version data")
	}

	return ord.States[length-1], nil
}

// 添加新版本，保证所有版本顺序递增
// 这里不考虑外部订单，外部订单到本地订单的转换在doMethod完成
func (ord *OrderState) AddVersion(currentVd VersionData) error {
	preventVd, err := ord.LatestVersion()

	if err != nil {
		vd := VersionData{}
		vd.RemainedAmountB = big.NewInt(0)
		vd.RemainedAmountS = ord.RawOrder.AmountS
		vd.Block = big.NewInt(0)
		vd.Status = ORDER_NEW
		ord.States = append(ord.States, vd)

		log.Debugf("ipfs new order add version data:%s", ord.RawOrder.Hash.Hex())
		return nil
	}

	if currentVd.Block.Cmp(preventVd.Block) < 0 {
		return nil
	}

	ord.States = append(ord.States, currentVd)
	log.Debugf("chain order add version data %s->%d", ord.RawOrder.Hash.Hex(), currentVd.Status)
	return nil
}

// 放到common package 根据配置决定状态
func (ord *OrderState) SettleStatus() {

}
