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
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/types"
)

//代理，控制整个match流程，其中会提供几种实现，如bucket、realtime，etc。

type RingSubmitFailedChan chan *types.RingState

var Loopring *chainclient.Loopring

type Proxy interface {
	Start()
	Stop()
	AddFilter()
}

func Initialize(options config.MinerOptions, client *chainclient.Client) {
	Loopring = &chainclient.Loopring{}
	Loopring.Client = client
	Loopring.Tokens = make(map[types.Address]*chainclient.Erc20Token)

	protocolImps := make(map[types.Address]*chainclient.LoopringProtocolImpl)
	fingerprints := make(map[types.Address]*chainclient.LoopringFingerprintRegistry)

	//todo:change it
	for _, impOpts := range options.LoopringImps {
		imp := &chainclient.LoopringProtocolImpl{}
		eth.NewContract(&imp, impOpts.Address, impOpts.Abi)
		addr := &types.Address{}
		addr.SetBytes([]byte(impOpts.Address))
		protocolImps[*addr] = imp
	}
	for _, impOpts := range options.LoopringFingerprints {
		imp := &chainclient.LoopringFingerprintRegistry{}
		eth.NewContract(&imp, impOpts.Address, impOpts.Abi)
		addr := &types.Address{}
		addr.SetBytes([]byte(impOpts.Address))
		fingerprints[*addr] = imp
	}
	Loopring.LoopringFingerprints = fingerprints
	Loopring.LoopringImpls = protocolImps
}
