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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"sync"
)

//保存ring，并将ring发送到区块链，同样需要分为待完成和已完成
type RingSubmitter struct {
	Accessor           *ethaccessor.EthNodeAccessor
	miner              accounts.Account
	ks                 *keystore.KeyStore
	feeReceipt         types.Address //used to receive fee
	ifRegistryRingHash bool

	stopChan chan bool

	mtx *sync.RWMutex

	//todo:
	registeredRings map[types.Hash]types.RingForSubmit
}

type RingSubmitFailed struct {
	RingState *types.Ring
	err       error
}

func NewSubmitter(options config.MinerOptions, ks *keystore.KeyStore, accessor *ethaccessor.EthNodeAccessor) *RingSubmitter {
	submitter := &RingSubmitter{}
	submitter.Accessor = accessor
	submitter.mtx = &sync.RWMutex{}
	submitter.ks = ks
	submitter.miner = accounts.Account{Address: common.HexToAddress(options.Miner)}

	submitter.feeReceipt = types.HexToAddress(options.FeeRecepient)
	submitter.ifRegistryRingHash = options.IfRegistryRingHash

	submitter.registeredRings = make(map[types.Hash]types.RingForSubmit)

	return submitter
}

func (submitter *RingSubmitter) newRings(eventData eventemitter.EventData) error {
	submitter.mtx.Lock()
	defer submitter.mtx.Unlock()

	rings := eventData.([]*types.RingForSubmit)
	if submitter.ifRegistryRingHash {
		if len(rings) == 1 {
			return submitter.ringhashRegistry(rings[0])
		} else {
			return submitter.batchRinghashRegistry(rings)
		}
	} else {
		for _, ringState := range rings {
			if err := submitter.submitRing(ringState); nil != err {
				//todo:index
				return err
			}
		}
		return nil
	}
}

func isOrdersRemined(ring *types.Ring) bool {
	//todo:args validator
	return true
}

//todo: 不在submit中的才会提交
func (submitter *RingSubmitter) canSubmit(ringState *types.RingForSubmit) error {
	return errors.New("had been processed")
}

func (submitter *RingSubmitter) batchRinghashRegistry(ringState []*types.RingForSubmit) error {
	return nil
}

func (submitter *RingSubmitter) ringhashRegistry(ringState *types.RingForSubmit) error {
	contractAddress := ringState.ProtocolAddress
	ringhashRegistryAddress := submitter.Accessor.ProtocolImpls[contractAddress].RinghashRegistryAddress
	if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.miner, ringhashRegistryAddress, ringState.RegistryGas, ringState.RegistryGasPrice, ringState.RegistryData); nil != err {
		return err
	} else {
		ringState.RegistryTxHash = types.HexToHash(txHash)
	}
	return nil
}

func (submitter *RingSubmitter) submitRing(ringSate *types.RingForSubmit) error {
	if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.miner, ringSate.ProtocolAddress, ringSate.ProtocolGas, ringSate.ProtocolGasPrice, ringSate.ProtocolData); nil != err {
		return err
	} else {
		ringSate.SubmitTxHash = types.HexToHash(txHash)
	}
	return nil
}

func (submitter *RingSubmitter) handleSubmitRingEvent(e eventemitter.EventData) error {
	if nil != e {
		contractEventData := e.(ethaccessor.ContractData)
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
		contractEventData := e.(ethaccessor.ContractData)
		//registry failed
		if nil == contractEventData.Event {
			submitter.submitFailed(contractEventData.TxHash)
		} else {
			event := contractEventData.Event.(ethaccessor.RinghashSubmitted)
			ringHash := types.BytesToHash(event.RingHash)
			println("ringHash.HexringHash.Hex", ringHash.Hex())
			//todo:change to dao
			ringData := []byte{}
			if nil != ringData {
				ringState := &types.RingForSubmit{}
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

func (submitter *RingSubmitter) GenerateRingSubmitArgs(ringState *types.Ring) (*types.RingForSubmit, error) {
	var err error
	protocolAddress := ringState.Orders[0].OrderState.RawOrder.Protocol
	protocolAbi := submitter.Accessor.ProtocolImpls[protocolAddress].ProtocolImplAbi

	ringForSubmit := &types.RingForSubmit{}
	ringForSubmit.ProtocolAddress = protocolAddress
	ringForSubmit.OrdersCount = big.NewInt(int64(len(ringState.Orders)))
	ringForSubmit.Ringhash = ringState.Hash

	if submitter.ifRegistryRingHash {
		ringhashRegistryAbi := submitter.Accessor.ProtocolImpls[protocolAddress].RinghashRegistryAbi
		ringhashRegistryAddress := submitter.Accessor.ProtocolImpls[protocolAddress].RinghashRegistryAddress

		ringForSubmit.RegistryData, err = ringhashRegistryAbi.Pack("submitRinghash", ringForSubmit.OrdersCount,
			submitter.miner.Address,
			ringForSubmit.Ringhash)
		if nil != err {
			return nil, err
		}

		ringForSubmit.RegistryGas, ringForSubmit.RegistryGasPrice, err = submitter.Accessor.EstimateGas(ringForSubmit.RegistryData, ringhashRegistryAddress)
		if nil != err {
			return nil, err
		}
	}

	ringSubmitArgs := ringState.GenerateSubmitArgs(submitter.miner.Address.Hex(), submitter.feeReceipt)
	ringForSubmit.ProtocolData, err = protocolAbi.Pack("submitRing",
		ringSubmitArgs.AddressList,
		ringSubmitArgs.UintArgsList,
		ringSubmitArgs.Uint8ArgsList,
		ringSubmitArgs.BuyNoMoreThanAmountBList,
		ringSubmitArgs.VList,
		ringSubmitArgs.RList,
		ringSubmitArgs.SList,
		ringSubmitArgs.Ringminer,
		ringSubmitArgs.FeeRecepient,
	)
	if nil != err {
		return nil, err
	}
	ringForSubmit.ProtocolGas, ringForSubmit.ProtocolGasPrice, err = submitter.Accessor.EstimateGas(ringForSubmit.ProtocolData, protocolAddress)
	if nil != err {
		return nil, err
	}
	return ringForSubmit, nil
}

func (submitter *RingSubmitter) stop() {
	//todo
}

func (submitter *RingSubmitter) start() {
	newRingWatcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.newRings}
	eventemitter.On(eventemitter.Miner_NewRing, newRingWatcher)

	watcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.handleRegistryEvent}
	for _, imp := range submitter.Accessor.ProtocolImpls {
		registryAddress := imp.RinghashRegistryAddress
		eventId := imp.RinghashRegistryAbi.Events["ringhashSubmitted"].Id()
		topic := registryAddress.Hex() + eventId.Hex()
		eventemitter.On(topic, watcher)
	}
}
