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

package miner

import (
	"encoding/json"
	"errors"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"sync"
)

//保存ring，并将ring发送到区块链，同样需要分为待完成和已完成
type RingSubmitter struct {
	Chainclient *chainclient.Client
	store       db.Database

	submitedRingsStore db.Database

	unSubmitedRingsStore db.Database

	txToRingHashIndexStore db.Database

	MinerPrivateKey                             []byte        //used to sign the ring
	feeRecepient                                types.Address //used to receive fee
	ifRegistryRingHash, throwIfLrcIsInsuffcient bool

	stopChan chan bool

	mtx *sync.RWMutex
}

type RingSubmitFailed struct {
	RingState *types.RingState
	err       error
}

func NewSubmitter(options config.MinerOptions, commOpts config.CommonOptions, database db.Database, client *chainclient.Client) *RingSubmitter {
	submitter := &RingSubmitter{}
	submitter.Chainclient = client
	submitter.store = database
	submitter.unSubmitedRingsStore = db.NewTable(submitter.store, "unsubmited")
	submitter.submitedRingsStore = db.NewTable(submitter.store, "submited")

	submitter.txToRingHashIndexStore = db.NewTable(submitter.store, "txToRing")
	submitter.mtx = &sync.RWMutex{}

	passphrase := &types.Passphrase{}
	passphrase.SetBytes([]byte(commOpts.Passphrase))
	var err error
	submitter.MinerPrivateKey, err = crypto.AesDecrypted(passphrase.Bytes(), types.FromHex(options.Miner))
	if nil != err {
		panic(err)
	}
	submitter.feeRecepient = types.HexToAddress(options.FeeRecepient)
	submitter.ifRegistryRingHash = options.IfRegistryRingHash
	submitter.throwIfLrcIsInsuffcient = options.ThrowIfLrcIsInsuffcient

	return submitter
}

func (submitter *RingSubmitter) NewRing(ringState *types.RingState) error {
	submitter.mtx.Lock()
	defer submitter.mtx.Unlock()

	if err := submitter.canSubmit(ringState); nil == err {
		if ringBytes, err := json.Marshal(ringState); err == nil {
			submitter.unSubmitedRingsStore.Put(ringState.RawRing.Hash.Bytes(), ringBytes)
			ringData, _ := json.Marshal(ringState)
			log.Debugf("ringState:%s", string(ringData))
			if submitter.ifRegistryRingHash {
				return submitter.sendRinghashRegistry(ringState)
			} else {
				return submitter.submitRing(ringState)
			}
		} else {
			return err
		}
	} else {
		return err
	}
}

func isOrdersRemined(ring *types.RingState) bool {
	//todo:args validator
	return true
}

//todo: 不在submit中的才会提交
func (submitter *RingSubmitter) canSubmit(ringState *types.RingState) error {
	ringData, _ := submitter.unSubmitedRingsStore.Get(ringState.RawRing.Hash.Bytes())
	if nil == ringData || len(ringData) == 0 {
		ringData, _ = submitter.submitedRingsStore.Get(ringState.RawRing.Hash.Bytes())
		if nil == ringData || len(ringData) == 0 {
			return nil
		}
	}
	return errors.New("had been processed")
}

//send Fingerprint to block chain
func (submitter *RingSubmitter) sendRinghashRegistry(ringState *types.RingState) error {
	ring := ringState.RawRing
	contractAddress := ring.Orders[0].OrderState.RawOrder.Protocol
	ringRegistryArgs := ring.GenerateSubmitArgs(submitter.MinerPrivateKey)

	if txHash, err := MinerInstance.Loopring.LoopringImpls[contractAddress].RingHashRegistry.SubmitRinghash.SendTransaction(types.HexToAddress("0x"),
		big.NewInt(int64(len(ring.Orders))),
		ringRegistryArgs.Ringminer,
		ringRegistryArgs.VList,
		ringRegistryArgs.RList,
		ringRegistryArgs.SList,
	); nil != err {
		return err
	} else {
		ringState.RegistryTxHash = types.HexToHash(txHash)
		if ringBytes, err := json.Marshal(ringState); nil != err {
			return err
		} else {
			return submitter.unSubmitedRingsStore.Put(ringState.RawRing.Hash.Bytes(), ringBytes)
		}
	}
}

func (submitter *RingSubmitter) submitRing(ringSate *types.RingState) error {
	ring := ringSate.RawRing
	ring.ThrowIfLrcIsInsuffcient = submitter.throwIfLrcIsInsuffcient
	ring.FeeRecepient = submitter.feeRecepient
	contractAddress := ring.Orders[0].OrderState.RawOrder.Protocol
	log.Debugf("submitRing ringState lrcFee:%s", ringSate.LegalFee)
	ringSubmitArgs := ring.GenerateSubmitArgs(submitter.MinerPrivateKey)
	if txHash, err := MinerInstance.Loopring.LoopringImpls[contractAddress].SubmitRing.SendTransaction(types.HexToAddress("0x"),
		ringSubmitArgs.AddressList,
		ringSubmitArgs.UintArgsList,
		ringSubmitArgs.Uint8ArgsList,
		ringSubmitArgs.BuyNoMoreThanAmountBList,
		ringSubmitArgs.VList,
		ringSubmitArgs.RList,
		ringSubmitArgs.SList,
		ringSubmitArgs.Ringminer,
		ringSubmitArgs.FeeRecepient,
		ringSubmitArgs.ThrowIfLRCIsInsuffcient,
	); nil != err {
		return err
	} else {
		//标记为已删除,迁移到已完成的列表中
		if err := submitter.unSubmitedRingsStore.Delete(ring.Hash.Bytes()); nil != err {
			return err
		}
		ringSate.SubmitTxHash = types.HexToHash(txHash)
		if data, err := json.Marshal(ringSate); nil != err {
			return err
		} else {
			return submitter.submitedRingsStore.Put(ring.Hash.Bytes(), data)
		}
	}
}

