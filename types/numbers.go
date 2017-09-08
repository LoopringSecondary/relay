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

package types

import "math/big"

//todo:test and fix bug (bug exists)
type EnlargedInt struct {
	Value *big.Int
	Decimals *big.Int
}

func (ei *EnlargedInt) Div(x, y *EnlargedInt) *EnlargedInt {
	if (ei.Value == nil) {
		ei.Value = big.NewInt(0)
	}
	if (ei.Decimals == nil) {
		ei.Decimals = big.NewInt(0)
	}
	ei.Value.Div(x.Value, y.Value)
	ei.Decimals.Div(x.Decimals, y.Decimals)
	return ei
}

func (ei *EnlargedInt) DivBigInt(x *EnlargedInt, y *big.Int) *EnlargedInt {
	if (ei.Value == nil) {
		ei.Value = big.NewInt(1)
	}
	if (ei.Decimals == nil) {
		ei.Decimals = big.NewInt(1)
	}
	ei.Value.Div(x.Value, y)
	ei.Decimals = ei.Decimals.Mul(ei.Decimals, x.Decimals)
	return ei
}

func (ei *EnlargedInt) Mul(x, y *EnlargedInt) {
	if (ei.Value == nil) {
		ei.Value = big.NewInt(1)
	}
	if (ei.Decimals == nil) {
		ei.Decimals = big.NewInt(1)
	}
	ei.Value.Mul(x.Value, y.Value)
	ei.Decimals.Mul(x.Decimals, y.Decimals)
}

func (ei *EnlargedInt) MulBigInt(x *EnlargedInt, y *big.Int) *EnlargedInt {
	if (ei.Value == nil) {
		ei.Value = big.NewInt(1)
	}
	if (ei.Decimals == nil) {
		ei.Decimals = big.NewInt(1)
	}
	ei.Value.Mul(x.Value, y)
	ei.Decimals = ei.Decimals.Mul(ei.Decimals, x.Decimals)
	return ei
}

func (ei *EnlargedInt) Sub(x,y *EnlargedInt) *EnlargedInt {

	if (ei.Value == nil) {
		ei.Value = big.NewInt(0)
	}
	if (ei.Decimals == nil) {
		ei.Decimals = big.NewInt(1)
	}
	if (x.Decimals.Cmp(y.Decimals) == 0) {
		ei.Value.Sub(x.Value,  y.Value)
		ei.Decimals = x.Decimals
	} else if (x.Decimals.Cmp(y.Decimals) > 0) {
		decimals := big.NewInt(1)
		decimals.Div(x.Decimals, y.Decimals)
		value := big.NewInt(1)
		value.Mul(y.Value, decimals)

		ei.Value.Sub(x.Value, value)
		ei.Decimals = x.Decimals
	} else {
		decimals := big.NewInt(1)
		decimals.Div(y.Decimals, x.Decimals)
		value := big.NewInt(1)
		value.Mul(x.Value, decimals)

		ei.Value.Sub(value, y.Value)
		ei.Decimals = y.Decimals
	}

	return ei
}
func (ei *EnlargedInt) Add(x,y *EnlargedInt) *EnlargedInt {

	if (ei.Value == nil) {
		ei.Value = big.NewInt(0)
	}
	if (ei.Decimals == nil) {
		ei.Decimals = big.NewInt(1)
	}
	if (x.Decimals.Cmp(y.Decimals) == 0) {
		ei.Value.Add(x.Value,  y.Value)
		ei.Decimals = x.Decimals
	} else if (x.Decimals.Cmp(y.Decimals) > 0) {
		decimals := big.NewInt(1)
		decimals.Div(x.Decimals, y.Decimals)
		value := big.NewInt(1)
		value.Mul(y.Value, decimals)

		ei.Value.Add(x.Value, value)
		ei.Decimals = x.Decimals
	} else {
		decimals := big.NewInt(1)
		decimals.Div(y.Decimals, x.Decimals)
		value := big.NewInt(1)
		value.Mul(x.Value, decimals)

		ei.Value.Add(value, y.Value)
		ei.Decimals = y.Decimals
	}

	return ei
}

func (ei *EnlargedInt) RealValue() *big.Int {
	realValue := big.NewInt(0)
	return realValue.Div(ei.Value, ei.Decimals)
}

func (ei *EnlargedInt) Cmp(x *EnlargedInt) int {
	return ei.RealValue().Cmp(x.RealValue())
}

func (ei *EnlargedInt) CmpBigInt(x *big.Int) int {
	return ei.RealValue().Cmp(x)
}

func NewEnlargedInt(value *big.Int) *EnlargedInt {
	return &EnlargedInt{Value:value, Decimals:big.NewInt(1)}
}