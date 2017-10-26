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
type RingClient struct {
	Chainclient *chainclient.Client
	store       db.Database

	submitedRingsStore db.Database

	unSubmitedRingsStore db.Database

	ringhashRegistryChan chan *chainclient.RinghashSubmitted

	//ring 的失败包括：提交失败，ring的合约执行时失败，执行时包括：gas不足，以及其他失败
	ringSubmitFailedChans []RingSubmitFailedChan

	stopChan chan bool

	mtx *sync.RWMutex
}

func NewRingClient(database db.Database, client *chainclient.Client) *RingClient {
	ringClient := &RingClient{}
	ringClient.Chainclient = client
	ringClient.store = database
	ringClient.unSubmitedRingsStore = db.NewTable(ringClient.store, "unsubmited")
	ringClient.submitedRingsStore = db.NewTable(ringClient.store, "submited")
	ringClient.mtx = &sync.RWMutex{}
	ringClient.ringSubmitFailedChans = make([]RingSubmitFailedChan, 0)
	return ringClient
}

func (ringClient *RingClient) AddRingSubmitFailedChan(c RingSubmitFailedChan) {
	ringClient.mtx.Lock()
	defer ringClient.mtx.Unlock()
	ringClient.ringSubmitFailedChans = append(ringClient.ringSubmitFailedChans, c)
}

func (ringClient *RingClient) DeleteRingSubmitFailedChan(c RingSubmitFailedChan) {
	ringClient.mtx.Lock()
	defer ringClient.mtx.Unlock()

	chans := make([]RingSubmitFailedChan, 0)
	for _, v := range ringClient.ringSubmitFailedChans {
		if v != c {
			chans = append(chans, v)
		}
	}
	ringClient.ringSubmitFailedChans = chans
}

func (ringClient *RingClient) NewRing(ringState *types.RingState) error {
	ringClient.mtx.Lock()
	defer ringClient.mtx.Unlock()

	if err := ringClient.canSubmit(ringState); nil == err {
		if ringBytes, err := json.Marshal(ringState); err == nil {
			ringClient.unSubmitedRingsStore.Put(ringState.RawRing.Hash.Bytes(), ringBytes)
			ringData, _ := json.Marshal(ringState)
			log.Debugf("ringState:%x", ringData)
			if IfRegistryRingHash {
				return ringClient.sendRinghashRegistry(ringState)
			} else {
				return ringClient.submitRing(ringState)
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
func (ringClient *RingClient) canSubmit(ringState *types.RingState) error {
	ringData, _ := ringClient.unSubmitedRingsStore.Get(ringState.RawRing.Hash.Bytes())
	if nil == ringData || len(ringData) == 0 {
		ringData, _ = ringClient.submitedRingsStore.Get(ringState.RawRing.Hash.Bytes())
		if nil == ringData || len(ringData) == 0 {
			return nil
		}
	}
	return errors.New("had been processed it")
}

//send Fingerprint to block chain
func (ringClient *RingClient) sendRinghashRegistry(ringState *types.RingState) error {
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
			return ringClient.unSubmitedRingsStore.Put(ringState.RawRing.Hash.Bytes(), ringBytes)
		}
	}
}

func (ringClient *RingClient) submitRing(ringSate *types.RingState) error {
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
func (ringClient *RingClient) recoverRing() {

	//Traversal the uncompelete rings
	iterator := ringClient.unSubmitedRingsStore.NewIterator(nil, nil)
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
								ringClient.submitRing(ring)
							} else {

							}
						}
					} else {
						ringClient.NewRing(ring)
					}
				}
			} else {
				for _, c := range ringClient.ringSubmitFailedChans {
					c <- ring
				}
			}
		}
	}
}

func (ringClient *RingClient) processSubmitRingEvent(e chainclient.AbiEvent) {
	if nil == e {
		//for _,failedChan := range ringClient.ringSubmitFailedChans {
		//				//todo: generate or get ringstate
		//				failedChan <- nil
		//			}
	}
}

func (ringClient *RingClient) processRegistryEvent(e eventemitter.EventData) error {

	if nil == e {
		//for _,failedChan := range ringClient.ringSubmitFailedChans {
		//	//todo: generate or get ringstate
		//	log.Errorf("errhhhhhhhhhh:%s", tx.Hash)
		//	failedChan <- nil
		//}
	} else {
		event := e.(chainclient.RinghashSubmitted)
		ringHash := types.BytesToHash(event.RingHash)
		println("ringHash.HexringHash.Hex", ringHash.Hex())
		ringData, _ := ringClient.unSubmitedRingsStore.Get(ringHash.Bytes())
		if nil != ringData {
			ring := &types.RingState{}
			if err := json.Unmarshal(ringData, ring); nil != err {
				log.Errorf("error:%s", err.Error())
			} else {
				log.Debugf("ringhashRegistry:%s", string(ringData))
				//todo:need pre condition
				//if err := ringClient.submitRing(ring); nil != err {
				//	log.Errorf("error:%s", err.Error())
				//}
			}
		}
	}
	return nil

}

func (ringClient *RingClient) Start() {
	//ringClient.recoverRing()

	watcher := &eventemitter.Watcher{Concurrent: false, Handle: ringClient.processRegistryEvent}
	for _, imp := range LoopringInstance.LoopringImpls {
		e := imp.RingHashRegistry.RinghashSubmittedEvent
		topic := e.Address().Hex() + e.Id()
		eventemitter.On(topic, watcher)
	}
}
