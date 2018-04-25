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

//import (
//	"github.com/Loopring/relay/log"
//	"github.com/Loopring/relay/types"
//	"github.com/ethereum/go-ethereum/accounts/abi"
//	"github.com/ethereum/go-ethereum/common"
//	"github.com/ethereum/go-ethereum/rpc"
//	"math/big"
//	"math/rand"
//	"sort"
//	"strings"
//	"sync"
//	"time"
//)
//
//type MutilClient struct {
//	mtx          sync.RWMutex
//	clients      SortedClients
//	latestMaxIdx int
//}
//
//type SortedClients []*RpcClient
//
//func (clients SortedClients) Len() int {
//	return len(clients)
//}
//func (clients SortedClients) Swap(i, j int) {
//	clients[i], clients[j] = clients[j], clients[i]
//}
//func (clients SortedClients) Less(i, j int) bool {
//	return clients[i].syncingResult.CurrentBlock.BigInt().Cmp(clients[j].syncingResult.CurrentBlock.BigInt()) > 0
//}
//
//type RpcClient struct {
//	url           string
//	syncingResult *SyncingResult
//	client        *rpc.Client
//}
//
//type SyncingResult struct {
//	StartingBlock types.Big
//	CurrentBlock  types.Big
//	HighestBlock  types.Big
//}
//
//func (sr *SyncingResult) isSynced() bool {
//	return sr.CurrentBlock.BigInt().Cmp(sr.HighestBlock.BigInt()) >= 0
//}
//
//func (mc *MutilClient) startSyncStatus() {
//	go func() {
//		for {
//			select {
//			case <-time.After(time.Duration(15 * time.Second)):
//				mc.syncStatus()
//			}
//		}
//	}()
//}
//
//func (mc *MutilClient) syncStatus() {
//	mc.mtx.Lock()
//	defer mc.mtx.Unlock()
//
//	highest := big.NewInt(int64(0))
//	log.Debugf("#####, syncStatus, %s", highest.String())
//	for _, client := range mc.clients {
//		var status bool
//
//		sr := &SyncingResult{}
//
//		if err := client.client.Call(&status, "eth_syncing"); nil != err {
//			if err := client.client.Call(&sr, "eth_syncing"); nil != err {
//				log.Errorf("err:%s", err.Error())
//			} else {
//				if highest.Cmp(sr.HighestBlock.BigInt()) < 0 {
//					highest.Set(sr.HighestBlock.BigInt())
//				}
//			}
//		} else {
//			var blockNumber types.Big
//			if err := client.client.Call(&blockNumber, "eth_blockNumber"); nil != err {
//				log.Errorf("err:%s", err.Error())
//			}
//			if highest.Cmp(blockNumber.BigInt()) < 0 {
//				highest.Set(blockNumber.BigInt())
//			}
//			sr.CurrentBlock = blockNumber
//		}
//		client.syncingResult = sr
//	}
//
//	for _, c := range mc.clients {
//		c.syncingResult.HighestBlock = new(types.Big).SetInt(highest)
//	}
//	sort.Sort(mc.clients)
//
//	if len(mc.clients) > 0 {
//		latestBlockNumber := mc.clients[0].syncingResult.CurrentBlock.Int()
//		for idx, c := range mc.clients {
//			if latestBlockNumber <= c.syncingResult.CurrentBlock.Int() {
//				mc.latestMaxIdx = idx
//			}
//		}
//	}
//}
//
//func (mc *MutilClient) bestClient(routeParam string) *RpcClient {
//	var idx int
//	//latest,pending
//	lastIdx := 0
//
//	if "latest" == routeParam || "" == routeParam {
//		lastIdx = mc.latestMaxIdx
//	} else if strings.Contains(routeParam, ":") {
//		//specific node
//		for _, c := range mc.clients {
//			if routeParam == c.url {
//				return c
//			}
//		}
//	} else {
//		var blockNumberForRouteBig *big.Int
//		if strings.HasPrefix(routeParam, "0x") {
//			blockNumberForRouteBig = types.HexToBigint(routeParam)
//		} else {
//			blockNumberForRouteBig = new(big.Int)
//			blockNumberForRouteBig.SetString(routeParam, 0)
//		}
//		for curIdx, c := range mc.clients {
//			//todo:request from synced client
//			if blockNumberForRouteBig.Cmp(c.syncingResult.CurrentBlock.BigInt()) <= 0 {
//				lastIdx = curIdx
//			} else {
//				break
//			}
//		}
//
//	}
//	if lastIdx > 0 {
//		idx = rand.Intn(lastIdx)
//	}
//	client := mc.clients[idx]
//	return client
//}
//
//func (mc *MutilClient) Dail(urls []string) {
//	for _, url := range urls {
//		if client, err := rpc.Dial(url); nil != err {
//			log.Errorf("rpc.Dail err : %s, url:%s", err.Error(), url)
//		} else {
//			rpcClient := &RpcClient{}
//			rpcClient.client = client
//			rpcClient.url = url
//			mc.clients = append(mc.clients, rpcClient)
//		}
//	}
//	mc.syncStatus()
//}
//
//func (mc *MutilClient) Call(routeParam string, result interface{}, method string, args ...interface{}) (node string, err error) {
//	rpcClient := mc.bestClient(routeParam)
//	err = rpcClient.client.Call(result, method, args...)
//	return rpcClient.url, err
//}
//
//func (mc *MutilClient) BatchCall(routeParam string, b []rpc.BatchElem) (node string, err error) {
//	rpcClient := mc.bestClient(routeParam)
//	log.Debugf("client:%s", rpcClient.url)
//	err = rpcClient.client.BatchCall(b)
//	return rpcClient.url, err
//}
//
//func (mc *MutilClient) Close() {
//	for _, c := range mc.clients {
//		c.client.Close()
//	}
//}
//
//func (mc *MutilClient) Synced() bool {
//	return mc.clients[0].syncingResult.isSynced()
//}
