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
	"github.com/Loopring/relay/config"
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

func (mc *MutilClient) syncStatus() {
	go func() {
		for {
			select {
			case <-time.After(time.Duration(30 * time.Second)):
				mc.mtx.Lock()
				defer mc.mtx.Unlock()

				for _, client := range mc.clients {
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
		for _, c := range mc.clients {
			if routeParam == c.url {
				return c
			}
		}
	} else {
		blockNumberForRouteBig, _ := new(big.Int).SetString(routeParam, 0)
		lastIdx := 0
		for curIdx, c := range mc.clients {
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
	for _, url := range urls {
		if client, err := rpc.Dial(url); nil != err {
			log.Errorf("rpc.Dail err : %s, url:%s", err.Error(), url)
		} else {
			rpcClient := &RpcClient{}
			rpcClient.client = client
			rpcClient.url = url
			sr := &SyncingResult{}
			if err := client.Call(sr, "eth_syncing"); nil != err {
				//todo:
			} else {
				rpcClient.syncingResult = sr
				mc.clients = append(mc.clients, rpcClient)
			}
		}
	}
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

type EthNodeAccessor struct {
	Erc20Abi            *abi.ABI
	ProtocolImplAbi     *abi.ABI
	DelegateAbi         *abi.ABI
	RinghashRegistryAbi *abi.ABI
	TokenRegistryAbi    *abi.ABI
	WethAbi             *abi.ABI
	WethAddress         common.Address
	ProtocolAddresses   map[common.Address]*ProtocolAddress
	*MutilClient
}

func NewAccessor(accessorOptions config.AccessorOptions, commonOptions config.CommonOptions, wethAddress common.Address) (*EthNodeAccessor, error) {
	var err error
	accessor := &EthNodeAccessor{}
	//accessor.Client, err = rpc.Dial(accessorOptions.RawUrl)
	accessor.MutilClient = &MutilClient{}
	accessor.MutilClient.Dail(accessorOptions.RawUrls)
	if nil != err {
		return nil, err
	}

	if accessor.Erc20Abi, err = NewAbi(commonOptions.Erc20Abi); nil != err {
		return nil, err
	}

	if accessor.WethAbi, err = NewAbi(commonOptions.WethAbi); nil != err {
		return nil, err
	}
	accessor.WethAddress = wethAddress

	accessor.ProtocolAddresses = make(map[common.Address]*ProtocolAddress)

	if protocolImplAbi, err := NewAbi(commonOptions.ProtocolImpl.ImplAbi); nil != err {
		return nil, err
	} else {
		accessor.ProtocolImplAbi = protocolImplAbi
	}
	if registryAbi, err := NewAbi(commonOptions.ProtocolImpl.RegistryAbi); nil != err {
		return nil, err
	} else {
		accessor.RinghashRegistryAbi = registryAbi
	}
	if transferDelegateAbi, err := NewAbi(commonOptions.ProtocolImpl.DelegateAbi); nil != err {
		return nil, err
	} else {
		accessor.DelegateAbi = transferDelegateAbi
	}
	if tokenRegistryAbi, err := NewAbi(commonOptions.ProtocolImpl.TokenRegistryAbi); nil != err {
		return nil, err
	} else {
		accessor.TokenRegistryAbi = tokenRegistryAbi
	}

	for version, address := range commonOptions.ProtocolImpl.Address {
		impl := &ProtocolAddress{Version: version, ContractAddress: common.HexToAddress(address)}
		callMethod := accessor.ContractCallMethod("latest", accessor.ProtocolImplAbi, impl.ContractAddress)
		var addr string
		if err := callMethod(&addr, "lrcTokenAddress", "latest"); nil != err {
			return nil, err
		} else {
			impl.LrcTokenAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "ringhashRegistryAddress", "latest"); nil != err {
			return nil, err
		} else {
			impl.RinghashRegistryAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "tokenRegistryAddress", "latest"); nil != err {
			return nil, err
		} else {
			impl.TokenRegistryAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "delegateAddress", "latest"); nil != err {
			return nil, err
		} else {
			impl.DelegateAddress = common.HexToAddress(addr)
		}
		accessor.ProtocolAddresses[impl.ContractAddress] = impl
	}

	return accessor, nil
}
