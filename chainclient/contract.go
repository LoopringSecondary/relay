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

package chainclient

import (
	"github.com/Loopring/ringminer/types"
	"math/big"
)

type AbiMethod interface {
	Call(result interface{}, blockParameter string, args ...interface{}) error
	SendTransactionWithSpecificGas(from string, gas, gasPrice *big.Int, args ...interface{}) (string, error)
	SendTransaction(from string, args ...interface{}) (string, error)
}

//the base info of contract
type Contract struct {
	Abi     interface{}
	Address string
}

type Erc20Token struct {
	Contract
	Name         string
	TotalSupply  AbiMethod
	BalanceOf    AbiMethod
	Transfer     AbiMethod
	TransferFrom AbiMethod
	Approve      AbiMethod
	Allowance    AbiMethod
}

type LoopringProtocolImpl struct {
	Contract

	RemainAmount AbiMethod //todo:

	SubmitRing                   AbiMethod
	SubmitRingFingerPrint        AbiMethod
	CancelOrder                  AbiMethod
	VerifyTokensRegistered       AbiMethod
	CalculateSignerAddress       AbiMethod
	CalculateOrderHash           AbiMethod
	ValidateOrder                AbiMethod
	AssembleOrders               AbiMethod
	CalculateOrderFillAmount     AbiMethod
	CalculateRingFillAmount      AbiMethod
	CalculateRingFees            AbiMethod
	VerifyMinerSuppliedFillRates AbiMethod
	SettleRing                   AbiMethod
	VerifyRingHasNoSubRing       AbiMethod
}

type LoopringFingerprintRegistry struct {
	Contract
	SubmitRingFingerprint AbiMethod
	CanSubmit             AbiMethod
	FingerprintFound      AbiMethod
	IsExpired             AbiMethod
	GetRingHash           AbiMethod
}

type FingerprintEvent struct {
	RingHash *types.Hash
}

type TokenRegistry struct {
	Contract
	RegisterToken     AbiMethod
	UnregisterToken   AbiMethod
	IsTokenRegistered AbiMethod
}

//todo:need perfect name
type LoopringProtocolImplMap map[types.Address]*LoopringProtocolImpl

type LoopringFingerprintRegistryMap map[types.Address]*LoopringFingerprintRegistry

type Loopring struct {
	Client               *Client
	Tokens               map[types.Address]*Erc20Token
	LoopringImpls        LoopringProtocolImplMap
	LoopringFingerprints LoopringFingerprintRegistryMap
}
