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
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"sync"
)

//保存ring，并将ring发送到区块链，同样需要分为待完成和已完成
type RingSubmitClient struct {
	Chainclient *chainclient.Client
	store       db.Database

	submitedRingsStore db.Database

	unSubmitedRingsStore db.Database

	txToRingHashIndexStore db.Database

	//ring 的失败包括：提交失败，ring的合约执行时失败，执行时包括：gas不足，以及其他失败
	//ringSubmitFailedChans []RingSubmitFailedChan

	stopChan chan bool

	mtx *sync.RWMutex
}

type RingSubmitFailed struct {
	RingState *types.RingState
	err	error
}

func NewRingSubmitClient(database db.Database, client *chainclient.Client) *RingSubmitClient {
	submitClient := &RingSubmitClient{}
	submitClient.Chainclient = client
	submitClient.store = database
	submitClient.unSubmitedRingsStore = db.NewTable(submitClient.store, "unsubmited")
	submitClient.submitedRingsStore = db.NewTable(submitClient.store, "submited")

	submitClient.txToRingHashIndexStore = db.NewTable(submitClient.store, "txToRing")
	submitClient.mtx = &sync.RWMutex{}
	//ringClient.ringSubmitFailedChans = make([]RingSubmitFailedChan, 0)
	return submitClient
}

