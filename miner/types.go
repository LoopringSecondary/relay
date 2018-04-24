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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type NormalSenderAddress struct {
	Address         common.Address
	GasPriceLimit   *big.Int
	MaxPendingTtl   int
	MaxPendingCount int64

	Nonce *big.Int
}

type SplitMinerAddress struct {
	Address    common.Address
	FeePercent float64
	StartFee   float64

	Nonce *big.Int
}

func NewRing(filledOrders []*types.FilledOrder) *types.Ring {
	ring := &types.Ring{}
	ring.Orders = filledOrders
	return ring
}
