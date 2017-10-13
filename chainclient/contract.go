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
	SendTransactionWithSpecificGas(from types.Address, gas, gasPrice *big.Int, args ...interface{}) (string, error)
	SendTransaction(from types.Address, args ...interface{}) (string, error)
}

type AbiEvent interface {
	Subscribe() //对事件进行订阅
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

const (
	Erc20TokenAbiStr              string = `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"who","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	CurrentImplAbiStr             string = `[{"constant":true,"inputs":[{"name":"signer","type":"address"},{"name":"hash","type":"bytes32"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"verifySignature","outputs":[],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"orderHash","type":"bytes32"}],"name":"getOrderCancelled","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_MAX_VALUE","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"filled","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"cancelled","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"MARGIN_SPLIT_PERCENTAGE_BASE","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"ringIndex","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"addresses","type":"address[3]"},{"name":"orderValues","type":"uint256[7]"},{"name":"buyNoMoreThanAmountB","type":"bool"},{"name":"marginSplitPercentage","type":"uint8"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"cancelOrder","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"RATE_RATIO_SCALE","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"lrcTokenAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"tokenRegistryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"addressList","type":"address[2][]"},{"name":"uintArgsList","type":"uint256[7][]"},{"name":"uint8ArgsList","type":"uint8[2][]"},{"name":"buyNoMoreThanAmountBList","type":"bool[]"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"},{"name":"ringminer","type":"address"},{"name":"feeRecepient","type":"address"},{"name":"throwIfLRCIsInsuffcient","type":"bool"}],"name":"submitRing","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"delegateAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"orderHash","type":"bytes32"}],"name":"getOrderFilled","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"maxRingSize","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"ringhashRegistryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"cutoff","type":"uint256"}],"name":"setCutoff","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_LRC","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"cutoffs","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"rateRatioCVSThreshold","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_MARGIN_SPLIT","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_lrcTokenAddress","type":"address"},{"name":"_tokenRegistryAddress","type":"address"},{"name":"_ringhashRegistryAddress","type":"address"},{"name":"_delegateAddress","type":"address"},{"name":"_maxRingSize","type":"uint256"},{"name":"_rateRatioCVSThreshold","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_ringIndex","type":"uint256"},{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_ringhash","type":"bytes32"},{"indexed":true,"name":"_miner","type":"address"},{"indexed":true,"name":"_feeRecepient","type":"address"},{"indexed":false,"name":"_ringhashFound","type":"bool"}],"name":"RingMined","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_ringIndex","type":"uint256"},{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_ringhash","type":"bytes32"},{"indexed":false,"name":"_prevOrderHash","type":"bytes32"},{"indexed":true,"name":"_orderHash","type":"bytes32"},{"indexed":false,"name":"_nextOrderHash","type":"bytes32"},{"indexed":false,"name":"_amountS","type":"uint256"},{"indexed":false,"name":"_amountB","type":"uint256"},{"indexed":false,"name":"_lrcReward","type":"uint256"},{"indexed":false,"name":"_lrcFee","type":"uint256"}],"name":"OrderFilled","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_orderHash","type":"bytes32"},{"indexed":false,"name":"_amountCancelled","type":"uint256"}],"name":"OrderCancelled","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_address","type":"address"},{"indexed":false,"name":"_cutoff","type":"uint256"}],"name":"CutoffTimestampChanged","type":"event"}]`
	CurrentRinghashRegistryAbiStr string = `[{"constant":true,"inputs":[{"name":"ringSize","type":"uint256"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"}],"name":"calculateRinghash","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"ringhash","type":"bytes32"}],"name":"ringhashFound","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"blocksToLive","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"ringhash","type":"bytes32"},{"name":"feeRecepient","type":"address"}],"name":"canSubmit","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"ringSize","type":"uint256"},{"name":"feeRecepient","type":"address"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"}],"name":"submitRinghash","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"inputs":[{"name":"_blocksToLive","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"}]`
)

type LoopringProtocolImpl struct {
	Contract

	RingHashRegistry *LoopringRinghashRegistry

	LrcTokenAddress         AbiMethod
	TokenRegistryAddress    AbiMethod
	RinghashRegistryAddress AbiMethod
	DelegateAddress         AbiMethod
	MaxRingSize             AbiMethod
	RateRatioCVSThreshold   AbiMethod

	Filled   AbiMethod
	Canceled AbiMethod
	Cutoffs  AbiMethod

	SubmitRing             AbiMethod
	CancelOrder            AbiMethod
	SetCutoff              AbiMethod
	VerifyTokensRegistered AbiMethod
	VerifySignature        AbiMethod
	GetOrderFilled         AbiMethod
	GetOrderCancelled      AbiMethod

	//event
	RingMined struct {
		AbiEvent
		hash string
	}

	OrderFilled struct {
	}

	OrderCancelled struct {
	}

	CutoffTimestampChanged struct {
	}
}

type LoopringRinghashRegistry struct {
	Contract
	SubmitRinghash    AbiMethod
	CanSubmit         AbiMethod
	RinghashFound     AbiMethod
	BlocksToLive      AbiMethod
	CalculateRinghash AbiMethod
}

type RinghashRegistryEvent struct {
	AbiEvent
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

type Loopring struct {
	Client        *Client
	Tokens        map[types.Address]*Erc20Token
	LoopringImpls LoopringProtocolImplMap
}
