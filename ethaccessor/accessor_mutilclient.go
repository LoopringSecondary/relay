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
	"errors"
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"math/rand"
	"strings"
	"sync"
	"time"
)

const (
	USAGE_CLIENT_BLOCK = "usage_client_block_"
	BLOCKS             = "blocks_"
	blocks_count       = int64(2000)
	cacheDuration      = 86400 * 3
)

type MutilClient struct {
	clients       map[string]*RpcClient
	downedClients map[string]*RpcClient
}

type RpcClient struct {
	url         string
	client      *rpc.Client
	blockNumber *big.Int
}

type SyncingResult struct {
	StartingBlock types.Big
	CurrentBlock  types.Big
	HighestBlock  types.Big
}

//将最近的块放入redis中，获取时，从redis中按照块号获取可用的client与本地保存做交集，然后随机选取client，请求节点
func NewMutilClient(urls []string) *MutilClient {
	mc := &MutilClient{}
	mc.clients = make(map[string]*RpcClient)
	mc.downedClients = make(map[string]*RpcClient)
	for _, url := range urls {
		mc.newRpcClient(url)
	}
	return mc
}

func (mc *MutilClient) newRpcClient(url string) {
	rpcClient := &RpcClient{}
	rpcClient.url = url
	if client, err := rpc.DialHTTP(url); nil != err {
		log.Errorf("rpc.Dail err : %s, url:%s", err.Error(), url)
		mc.downedClients[url] = rpcClient
	} else {
		rpcClient.client = client
		mc.clients[url] = rpcClient
	}
}

func (mc *MutilClient) bestClient(routeParam string) *RpcClient {
	//latest,pending

	var blockNumber types.Big
	if "latest" == routeParam || "" == routeParam {
		//lastIdx = mc.latestMaxIdx
		mc.BlockNumber(&blockNumber)
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
		blockNumber = *types.NewBigPtr(blockNumberForRouteBig)
	}

	urls, _ := mc.useageClient(blockNumber.BigInt().String())

	for _, url := range urls {
		if _, exists := mc.clients[url]; !exists {
			mc.newRpcClient(url)
		}
	}

	if len(urls) <= 0 {
		for url, client := range mc.clients {
			if _, exists := mc.downedClients[url]; !exists && (nil == client.blockNumber || client.blockNumber.Cmp(blockNumber.BigInt()) >= 0) {
				urls = append(urls, url)
			}
		}
	}

	if len(urls) == 0 {
		log.Debugf("len(urls) == 0")
		mc.syncBlockNumber()
		for url, client := range mc.clients {
			if _, exists := mc.downedClients[url]; !exists && (nil == client.blockNumber || client.blockNumber.Cmp(blockNumber.BigInt()) >= 0) {
				urls = append(urls, url)
			}
		}
		log.Debugf("after syncBlockNumber len(urls) == %d", len(urls))
	}

	if len(urls) > 0 {
		idx := 0
		idx = rand.Intn(len(urls))
		client := mc.clients[urls[idx]]
		return client
	} else {
		return nil
	}
}

func (mc *MutilClient) syncBlockNumber() {
	for _, client := range mc.clients {
		var blockNumber types.Big
		if err := client.client.Call(&blockNumber, "eth_blockNumber"); nil != err {
			mc.downedClients[client.url] = client
		} else {
			delete(mc.downedClients, client.url)
			client.blockNumber = blockNumber.BigInt()
			blockNumberStr := blockNumber.BigInt().String()
			cache.SAdd(USAGE_CLIENT_BLOCK+blockNumberStr, cacheDuration, []byte(client.url))
			cache.ZAdd(BLOCKS, int64(0), []byte(blockNumberStr), []byte(blockNumberStr))
			cache.ZRemRangeByScore(BLOCKS, int64(0), blockNumber.Int64()-blocks_count)
		}
	}
}

func (mc *MutilClient) startSyncBlockNumber() {
	go func() {
		for {
			select {
			case <-time.After(time.Duration(3 * time.Second)):
				mc.syncBlockNumber()
			}
		}
	}()
}

func (mc *MutilClient) BlockNumber(result interface{}) error {
	if data, err := cache.ZRange(BLOCKS, -1, -1, false); nil != err {
		return err
	} else {
		if len(data) > 0 && len(data[0]) > 0 {
			if r, ok := result.(*types.Big); ok {
				blockNumber := new(big.Int)
				blockNumber.SetString(string(data[0]), 0)
				r.SetInt(blockNumber)
			} else {
				errors.New("Wrong `result` type, please use types.Big ")
			}
			return nil
		} else {
			return errors.New("BlockNumber can't get from cache")
		}
	}
}

func (mc *MutilClient) useageClient(blockNumberStr string) ([]string, error) {
	urls := []string{}
	if data, err := cache.SMembers(USAGE_CLIENT_BLOCK + blockNumberStr); nil != err {
		return urls, err
	} else {
		if len(data) > 0 {
			for _, d := range data {
				if len(d) > 0 {
					//log.Debugf("useageClient:%s, %s", string(d), blockNumberStr)
					urls = append(urls, string(d))
				} else {
					log.Debug("useageClient len(d) == 0")
				}
			}
		} else {
			return urls, errors.New("cant get client by blocknumer:" + blockNumberStr)
		}
	}
	return urls, nil
}

func (mc *MutilClient) Call(routeParam string, result interface{}, method string, args ...interface{}) (node string, err error) {
	//blocknumber 特殊处理下
	if "eth_blockNumber" == method {
		err = mc.BlockNumber(result)
	}
	if "eth_blockNumber" == method && nil == err {
		return "", nil
	} else if "eth_sendRawTransaction" == method {
		var (
			sendSuccess bool
			err error
		)
		for _,client := range mc.clients {
			if err1 := client.client.Call(result, method, args...); nil == err1 {
				sendSuccess = true
			} else {
				err = err1
			}
		}
		if !sendSuccess {
			return "", err
		} else {
			return "", nil
		}
	} else {
		rpcClient := mc.bestClient(routeParam)
		if nil == rpcClient {
			return "", errors.New("there isn't an usable ethnode")
		}
		log.Debugf("rpcClient:%s, %s", rpcClient.url, routeParam)
		err = rpcClient.client.Call(result, method, args...)
		return rpcClient.url, err
	}
}

func (mc *MutilClient) BatchCall(routeParam string, b []rpc.BatchElem) (node string, err error) {
	rpcClient := mc.bestClient(routeParam)
	if nil == rpcClient {
		return "", errors.New("there isn't an usable ethnode")
	}
	err = rpcClient.client.BatchCall(b)
	return rpcClient.url, err
}

type ethNodeAccessor struct {
	Erc20Abi         *abi.ABI
	ProtocolImplAbi  *abi.ABI
	DelegateAbi      *abi.ABI
	TokenRegistryAbi *abi.ABI
	//NameRegistryAbi   *abi.ABI
	WethAbi           *abi.ABI
	WethAddress       common.Address
	ProtocolAddresses map[common.Address]*ProtocolAddress
	DelegateAddresses map[common.Address]bool

	*MutilClient
	gasPriceEvaluator *GasPriceEvaluator
	mtx               sync.RWMutex
	AddressNonce      map[common.Address]*big.Int
	fetchTxRetryCount int
}

type AddressNonce struct {
	Address common.Address
	Nonce   *big.Int
	mtx     sync.RWMutex
}