func (submitClient *RingSubmitClient) NewRing(ringState *types.RingState) error {
	submitClient.mtx.Lock()
	defer submitClient.mtx.Unlock()

	if err := submitClient.canSubmit(ringState); nil == err {
		if ringBytes, err := json.Marshal(ringState); err == nil {
			submitClient.unSubmitedRingsStore.Put(ringState.RawRing.Hash.Bytes(), ringBytes)
			ringData, _ := json.Marshal(ringState)
			log.Debugf("ringState:%x", ringData)
			if IfRegistryRingHash {
				return submitClient.sendRinghashRegistry(ringState)
			} else {
				return submitClient.submitRing(ringState)
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
func (submitClient *RingSubmitClient) canSubmit(ringState *types.RingState) error {
	ringData, _ := submitClient.unSubmitedRingsStore.Get(ringState.RawRing.Hash.Bytes())
	if nil == ringData || len(ringData) == 0 {
		ringData, _ = submitClient.submitedRingsStore.Get(ringState.RawRing.Hash.Bytes())
		if nil == ringData || len(ringData) == 0 {
			return nil
		}
	}
	return errors.New("had been processed")
}

//send Fingerprint to block chain
func (submitClient *RingSubmitClient) sendRinghashRegistry(ringState *types.RingState) error {
	ring := ringState.RawRing
	contractAddress := ring.Orders[0].OrderState.RawOrder.Protocol
	ringRegistryArgs := ring.GenerateSubmitArgs(MinerPrivateKey)

	if txHash, err := LoopringInstance.LoopringImpls[contractAddress].RingHashRegistry.SubmitRinghash.SendTransaction(types.HexToAddress("0x"),
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
			return submitClient.unSubmitedRingsStore.Put(ringState.RawRing.Hash.Bytes(), ringBytes)
		}
	}
}

func (ringClient *RingSubmitClient) submitRing(ringSate *types.RingState) error {
	ring := ringSate.RawRing
	ring.ThrowIfLrcIsInsuffcient = ThrowIfLrcIsInsuffcient
	ring.FeeRecepient = FeeRecepient
	contractAddress := ring.Orders[0].OrderState.RawOrder.Protocol
	log.Debugf("submitRing ringState lrcFee:%s", ringSate.LegalFee)
	ringSubmitArgs := ring.GenerateSubmitArgs(MinerPrivateKey)
	if txHash, err := LoopringInstance.LoopringImpls[contractAddress].SubmitRing.SendTransaction(types.HexToAddress("0x"),
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
		if err := ringClient.unSubmitedRingsStore.Delete(ring.Hash.Bytes()); nil != err {
			return err
		}
		ringSate.SubmitTxHash = types.HexToHash(txHash)
		if data, err := json.Marshal(ringSate); nil != err {
			return err
		} else {
			return ringClient.submitedRingsStore.Put(ring.Hash.Bytes(), data)
		}
	}
}

//recover after restart
func (submitClient *RingSubmitClient) recoverRing() {

	//Traversal the uncompelete rings
	iterator := submitClient.unSubmitedRingsStore.NewIterator(nil, nil)
	for iterator.Next() {
		dataBytes := iterator.Value()
		ring := &types.RingState{}
		if err := json.Unmarshal(dataBytes, ring); nil != err {
			log.Errorf("error:%s", err.Error())
		} else {
			contractAddress := ring.RawRing.Orders[0].OrderState.RawOrder.Protocol
			var isRinghashRegistered types.Big
			if isOrdersRemined(ring) {
				if err := LoopringInstance.LoopringImpls[contractAddress].RingHashRegistry.RinghashFound.Call(&isRinghashRegistered, "latest", ring.RawRing.Hash); err != nil {
					log.Errorf("error:%s", err.Error())
				} else {
					if isRinghashRegistered.Int() > 0 {
						var canSubmit types.Big
						if err := LoopringInstance.LoopringImpls[contractAddress].RingHashRegistry.CanSubmit.Call(&canSubmit, "latest", ring.RawRing.Hash, ring.RawRing.Miner); err != nil {
							log.Errorf("error:%s", err.Error())
						} else {
							if canSubmit.Int() > 0 {
								submitClient.submitRing(ring)
							} else {

							}
						}
					} else {
						submitClient.NewRing(ring)
					}
				}
			} else {
				failedEvent := &RingSubmitFailed{RingState:ring, err:errors.New("submit ring failed")}
				eventemitter.Emit(eventemitter.RingSubmitFailed, failedEvent)
			}
		}
	}
}

func (submitClient *RingSubmitClient) handleSubmitRingEvent(e eventemitter.EventData) error {
	if nil != e {
		contractEventData := e.(chainclient.ContractData)
		event := contractEventData.Event
		//excute ring failed
		if nil == event {
			submitClient.submitFailed(contractEventData.TxHash)
		}
	}
	return nil
}

func (submitClient *RingSubmitClient) submitFailed(txHash types.Hash) {
	ringHashBytes, _ := submitClient.txToRingHashIndexStore.Get(txHash.Bytes())
	if nil != ringHashBytes && len(ringHashBytes) > 0 {
		if ringData, _ := submitClient.unSubmitedRingsStore.Get(ringHashBytes); nil == ringData || len(ringData) == 0 {
			if ringData,_ = submitClient.submitedRingsStore.Get(ringHashBytes); nil != ringData || len(ringData) > 0 {
				var ringState *types.RingState
				if err := json.Unmarshal(ringData, ringState); nil == err {
					failedEvent := &RingSubmitFailed{RingState:ringState, err:errors.New("execute ring failed")}
					eventemitter.Emit(eventemitter.RingSubmitFailed, failedEvent)
				}
			}
		}

	}
}

func (submitClient *RingSubmitClient) handleRegistryEvent(e eventemitter.EventData) error {

	if nil != e {
		contractEventData := e.(chainclient.ContractData)
		//registry failed
		if nil == contractEventData.Event {
			submitClient.submitFailed(contractEventData.TxHash)
		} else {
			event := contractEventData.Event.(chainclient.RinghashSubmitted)
			ringHash := types.BytesToHash(event.RingHash)
			println("ringHash.HexringHash.Hex", ringHash.Hex())
			ringData, _ := submitClient.unSubmitedRingsStore.Get(ringHash.Bytes())
			if nil != ringData {
				ring := &types.RingState{}
				if err := json.Unmarshal(ringData, ring); nil != err {
					log.Errorf("error:%s", err.Error())
				} else {
					log.Debugf("ringhashRegistry:%s", string(ringData))
					//todo:need pre condition
					if err := submitClient.submitRing(ring); nil != err {
						log.Errorf("error:%s", err.Error())
					}
				}
			}
		}
	}

	return nil
}

func (submitClient *RingSubmitClient) Start() {
	//ringClient.recoverRing()

	watcher := &eventemitter.Watcher{Concurrent: false, Handle: submitClient.handleRegistryEvent}
	for _, imp := range LoopringInstance.LoopringImpls {
		e := imp.RingHashRegistry.RinghashSubmittedEvent
		topic := e.Address().Hex() + e.Id()
		eventemitter.On(topic, watcher)
	}
}
