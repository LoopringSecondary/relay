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
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/types"
	"sync"
)

/**
区块链的listener, 得到order以及ring的事件，
*/

type Whisper struct {
	ChainOrderChan chan *types.OrderMined
}

// TODO(fukun):不同的channel，应当交给orderbook统一进行后续处理，可以将channel作为函数返回值、全局变量、参数等方式
type EthClientListener struct {
	options   config.ChainClientOptions
	whisper   *Whisper
	stop      chan struct{}
	lock      sync.RWMutex
	filterIds map[string]string
	ethClient *chainclient.Client
}

func NewListener(options config.ChainClientOptions, whisper *Whisper, ethClient *chainclient.Client) *EthClientListener {
	var l EthClientListener
	l.options = options

	l.whisper = whisper
	l.ethClient = ethClient

	return &l
}

// TODO(fukun): 这里调试调不通,应当返回channel
func (l *EthClientListener) Start() {
	l.stop = make(chan struct{})

	// TODO(fukun): add filterId
	filterId := ""

	ethlog := make(chan []eth.Log)
	err := l.ethClient.Subscribe(&ethlog, filterId)
	if err != nil {
		panic(err)
	}

	// TODO(fukun): 解析log->ORDERMINED
	for {
		select {
		case l.whisper.ChainOrderChan <- &types.OrderMined{}:
			println("----")
		}
	}

	defer l.ethClient.UninstallFilter(filterId)
}

// todo(fk): convert height to big int & get token address
func (l *EthClientListener) newFilter() (string, error) {
	var filterId string

	// 使用jsonrpc的方式调用newFilter
	filter := eth.FilterQuery{}
	filter.FromBlock = types.Int2BlockNumHex(int64(height))
	filter.ToBlock = "latest"
	filter.Address = TransferTokenAddress

	err := l.ethClient.Call(&filterId, "eth_newFilter", &filter)
	if err != nil {
		return "", err
	}

	return filterId, nil
}

func (l *EthClientListener) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	close(l.stop)
}

func (l *EthClientListener) Name() string {
	return "eth-listener"
}
