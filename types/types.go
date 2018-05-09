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
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

const (
	Bytes32Length = 32
)

type Bytes32 [Bytes32Length]byte

func BitToBytes32(b *big.Int) Bytes32 { return BytesToBytes32(b.Bytes()[:]) }
func HexToBytes32(s string) Bytes32   { return BytesToBytes32(common.FromHex(s)) }

//MarshalJson
func (a *Bytes32) MarshalText() ([]byte, error) {
	return []byte(a.Hex()), nil
}

func (a *Bytes32) UnmarshalText(input []byte) error {
	a.SetBytes(HexToBytes32(string(input)).Bytes())
	return nil
}

func (s Bytes32) Str() string       { return string(s[:]) }
func (s Bytes32) Bytes() []byte     { return s[:] }
func (s Bytes32) Bytes32() [32]byte { return s }
func (s Bytes32) Big() *big.Int     { return new(big.Int).SetBytes(s[:]) }
func (s Bytes32) Hex() string       { return common.ToHex(s[:]) }

func BytesToBytes32(b []byte) Bytes32 {
	var s Bytes32
	s.SetBytes(b)
	return s
}

func (s *Bytes32) SetBytes(b []byte) {
	if len(b) > len(s) {
		b = b[len(b)-Bytes32Length:]
	}
	copy(s[Bytes32Length-len(b):], b)
}

func IsZeroHash(hash common.Hash) bool {
	return hash == common.HexToHash("0x")
}

func IsZeroAddress(addr common.Address) bool {
	return addr == common.HexToAddress("0x")
}

func BigintToHex(b *big.Int) string {
	if nil == b {
		b = big.NewInt(0)
	}
	return fmt.Sprintf("%#x", b)
}

func HexToBigint(h string) *big.Int {
	return new(big.Int).SetBytes(common.FromHex(h))
}

type CheckNull interface {
	IsNull() bool
}

var (
	NilHash    = common.HexToHash("0x")
	NilAddress = common.HexToAddress("0x")
)
