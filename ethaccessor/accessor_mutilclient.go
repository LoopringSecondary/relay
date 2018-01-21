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
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/Loopring/relay/log"
	"sync"
	"time"
	"github.com/Loopring/relay/types"
	"sort"
	"math/big"
	"math/rand"
	"strings"
)

type MutilClient struct {
	mtx sync.RWMutex
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
	url string
	syncingResult *SyncingResult
	client *rpc.Client
}

type SyncingResult struct {
	StartingBlock types.Big
	CurrentBlock types.Big
	HighestBlock types.Big
}

func (sr *SyncingResult) isSynced() bool {
	//todo:
	return true
}

func (mc *MutilClient) syncStatus() {
	go func() {
		for {
			select {
			case <- time.After(time.Duration(30 * time.Second)):
				mc.mtx.Lock()
				defer mc.mtx.Unlock()

				for _,client := range mc.clients {
					sr := &SyncingResult{}
					if err := client.client.Call(sr, "eth_syncing"); nil != err {
						//todo:
					} else {
						client.syncingResult = sr
					}
				}
				sort.Sort(mc.clients)
			}
		}
	}()
}

func (mc *MutilClient) bestClient(routeParam string) *RpcClient {
	var idx int
	//latest,pending
	if "latest" == routeParam {
		idx = 0
	} else if strings.Contains(routeParam, ":") {
		//specific node
		for _,c := range mc.clients {
			if routeParam == c.url {
				return c
			}
		}
	} else {
		blockNumberForRouteBig,_ := new(big.Int).SetString(routeParam, 0)
		lastIdx := 0
		for curIdx,c := range mc.clients {
			if blockNumberForRouteBig.Cmp(c.syncingResult.CurrentBlock.BigInt()) <= 0 {
				lastIdx = curIdx
			} else {
				break
			}
		}
		idx = rand.Intn(lastIdx)
	}
	return mc.clients[idx]
}

func (mc *MutilClient) Dail(urls []string) {
	for _,url := range urls {
		if client,err := rpc.Dial(url); nil != err {
			log.Errorf("rpc.Dail err : %s, url:%s", err.Error(), url)
		} else {
			mc = append(mc, client)
		}
	}
}

func (mc *MutilClient) Call(routeParam string, result interface{}, method string, args ...interface{}) (node string, err error) {
	rpcClient := mc.bestClient(routeParam)
	err = rpcClient.client.Call(result , method , args...)
	return rpcClient.url, err
}

func (mc *MutilClient) BatchCall(routeParam string, b []rpc.BatchElem) (node string, err error) {
	rpcClient := mc.bestClient(routeParam)
	err = rpcClient.client.BatchCall(b)
	return rpcClient.url,err
}

func (mc *MutilClient) Close() {
	for _,c := range *mc.clients {
		c.client.Close()
	}
}

func (mc *MutilClient) Synced() bool {
	return mc.clients[0].syncingResult.isSynced()
}

type NodeAccessor struct {

}