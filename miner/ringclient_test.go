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
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
	"testing"
)

func TestRingClient(t *testing.T) {

	ring := &types.RingState{}
	hash := &types.Hash{}
	hash.SetBytes([]byte("testtesthash"))
	ring.Hash = *hash
	ring.FeeMode = 1
	loopring := &chainclient.Loopring{}
	loopring.LoopringImpls = make(map[types.Address]*chainclient.LoopringProtocolImpl)
	loopring.LoopringFingerprints = make(map[types.Address]*chainclient.LoopringFingerprintRegistry)
	loopring.Tokens = make(map[types.Address]*chainclient.Erc20Token)
	loopring.Client = eth.EthClientInstance

	ringClient := miner.NewRingClient(loopring)
	ringClient.Start()

	ringClient.NewRing(ring)
}
