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

package eth

import (
	"crypto/ecdsa"
	"github.com/Loopring/ringminer/crypto"
	"github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
)

//address -> account
var Accounts map[string]*Account

var passphrase []byte //as aes key

type Account struct {
	PrivKey          *ecdsa.PrivateKey
	PubKey           *ecdsa.PublicKey
	Address          common.Address
	EncryptedPrivKey []byte
}

func (account *Account) Encrypted(passphrase []byte) ([]byte, error) {
	encrypted, err := crypto.AesEncrypted(passphrase, account.PrivKey.D.Bytes())
	if nil != err {
		return nil, err
	}
	account.EncryptedPrivKey = encrypted
	return encrypted, nil
}

func (account *Account) Decrypted(passphrase []byte) ([]byte, error) {
	decrypted, err := crypto.AesDecrypted(account.EncryptedPrivKey, passphrase)
	if nil != err {
		return nil, err
	}
	account.PrivKey, err = ethCrypto.ToECDSA(decrypted)
	account.PubKey = &account.PrivKey.PublicKey
	account.Address = ethCrypto.PubkeyToAddress(*account.PubKey)
	return decrypted, nil
}
