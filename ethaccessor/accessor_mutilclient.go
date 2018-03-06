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

package ethaccessor

import (
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
	//"encoding/json"
)

type MutilClient struct {
	mtx     sync.RWMutex
	clients SortedClients
}

type SortedClients []*RpcClient

func (clients SortedClients) Len() int {
	return len(clients)
}
func (clients SortedClients) Swap(i, j int) {
	clients[i], clients[j] = clients[j], clients[i]
}
func (clients SortedClients) Less(i, j int) bool {
	return clients[i].syncingResult.CurrentBlock.BigInt().Cmp(clients[j].syncingResult.CurrentBlock.BigInt()) > 0
}

type RpcClient struct {
	url           string
	syncingResult *SyncingResult
	client        *rpc.Client
}

type SyncingResult struct {
	StartingBlock types.Big
	CurrentBlock  types.Big
	HighestBlock  types.Big
}

func (sr *SyncingResult) isSynced() bool {
	//todo:
	return true
}

func (mc *MutilClient) startSyncStatus() {
	go func() {
		for {
			select {
			case <-time.After(time.Duration(10 * time.Second)):
				mc.syncStatus()
			}
		}
	}()
}

func (mc *MutilClient) syncStatus() {
	mc.mtx.Lock()
	defer mc.mtx.Unlock()

	highest := big.NewInt(int64(0))
	for _, client := range mc.clients {
		var blockNumber types.Big
		if err := client.client.Call(&blockNumber, "eth_blockNumber"); nil != err {
			//todo:
		}
		if highest.Cmp(blockNumber.BigInt()) < 0 {
			highest.Set(blockNumber.BigInt())
		}
		var status bool

		sr := &SyncingResult{}
		sr.CurrentBlock = blockNumber

		if err := client.client.Call(&status, "eth_syncing"); nil != err {
			//todo:
			if err := client.client.Call(&sr, "eth_syncing"); nil != err {
				//todo:
			}
		}
		client.syncingResult = sr
	}

	for _, c := range mc.clients {
		c.syncingResult.HighestBlock = new(types.Big).SetInt(highest)
	}
	sort.Sort(mc.clients)
}

func (mc *MutilClient) bestClient(routeParam string) *RpcClient {
	var idx int
	//latest,pending
	if "latest" == routeParam || "" == routeParam {
		idx = 0
	} else if strings.Contains(routeParam, ":") {
		//specific node
		for _, c := range mc.clients {
			if routeParam == c.url {
				return c
			}
		}
	} else {
		var blockNumberForRouteBig *big.Int
		if strings.HasPrefix(routeParam, "0x") {
			blockNumberForRouteBig = types.HexToBigint(routeParam)
		} else {
			blockNumberForRouteBig = new(big.Int)
			blockNumberForRouteBig.SetString(routeParam, 0)
		}
		lastIdx := 0
		for curIdx, c := range mc.clients {
			//todo:request from synced client
			if blockNumberForRouteBig.Cmp(c.syncingResult.CurrentBlock.BigInt()) <= 0 {
				lastIdx = curIdx
			} else {
				break
			}
		}
		if lastIdx > 0 {
			idx = rand.Intn(lastIdx)
		}
	}
	return mc.clients[idx]
}

func (mc *MutilClient) Dail(urls []string) {
	for _, url := range urls {
		if client, err := rpc.Dial(url); nil != err {
			log.Errorf("rpc.Dail err : %s, url:%s", err.Error(), url)
		} else {
			rpcClient := &RpcClient{}
			rpcClient.client = client
			rpcClient.url = url
			mc.clients = append(mc.clients, rpcClient)
		}
	}
	mc.syncStatus()
}

func (mc *MutilClient) Call(routeParam string, result interface{}, method string, args ...interface{}) (node string, err error) {
	rpcClient := mc.bestClient(routeParam)
	err = rpcClient.client.Call(result, method, args...)
	return rpcClient.url, err
}

func (mc *MutilClient) BatchCall(routeParam string, b []rpc.BatchElem) (node string, err error) {
	rpcClient := mc.bestClient(routeParam)
	err = rpcClient.client.BatchCall(b)
	return rpcClient.url, err
}

func (mc *MutilClient) Close() {
	for _, c := range mc.clients {
		c.client.Close()
	}
}

func (mc *MutilClient) Synced() bool {
	return mc.clients[0].syncingResult.isSynced()
}

type ethNodeAccessor struct {
	Erc20Abi          *abi.ABI
	ProtocolImplAbi   *abi.ABI
	DelegateAbi       *abi.ABI
	TokenRegistryAbi  *abi.ABI
	NameRegistryAbi   *abi.ABI
	WethAbi           *abi.ABI
	WethAddress       common.Address
	ProtocolAddresses map[common.Address]*ProtocolAddress
	*MutilClient
	gasPriceEvaluator *GasPriceEvaluator
}
