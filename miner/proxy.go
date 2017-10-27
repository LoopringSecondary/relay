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

var LoopringInstance *chainclient.Loopring

var MinerPrivateKey []byte     //used to sign the ring
var FeeRecepient types.Address //used to receive fee
var IfRegistryRingHash, ThrowIfLrcIsInsuffcient bool

type Proxy interface {
	Start()
	Stop()
	AddFilter()
}

func Initialize(options config.MinerOptions, commOpts config.CommonOptions, client *chainclient.Client) {
	LoopringInstance = &chainclient.Loopring{}
	LoopringInstance.Client = client
	LoopringInstance.Tokens = make(map[types.Address]*chainclient.Erc20Token)

	protocolImps := make(map[types.Address]*chainclient.LoopringProtocolImpl)

	for _, implAddress := range commOpts.LoopringImpAddresses {
		imp := &chainclient.LoopringProtocolImpl{}
		client.NewContract(imp, implAddress, chainclient.ImplAbiStr)
		addr := types.HexToAddress(implAddress)

		var lrcTokenAddressHex string
		imp.LrcTokenAddress.Call(&lrcTokenAddressHex, "latest")
		lrcTokenAddress := types.HexToAddress(lrcTokenAddressHex)
		lrcToken := &chainclient.Erc20Token{}
		client.NewContract(lrcToken, lrcTokenAddress.Hex(), chainclient.Erc20TokenAbiStr)
		LoopringInstance.Tokens[lrcTokenAddress] = lrcToken

		var registryAddressHex string
		imp.RinghashRegistryAddress.Call(&registryAddressHex, "latest")
		registryAddress := types.HexToAddress(registryAddressHex)
		registry := &chainclient.LoopringRinghashRegistry{}
		client.NewContract(registry, registryAddress.Hex(), chainclient.RinghashRegistryAbiStr)
		imp.RingHashRegistry = registry

		var delegateAddressHex string
		imp.DelegateAddress.Call(&delegateAddressHex, "latest")
		delegateAddress := types.HexToAddress(delegateAddressHex)
		delegate := &chainclient.TransferDelegate{}
		client.NewContract(delegate, delegateAddress.Hex(), chainclient.TransferDelegateAbiStr)
		imp.TokenTransferDelegate = delegate

		protocolImps[addr] = imp
	}
	LoopringInstance.LoopringImpls = protocolImps

	passphrase := &types.Passphrase{}
	passphrase.SetBytes([]byte(commOpts.Passphrase))
	var err error
	MinerPrivateKey, err = crypto.AesDecrypted(passphrase.Bytes(), types.FromHex(options.Miner))
	if nil != err {
		panic(err)
	}
	FeeRecepient = types.HexToAddress(options.FeeRecepient)
	IfRegistryRingHash = options.IfRegistryRingHash

	RateProvider = NewExchangeRateProvider(options)

	RateRatioCVSThreshold = options.RateRatioCVSThreshold

	ThrowIfLrcIsInsuffcient = options.ThrowIfLrcIsInsuffcient
}
