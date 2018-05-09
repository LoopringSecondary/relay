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
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type NameRegistryInfo struct {
	Name          string
	Owner         common.Address
	FeeRecipient  common.Address
	Signer        common.Address
	ParticipantId *big.Int
}

// 旷工在成本节约和fee上二选一，撮合者计算出:
// 1.fee(lrc)的市场价(法币交易价格)
// 2.成本节约(savingShare)的市场价(法币交易价格)
// 撮合者在fee和savingShare中二选一，作为自己的利润，
// 如果撮合者选择fee，则成本节约分给订单发起者，如果选择成本节约，则需要返还给用户一定的lrc
// 这样一来，撮合者的利润判断公式应该是max(fee, savingShare - fee * s),s为固定比例
// 此外，在选择最优环路的时候，撮合者会在确定了选择fee/savingShare后，选择某个具有最大利润的环路
// 但是，根据谷歌竞拍法则(A出价10,B出价20,最终成交价为10)，撮合者最终获得的利润只能是利润最小的环路利润

type Ring struct {
	Orders      []*FilledOrder `json:"orderes"`
	V           uint8          `json:"v"`
	R           Bytes32        `json:"r"`
	S           Bytes32        `json:"s"`
	Hash        common.Hash    `json:"hash"`
	ReducedRate *big.Rat       `json:"reducedRate"` //成环之后，折价比例
	LegalFee    *big.Rat       `json:"legalFee"`    //法币计算的fee
	UniqueId    common.Hash    `json:"uniquedId"`

	//
	Received  *big.Rat
	LegalCost *big.Rat
	Gas       *big.Int
	GasPrice  *big.Int
}

func (ring *Ring) FeeSelections() *big.Int {
	feeSelections := big.NewInt(int64(0))
	for idx, filledOrder := range ring.Orders {
		if filledOrder.FeeSelection > 0 {
			feeSelections.Or(feeSelections, big.NewInt(int64(1<<uint(idx))))
		}
	}
	return feeSelections
}

func (ring *Ring) GenerateUniqueId() common.Hash {
	if IsZeroHash(ring.UniqueId) {
		orderHashBytes := ring.Orders[0].OrderState.RawOrder.Hash.Bytes()
		for idx, order := range ring.Orders {
			if idx > 0 {
				orderHashBytes = Xor(orderHashBytes, order.OrderState.RawOrder.Hash.Bytes())
			}
		}
		ring.UniqueId = common.BytesToHash(orderHashBytes)
	}
	return ring.UniqueId
}

func (ring *Ring) GenerateHash(feeReceipt common.Address) common.Hash {
	hashBytes := crypto.GenerateHash(
		ring.GenerateUniqueId().Bytes(),
		feeReceipt.Bytes(),
		common.LeftPadBytes(ring.FeeSelections().Bytes(), 2),
	)
	return common.BytesToHash(hashBytes)
}

//func (ring *Ring) GenerateAndSetSignature(miner common.Address) error {
//	if IsZeroHash(ring.Hash) {
//		ring.Hash = ring.GenerateHash(miner)
//	}
//
//	if sig, err := crypto.Sign(ring.Hash.Bytes(), miner); nil != err {
//		return err
//	} else {
//		v, r, s := crypto.SigToVRS(sig)
//		ring.V = uint8(v)
//		ring.R = BytesToBytes32(r)
//		ring.S = BytesToBytes32(s)
//		return nil
//	}
//}

func (ring *Ring) ValidSinceTime() int64 {
	latestValidSince := int64(0)
	if nil != ring && len(ring.Orders) > 0 {
		for _, order := range ring.Orders {
			thisTime := order.OrderState.RawOrder.ValidSince.Int64()
			if latestValidSince <= thisTime {
				latestValidSince = thisTime
			}
		}
	}

	return latestValidSince
}

