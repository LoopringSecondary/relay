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

package node

import (
	"sync"
	"github.com/Loopring/ringminer/matchengine"
	"go.uber.org/zap"
	"github.com/Loopring/ringminer/orderbook"
	"github.com/Loopring/ringminer/p2p"
	"github.com/Loopring/ringminer/types"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/chainclient/eth"
)

// TODO(fk): add services
type Node struct {
	options *config.GlobalConfig
	server *matchengine.Proxy
	p2pListener p2p.Listener
	ethListener eth.Listener
	orderbook *orderbook.OrderBook
	whisper *types.Whispers
	stop chan struct{}
	lock sync.RWMutex
	logger *zap.Logger
}

// TODO(fk): inject whisper and logger
func NewNode(logger *zap.Logger) *Node {
	n := &Node{}

	whisper := &types.Whispers{}
	whisper.PeerOrderChan = make(chan *types.Order)
	whisper.ChainOrderChan = make(chan *types.OrderMined)
	whisper.EngineOrderChan = make(chan *types.OrderState)

	n.whisper = whisper
	n.logger = logger
	n.options = config.LoadConfig()

	n.registerP2PListener()
	n.registerOrderBook()

	return n
}

func (n *Node) Start() {
	n.p2pListener.Start()
	n.orderbook.Start()

	// TODO(fk): start eth client
	//n.ethListener.Start()
}

func (n *Node) Wait() {
	n.lock.RLock()

	// TODO(fk): states should be judged

	stop := n.stop
	n.lock.RUnlock()

	<-stop
}

func (n *Node) Stop() {
	n.lock.RLock()

	n.p2pListener.Stop()
	n.ethListener.Stop()
	close(n.stop)

	n.lock.RUnlock()
}

func (n *Node) registerP2PListener() {
	n.p2pListener = p2p.NewListener(n.whisper, n.options.Ipfs)
}

func (n *Node) registerOrderBook() {
	n.orderbook = orderbook.NewOrderBook(n.whisper, n.options.Database)
}

func (n *Node) registerEthClient() {
	n.ethListener = eth.NewListener(n.whisper, n.options.EthClient)
}