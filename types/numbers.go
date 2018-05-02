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
	"math/big"
)

type Big big.Int

func NewBigPtr(v *big.Int) *Big {
	n := new(Big)
	n.SetInt(v)
	return n
}

func NewBigWithInt(v int) *Big {
	n := new(Big)
	n.SetInt(big.NewInt(int64(v)))
	return n
}

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

func (h *Big) SetInt(v *big.Int) Big {
	(*big.Int)(h).Set(v)
	return *h
}

type Rat big.Rat

func (r *Rat) UnmarshalText(input []byte) error {
	rn := (*big.Rat)(r)
	rn.SetString(string(input))
	return nil
}

func (r *Rat) MarshalText() ([]byte, error) {
	rn := (*big.Rat)(r)
	bytes := []byte(rn.RatString())
	return bytes, nil
}

func (r *Rat) BigRat() *big.Rat {
	return (*big.Rat)(r)
}
func NewBigRat(v *big.Rat) *Rat {
	r := new(Rat)
	(*big.Rat)(r).Set(v)
	return r
}

var MaxUint256 = maxUint256()

func maxUint256() *big.Int {
	maxBytes := [32]uint8{}
	for idx, _ := range maxBytes {
		maxBytes[idx] = uint8(255)
	}
	maxBig := new(big.Int).SetBytes(maxBytes[:])
	return maxBig
}
