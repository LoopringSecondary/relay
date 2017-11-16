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

package types_test

import (
	"github.com/Loopring/relay/types"
	"math/big"
	"testing"
)

func TestOrder_GeneratePrice(t *testing.T) {
	ord := types.Order{}
	ord.AmountB = big.NewInt(100)
	ord.AmountS = big.NewInt(5)
	ord.GeneratePrice()

	t.Log(ord.Price.String())
}
