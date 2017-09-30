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
	"github.com/Loopring/ringminer/log"
	"math/big"
)

//this should be diff value for diff chain
const (
	HashLength    = 32 //todo：this is eth value
	AddressLength = 20
	SignLength    = 32
)

type Sign [SignLength]byte

func StringToSign(s string) Sign { return BytesToSign([]byte(s)) }
func BitToSign(b *big.Int) Sign  { return BytesToSign(b.Bytes()) }
func HexToSign(s string) Sign    { return BytesToSign(FromHex(s)) }

//MarshalJson
func (a *Sign) MarshalText() ([]byte, error) {
	return []byte(a.Hex()), nil
}

func (a *Sign) UnmarshalText(input []byte) error {
	a.SetBytes(HexToSign(string(input)).Bytes())
	return nil
}

func (s Sign) Str() string   { return string(s[:]) }
func (s Sign) Bytes() []byte { return s[:] }
func (s Sign) Big() *big.Int { return new(big.Int).SetBytes(s[:]) }
func (s Sign) Hex() string   { return ToHex(s[:]) }

func BytesToSign(b []byte) Sign {
	var s Sign
	s.SetBytes(b)
	return s
}

func (s *Sign) SetBytes(b []byte) {
	if len(b) > len(s) {
		b = b[len(b)-SignLength:]
	}
	copy(s[SignLength-len(b):], b)
}

type Hash [HashLength]byte

func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return ToHex(h[:]) }

//MarshalJson
func (a *Hash) MarshalText() ([]byte, error) {
	return []byte(a.Hex()), nil
}

func (a *Hash) UnmarshalText(input []byte) error {
	a.SetBytes(HexToHash(string(input)).Bytes())
	return nil
}

func StringToHash(s string) Hash { return BytesToHash([]byte(s)) }
func BigToHash(b *big.Int) Hash  { return BytesToHash(b.Bytes()) }
func HexToHash(s string) Hash    { return BytesToHash(FromHex(s)) }

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}
	copy(h[HashLength-len(b):], b)
}

type Address [AddressLength]byte

func (a Address) Str() string   { return string(a[:]) }
func (a Address) Bytes() []byte { return a[:] }
func (a Address) Big() *big.Int { return new(big.Int).SetBytes(a[:]) }
func (a Address) Hex() string   { return ToHex(a[:]) }

func (a *Address) MarshalText() ([]byte, error) {
	return []byte(a.Hex()), nil
}

func (a *Address) UnmarshalText(input []byte) error {
	a.SetBytes(HexToAddress(string(input)).Bytes())
	return nil
}

func StringToAddress(s string) Address { return BytesToAddress([]byte(s)) }
func BigToAddress(b *big.Int) Address  { return BytesToAddress(b.Bytes()) }
func HexToAddress(s string) Address    { return BytesToAddress(FromHex(s)) }

func BytesToAddress(b []byte) Address {
	var a Address
	a.SetBytes(b)
	return a
}

// Sets the address to the value of b. If b is larger than len(h) it will panic
func (a *Address) SetBytes(b []byte) {
	if len(b) > len(a) {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
}

type Passphrase [32]byte

func (p *Passphrase) SetBytes(b []byte) {
	if len(b) > 32 {
		log.Info("the passphrase will only use 32 bytes ")
	}
	copy(p[32-len(b):], b)
}

func (p *Passphrase) Bytes() []byte {
	return p[:]
}

func BigintToHex(b *big.Int) string {
	if nil == b {
		b = big.NewInt(0)
	}
	return ToHex(b.Bytes())
}

func HexToBigint(h string) *big.Int {
	return new(big.Int).SetBytes(FromHex(h))
}
