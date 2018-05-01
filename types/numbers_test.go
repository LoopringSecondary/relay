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
	"github.com/Loopring/relay/ethaccessor"
	"testing"
)

func TestBig_UnmarshalText(t *testing.T) {
	//n := types.NewBigWithInt(100)
	//bs, _ := n.MarshalText()
	//
	//if err := n.UnmarshalText(bs); err != nil {
	//	t.Fatalf(err.Error())
	//}
	//
	//t.Log(n.BigInt().String())
	var b ethaccessor.TransactionReceipt

	println(b.Status.IsNil())
	//if b.Status {
	//	t.Log("#####")
	//} else if b.BigInt() == nil {
	//	t.Log("!!!!")
	//} else {
	//	t.Log("iiiiiii")
	//}

}
