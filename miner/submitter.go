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
	"github.com/Loopring/relay/chainclient"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"math/big"
	"sync"
)

//保存ring，并将ring发送到区块链，同样需要分为待完成和已完成
type RingSubmitter struct {
	Chainclient *chainclient.Client

	MinerPrivateKey    []byte //used to sign the ring
	MinerAddress       types.Address
	feeRecepient       types.Address //used to receive fee
	ifRegistryRingHash bool

	stopChan chan bool

	mtx *sync.RWMutex

	//todo:
	registeredRings map[types.Hash]types.RingState
}

type RingSubmitFailed struct {
	RingState *types.RingState
	err       error
}

func NewSubmitter(options config.MinerOptions, commOpts config.CommonOptions, client *chainclient.Client) *RingSubmitter {
	submitter := &RingSubmitter{}
	submitter.Chainclient = client
	submitter.mtx = &sync.RWMutex{}

	passphrase := &types.Passphrase{}
	passphrase.SetBytes(commOpts.Passphrase)
	var err error
	submitter.MinerPrivateKey, err = crypto.AesDecrypted(passphrase.Bytes(), types.FromHex(options.Miner))
	if nil != err {
		panic(err)
	}
	submitter.feeRecepient = types.HexToAddress(options.FeeRecepient)
	submitter.ifRegistryRingHash = options.IfRegistryRingHash

	submitter.registeredRings = make(map[types.Hash]types.RingState)

	return submitter
}

func (submitter *RingSubmitter) newRings(eventData eventemitter.EventData) error {

	submitter.mtx.Lock()
	defer submitter.mtx.Unlock()
	ringStates := eventData.([]*types.RingState)
	if submitter.ifRegistryRingHash {
		println("ppppppppppppppp")

		if len(ringStates) == 1 {
			return submitter.ringhashRegistry(ringStates[0])
		} else {
			return submitter.batchRinghashRegistry(ringStates)
		}
	} else {
		println("dddddd")
		for _, ringState := range ringStates {
			if err := submitter.submitRing(ringState); nil != err {
				//todo:index
				return err
			}
		}
		return nil
	}
}

func isOrdersRemined(ring *types.RingState) bool {
	//todo:args validator
	return true
}

//todo: 不在submit中的才会提交
func (submitter *RingSubmitter) canSubmit(ringState *types.RingState) error {
	return errors.New("had been processed")
}

func (submitter *RingSubmitter) batchRinghashRegistry(ringState []*types.RingState) error {
	return nil
}

func (submitter *RingSubmitter) ringhashRegistry(ringState *types.RingState) error {
	ring := ringState.RawRing
	contractAddress := ring.Orders[0].OrderState.RawOrder.Protocol

	if txHash, err := chainclient.LoopringInstance.LoopringImpls[contractAddress].RingHashRegistry.SubmitRinghash.SendTransaction(types.HexToAddress("0x"),
		big.NewInt(int64(len(ring.Orders))),
		submitter.MinerAddress,
		ringState.RawRing.Hash,
	); nil != err {
		return err
	} else {
		ringState.RegistryTxHash = types.HexToHash(txHash)
		if _, err := json.Marshal(ringState); nil != err {
			return err
		}
	}
	return nil
}

func (submitter *RingSubmitter) submitRing(ringSate *types.RingState) error {
	ring := ringSate.RawRing
	ring.FeeRecepient = submitter.feeRecepient
	contractAddress := ring.Orders[0].OrderState.RawOrder.Protocol
	log.Debugf("submitRing ringState lrcFee:%s", ringSate.LegalFee)
	ringSubmitArgs := ring.GenerateSubmitArgs(submitter.MinerPrivateKey)
	if _, err := chainclient.LoopringInstance.LoopringImpls[contractAddress].SubmitRing.SendTransaction(types.HexToAddress("0x"),
		ringSubmitArgs.AddressList,
		ringSubmitArgs.UintArgsList,
		ringSubmitArgs.Uint8ArgsList,
		ringSubmitArgs.BuyNoMoreThanAmountBList,
		ringSubmitArgs.VList,
		ringSubmitArgs.RList,
		ringSubmitArgs.SList,
		ringSubmitArgs.Ringminer,
		ringSubmitArgs.FeeRecepient,
	); nil != err {
		return err
	}
	return nil
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
	//ringHashBytes, _ := submitter.txToRingHashIndexStore.Get(txHash.Bytes())
	//if nil != ringHashBytes && len(ringHashBytes) > 0 {
	//	if ringData, _ := submitter.unSubmitedRingsStore.Get(ringHashBytes); nil == ringData || len(ringData) == 0 {
	//		if ringData, _ = submitter.submitedRingsStore.Get(ringHashBytes); nil != ringData || len(ringData) > 0 {
	//			var ringState *types.RingState
	//			if err := json.Unmarshal(ringData, ringState); nil == err {
	//				failedEvent := &RingSubmitFailed{RingState: ringState, err: errors.New("execute ring failed")}
	//				eventemitter.Emit(eventemitter.RingSubmitFailed, failedEvent)
	//			}
	//		}
	//	}
	//}
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
			//todo:
			ringData := []byte{}
			if nil != ringData {
				ringState := &types.RingState{}
				if err := json.Unmarshal(ringData, ringState); nil != err {
					log.Errorf("error:%s", err.Error())
				} else {
					log.Debugf("ringhashRegistry:%s", string(ringData))
					//todo:need pre condition
					if err := submitter.submitRing(ringState); nil != err {
						log.Errorf("error:%s", err.Error())
					}
				}
			}
		}
	}

	return nil
}

func (submitter *RingSubmitter) stop() {
	//todo
}

func (submitter *RingSubmitter) start() {
	newRingWatcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.newRings}
	eventemitter.On(eventemitter.Miner_NewRing, newRingWatcher)

	watcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.handleRegistryEvent}
	for _, imp := range chainclient.LoopringInstance.LoopringImpls {
		e := imp.RingHashRegistry.RinghashSubmittedEvent
		topic := e.Address().Hex() + e.Id()
		eventemitter.On(topic, watcher)
	}
}
