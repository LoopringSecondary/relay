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

package miner_test

import (
	"github.com/Loopring/relay/chainclient"
	"github.com/Loopring/relay/chainclient/eth"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/miner/timing_matcher"
	"github.com/Loopring/relay/test"
	"testing"
	"time"
)

func TestMatch(t *testing.T) {
	test.LoadConfigAndGenerateTestParams()

	cfg := test.LoadConfig()
	ethClient := eth.NewChainClient(cfg.ChainClient, []byte("sa"))
	submitter := miner.NewSubmitter(cfg.Miner, cfg.Common, ethClient.Client)
	chainclient.LoopringInstance = chainclient.NewLoopringInstance(cfg.Common, ethClient.Client)

	matcher := timing_matcher.NewTimingMatcher()
	marketCapProvider := &market.MarketCapProvider{}

	miner.MinerInstance = miner.NewMinerInstance(cfg.Miner, submitter, matcher, nil, marketCapProvider)
	miner.MinerInstance.Start()

	time.Sleep(1 * time.Minute)
	//matcher.Start()
}
