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
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"reflect"
)

type AbiMethod interface {
	MethodId() string
	Address() types.Address
	Call(result interface{}, blockParameter string, args ...interface{}) error
	SendTransactionWithSpecificGas(from types.Address, gas, gasPrice *big.Int, args ...interface{}) (string, error)
	SendTransaction(from types.Address, args ...interface{}) (string, error)
	Unpack(v interface{}, hex string) error
	WatcherTopic() string //watch topic
}

type AbiEvent interface {
	Id() string                                                   //返回事件id
	Name() string                                                 //返回事件名
	Address() types.Address                                       //事件地址
	Subscribe(eventChan reflect.Value, fromBlock, toBlock string) //对事件进行订阅
	Unpack(v interface{}, output []byte, topics []string) error   //事件解析
	WatcherTopic() string                                         //watch topic
}

//the base info of contract
type Contract struct {
	Abi     interface{}
	Address types.Address
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
	Erc20TokenAbiStr       string = `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"who","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	ImplAbiStr             string = `[{"constant":true,"inputs":[{"name":"signer","type":"address"},{"name":"hash","type":"bytes32"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"verifySignature","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"orderHash","type":"bytes32"}],"name":"getOrderCancelled","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_MAX_VALUE","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"filled","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"cancelled","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"MARGIN_SPLIT_PERCENTAGE_BASE","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"ringIndex","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"addresses","type":"address[3]"},{"name":"orderValues","type":"uint256[7]"},{"name":"buyNoMoreThanAmountB","type":"bool"},{"name":"marginSplitPercentage","type":"uint8"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"cancelOrder","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"RATE_RATIO_SCALE","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"lrcTokenAddress","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"tokenRegistryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"addressList","type":"address[2][]"},{"name":"uintArgsList","type":"uint256[7][]"},{"name":"uint8ArgsList","type":"uint8[2][]"},{"name":"buyNoMoreThanAmountBList","type":"bool[]"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"},{"name":"ringminer","type":"address"},{"name":"feeRecepient","type":"address"},{"name":"throwIfLRCIsInsuffcient","type":"bool"}],"name":"submitRing","outputs":[{"name":"","type":"bytes32"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"delegateAddress","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"orderHash","type":"bytes32"}],"name":"getOrderFilled","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"tokenAddress","type":"address"},{"name":"tokenOwner","type":"address"}],"name":"getSpendable","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"maxRingSize","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"ringhashRegistryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"cutoff","type":"uint256"}],"name":"setCutoff","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_LRC","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"cutoffs","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"rateRatioCVSThreshold","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_MARGIN_SPLIT","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"inputs":[{"name":"_lrcTokenAddress","type":"address"},{"name":"_tokenRegistryAddress","type":"address"},{"name":"_ringhashRegistryAddress","type":"address"},{"name":"_delegateAddress","type":"address"},{"name":"_maxRingSize","type":"uint256"},{"name":"_rateRatioCVSThreshold","type":"uint256"}],"payable":false,"type":"constructor"},{"payable":true,"type":"fallback"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_ringIndex","type":"uint256"},{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_ringhash","type":"bytes32"},{"indexed":true,"name":"_miner","type":"address"},{"indexed":true,"name":"_feeRecepient","type":"address"},{"indexed":false,"name":"_ringhashFound","type":"bool"}],"name":"RingMinedEvent","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_ringIndex","type":"uint256"},{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_ringhash","type":"bytes32"},{"indexed":false,"name":"_prevOrderHash","type":"bytes32"},{"indexed":true,"name":"_orderHash","type":"bytes32"},{"indexed":false,"name":"_nextOrderHash","type":"bytes32"},{"indexed":false,"name":"_amountS","type":"uint256"},{"indexed":false,"name":"_amountB","type":"uint256"},{"indexed":false,"name":"_lrcReward","type":"uint256"},{"indexed":false,"name":"_lrcFee","type":"uint256"}],"name":"OrderFilled","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_orderHash","type":"bytes32"},{"indexed":false,"name":"_amountCancelled","type":"uint256"}],"name":"OrderCancelled","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_address","type":"address"},{"indexed":false,"name":"_cutoff","type":"uint256"}],"name":"CutoffTimestampChanged","type":"event"}]`
	RinghashRegistryAbiStr string = `[{"constant":true,"inputs":[{"name":"ringminer","type":"address"},{"name":"ringSize","type":"uint256"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"}],"name":"calculateRinghash","outputs":[{"name":"","type":"bytes32"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"ringhash","type":"bytes32"}],"name":"ringhashFound","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"blocksToLive","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"ringhash","type":"bytes32"},{"name":"ringminer","type":"address"}],"name":"canSubmit","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"ringSize","type":"uint256"},{"name":"ringminer","type":"address"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"}],"name":"submitRinghash","outputs":[],"payable":false,"type":"function"},{"inputs":[{"name":"_blocksToLive","type":"uint256"}],"payable":false,"type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_ringminer","type":"address"},{"indexed":true,"name":"_ringhash","type":"bytes32"}],"name":"RinghashSubmitted","type":"event"}]`
	TransferDelegateAbiStr string = `[{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"addVersion","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"token","type":"address"},{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferToken","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"versioned","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"getVersions","outputs":[{"name":"","type":"address[]"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"versions","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"removeVersion","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"tokenAddress","type":"address"},{"name":"_owner","type":"address"}],"name":"getSpendable","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"addr","type":"address"},{"indexed":false,"name":"version","type":"uint256"}],"name":"VersionAdded","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"addr","type":"address"},{"indexed":false,"name":"version","type":"uint256"}],"name":"VersionRemoved","type":"event"}]`
	TokenRegistryAbiStr    string = `[{"constant":false,"inputs":[{"name":"_token","type":"address"},{"name":"_symbol","type":"string"}],"name":"unregisterToken","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"symbol","type":"string"}],"name":"getAddressBySymbol","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_token","type":"address"}],"name":"isTokenRegistered","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"tokens","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_token","type":"address"},{"name":"_symbol","type":"string"}],"name":"registerToken","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"symbol","type":"string"}],"name":"isTokenRegisteredBySymbol","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"}]`
)

type LoopringProtocolImpl struct {
	Contract

	RingHashRegistry      *LoopringRinghashRegistry
	TokenTransferDelegate *TransferDelegate

	LrcTokenAddress         AbiMethod
	TokenRegistryAddress    AbiMethod
	RinghashRegistryAddress AbiMethod
	DelegateAddress         AbiMethod
	MaxRingSize             AbiMethod
	RateRatioCVSThreshold   AbiMethod

	Filled   AbiMethod
	Canceled AbiMethod
	Cutoffs  AbiMethod

	SubmitRing             SubmitRingMethod
	CancelOrder            AbiMethod
	SetCutoff              AbiMethod
	VerifyTokensRegistered AbiMethod
	VerifySignature        AbiMethod
	GetOrderFilled         AbiMethod
	GetOrderCancelled      AbiMethod
	GetCancelledOrFilled   AbiMethod `parameterName:"cancelledOrFilled"`

	//event
	RingMinedEvent              RingMinedEvent
	OrderFilledEvent            OrderFilledEvent
	OrderCancelledEvent         OrderCancelledEvent
	CutoffTimestampChangedEvent CutoffTimestampChangedEvent
}

type SubmitRingMethod struct {
	AbiMethod
	Inputs  types.RingSubmitInputs
	Outputs types.RingSubmitOuts
}

// todo(fk): []byte to hash
//go:generate gencodec -type RingMinedEvent -field-override ringMinedEventMarshaling -out gen_ringminedevent_json.go
type RingMinedEvent struct {
	AbiEvent
	RingIndex     *big.Int      `json:"ringIndex" alias:"_ringIndex" gencodec:"required"`
	Time          *big.Int      `json:"time" alias:"_time" gencodec:"required"`
	Blocknumber   *big.Int      `json:"blockNumber" alias:"_blocknumber" gencodec:"required"`
	Ringhash      types.Hash    `json:"ringHash" alias:"_ringhash" gencodec:"required"`
	Miner         types.Address `json:"miner" alias:"_miner" gencodec:"required"`
	FeeRecepient  types.Address `json:"feeRecepient" alias:"_feeRecepient" gencodec:"required"`
	RinghashFound bool          `json:"ringHashFound" alias:"_ringhashFound" gencodec:"required"`
}

type ringMinedEventMarshaling struct {
	RingIndex   *types.Big
	Time        *types.Big
	Blocknumber *types.Big
}

//go:generate gencodec -type OrderFilledEvent -field-override orderFilledEventMarshaling -out gen_orderfilledevent_json.go
type OrderFilledEvent struct {
	AbiEvent
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

//go:generate gencodec -type OrderCancelledEvent -field-override orderCancelledEventMarshaling -out gen_ordercancelledevent_json.go
type OrderCancelledEvent struct {
	AbiEvent
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

//go:generate gencodec -type CutoffTimestampChangedEvent -field-override cutoffTimestampChangedEventtMarshaling -out gen_cutofftimestampevent_json.go
type CutoffTimestampChangedEvent struct {
	AbiEvent
	Time        *big.Int      `json:"time" alias:"_time" gencodec:"required"`
	Blocknumber *big.Int      `json:"blockNumber" alias:"_blocknumber" gencodec:"required"`
	Address     types.Address `json:"address" alias:"_address" gencodec:"required"`
	Cutoff      *big.Int      `json:"cutoff" alias:"_cutoff" gencodec:"required"`
}

type cutoffTimestampChangedEventtMarshaling struct {
	Time        *types.Big
	Blocknumber *types.Big
	Cutoff      *types.Big
}

type LoopringRinghashRegistry struct {
	Contract
	SubmitRinghash    AbiMethod
	CanSubmit         AbiMethod
	RinghashFound     AbiMethod
	BlocksToLive      AbiMethod
	CalculateRinghash AbiMethod

	RinghashSubmittedEvent RinghashSubmitted
}

type RinghashSubmitted struct {
	AbiEvent
	RingHash  []byte         `alias:"_ringhash"`
	RingMiner common.Address `alias:"_ringminer"`
}

type TokenRegistry struct {
	Contract
	RegisterToken     AbiMethod
	UnregisterToken   AbiMethod
	IsTokenRegistered AbiMethod
}

type TransferDelegate struct {
	Contract
	AddVersion AbiMethod
}

type Loopring struct {
	Client        *Client
	Tokens        map[types.Address]*Erc20Token
	LoopringImpls map[types.Address]*LoopringProtocolImpl
}

func NewLoopringInstance(commOpts config.CommonOptions, client *Client) *Loopring {
	loopringInstance := &Loopring{}
	loopringInstance.Client = client
	loopringInstance.Tokens = make(map[types.Address]*Erc20Token)

	protocolImps := make(map[types.Address]*LoopringProtocolImpl)

	for _, implAddress := range commOpts.LoopringImpAddresses {
		imp := &LoopringProtocolImpl{}
		client.NewContract(imp, implAddress, ImplAbiStr)
		addr := types.HexToAddress(implAddress)

		var lrcTokenAddressHex string
		imp.LrcTokenAddress.Call(&lrcTokenAddressHex, "latest")
		lrcTokenAddress := types.HexToAddress(lrcTokenAddressHex)
		lrcToken := &Erc20Token{}
		client.NewContract(lrcToken, lrcTokenAddress.Hex(), Erc20TokenAbiStr)
		loopringInstance.Tokens[lrcTokenAddress] = lrcToken

		var registryAddressHex string
		imp.RinghashRegistryAddress.Call(&registryAddressHex, "latest")
		registryAddress := types.HexToAddress(registryAddressHex)
		registry := &LoopringRinghashRegistry{}
		client.NewContract(registry, registryAddress.Hex(), RinghashRegistryAbiStr)
		imp.RingHashRegistry = registry

		var delegateAddressHex string
		imp.DelegateAddress.Call(&delegateAddressHex, "latest")
		delegateAddress := types.HexToAddress(delegateAddressHex)
		delegate := &TransferDelegate{}
		client.NewContract(delegate, delegateAddress.Hex(), TransferDelegateAbiStr)
		imp.TokenTransferDelegate = delegate

		protocolImps[addr] = imp
	}
	loopringInstance.LoopringImpls = protocolImps

	return loopringInstance
}

func (loopring *Loopring) AddToken(tokenAddress types.Address) {
	if _, ok := loopring.Tokens[tokenAddress]; !ok {
		tokenContract := &Erc20Token{}
		if err := loopring.Client.NewContract(tokenContract, tokenAddress.Hex(), Erc20TokenAbiStr); nil != err {

		} else {
			loopring.Tokens[tokenAddress] = tokenContract
		}
	}
}

type ContractData struct {
	Method AbiMethod
	Event  AbiEvent
	TxHash types.Hash
}

type sd interface {
	GetErc20Balance()
	GetAllownce()
}
