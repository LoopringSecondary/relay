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

import "github.com/ethereum/go-ethereum/common"

func Xor(bytes1, bytes2 []byte) []byte {
	bs1Length := len(bytes1)
	bs2Length := len(bytes2)
	var bytesTmp []byte
	bytesTmp = make([]byte, bs1Length)
	if bs1Length > bs2Length {
		bytes2 = common.LeftPadBytes(bytes2, bs1Length)
	} else if bs1Length < bs2Length {
		bytes1 = common.LeftPadBytes(bytes1, bs2Length)
		bytesTmp = make([]byte, bs2Length)
	}

	for idx, _ := range bytesTmp {
		bytesTmp[idx] = bytes1[idx] ^ bytes2[idx]
	}
	return bytesTmp
}
