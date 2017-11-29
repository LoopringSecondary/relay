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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/extractor"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/miner/timing_matcher"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/usermanager"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"go.uber.org/zap"
	"sync"
	"strconv"
)

// TODO(fk): add services
type Node struct {
	globalConfig     *config.GlobalConfig
	rdsService       dao.RdsService
	ipfsSubService   gateway.IPFSSubService
	accessor         *ethaccessor.EthNodeAccessor
	extractorService extractor.ExtractorService
	orderManager     ordermanager.OrderManager
	userManager      usermanager.UserManager
	miner            *miner.Miner
	stop             chan struct{}
	lock             sync.RWMutex
	logger           *zap.Logger
	trendManager     market.TrendManager
	accountManager   market.AccountManager
	jsonRpcService   gateway.JsonrpcServiceImpl
}

func NewEthNode(logger *zap.Logger, globalConfig *config.GlobalConfig) *Node {
	n := &Node{}
	n.logger = logger
	n.globalConfig = globalConfig

	ks := keystore.NewKeyStore(n.globalConfig.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	accessor, err := ethaccessor.NewAccessor(globalConfig.Accessor, globalConfig.Common, ks)
	if nil != err {
		panic(err)
	}
	n.accessor = accessor

	marketCapProvider := marketcap.NewMarketCapProvider(globalConfig.Miner)

	n.registerCrypto(ks)
	n.registerMysql()
	n.registerUserManager()
	n.registerIPFSSubService()
	n.registerMiner(accessor, ks, marketCapProvider)
	n.registerExtractor()
	n.registerAccountManager(accessor)
	n.registerMiner(accessor, ks, marketCapProvider)
	n.registerExtractor()
	n.registerOrderManager()
	n.registerGateway()
	//n.registerTrendManager()
	//n.registerJsonRpcService()
	return n
}

func (n *Node) Start() {
	//n.extractorService.Start()
	n.ipfsSubService.Start()
	//n.miner.Start()
	//gateway.NewJsonrpcService("8080").Start()
	//n.orderManager.Start()
	n.jsonRpcService.Start()
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

func (n *Node) registerCrypto(ks *keystore.KeyStore) {
	c := crypto.NewCrypto(true, ks)
	crypto.Initialize(c)
}

func (n *Node) registerMysql() {
	n.rdsService = dao.NewRdsService(n.globalConfig.Mysql)
	n.rdsService.Prepare()
}

func (n *Node) registerAccessor() {
	accessor, _ := ethaccessor.NewAccessor(n.globalConfig.Accessor, n.globalConfig.Common, nil)
	n.accessor = accessor
}

func (n *Node) registerExtractor() {
	n.extractorService = extractor.NewExtractorService(n.globalConfig.Accessor, n.globalConfig.Common, n.accessor, n.rdsService)
}

func (n *Node) registerIPFSSubService() {
	n.ipfsSubService = gateway.NewIPFSSubService(n.globalConfig.Ipfs)
}

func (n *Node) registerOrderManager() {
	n.orderManager = ordermanager.NewOrderManager(n.globalConfig.OrderManager, &n.globalConfig.Common, n.rdsService, n.userManager, n.accessor)
}

func (n *Node) registerTrendManager() {
	n.trendManager = market.NewTrendManager(n.rdsService)
}

func (n *Node) registerAccountManager(accessor *ethaccessor.EthNodeAccessor) {
	n.accountManager = market.NewAccountManager(accessor)
}

func (n *Node) registerJsonRpcService() {
	ethForwarder := gateway.EthForwarder{Accessor:*n.accessor}
	n.jsonRpcService = *gateway.NewJsonrpcService(strconv.Itoa(n.globalConfig.Jsonrpc.Port), n.trendManager, n.orderManager, n.accountManager, &ethForwarder)
}

func (n *Node) registerMiner(accessor *ethaccessor.EthNodeAccessor, ks *keystore.KeyStore, marketCapProvider *marketcap.MarketCapProvider) {
	submitter := miner.NewSubmitter(n.globalConfig.Miner, ks, accessor, n.rdsService, marketCapProvider)
	evaluator := miner.NewEvaluator(marketCapProvider, n.globalConfig.Miner.RateRatioCVSThreshold, accessor)
	matcher := timing_matcher.NewTimingMatcher(submitter, evaluator, n.orderManager)
	n.miner = miner.NewMiner(submitter, matcher, evaluator, accessor, marketCapProvider)
}

func (n *Node) registerGateway() {
	gateway.Initialize(&n.globalConfig.GatewayFilters, &n.globalConfig.Gateway, &n.globalConfig.Ipfs, n.orderManager)
}

func (n *Node) registerUserManager() {
	n.userManager = usermanager.NewUserManager(n.rdsService)
}
