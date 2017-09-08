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
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	HashLength    = 66  // 以太坊中为
	AddressLength = 42  // 以太坊中为20 20*2 + "0x"
	SignLength    = 32
)

// TODO(fukun): 后续可能要整理下，有些东西没有用到

// Hash represents the 32 byte Keccak256 hash of arbitrary data.
type Hash [HashLength]byte

// Address represents the 20 byte address of an Ethereum account.
type Address [AddressLength]byte

// Sign represents the 32 byte of an ECDSA r/s
type Sign [SignLength]byte

// Get the string representation of the underlying hash
func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return hexutil.Encode(h[:]) }

// Get the string representation of the underlying address
func (a Address) Str() string   { return string(a[:]) }
func (a Address) Bytes() []byte { return a[:] }
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }
func (a Address) Hex() string   { return hexutil.Encode(a[:]) }

// Get the string representation of the underlying sign
func (s Sign) Str() string   { return string(s[:]) }
func (s Sign) Bytes() []byte { return s[:] }
func (s Sign) Big() *big.Int { return new(big.Int).SetBytes(s[:]) }
func (s Sign) Hex() string   { return hexutil.Encode(s[:]) }

func StringToHash(s string) Hash { return BytesToHash([]byte(s)) }
func BigToHash(b *big.Int) Hash  { return BytesToHash(b.Bytes()) }
func HexToHash(s string) Hash    { return BytesToHash(FromHex(s)) }

func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }
func HexToAddress(s string) Address    { return BytesToAddress(FromHex(s)) }

func StringToSign(s string) Sign { return BytesToSign([]byte(s)) }
func BitToSign(b *big.Int) Sign  { return BytesToSign(b.Bytes()) }
func HexToSign(s string) Sign    { return BytesToSign(FromHex(s)) }

func IntToBig(i int64) *big.Int { return big.NewInt(i) }

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

func BytesToSign(b []byte) Sign {
	var s Sign
	s.SetBytes(b)
	return s
}

// Sets the hash to the value of b. If b is larger than len(h) it will panic
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}

// Sets the address to the value of b. If b is larger than len(h) it will panic
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

// Sets the sign to the value of b. If b is larger than len(h) it will panic
func (s *Sign) SetBytes(b []byte) {
	if len(b) > len(s) {
		b = b[len(b)-SignLength:]
	}
	copy(s[SignLength-len(b):], b)
}