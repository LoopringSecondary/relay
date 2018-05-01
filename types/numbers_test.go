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
	"testing"
	"math/big"
)

func TestBig_UnmarshalText(t *testing.T) {
	n := types.NewBigWithInt(100)
	bs, _ := n.MarshalText()
	a := n.BigInt()
	a.Mul(a, big.NewInt(-1))
	println(a.String())

	if err := n.UnmarshalText(bs); err != nil {
		t.Fatalf(err.Error())
	}
	//type A struct {
	//	Pk        crypto.EthPrivateKeyCrypto
	//}
	//if c,err := crypto.NewPrivateKeyCrypto(true, "0x00ded40c7e1b5111a75c998368abeab124596826f894194cc77ff59e7b504372"); nil != err {
	//	t.Log(err.Error())
	//} else {
	//	t.Log(c.Address().Hex())
	//	data,_ := c.MarshalText()
	//	t.Log(string(data))
	//}
	//
	//a := common.HexToAddress("0x92B80A4d1cBB26028704cB78EFd9fC03eA0eAc60")
	//t.Log(a.Hex())
	//s := []byte("matcher_ringhash_0x33ab36dfe73bb98660f35e2ae52aa826c12da5a225c18aa29c02f01359daa1f3")
	//
	//prefix := []byte("matcher_ringhash_")
	//s1 := s[len(prefix):]
	//
	//ringhash := common.HexToHash(string(s1))
	//t.Log(ringhash.Hex())

	s := "0x2215ba88c1deea20980"
	b := types.HexToBigint(s)
	println(b.String())
}
