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
	"fmt"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"reflect"
	"time"
)

type EthClient struct {
	*chainclient.Client
	runtimedb *db.Database
	signer    *ethTypes.HomesteadSigner
	senders   map[types.Address]*Account
	rpcClient *rpc.Client
}

func (ethClient *EthClient) newRpcMethod(name string) func(result interface{}, args ...interface{}) error {
	return func(result interface{}, args ...interface{}) error {
		return ethClient.rpcClient.Call(result, name, args...)
	}
}

type CallArg struct {
	From     types.Address `json:"from"`
	To       types.Address `json:"to"`
	Gas      types.Big     `json:"gas"`
	GasPrice types.Big     `json:"gasPrice"`
	Value    types.Big     `json:"value"`
	Data     string        `json:"data"`
	Nonce    types.Big     `json:"nonce"`
}

func NewChainClient(clientConfig config.ChainClientOptions, passphraseStr string) *EthClient {
	ethClient := &EthClient{}
	var err error
	ethClient.rpcClient, err = rpc.Dial(clientConfig.RawUrl)
	if nil != err {
		panic(err)
	}

	ethClient.Client = &chainclient.Client{}

	ethClient.applyMethod()
	ethClient.Subscribe = ethClient.subscribe
	ethClient.SignAndSendTransaction = ethClient.signAndSendTransaction
	ethClient.NewContract = ethClient.newContract
	ethClient.StartForkDetect = ethClient.startForkDetect
	ethClient.BlockIterator = ethClient.blockIterator

	ethClient.signer = &ethTypes.HomesteadSigner{}

	passphrase := &types.Passphrase{}
	passphrase.SetBytes([]byte(passphraseStr))
	if accounts, err := DecryptAccounts(passphrase, clientConfig.Senders); nil != err {
		panic(err)
	} else {
		ethClient.senders = accounts
	}

	return ethClient
}

func (ethClient *EthClient) signAndSendTransaction(result interface{}, from types.Address, tx interface{}) error {
	transaction := tx.(*ethTypes.Transaction)
	if account, ok := ethClient.senders[from]; !ok {
		return errors.New("there isn't a private key for this address:" + from.Hex())
	} else {
		signer := &ethTypes.HomesteadSigner{}

		signature, err := crypto.Sign(signer.Hash(transaction).Bytes(), account.PrivKey)

		log.Debugf("hash:%s, sig:%s, value%s, gas:%s, gasPrice:%s", signer.Hash(transaction).Hex(), common.ToHex(signature), transaction.Value().String(), transaction.Gas().String(), transaction.GasPrice().String())
		if nil != err {
			return err
		}
		if transaction, err = transaction.WithSignature(signer, signature); nil != err {
			return err
		} else {
			if txData, err := rlp.EncodeToBytes(transaction); nil != err {
				return err
			} else {
				err = ethClient.SendRawTransaction(result, common.ToHex(txData))
				return err
			}
		}
	}
}

func (ethClient *EthClient) doSubscribe(chanVal reflect.Value, filterId string) {
	for {
		select {
		case <-time.Tick(1000000000):
			v := chanVal.Type().Elem()
			result := reflect.New(v)
			if err := ethClient.GetFilterChanges(result.Interface(), filterId); nil != err {
				log.Errorf("error:%s", err.Error())
				break
			} else {
				chanVal.Send(result.Elem())
			}
		}
	}
}

func (ethClient *EthClient) subscribe(result interface{}, filterId string) error {
	//todo:should check result is a chan
	chanVal := reflect.ValueOf(result).Elem()
	go ethClient.doSubscribe(chanVal, filterId)
	return nil
}

func (ethClient *EthClient) applyMethod() error {
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
	v := reflect.ValueOf(ethClient.Client).Elem()
	for i := 0; i < v.NumField(); i++ {
		fieldV := v.Field(i)
		if methodName, ok := methodNameMap[v.Type().Field(i).Tag.Get("methodName")]; ok && methodName != "" {
			fieldV.Set(reflect.ValueOf(ethClient.newRpcMethod(methodName)))
		}
	}
	return nil
}

type BlockIterator struct {
	startNumber   *big.Int
	endNumber     *big.Int
	currentNumber *big.Int
	ethClient     *EthClient
}

func (iterator *BlockIterator) Next() (interface{}, error) {
	block := &BlockWithTxObject{}
	if nil != iterator.endNumber && iterator.endNumber.Cmp(big.NewInt(0)) > 0 && iterator.endNumber.Cmp(iterator.currentNumber) < 0 {
		return nil, errors.New("finished")
	}
	if err := iterator.ethClient.GetBlockByNumber(&block, fmt.Sprintf("%#x", iterator.currentNumber), true); nil != err {
		return nil, err
	} else {
		if nil == block {
		hasNext:
			for {
				select {
				// todo(fk):modify this duration
				case <-time.After(time.Duration(5 * time.Second)):
					if err1 := iterator.ethClient.GetBlockByNumber(&block, fmt.Sprintf("%#x", iterator.currentNumber), true); nil == err1 && nil != block {
						break hasNext
					}
				}
			}
		}
		iterator.currentNumber.Add(iterator.currentNumber, big.NewInt(1))
		return *block, nil
	}
}

func (iterator *BlockIterator) Prev() (interface{}, error) {
	block := &Block{}
	if nil != iterator.startNumber && iterator.startNumber.Cmp(big.NewInt(0)) > 0 && iterator.startNumber.Cmp(iterator.currentNumber) > 0 {
		return nil, errors.New("finished")
	}
	prevNumber := new(big.Int).Sub(iterator.currentNumber, big.NewInt(1))
	if err := iterator.ethClient.GetBlockByNumber(&block, fmt.Sprintf("%#x", prevNumber), false); nil != err {
		return nil, err
	} else {
		if nil == block {
			return nil, errors.New("there isn't a block with number:" + prevNumber.String())
		}
		iterator.currentNumber.Sub(iterator.currentNumber, big.NewInt(1))
		return *block, nil
	}
}

func (ethClient *EthClient) blockIterator(startNumber, endNumber *big.Int) chainclient.BlockIterator {
	iterator := &BlockIterator{
		startNumber:   new(big.Int).Set(startNumber),
		endNumber:     endNumber,
		currentNumber: new(big.Int).Set(startNumber),
		ethClient:     ethClient,
	}
	return iterator
}
