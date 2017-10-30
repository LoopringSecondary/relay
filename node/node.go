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
	"github.com/Loopring/ringminer/chainclient"
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
	"go.uber.org/zap"
	"sync"
)

// TODO(fk): add services
type Node struct {
	globalConfig  *config.GlobalConfig
	p2pListener   listener.Listener
	chainListener listener.Listener
	orderbook     *orderbook.OrderBook
	miner         *miner.Miner
	stop          chan struct{}
	lock          sync.RWMutex
	logger        *zap.Logger
}

func NewEthNode(logger *zap.Logger, globalConfig *config.GlobalConfig) *Node {
	n := &Node{}
	n.logger = logger
	n.globalConfig = globalConfig

	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	ethClient := ethClientLib.NewChainClient(globalConfig.ChainClient, globalConfig.Common.Passphrase)

	database := db.NewDB(globalConfig.Database)

	n.registerP2PListener()
	n.registerOrderBook(database)
	n.registerMiner(ethClient.Client, database)
	n.registerEthListener(ethClient, database)

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

func (n *Node) registerEthListener(client *ethClientLib.EthClient, database db.Database) {
	n.chainListener = ethListenerLib.NewListener(n.globalConfig.ChainClient, n.globalConfig.Common, client, n.orderbook, database)
}

func (n *Node) registerP2PListener() {
	n.p2pListener = ipfsListenerLib.NewListener(n.globalConfig.Ipfs)
}

func (n *Node) registerOrderBook(database db.Database) {
	n.orderbook = orderbook.NewOrderBook(n.globalConfig.Orderbook, n.globalConfig.Common, database)
}

func (n *Node) registerMiner(client *chainclient.Client, database db.Database) {
	loopringInstance := chainclient.NewLoopringInstance(n.globalConfig.Common, client)
	submitter := miner.NewRingSubmitClient(n.globalConfig.Miner, n.globalConfig.Common, database, client)
	rateProvider := miner.NewLegalRateProvider(n.globalConfig.Miner)
	matcher := bucket.NewBucketMatcher(submitter, n.globalConfig.Miner.RingMaxLength)
	minerInstance := miner.NewMiner(n.globalConfig.Miner, submitter, matcher, loopringInstance, rateProvider)
	miner.Initialize(minerInstance)

	n.miner = minerInstance
}
