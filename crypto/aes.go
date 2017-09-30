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
	"crypto/aes"
	"crypto/cipher"
)

func AesEncrypted(key, data []byte) (encrypted []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv := []byte(key)[:aes.BlockSize]
	encrypter := cipher.NewCFBEncrypter(aesBlock, iv)
	encrypted = make([]byte, len(data))
	encrypter.XORKeyStream(encrypted, data)
	return encrypted, err
}

func AesDecrypted(key, encrypted []byte) (decrypted []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	var iv = []byte(key)[:aes.BlockSize]
	decrypted = make([]byte, len(encrypted))
	var aesBlockDecrypter cipher.Block
	aesBlockDecrypter, err = aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}
	aesDecrypter := cipher.NewCFBDecrypter(aesBlockDecrypter, iv)
	aesDecrypter.XORKeyStream(decrypted, encrypted)
	return decrypted, nil
}
