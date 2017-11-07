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

package miner

import (
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/config"
)

var MinerInstance *Miner

type Miner struct {
	Loopring              *chainclient.Loopring
	matcher               Matcher
	submitter             *RingSubmitter
	rateRatioCVSThreshold int64
	legalRateProvider     *LegalRateProvider
}

func (minerInstance *Miner) Start() {
	minerInstance.matcher.Start()
	minerInstance.submitter.start()
	minerInstance.legalRateProvider.start()
}

func (minerInstance *Miner) Stop() {
	minerInstance.matcher.Stop()
	minerInstance.submitter.stop()
	minerInstance.legalRateProvider.stop()
}

func NewMiner(options config.MinerOptions, submitter *RingSubmitter, matcher Matcher, loopringInstance *chainclient.Loopring, rateProvider *LegalRateProvider) *Miner {
	rateRatioCVSThreshold := options.RateRatioCVSThreshold
	return &Miner{legalRateProvider: rateProvider,
		submitter:             submitter,
		matcher:               matcher,
		rateRatioCVSThreshold: rateRatioCVSThreshold,
		Loopring:              loopringInstance,
	}
}

func Initialize(minerInstance *Miner) {
	MinerInstance = minerInstance
}
