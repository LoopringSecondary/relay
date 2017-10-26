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
	ethClientLib "github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	ethCryptoLib "github.com/Loopring/ringminer/crypto/eth"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/listener"
	ethListenerLib "github.com/Loopring/ringminer/listener/chain/eth"
	ipfsListenerLib "github.com/Loopring/ringminer/listener/p2p/ipfs"
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

func NewEthNode(logger *zap.Logger, globalConfig *config.GlobalConfig) *Node {
	n := &Node{}
	n.logger = logger
	n.globalConfig = globalConfig

	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	bucket.RingLength = globalConfig.Miner.RingMaxLength

	ethClient := ethClientLib.NewChainClient(globalConfig.ChainClient, globalConfig.Common.Passphrase)

	database := db.NewDB(globalConfig.Database)
	//forkDetectChans := []chan chainclient.ForkedEvent{make(chan chainclient.ForkedEvent, 10)}
	//ethClient.StartForkDetect(forkDetectChans, database)
	ringClient := miner.NewRingClient(database, ethClient.Client)

	miner.Initialize(n.globalConfig.Miner, n.globalConfig.Common, ringClient.Chainclient)

	peerOrderChan := make(chan *types.Order)
	chainOrderChan := make(chan *types.OrderState)
	engineOrderChan := make(chan *types.OrderState)

	n.registerP2PListener(peerOrderChan)
	n.registerOrderBook(database, peerOrderChan, chainOrderChan, engineOrderChan)
	n.registerMiner(ringClient, engineOrderChan)
	n.registerEthListener(ethClient, database, chainOrderChan)

	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	return n
}

func (n *Node) Start() {

	n.chainListener.Start()
	n.p2pListener.Start()
	n.miner.Start()

	n.orderbook.Start()

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

func (n *Node) registerEthListener(client *ethClientLib.EthClient, database db.Database, chainOrderChan chan *types.OrderState) {
	whisper := &ethListenerLib.Whisper{chainOrderChan}
	n.chainListener = ethListenerLib.NewListener(n.globalConfig.ChainClient, n.globalConfig.Common, whisper, client, n.orderbook, database)
}

func (n *Node) registerP2PListener(peerOrderChan chan *types.Order) {
	whisper := &ipfsListenerLib.Whisper{peerOrderChan}
	n.p2pListener = ipfsListenerLib.NewListener(n.globalConfig.Ipfs, whisper)
}

func (n *Node) registerOrderBook(database db.Database, peerOrderChan chan *types.Order, chainOrderChan chan *types.OrderState, engineOrderChan chan *types.OrderState) {
	whisper := &orderbook.Whisper{PeerOrderChan: peerOrderChan, EngineOrderChan: engineOrderChan, ChainOrderChan: chainOrderChan}
	n.orderbook = orderbook.NewOrderBook(n.globalConfig.Orderbook, n.globalConfig.Common, database, whisper)
}

func (n *Node) registerMiner(ringClient *miner.RingClient, orderStateChan chan *types.OrderState) {
	whisper := bucket.Whisper{orderStateChan}
	n.miner = bucket.NewBucketProxy(ringClient, whisper)
}
