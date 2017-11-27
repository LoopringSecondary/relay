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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/miner/timing_matcher"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

func loadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/relay.toml"
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}
func TestMatch(t *testing.T) {

	cfg := loadConfig()
	accessor, _ := ethaccessor.NewAccessor(cfg.Accessor, cfg.Common, nil)
	submitter := miner.NewSubmitter(cfg.Miner, nil, accessor)

	evaluator := &miner.Evaluator{}

	matcher := timing_matcher.NewTimingMatcher(submitter, evaluator)
	marketCapProvider := &market.MarketCapProvider{}

	m := miner.NewMiner(submitter, matcher, evaluator, accessor, marketCapProvider)
	m.Start()
	time.Sleep(1 * time.Minute)

	//matcher.Start()
}
