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
	"github.com/Loopring/ringminer/crypto"
	"github.com/Loopring/ringminer/log"
	"math/big"
)

// 旷工在成本节约和fee上二选一，撮合者计算出:
// 1.fee(lrc)的市场价(法币交易价格)
// 2.成本节约(savingShare)的市场价(法币交易价格)
// 撮合者在fee和savingShare中二选一，作为自己的利润，
// 如果撮合者选择fee，则成本节约分给订单发起者，如果选择成本节约，则需要返还给用户一定的lrc
// 这样一来，撮合者的利润判断公式应该是max(fee, savingShare - fee * s),s为固定比例
// 此外，在选择最优环路的时候，撮合者会在确定了选择fee/savingShare后，选择某个具有最大利润的环路
// 但是，根据谷歌竞拍法则(A出价10,B出价20,最终成交价为10)，撮合者最终获得的利润只能是利润最小的环路利润

type Ring struct {
	Orders                                      []*FilledOrder `json:"orderes"`
	Miner                                       Address        `json:"miner"`
	FeeRecepient                                Address        `json:"feeRecepient"`
	ThrowIfTokenAllowanceOrBalanceIsInsuffcient bool           `json:"throwIfTokenAllowanceOrBalanceIsInsuffcient"`
	V                                           uint8          `json:"v"`
	R                                           Sign           `json:"r"`
	S                                           Sign           `json:"s"`
	Hash                                        Hash           `json:"hash"`
}

func (ring *Ring) GenerateHash() Hash {
	vBytes := []byte{byte(ring.Orders[0].OrderState.RawOrder.V)}
	rBytes := ring.Orders[0].OrderState.RawOrder.R.Bytes()
	sBytes := ring.Orders[0].OrderState.RawOrder.S.Bytes()
	for idx, order := range ring.Orders {
		if idx > 0 {
			vBytes = Xor(vBytes, []byte{byte(order.OrderState.RawOrder.V)})
			rBytes = Xor(rBytes, order.OrderState.RawOrder.R.Bytes())
			sBytes = Xor(sBytes, order.OrderState.RawOrder.S.Bytes())
		}
	}
	hashBytes := crypto.CryptoInstance.GenerateHash(vBytes, rBytes, sBytes)
	return BytesToHash(hashBytes)
}

func (ring *Ring) GenerateAndSetSignature(pkBytes []byte) error {
	if ring.Hash.IsZero() {
		ring.Hash = ring.GenerateHash()
	}
	if sig, err := crypto.CryptoInstance.Sign(ring.Hash.Bytes(), pkBytes); nil != err {
		return err
	} else {
		v, r, s := crypto.CryptoInstance.SigToVRS(sig)
		ring.V = uint8(v)
		ring.R = BytesToSign(r)
		ring.S = BytesToSign(s)
		return nil
	}
}

func (ring *Ring) ValidateSignatureValues() bool {
	return crypto.CryptoInstance.ValidateSignatureValues(byte(ring.V), ring.R.Bytes(), ring.S.Bytes())
}

func (ring *Ring) SignerAddress() (Address, error) {
	address := &Address{}
	hash := ring.Hash
	if hash.IsZero() {
		hash = ring.GenerateHash()
	}

	sig, _ := crypto.CryptoInstance.VRSToSig(ring.V, ring.R.Bytes(), ring.S.Bytes())
	log.Debugf("orderstate.hash:%s", hash.Hex())

	if addressBytes, err := crypto.CryptoInstance.SigToAddress(hash.Bytes(), sig); nil != err {
		log.Errorf("error:%s", err.Error())
		return *address, err
	} else {
		address.SetBytes(addressBytes)
		return *address, nil
	}
}

func (ring *Ring) GenerateSubmitArgs(minerPk []byte) *RingSubmitArgs {
	ringSubmitArgs := emptyRingSubmitArgs()

	for _, filledOrder := range ring.Orders {
		order := filledOrder.OrderState.RawOrder
		ringSubmitArgs.AddressList = append(ringSubmitArgs.AddressList, [2]Address{order.Owner, order.TokenS})
		rateAmountS, _ := new(big.Int).SetString(filledOrder.RateAmountS.FloatString(0), 10)
		ringSubmitArgs.UintArgsList = append(ringSubmitArgs.UintArgsList, [7]*big.Int{order.AmountS, order.AmountB, order.Timestamp, order.Ttl, order.Salt, order.LrcFee, rateAmountS})

		ringSubmitArgs.Uint8ArgsList = append(ringSubmitArgs.Uint8ArgsList, [2]uint8{order.MarginSplitPercentage, filledOrder.FeeSelection})

		ringSubmitArgs.BuyNoMoreThanAmountBList = append(ringSubmitArgs.BuyNoMoreThanAmountBList, order.BuyNoMoreThanAmountB)

		ringSubmitArgs.VList = append(ringSubmitArgs.VList, order.V)
		ringSubmitArgs.RList = append(ringSubmitArgs.RList, order.R.Bytes())
		ringSubmitArgs.SList = append(ringSubmitArgs.SList, order.S.Bytes())
	}

	ringSubmitArgs.ThrowIfLRCIsInsuffcient = ring.ThrowIfTokenAllowanceOrBalanceIsInsuffcient

	if err := ring.GenerateAndSetSignature(minerPk); nil != err {
		log.Error(err.Error())
	} else {
		ringSubmitArgs.VList = append(ringSubmitArgs.VList, ring.V)
		ringSubmitArgs.RList = append(ringSubmitArgs.RList, ring.R.Bytes())
		ringSubmitArgs.SList = append(ringSubmitArgs.SList, ring.S.Bytes())
	}
	ringminer, _ := ring.SignerAddress()
	ringSubmitArgs.Ringminer = ringminer
	return ringSubmitArgs
}

// todo:unpack transaction data to ring,finally get orders

type RingState struct {
	RawRing        *Ring    `json:"rawRing"`
	ReducedRate    *big.Rat `json:"reducedRate"` //成环之后，折价比例
	LegalFee       *big.Rat `json:"legalFee"`    //法币计算的fee
	FeeMode        int      `json:"feeMode"`     //收费方式，0 lrc 1 share
	SubmitTxHash   Hash     `json:"submitTxHash"`
	RegistryTxHash Hash     `json:"registryTxHash"`
}

type RingSubmitArgs struct {
	AddressList              [][2]Address  `alias:"addressList"`
	UintArgsList             [][7]*big.Int `alias:"uintArgsList"`
	Uint8ArgsList            [][2]uint8    `alias:"uint8ArgsList"`
	BuyNoMoreThanAmountBList []bool        `alias:"buyNoMoreThanAmountBList"`
	VList                    []uint8       `alias:"vList"`
	RList                    [][]byte      `alias:"rList"`
	SList                    [][]byte      `alias:"sList"`
	Ringminer                Address       `alias:"ringminer"`
	FeeRecepient             Address       `alias:"feeRecepient"`
	ThrowIfLRCIsInsuffcient  bool          `alias:"throwIfLRCIsInsuffcient"`
}

func emptyRingSubmitArgs() *RingSubmitArgs {
	return &RingSubmitArgs{
		AddressList:              [][2]Address{},
		UintArgsList:             [][7]*big.Int{},
		Uint8ArgsList:            [][2]uint8{},
		BuyNoMoreThanAmountBList: []bool{},
		VList: []uint8{},
		RList: [][]byte{},
		SList: [][]byte{},
	}
}
