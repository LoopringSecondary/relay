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
	"github.com/Loopring/ringminer/types"
	"github.com/Loopring/ringminer/crypto"
)

//代理，控制整个match流程，其中会提供几种实现，如bucket、realtime，etc。

type RingSubmitFailedChan chan *types.RingState

var Loopring *chainclient.Loopring

var Miner []byte  //used to sign the ring
var FeeRecepient types.Address //used to receive fee

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
		client.NewContract(&imp, impOpts.Address, impOpts.Abi)
		addr := &types.Address{}
		addr.SetBytes([]byte(impOpts.Address))
		protocolImps[*addr] = imp
	}
	for _, impOpts := range options.LoopringFingerprints {
		imp := &chainclient.LoopringFingerprintRegistry{}
		client.NewContract(&imp, impOpts.Address, impOpts.Abi)
		addr := &types.Address{}
		addr.SetBytes([]byte(impOpts.Address))
		fingerprints[*addr] = imp
	}
	Loopring.LoopringFingerprints = fingerprints
	Loopring.LoopringImpls = protocolImps

	passphrase := &types.Passphrase{}
	passphrase.SetBytes([]byte(options.Passphrase))
	var err error
	Miner,err = crypto.AesDecrypted(passphrase.Bytes(), types.FromHex(options.Miner))
	if nil != err {
		panic(err)
	}
	FeeRecepient = types.HexToAddress(options.FeeRecepient)
}