//func (ring *Ring) GenerateSubmitArgs(miner common.Address) (*RingSubmitInputs, error) {
//	ringSubmitArgs := emptyRingSubmitArgs(miner)
//	authVList := []uint8{}
//	authRList := []Bytes32{}
//	authSList := []Bytes32{}
//	ring.Hash = ring.GenerateHash(miner)
//	for _, filledOrder := range ring.Orders {
//		order := filledOrder.OrderState.RawOrder
//		ringSubmitArgs.AddressList = append(ringSubmitArgs.AddressList, [4]common.Address{order.Owner, order.TokenS, order.WalletAddress, order.AuthAddr})
//		rateAmountS, _ := new(big.Int).SetString(filledOrder.RateAmountS.FloatString(0), 10)
//		ringSubmitArgs.UintArgsList = append(ringSubmitArgs.UintArgsList, [6]*big.Int{order.AmountS, order.AmountB, order.ValidSince, order.ValidUntil, order.LrcFee, rateAmountS})
//		ringSubmitArgs.Uint8ArgsList = append(ringSubmitArgs.Uint8ArgsList, [1]uint8{order.MarginSplitPercentage})
//
//		ringSubmitArgs.BuyNoMoreThanAmountBList = append(ringSubmitArgs.BuyNoMoreThanAmountBList, order.BuyNoMoreThanAmountB)
//
//		ringSubmitArgs.VList = append(ringSubmitArgs.VList, order.V)
//		ringSubmitArgs.RList = append(ringSubmitArgs.RList, order.R)
//		ringSubmitArgs.SList = append(ringSubmitArgs.SList, order.S)
//
//		//sign By authPrivateKey
//		if signBytes, err := order.AuthPrivateKey.Sign(ring.Hash.Bytes(), order.AuthPrivateKey.Address()); nil == err {
//			v, r, s := crypto.SigToVRS(signBytes)
//			authVList = append(authVList, v)
//			authRList = append(authRList, BytesToBytes32(r))
//			authSList = append(authSList, BytesToBytes32(s))
//		} else {
//			return nil, err
//		}
//	}
//
//	ringSubmitArgs.VList = append(ringSubmitArgs.VList, authVList...)
//	ringSubmitArgs.RList = append(ringSubmitArgs.RList, authRList...)
//	ringSubmitArgs.SList = append(ringSubmitArgs.SList, authSList...)
//
//	ringSubmitArgs.FeeSelections = ring.FeeSelections()
//	//if err := ring.GenerateAndSetSignature(miner); nil != err {
//	//	return nil, err
//	//} else {
//	//	ringSubmitArgs.VList = append(ringSubmitArgs.VList, ring.V)
//	//	ringSubmitArgs.RList = append(ringSubmitArgs.RList, ring.R)
//	//	ringSubmitArgs.SList = append(ringSubmitArgs.SList, ring.S)
//	//}
//
//	return ringSubmitArgs, nil
//}

type RingSubmitInfo struct {
	RawRing *Ring

	//todo:remove it
	Miner            common.Address
	ProtocolAddress  common.Address
	Ringhash         common.Hash
	OrdersCount      *big.Int
	ProtocolData     []byte
	ProtocolGas      *big.Int
	ProtocolUsedGas  *big.Int
	ProtocolGasPrice *big.Int

	SubmitTxHash common.Hash
}

//
//type RingSubmitInputs struct {
//	AddressList              [][4]common.Address
//	UintArgsList             [][6]*big.Int
//	Uint8ArgsList            [][1]uint8
//	BuyNoMoreThanAmountBList []bool
//	VList                    []uint8
//	RList                    []Bytes32
//	SList                    []Bytes32
//	Miner                    common.Address
//	FeeSelections            *big.Int
//}
//
//func emptyRingSubmitArgs(miner common.Address) *RingSubmitInputs {
//	return &RingSubmitInputs{
//		AddressList:              [][4]common.Address{},
//		UintArgsList:             [][6]*big.Int{},
//		Uint8ArgsList:            [][1]uint8{},
//		BuyNoMoreThanAmountBList: []bool{},
//		VList: []uint8{},
//		RList: []Bytes32{},
//		SList: []Bytes32{},
//		Miner: miner,
//	}
//}
