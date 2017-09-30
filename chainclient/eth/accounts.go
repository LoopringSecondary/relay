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
	"github.com/Loopring/ringminer/types"
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

func (account *Account) Encrypted(passphrase *types.Passphrase) ([]byte, error) {
	encrypted, err := crypto.AesEncrypted(passphrase.Bytes(), account.PrivKey.D.Bytes())
	if nil != err {
		return nil, err
	}
	account.EncryptedPrivKey = encrypted
	return encrypted, nil
}

func (account *Account) Decrypted(passphrase *types.Passphrase) ([]byte, error) {
	decrypted, err := crypto.AesDecrypted(passphrase.Bytes(), account.EncryptedPrivKey)
	if nil != err {
		return nil, err
	}
	account.PrivKey, err = ethCrypto.ToECDSA(decrypted)
	account.PubKey = &account.PrivKey.PublicKey
	account.Address = ethCrypto.PubkeyToAddress(*account.PubKey)
	return decrypted, nil
}

//this is different from Client.NewAccount, that is stored in the keystore of chain
func NewAccount(pk string) (*Account, error) {
	var privKey *ecdsa.PrivateKey
	var err error
	if "" == pk {
		privKey, err = ethCrypto.GenerateKey()
	} else {
		privKey, err = ethCrypto.ToECDSA(types.FromHex(pk))
	}
	if nil != err {
		return nil, err
	} else {
		account := &Account{}
		account.PrivKey = privKey
		account.PubKey = &privKey.PublicKey
		account.Address = ethCrypto.PubkeyToAddress(*account.PubKey)
		return account, nil
	}
}
