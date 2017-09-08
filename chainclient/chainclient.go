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

import "github.com/Loopring/ringminer/types"

//提供接口，如：订阅事件、获取区块、获取交易、获取合约等接口

type RpcMethod func(result interface{}, args ...interface{}) error

type Client struct {
	Subscribe	RpcMethod	`methodName:"subscribe"`

	ClientVersion   RpcMethod       `methodName:"clientVersion"`
	Sha3    RpcMethod       `methodName:"sha3"`
	Version RpcMethod       `methodName:"version"`
	PeerCount       RpcMethod       `methodName:"peerCount"`
	Listening       RpcMethod       `methodName:"listening"`
	ProtocolVersion RpcMethod       `methodName:"protocolVersion"`
	Syncing RpcMethod       `methodName:"syncing"`
	Coinbase        RpcMethod       `methodName:"coinbase"`
	Mining  RpcMethod       `methodName:"mining"`
	Hashrate        RpcMethod       `methodName:"hashrate"`
	GasPrice        RpcMethod       `methodName:"gasPrice"`
	Accounts        RpcMethod       `methodName:"accounts"`
	BlockNumber     RpcMethod       `methodName:"blockNumber"`
	GetBalance      RpcMethod       `methodName:"getBalance"`
	GetStorageAt    RpcMethod       `methodName:"getStorageAt"`
	GetTransactionCount     RpcMethod       `methodName:"getTransactionCount"`
	GetBlockTransactionCountByHash  RpcMethod       `methodName:"getBlockTransactionCountByHash"`
	GetBlockTransactionCountByNumber        RpcMethod       `methodName:"getBlockTransactionCountByNumber"`
	GetUncleCountByBlockHash        RpcMethod       `methodName:"getUncleCountByBlockHash"`
	GetUncleCountByBlockNumber      RpcMethod       `methodName:"getUncleCountByBlockNumber"`
	GetCode RpcMethod       `methodName:"getCode"`
	Sign    RpcMethod       `methodName:"sign"`
	SendTransaction RpcMethod       `methodName:"sendTransaction"`
	SendRawTransaction      RpcMethod       `methodName:"sendRawTransaction"`
	Call    RpcMethod       `methodName:"call"`
	EstimateGas     RpcMethod       `methodName:"estimateGas"`
	GetBlockByHash  RpcMethod       `methodName:"getBlockByHash"`
	GetBlockByNumber        RpcMethod       `methodName:"getBlockByNumber"`
	GetTransactionByHash    RpcMethod       `methodName:"getTransactionByHash"`
	GetTransactionByBlockHashAndIndex       RpcMethod       `methodName:"getTransactionByBlockHashAndIndex"`
	GetTransactionByBlockNumberAndIndex     RpcMethod       `methodName:"getTransactionByBlockNumberAndIndex"`
	GetTransactionReceipt   RpcMethod       `methodName:"getTransactionReceipt"`
	GetUncleByBlockHashAndIndex     RpcMethod       `methodName:"getUncleByBlockHashAndIndex"`
	GetUncleByBlockNumberAndIndex   RpcMethod       `methodName:"getUncleByBlockNumberAndIndex"`
	GetCompilers    RpcMethod       `methodName:"getCompilers"`
	CompileLLL      RpcMethod       `methodName:"compileLLL"`
	CompileSolidity RpcMethod       `methodName:"compileSolidity"`
	CompileSerpent  RpcMethod       `methodName:"compileSerpent"`
	NewFilter       RpcMethod       `methodName:"newFilter"`
	NewBlockFilter  RpcMethod       `methodName:"newBlockFilter"`
	NewPendingTransactionFilter     RpcMethod       `methodName:"newPendingTransactionFilter"`
	UninstallFilter RpcMethod       `methodName:"uninstallFilter"`
	GetFilterChanges        RpcMethod       `methodName:"getFilterChanges"`
	GetFilterLogs   RpcMethod       `methodName:"getFilterLogs"`
	GetLogs RpcMethod       `methodName:"getLogs"`
	GetWork RpcMethod       `methodName:"getWork"`
	SubmitWork      RpcMethod       `methodName:"submitWork"`
	SubmitHashrate  RpcMethod       `methodName:"submitHashrate"`

	NewAccount	RpcMethod	`methodName:"newAccount"`
	UnlockAccount	RpcMethod	`methodName:"unlockAccount"`

	//发送环路
	SendRingHash RpcMethod	`methodName:"sendRingHash"`//发送环路凭证

	SendRing RpcMethod	`methodName:"sendRing"`//发送环路
}

type AbiMethod interface {
	Call(result interface{}, blockParameter string, args ...interface{}) error
	SendTransaction(contractAddress string, args ...interface{}) error
}

//兼容不同区块链
type Contract interface {
	GetAbi() interface{}
	GetAddress()     string
}

type Erc20Token struct {
	Contract
	Name string
	TotalSupply AbiMethod
	BalanceOf AbiMethod
	Transfer AbiMethod
	TransferFrom AbiMethod
	Approve AbiMethod
	Allowance AbiMethod
}

type LoopringProtocolImpl struct {
	Contract

	RemainAmount AbiMethod //todo:

	SubmitRing AbiMethod
	SubmitRingFingerPrint AbiMethod
	CancelOrder AbiMethod
	VerifyTokensRegistered AbiMethod
	CalculateSignerAddress AbiMethod
	CalculateOrderHash AbiMethod
	ValidateOrder AbiMethod
	AssembleOrders AbiMethod
	CalculateOrderFillAmount AbiMethod
	CalculateRingFillAmount AbiMethod
	CalculateRingFees AbiMethod
	VerifyMinerSuppliedFillRates AbiMethod
	SettleRing AbiMethod
	VerifyRingHasNoSubRing AbiMethod
}

type LoopringFingerprintRegistry struct {
	Contract
	SubmitRingFingerprint AbiMethod
	CanSubmit AbiMethod
	FingerprintFound AbiMethod
	IsExpired AbiMethod
	GetRingHash AbiMethod
}

type TokenRegistry struct {
	Contract
	RegisterToken AbiMethod
	UnregisterToken AbiMethod
	IsTokenRegistered AbiMethod
}

type Loopring struct {
	Client *Client
	Tokens map[types.Address]*Erc20Token
	LoopringImpls map[types.Address]*LoopringProtocolImpl
	LoopringFingerprints map[types.Address]*LoopringFingerprintRegistry
}


