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

package timing_matcher

import (
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type minedRing struct {
	ringHash    common.Hash
	orderHashes []common.Hash
}

type RoundState struct {
	round          *big.Int
	ringHash       common.Hash
	tokenS         common.Address
	matchedAmountS *big.Rat
	matchedAmountB *big.Rat
}

type OrderMatchState struct {
	orderState types.OrderState
	rounds     map[common.Hash]*RoundState
}

type CandidateRing struct {
	filledOrders map[common.Hash]*big.Rat
	received     *big.Rat
	cost         *big.Rat
}

type CandidateRingList []CandidateRing

func (ringList CandidateRingList) Len() int {
	return len(ringList)
}
func (ringList CandidateRingList) Swap(i, j int) {
	ringList[i], ringList[j] = ringList[j], ringList[i]
}
func (ringList CandidateRingList) Less(i, j int) bool {
	return ringList[i].received.Cmp(ringList[j].received) > 0
}
