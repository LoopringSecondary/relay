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
	"github.com/Loopring/ringminer/crypto"
	"github.com/Loopring/ringminer/types"
)

//代理，控制整个match流程，其中会提供几种实现，如bucket、realtime，etc。

type RingSubmitFailedChan chan *types.RingState

var LoopringInstance *chainclient.Loopring

var MinerPrivateKey []byte     //used to sign the ring
var FeeRecepient types.Address //used to receive fee
var IfRegistryRingHash bool

type Proxy interface {
	Start()
	Stop()
	AddFilter()
}

func Initialize(options config.MinerOptions, client *chainclient.Client) {
	LoopringInstance = &chainclient.Loopring{}
	LoopringInstance.Client = client
	LoopringInstance.Tokens = make(map[types.Address]*chainclient.Erc20Token)

	protocolImps := make(map[types.Address]*chainclient.LoopringProtocolImpl)

	for _, implAddress := range options.LoopringImpAddresses {
		imp := &chainclient.LoopringProtocolImpl{}
		client.NewContract(imp, implAddress, chainclient.CurrentImplAbiStr)
		addr := types.HexToAddress(implAddress)

		var lrcTokenAddressHex string
		imp.LrcTokenAddress.Call(&lrcTokenAddressHex, "pending")
		lrcTokenAddress := types.HexToAddress(lrcTokenAddressHex)
		lrcToken := &chainclient.Erc20Token{}
		client.NewContract(lrcToken, lrcTokenAddress.Hex(), chainclient.Erc20TokenAbiStr)
		LoopringInstance.Tokens[lrcTokenAddress] = lrcToken

		var registryAddressHex string
		imp.RinghashRegistryAddress.coCall(&registryAddressHex, "pending")
		registryAddress := types.HexToAddress(registryAddressHex)
		registry := &chainclient.LoopringRinghashRegistry{}
		client.NewContract(registry, registryAddress.Hex(), chainclient.CurrentRinghashRegistryAbiStr)
		imp.RingHashRegistry = registry
		protocolImps[addr] = imp
	}
	LoopringInstance.LoopringImpls = protocolImps

	passphrase := &types.Passphrase{}
	passphrase.SetBytes([]byte(options.Passphrase))
	var err error
	MinerPrivateKey, err = crypto.AesDecrypted(passphrase.Bytes(), types.FromHex(options.Miner))
	if nil != err {
		panic(err)
	}
	FeeRecepient = types.HexToAddress(options.FeeRecepient)
	IfRegistryRingHash = options.IfRegistryRingHash
}
