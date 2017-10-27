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

//similar to web3
type RpcMethod func(result interface{}, args ...interface{}) error

type ForkedEvent struct {
	DetectedBlock *big.Int
	DetectedHash  types.Hash
	ForkBlock     *big.Int
	ForkHash      types.Hash
}

type BlockIterator interface {
	Next() (interface{}, error)
	Prev() (interface{}, error)
}

type Client struct {
	//subscribe, signAndSendTransaction and NewContract are customed
	//the first arg must be filterId in eth
	//Subscribe              func(callback func(args ...interface{}) error, filterArgs ...interface{}) error
	Subscribe              func(result interface{}, filterId string) error
	BlockIterator          func(startNumber, endNumber *big.Int, withTxData bool, confirms uint64) BlockIterator
	SignAndSendTransaction func(result interface{}, from types.Address, transaction interface{}) error
	NewContract            func(result interface{}, address, abiStr string) error

	//rpc method:
	ClientVersion                       RpcMethod `methodName:"clientVersion"`
	Sha3                                RpcMethod `methodName:"sha3"`
	Version                             RpcMethod `methodName:"version"`
	PeerCount                           RpcMethod `methodName:"peerCount"`
	Listening                           RpcMethod `methodName:"listening"`
	ProtocolVersion                     RpcMethod `methodName:"protocolVersion"`
	Syncing                             RpcMethod `methodName:"syncing"`
	Coinbase                            RpcMethod `methodName:"coinbase"`
	Mining                              RpcMethod `methodName:"mining"`
	Hashrate                            RpcMethod `methodName:"hashrate"`
	GasPrice                            RpcMethod `methodName:"gasPrice"`
	Accounts                            RpcMethod `methodName:"accounts"`
	BlockNumber                         RpcMethod `methodName:"blockNumber"`
	GetBalance                          RpcMethod `methodName:"getBalance"`
	GetStorageAt                        RpcMethod `methodName:"getStorageAt"`
	GetTransactionCount                 RpcMethod `methodName:"getTransactionCount"`
	GetBlockTransactionCountByHash      RpcMethod `methodName:"getBlockTransactionCountByHash"`
	GetBlockTransactionCountByNumber    RpcMethod `methodName:"getBlockTransactionCountByNumber"`
	GetUncleCountByBlockHash            RpcMethod `methodName:"getUncleCountByBlockHash"`
	GetUncleCountByBlockNumber          RpcMethod `methodName:"getUncleCountByBlockNumber"`
	GetCode                             RpcMethod `methodName:"getCode"`
	Sign                                RpcMethod `methodName:"sign"`
	SendTransaction                     RpcMethod `methodName:"sendTransaction"`
	SendRawTransaction                  RpcMethod `methodName:"sendRawTransaction"`
	Call                                RpcMethod `methodName:"call"`
	EstimateGas                         RpcMethod `methodName:"estimateGas"`
	GetBlockByHash                      RpcMethod `methodName:"getBlockByHash"`
	GetBlockByNumber                    RpcMethod `methodName:"getBlockByNumber"`
	GetTransactionByHash                RpcMethod `methodName:"getTransactionByHash"`
	GetTransactionByBlockHashAndIndex   RpcMethod `methodName:"getTransactionByBlockHashAndIndex"`
	GetTransactionByBlockNumberAndIndex RpcMethod `methodName:"getTransactionByBlockNumberAndIndex"`
	GetTransactionReceipt               RpcMethod `methodName:"getTransactionReceipt"`
	GetUncleByBlockHashAndIndex         RpcMethod `methodName:"getUncleByBlockHashAndIndex"`
	GetUncleByBlockNumberAndIndex       RpcMethod `methodName:"getUncleByBlockNumberAndIndex"`
	GetCompilers                        RpcMethod `methodName:"getCompilers"`
	CompileLLL                          RpcMethod `methodName:"compileLLL"`
	CompileSolidity                     RpcMethod `methodName:"compileSolidity"`
	CompileSerpent                      RpcMethod `methodName:"compileSerpent"`
	NewFilter                           RpcMethod `methodName:"newFilter"`
	NewBlockFilter                      RpcMethod `methodName:"newBlockFilter"`
	NewPendingTransactionFilter         RpcMethod `methodName:"newPendingTransactionFilter"`
	UninstallFilter                     RpcMethod `methodName:"uninstallFilter"`
	GetFilterChanges                    RpcMethod `methodName:"getFilterChanges"`
	GetFilterLogs                       RpcMethod `methodName:"getFilterLogs"`
	GetLogs                             RpcMethod `methodName:"getLogs"`
	GetWork                             RpcMethod `methodName:"getWork"`
	SubmitWork                          RpcMethod `methodName:"submitWork"`
	SubmitHashrate                      RpcMethod `methodName:"submitHashrate"`

	NewAccount    RpcMethod `methodName:"newAccount"`
	UnlockAccount RpcMethod `methodName:"unlockAccount"`
}