//recover after restart
func (submitter *RingSubmitter) recoverRing() {

	//Traversal the uncompelete rings
	iterator := submitter.unSubmitedRingsStore.NewIterator(nil, nil)
	for iterator.Next() {
		dataBytes := iterator.Value()
		ring := &types.RingState{}
		if err := json.Unmarshal(dataBytes, ring); nil != err {
			log.Errorf("error:%s", err.Error())
		} else {
			contractAddress := ring.RawRing.Orders[0].OrderState.RawOrder.Protocol
			var isRinghashRegistered types.Big
			if isOrdersRemined(ring) {
				if err := MinerInstance.Loopring.LoopringImpls[contractAddress].RingHashRegistry.RinghashFound.Call(&isRinghashRegistered, "latest", ring.RawRing.Hash); err != nil {
					log.Errorf("error:%s", err.Error())
				} else {
					if isRinghashRegistered.Int() > 0 {
						var canSubmit types.Big
						if err := MinerInstance.Loopring.LoopringImpls[contractAddress].RingHashRegistry.CanSubmit.Call(&canSubmit, "latest", ring.RawRing.Hash, ring.RawRing.Miner); err != nil {
							log.Errorf("error:%s", err.Error())
						} else {
							if canSubmit.Int() > 0 {
								submitter.submitRing(ring)
							} else {

							}
						}
					} else {
						submitter.NewRing(ring)
					}
				}
			} else {
				failedEvent := &RingSubmitFailed{RingState: ring, err: errors.New("submit ring failed")}
				eventemitter.Emit(eventemitter.RingSubmitFailed, failedEvent)
			}
		}
	}
}

func (submitter *RingSubmitter) handleSubmitRingEvent(e eventemitter.EventData) error {
	if nil != e {
		contractEventData := e.(chainclient.ContractData)
		event := contractEventData.Event
		//excute ring failed
		if nil == event {
			submitter.submitFailed(contractEventData.TxHash)
		}
	}
	return nil
}

func (submitter *RingSubmitter) submitFailed(txHash types.Hash) {
	ringHashBytes, _ := submitter.txToRingHashIndexStore.Get(txHash.Bytes())
	if nil != ringHashBytes && len(ringHashBytes) > 0 {
		if ringData, _ := submitter.unSubmitedRingsStore.Get(ringHashBytes); nil == ringData || len(ringData) == 0 {
			if ringData, _ = submitter.submitedRingsStore.Get(ringHashBytes); nil != ringData || len(ringData) > 0 {
				var ringState *types.RingState
				if err := json.Unmarshal(ringData, ringState); nil == err {
					failedEvent := &RingSubmitFailed{RingState: ringState, err: errors.New("execute ring failed")}
					eventemitter.Emit(eventemitter.RingSubmitFailed, failedEvent)
				}
			}
		}

	}
}

func (submitter *RingSubmitter) handleRegistryEvent(e eventemitter.EventData) error {

	if nil != e {
		contractEventData := e.(chainclient.ContractData)
		//registry failed
		if nil == contractEventData.Event {
			submitter.submitFailed(contractEventData.TxHash)
		} else {
			event := contractEventData.Event.(chainclient.RinghashSubmitted)
			ringHash := types.BytesToHash(event.RingHash)
			println("ringHash.HexringHash.Hex", ringHash.Hex())
			ringData, _ := submitter.unSubmitedRingsStore.Get(ringHash.Bytes())
			if nil != ringData {
				ring := &types.RingState{}
				if err := json.Unmarshal(ringData, ring); nil != err {
					log.Errorf("error:%s", err.Error())
				} else {
					log.Debugf("ringhashRegistry:%s", string(ringData))
					//todo:need pre condition
					if err := submitter.submitRing(ring); nil != err {
						log.Errorf("error:%s", err.Error())
					}
				}
			}
		}
	}

	return nil
}

func (submitter *RingSubmitter) stop() {
	//todp
}

func (submitter *RingSubmitter) start() {
	//ringClient.recoverRing()

	watcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.handleRegistryEvent}
	for _, imp := range MinerInstance.Loopring.LoopringImpls {
		e := imp.RingHashRegistry.RinghashSubmittedEvent
		topic := e.Address().Hex() + e.Id()
		eventemitter.On(topic, watcher)
	}
}
