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
	"errors"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"reflect"
	"time"
)

var rpcClient *rpc.Client

var EthClient *chainclient.Client

func newRpcMethod(name string) func(result interface{}, args ...interface{}) error {
	return func(result interface{}, args ...interface{}) error {
		return rpcClient.Call(result, name, args...)
	}
}

type CallArg struct {
	From     string
	To       string
	Gas      types.Big
	GasPrice types.Big
	Value    types.Big
	Data     string
}

func NewClient() *chainclient.Client {
	client := &chainclient.Client{}

	//set rpcmethod
	applyMethod(client)

	//Subscribe
	client.Subscribe = subscribe

	//SignAndSendTransaction
	client.SignAndSendTransaction = signAndSendTransaction
	return client
}

func signAndSendTransaction(result interface{}, args ...interface{}) error {
	from := args[0].(string)
	transaction := args[1].(*ethTypes.Transaction)
	if account, ok := Accounts[from]; !ok {
		return errors.New("there isn't a private key for this address:" + from)
	} else {
		signer := &ethTypes.HomesteadSigner{}

		signature, err := crypto.Sign(signer.Hash(transaction).Bytes(), account.PrivKey)

		log.Debugf("hash:%s, sig:%s", signer.Hash(transaction).Hex(), common.Bytes2Hex(signature))
		if nil != err {
			return err
		}
		if transaction, err = transaction.WithSignature(signer, signature); nil != err {
			return err
		} else {
			if txData, err := rlp.EncodeToBytes(transaction); nil != err {
				return err
			} else {
				err = EthClient.SendRawTransaction(result, common.ToHex(txData))
				return err
			}
		}
	}
}

func doSubscribe(chanVal reflect.Value, filterId string) {
	for {
		select {
		case <-time.Tick(1000000000):
			v := chanVal.Type().Elem()
			result := reflect.New(v)
			if err := EthClient.GetFilterChanges(result.Interface(), filterId); nil != err {
				log.Errorf("error:%s", err.Error())
				break
			} else {
				chanVal.Send(result.Elem())
			}
		}
	}
}

func subscribe(result interface{}, args ...interface{}) error {
	//the first arg must be filterId
	filterId := args[0].(string)
	//todo:should check result is a chan
	chanVal := reflect.ValueOf(result).Elem()
	go doSubscribe(chanVal, filterId)
	return nil
}

func applyMethod(client *chainclient.Client) error {
	//todo:is it should be in config ?
	methodNameMap := map[string]string{
		"clientVersion":                       "web3_clientVersion",
		"sha3":                                "web3_sha3",
		"version":                             "net_version",
		"peerCount":                           "net_peerCount",
		"listening":                           "net_listening",
		"protocolVersion":                     "eth_protocolVersion",
		"syncing":                             "eth_syncing",
		"coinbase":                            "eth_coinbase",
		"mining":                              "eth_mining",
		"hashrate":                            "eth_hashrate",
		"gasPrice":                            "eth_gasPrice",
		"accounts":                            "eth_accounts",
		"blockNumber":                         "eth_blockNumber",
		"getBalance":                          "eth_getBalance",
		"getStorageAt":                        "eth_getStorageAt",
		"getTransactionCount":                 "eth_getTransactionCount",
		"getBlockTransactionCountByHash":      "eth_getBlockTransactionCountByHash",
		"getBlockTransactionCountByNumber":    "eth_getBlockTransactionCountByNumber",
		"getUncleCountByBlockHash":            "eth_getUncleCountByBlockHash",
		"getUncleCountByBlockNumber":          "eth_getUncleCountByBlockNumber",
		"getCode":                             "eth_getCode",
		"sign":                                "eth_sign",
		"sendTransaction":                     "eth_sendTransaction",
		"sendRawTransaction":                  "eth_sendRawTransaction",
		"call":                                "eth_call",
		"estimateGas":                         "eth_estimateGas",
		"getBlockByHash":                      "eth_getBlockByHash",
		"getBlockByNumber":                    "eth_getBlockByNumber",
		"getTransactionByHash":                "eth_getTransactionByHash",
		"getTransactionByBlockHashAndIndex":   "eth_getTransactionByBlockHashAndIndex",
		"getTransactionByBlockNumberAndIndex": "eth_getTransactionByBlockNumberAndIndex",
		"getTransactionReceipt":               "eth_getTransactionReceipt",
		"getUncleByBlockHashAndIndex":         "eth_getUncleByBlockHashAndIndex",
		"getUncleByBlockNumberAndIndex":       "eth_getUncleByBlockNumberAndIndex",
		"getCompilers":                        "eth_getCompilers",
		"compileLLL":                          "eth_compileLLL",
		"compileSolidity":                     "eth_compileSolidity",
		"compileSerpent":                      "eth_compileSerpent",
		"newFilter":                           "eth_newFilter",
		"newBlockFilter":                      "eth_newBlockFilter",
		"newPendingTransactionFilter":         "eth_newPendingTransactionFilter",
		"uninstallFilter":                     "eth_uninstallFilter",
		"getFilterChanges":                    "eth_getFilterChanges",
		"getFilterLogs":                       "eth_getFilterLogs",
		"getLogs":                             "eth_getLogs",
		"getWork":                             "eth_getWork",
		"submitWork":                          "eth_submitWork",
		"submitHashrate":                      "eth_submitHashrate",

		"unlockAccount": "personal_unlockAccount",
		"newAccount":    "personal_newAccount",
	}
	v := reflect.ValueOf(client).Elem()
	for i := 0; i < v.NumField(); i++ {
		fieldV := v.Field(i)
		if methodName, ok := methodNameMap[v.Type().Field(i).Tag.Get("methodName")]; ok && methodName != "" {
			fieldV.Set(reflect.ValueOf(newRpcMethod(methodName)))
		}
	}
	return nil
}

func Initialize(clientConfig config.ChainClientOptions) {
	client, _ := rpc.Dial(clientConfig.RawUrl)
	rpcClient = client
	EthClient = NewClient()

	Accounts = make(map[string]*Account)
	passphrase := &types.Passphrase{}
	passphrase.SetBytes([]byte(clientConfig.Passphrase))

	for addr, p := range clientConfig.Eth.PrivateKeys {
		account := &Account{EncryptedPrivKey: types.FromHex(p)}
		if _, err := account.Decrypted(passphrase); nil != err {
			log.Errorf("err:%s", err.Error())
			panic(err)
		}
		if account.Address.Hex() != addr {
			log.Errorf("address:%s and privkey:%s not match", addr, p)
			panic("address and privkey not match")
		}
		Accounts[addr] = account
	}
}
