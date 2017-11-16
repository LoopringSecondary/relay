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
	"fmt"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ipfs/go-ipfs-util"
	"github.com/lydy/go-ethereum/common/bitutil"
	"math/big"
	"testing"
)

func TestStringToAddress(t *testing.T) {
	str := "0xb"
	add := types.HexToAddress(str)
	t.Log(len([]byte(str)))
	t.Log(add.Hex())
	t.Log(len("0x08935625ce172eb3c6561404c09f130268808d08ba59dda70aefa0016619acbc"))
}

type Hash []byte

func (h Hash) Str() string   { return string(h[:]) }
func (h Hash) Bytes() []byte { return h[:] }
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }
func (h Hash) Hex() string   { return hexutil.Encode(h[:]) }

func HexToHash(s string) Hash { return BytesToHash(types.FromHex(s)) }

func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

func (h *Hash) SetBytes(b []byte) {
	//if len(b) > len(*h) {
	//	b = b[len(b)-32:]
	//}

	h1 := Hash(b)
	*h = h1
	//copy((*h)[32-len(b):], b)
	println(len(*h))
}

func TestHash(t *testing.T) {
	//s := "0x093e56de3901764da17fef7e89f016cfdd1a88b98b1f8e3d2ebda4aff2343380"
	//h := types.HexToHash(s)
	//t.Log(h.Hex())
	//println(fmt.Sprintf(`Header(%x)`, h.Bytes()))
}

func TestAddress(t *testing.T) {
	s := "0xc184dd351f215f689f481c329916bb33d8df8ced"
	addr := types.HexToAddress(s)
	//addr := &types.Address{}
	//addr.SetBytes(types.Hex2Bytes(s))
	t.Log(addr.Hex())
	bi := big.NewInt(10)
	println(fmt.Sprintf("%#x", bi.Bits()))
}
