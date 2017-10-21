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

import (
	"errors"
	"math/big"
)

//todo:test and fix bug (bug exists)
type EnlargedInt struct {
	Value    *big.Int
	Decimals *big.Int
}

func (ei *EnlargedInt) Div(x, y *EnlargedInt) *EnlargedInt {
	if ei.Value == nil {
		ei.Value = big.NewInt(1)
	}
	if ei.Decimals == nil {
		ei.Decimals = big.NewInt(1)
	}
	if x.Decimals.Cmp(big.NewInt(0)) == 0 || y.Value.Cmp(big.NewInt(0)) == 0 {
		panic(errors.New("division by zero"))
	}
	ei.Value.Mul(x.Value, y.Decimals)
	ei.Decimals.Mul(x.Decimals, y.Value)
	return ei
}

func (ei *EnlargedInt) DivBigInt(x *EnlargedInt, y *big.Int) *EnlargedInt {
	if ei.Value == nil {
		ei.Value = big.NewInt(1)
	}
	if ei.Decimals == nil {
		ei.Decimals = big.NewInt(1)
	}
	ei.Value.Div(x.Value, y)
	ei.Decimals = x.Decimals
	return ei
}

func (ei *EnlargedInt) Mul(x, y *EnlargedInt) {
	if ei.Value == nil {
		ei.Value = big.NewInt(1)
	}
	if ei.Decimals == nil {
		ei.Decimals = big.NewInt(1)
	}
	ei.Value.Mul(x.Value, y.Value)
	ei.Decimals.Mul(x.Decimals, y.Decimals)
}

func (ei *EnlargedInt) MulBigInt(x *EnlargedInt, y *big.Int) *EnlargedInt {
	if nil == ei.Value {
		ei.Value = big.NewInt(1)
	}
	ei.Decimals = new(big.Int).Set(x.Decimals)
	ei.Value.Mul(x.Value, y)
	return ei
}

func (ei *EnlargedInt) Sub(x, y *EnlargedInt) *EnlargedInt {

	if ei.Value == nil {
		ei.Value = big.NewInt(0)
	}
	if ei.Decimals == nil {
		ei.Decimals = big.NewInt(1)
	}
	if x.Decimals.Cmp(y.Decimals) == 0 {
		ei.Value.Sub(x.Value, y.Value)
		ei.Decimals = x.Decimals
	} else if x.Decimals.Cmp(y.Decimals) > 0 {
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

func (ei *EnlargedInt) Add(x, y *EnlargedInt) *EnlargedInt {

	if ei.Value == nil {
		ei.Value = big.NewInt(0)
	}
	if ei.Decimals == nil {
		ei.Decimals = big.NewInt(1)
	}
	if x.Decimals.Cmp(y.Decimals) == 0 {
		ei.Value.Add(x.Value, y.Value)
		ei.Decimals = x.Decimals
	} else if x.Decimals.Cmp(y.Decimals) > 0 {
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
	realValue := big.NewInt(1)
	return realValue.Div(ei.Value, ei.Decimals)
}

func (ei *EnlargedInt) Cmp(x *EnlargedInt) int {
	return ei.RealValue().Cmp(x.RealValue())
}

func (ei *EnlargedInt) CmpBigInt(x *big.Int) int {
	return ei.RealValue().Cmp(x)
}

func (ei *EnlargedInt) UnmarshalText(input []byte) error {
	bn := HexToBigint(string(input))
	ei.Value = bn
	ei.Decimals = big.NewInt(1)
	return nil
}

func (ei *EnlargedInt) MarshalText() ([]byte, error) {
	bn := ei.RealValue()
	bytes := []byte(BigintToHex(bn))
	return bytes, nil
}

func NewEnlargedInt(value *big.Int) *EnlargedInt {
	return &EnlargedInt{Value: value, Decimals: big.NewInt(1)}
}

type Big big.Int

func (h *Big) UnmarshalText(input []byte) error {
	//length := len(input)
	//if length >= 2 && input[0] == '"' && input[length-1] == '"' {
	//	input = input[1 : length-1]
	//}

	hn := (*big.Int)(h)
	hn.Set(HexToBigint(string(input)))
	return nil
}

func (h *Big) MarshalText() ([]byte, error) {
	hn := (*big.Int)(h)
	bytes := []byte(BigintToHex(hn))
	return bytes, nil
}

func (h *Big) Int() int {
	hn := (*big.Int)(h)
	return int(hn.Int64())
}

func (h *Big) Int64() int64 {
	hn := (*big.Int)(h)
	return hn.Int64()
}

func (h *Big) Uint() uint {
	hn := (*big.Int)(h)
	return uint(hn.Uint64())
}

func (h *Big) Uint64() uint64 {
	hn := (*big.Int)(h)
	return hn.Uint64()
}

func (h *Big) BigInt() *big.Int {
	return (*big.Int)(h)
}
