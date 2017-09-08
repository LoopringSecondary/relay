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

package eth

import (
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/Loopring/ringminer/chainclient"
	"reflect"
	"github.com/Loopring/ringminer/types"
	"time"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

/**
todo：未完成：
1、返回的struct定义
2、新订单、余额变动等事件定义
3、余额等变动的处理
4、应当给orderbook持有listener，然后在orderbook内部处理各种event，否则处理逻辑分散
 */

var RPCClient *rpc.Client

var EthClient *chainclient.Client

func newRpcMethod(name string) func(result interface{}, args ...interface{}) error {
	return func(result interface{}, args ...interface{}) error  {
		return RPCClient.Call(result, name, args...)
	}
}

//todo:hexutil.Big是否应被更合理地替换
type CallArgs struct {
	From	string
	To	string
	Gas      hexutil.Big
	GasPrice hexutil.Big
	Value	hexutil.Big
	Data	string
}

func NewClient() *chainclient.Client {
	client := &chainclient.Client{}

	//set rpcmethod
	applyMethod(client)
	//Subscribe
	client.Subscribe = subscribe
	return client
}

func dGetOrder(chanVal reflect.Value) {
	//todo:test
	i := 10
	for {
		i = i + 1
		select {
		case <-time.Tick(1000000):
			// Id:types.BytesToHash([]byte("idx:" + strconv.Itoa(i)))
			ord := &types.OrderState{RawOrder:types.Order{}}
			//(*r) <- ord
			chanVal.Send(reflect.ValueOf(ord))
		}
	}
}

func subscribe(result interface{}, args ...interface{}) error {
	//先获取filterId，然后定时更新filterChanges
	//todo:类型不定，需要根据不同情况返回不同值
	//r := result.(*chan *types.NewOrderEvent)
	chanVal := reflect.ValueOf(result).Elem()
	go dGetOrder(chanVal)
	return nil
}

func applyMethod(client *chainclient.Client) error {
	methodNameMap := map[string]string{
		"clientVersion":"web3_clientVersion",
		"sha3":"web3_sha3",
		"version":"net_version",
		"peerCount":"net_peerCount",
		"listening":"net_listening",
		"protocolVersion":"eth_protocolVersion",
		"syncing":"eth_syncing",
		"coinbase":"eth_coinbase",
		"mining":"eth_mining",
		"hashrate":"eth_hashrate",
		"gasPrice":"eth_gasPrice",
		"accounts":"eth_accounts",
		"blockNumber":"eth_blockNumber",
		"getBalance":"eth_getBalance",
		"getStorageAt":"eth_getStorageAt",
		"getTransactionCount":"eth_getTransactionCount",
		"getBlockTransactionCountByHash":"eth_getBlockTransactionCountByHash",
		"getBlockTransactionCountByNumber":"eth_getBlockTransactionCountByNumber",
		"getUncleCountByBlockHash":"eth_getUncleCountByBlockHash",
		"getUncleCountByBlockNumber":"eth_getUncleCountByBlockNumber",
		"getCode":"eth_getCode",
		"sign":"eth_sign",
		"sendTransaction":"eth_sendTransaction",
		"sendRawTransaction":"eth_sendRawTransaction",
		"call":"eth_call",
		"estimateGas":"eth_estimateGas",
		"getBlockByHash":"eth_getBlockByHash",
		"getBlockByNumber":"eth_getBlockByNumber",
		"getTransactionByHash":"eth_getTransactionByHash",
		"getTransactionByBlockHashAndIndex":"eth_getTransactionByBlockHashAndIndex",
		"getTransactionByBlockNumberAndIndex":"eth_getTransactionByBlockNumberAndIndex",
		"getTransactionReceipt":"eth_getTransactionReceipt",
		"getUncleByBlockHashAndIndex":"eth_getUncleByBlockHashAndIndex",
		"getUncleByBlockNumberAndIndex":"eth_getUncleByBlockNumberAndIndex",
		"getCompilers":"eth_getCompilers",
		"compileLLL":"eth_compileLLL",
		"compileSolidity":"eth_compileSolidity",
		"compileSerpent":"eth_compileSerpent",
		"newFilter":"eth_newFilter",
		"newBlockFilter":"eth_newBlockFilter",
		"newPendingTransactionFilter":"eth_newPendingTransactionFilter",
		"uninstallFilter":"eth_uninstallFilter",
		"getFilterChanges":"eth_getFilterChanges",
		"getFilterLogs":"eth_getFilterLogs",
		"getLogs":"eth_getLogs",
		"getWork":"eth_getWork",
		"submitWork":"eth_submitWork",
		"submitHashrate":"eth_submitHashrate",

		"unlockAccount":"personal_unlockAccount",
		"newAccount":"personal_newAccount",
	}
	v := reflect.ValueOf(client).Elem()
	for i:=0; i < v.NumField();i++ {
		fieldV := v.Field(i)
		methodName := methodNameMap[v.Type().Field(i).Tag.Get("methodName")]
		if (methodName != "") {
			fieldV.Set(reflect.ValueOf(newRpcMethod(methodName)))
		}
	}
	return nil
}

func init() {
	//TODO：change to inject
	EthClient = NewClient()
}