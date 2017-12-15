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
	"sync"
)

type TokenBalance struct {
	TokenAddress common.Address
	Balance      *big.Int
	Allowance    *big.Int
	mtx          sync.Mutex
}

func (t *TokenBalance) addAllowance(increment *big.Int) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.Allowance.Add(t.Allowance, increment)
}

func (t *TokenBalance) addBalance(increment *big.Int) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.Balance.Add(t.Balance, increment)
}

func (t *TokenBalance) Available() *big.Int {
	if t.Balance.Cmp(t.Allowance) > 0 {
		return new(big.Int).Set(t.Allowance)
	} else {
		return new(big.Int).Set(t.Balance)
	}
}

type Account struct {
	Address common.Address
	Tokens  map[common.Address]*TokenBalance
}

func NewRing(filledOrders []*types.FilledOrder) *types.Ring {
	ring := &types.Ring{}
	ring.Orders = filledOrders
	ring.Hash = ring.GenerateHash()
	return ring
}

func ConvertOrderStateToFilledOrder(orderState types.OrderState, lrcBalance, tokenSBalance *big.Int) *types.FilledOrder {
	filledOrder := &types.FilledOrder{}
	filledOrder.OrderState = orderState
	filledOrder.AvailableLrcBalance = new(big.Rat).SetInt(lrcBalance)
	filledOrder.AvailableTokenSBalance = new(big.Rat).SetInt(tokenSBalance)
	return filledOrder
}
