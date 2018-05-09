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
	"errors"
	"github.com/Loopring/relay/log"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

func ValidateSignatureValues(v byte, r, s []byte) bool {
	return crypto.ValidateSignatureValues(v, r, s)
}

func GenerateHash(data ...[]byte) []byte {
	return ethCrypto.Keccak256(data...)
}

func Sign(hash []byte, signer common.Address) ([]byte, error) {
	return crypto.Sign(hash, signer)
}

func SigToAddress(hash, sig []byte) ([]byte, error) {
	return crypto.SigToAddress(hash, sig)
}

func VRSToSig(v byte, r, s []byte) ([]byte, error) {
	return crypto.VRSToSig(v, r, s)
}

func SigToVRS(sig []byte) (v byte, r []byte, s []byte) {
	r = make([]byte, 32)
	s = make([]byte, 32)
	v = byte(uint8(sig[64]) + uint8(27))
	copy(r, sig[0:32])
	copy(s, sig[32:64])
	return v, r, s
}

func UnlockKSAccount(acc accounts.Account, passphrase string) error {
	if c, ok := crypto.(EthKSCrypto); ok {
		return c.UnlockAccount(acc, passphrase)
	} else {
		return errors.New("can't unlock ")
	}
}

func IsKSAccountUnlocked(addr common.Address) bool {
	if c, ok := crypto.(EthKSCrypto); ok {
		return c.IsUnlocked(addr)
	} else {
		log.Errorf("unable to get address :%s lock status", addr.Hex())
		return false
	}
}

func SignTx(a common.Address, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	return crypto.SignTx(a, tx, chainID)
}

func Initialize(c Crypto) {
	crypto = c
}
