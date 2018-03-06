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

package crypto

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

var crypto Crypto

type Crypto interface {
	//签名验证
	ValidateSignatureValues(v byte, r, s []byte) bool
	//生成hash
	GenerateHash(data ...[]byte) []byte
	//签名
	Sign(hash []byte, signer common.Address) ([]byte, error)
	//签名恢复到地址
	SigToAddress(hash, sig []byte) ([]byte, error)
	//生成sig
	VRSToSig(v byte, r, s []byte) ([]byte, error)

	SigToVRS(sig []byte) (v byte, r []byte, s []byte)

	SignTx(a common.Address, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
}
