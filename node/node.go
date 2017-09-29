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
	ethClient "github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	ethCrypto "github.com/Loopring/ringminer/crypto/eth"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/listener"
	ethListener "github.com/Loopring/ringminer/listener/chain/eth"
	ipfsListener "github.com/Loopring/ringminer/listener/p2p/ipfs"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/miner/bucket"
	"github.com/Loopring/ringminer/orderbook"
	"github.com/Loopring/ringminer/types"
	"go.uber.org/zap"
	"sync"
)

// TODO(fk): add services
type Node struct {
	globalConfig  *config.GlobalConfig
	p2pListener   listener.Listener
	chainListener listener.Listener
	orderbook     *orderbook.OrderBook
	miner         miner.Proxy
	stop          chan struct{}
	lock          sync.RWMutex
	logger        *zap.Logger
}

// TODO(fk): inject whisper
func NewEthNode(logger *zap.Logger, globalConfig *config.GlobalConfig) *Node {
	n := &Node{}
	n.logger = logger
	n.globalConfig = globalConfig
	ethClient.Initialize(n.globalConfig.ChainClient)

	database := db.NewDB(globalConfig.Database)
	ringClient := miner.NewRingClient(database, ethClient.EthClient)
	//
	miner.Initialize(n.globalConfig.Miner, ringClient.Chainclient)
	//
	peerOrderChan := make(chan *types.Order)
	chainOrderChan := make(chan *types.OrderMined)
	engineOrderChan := make(chan *types.OrderState)
	//
	n.registerP2PListener(peerOrderChan)
	//n.registerEthListener(chainOrderChan)
	//
	n.registerOrderBook(database, peerOrderChan, chainOrderChan, engineOrderChan)
	n.registerMiner(ringClient, engineOrderChan)

	crypto.CryptoInstance = &ethCrypto.EthCrypto{Homestead: false}

	return n
}

func (n *Node) Start() {
	//n.chainListener.Start()
	n.p2pListener.Start()
	//

	n.orderbook.Start()
	n.miner.Start()
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

	//
	//n.p2pListener.Stop()
	//n.chainListener.Stop()
	//n.orderbook.Stop()
	//n.miner.Stop()

	close(n.stop)

	n.lock.RUnlock()
}

func (n *Node) registerEthListener(chainOrderChan chan *types.OrderMined) {
	whisper := &ethListener.Whisper{chainOrderChan}
	n.chainListener = ethListener.NewListener(n.globalConfig.ChainClient, whisper)
}

func (n *Node) registerP2PListener(peerOrderChan chan *types.Order) {
	whisper := &ipfsListener.Whisper{peerOrderChan}
	n.p2pListener = ipfsListener.NewListener(n.globalConfig.Ipfs, whisper)
}

func (n *Node) registerOrderBook(database db.Database, peerOrderChan chan *types.Order, chainOrderChan chan *types.OrderMined, engineOrderChan chan *types.OrderState) {
	whisper := &orderbook.Whisper{PeerOrderChan: peerOrderChan, EngineOrderChan: engineOrderChan, ChainOrderChan: chainOrderChan}
	n.orderbook = orderbook.NewOrderBook(n.globalConfig.ObOptions, database, whisper)
}

func (n *Node) registerMiner(ringClient *miner.RingClient, orderStateChan chan *types.OrderState) {
	whisper := bucket.Whisper{orderStateChan}
	n.miner = bucket.NewBucketProxy(ringClient, whisper)
}
