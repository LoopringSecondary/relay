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
	"github.com/Loopring/relay/chainclient"
	ethClientLib "github.com/Loopring/relay/chainclient/eth"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	ethCryptoLib "github.com/Loopring/relay/crypto/eth"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/db"
	"github.com/Loopring/relay/extractor"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/miner/timing_matcher"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/usermanager"
	"go.uber.org/zap"
	"sync"
)

// TODO(fk): add services
type Node struct {
	globalConfig     *config.GlobalConfig
	rdsService       dao.RdsService
	ipfsSubService   gateway.IPFSSubService
	extractorService extractor.ExtractorService
	orderManager     ordermanager.OrderManager
	userManager      usermanager.UserManager
	miner            *miner.Miner
	stop             chan struct{}
	lock             sync.RWMutex
	logger           *zap.Logger
	trendManager     market.TrendManager
	jsonRpcService   gateway.JsonrpcServiceImpl
}

func NewEthNode(logger *zap.Logger, globalConfig *config.GlobalConfig) *Node {
	n := &Node{}
	n.logger = logger
	n.globalConfig = globalConfig

	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	ethClient := ethClientLib.NewChainClient(globalConfig.Accessor, globalConfig.Common.Passphrase)

	database := db.NewDB(globalConfig.Database)

	marketCapProvider := market.NewMarketCapProvider(globalConfig.Miner)

	//accessor := ethaccessor.NewAccessor(globalConfig.ChainClient)

	//n.registerMysql()
	n.registerUserManager()
	//n.registerIPFSSubService()
	n.registerGateway()
	n.registerMiner(ethClient.Client, marketCapProvider)
	n.registerExtractor(ethClient, database)
	n.registerOrderManager(database)
	n.registerTrendManager(database)
	n.registerJsonRpcService()

	return n
}

func (n *Node) Start() {
	//n.rdsService.Prepare()
	//n.extractorService.Start()
	//n.ipfsSubService.Start()
	//n.miner.Start()
	//gateway.NewJsonrpcService("8080").Start()
	//n.orderManager.Start()
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

func (n *Node) registerMysql() {
	n.rdsService = dao.NewRdsService(n.globalConfig.Mysql)
}

func (n *Node) registerExtractor(client *ethClientLib.EthClient, database db.Database) {
	n.extractorService = extractor.NewExtractorService(n.globalConfig.Accessor, n.globalConfig.Common, client, n.rdsService)
}

func (n *Node) registerIPFSSubService() {
	n.ipfsSubService = gateway.NewIPFSSubService(n.globalConfig.Ipfs)
}

func (n *Node) registerOrderManager(database db.Database) {
	n.orderManager = ordermanager.NewOrderManager(n.globalConfig.OrderManager, n.rdsService)
}

func (n *Node) registerTrendManager(database db.Database) {
	n.trendManager = market.NewTrendManager(n.rdsService)
}

func (n *Node) registerJsonRpcService() {
	n.jsonRpcService = *gateway.NewJsonrpcService(string(n.globalConfig.Jsonrpc.Port), n.trendManager)
}

func (n *Node) registerMiner(client *chainclient.Client, marketCapProvider *market.MarketCapProvider) {
	loopringInstance := chainclient.NewLoopringInstance(n.globalConfig.Common, client)
	submitter := miner.NewSubmitter(n.globalConfig.Miner, n.globalConfig.Common, client)
	matcher := timing_matcher.NewTimingMatcher()
	minerInstance := miner.NewMinerInstance(n.globalConfig.Miner, submitter, matcher, loopringInstance, marketCapProvider)
	miner.Initialize(minerInstance)

	n.miner = minerInstance
}

func (n *Node) registerGateway() {
	gateway.Initialize(&n.globalConfig.GatewayFilters)
}

func (n *Node) registerUserManager() {
	n.userManager = usermanager.NewUserManager(n.rdsService)
}
