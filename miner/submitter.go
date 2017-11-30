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
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/marketcap"
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
	feeReceipt         common.Address //used to receive fee
	ifRegistryRingHash bool

	stopChan chan bool

	mtx *sync.RWMutex

	//todo:
	registeredRings map[common.Hash]types.RingSubmitInfo

	dbService         dao.RdsService
	marketCapProvider *marketcap.MarketCapProvider
}

type RingSubmitFailed struct {
	RingState *types.Ring
	err       error
}

func NewSubmitter(options config.MinerOptions, ks *keystore.KeyStore, accessor *ethaccessor.EthNodeAccessor, dbService dao.RdsService, marketCapProvider *marketcap.MarketCapProvider) *RingSubmitter {
	submitter := &RingSubmitter{}
	submitter.dbService = dbService
	submitter.marketCapProvider = marketCapProvider
	submitter.Accessor = accessor
	submitter.mtx = &sync.RWMutex{}
	submitter.ks = ks
	submitter.miner = accounts.Account{Address: common.HexToAddress(options.Miner)}

	submitter.feeReceipt = common.HexToAddress(options.FeeRecepient)
	submitter.ifRegistryRingHash = options.IfRegistryRingHash

	submitter.registeredRings = make(map[common.Hash]types.RingSubmitInfo)

	return submitter
}

func (submitter *RingSubmitter) newRings(eventData eventemitter.EventData) error {
	submitter.mtx.Lock()
	defer submitter.mtx.Unlock()

	ringInfos := eventData.([]*types.RingSubmitInfo)

	for _, info := range ringInfos {
		daoInfo := &dao.RingSubmitInfo{}
		daoInfo.ConvertDown(info)
		if err := submitter.dbService.Add(daoInfo); nil != err {
			log.Errorf("err:%s", err.Error())
		}
	}
	if submitter.ifRegistryRingHash {
		if len(ringInfos) == 1 {
			return submitter.ringhashRegistry(ringInfos[0])
		} else {
			return submitter.batchRinghashRegistry(ringInfos)
		}
	} else {
		for _, ringState := range ringInfos {
			if err := submitter.submitRing(ringState); nil != err {
				//todo:index
				return err
			}
		}
		return nil
	}
}

//todo: 不在submit中的才会提交
func (submitter *RingSubmitter) canSubmit(ringState *types.RingSubmitInfo) error {
	return errors.New("had been processed")
}

func (submitter *RingSubmitter) batchRinghashRegistry(ringInfos []*types.RingSubmitInfo) error {
	infosMap := make(map[common.Address][]*types.RingSubmitInfo)
	for _, info := range ringInfos {
		if _, ok := infosMap[info.ProtocolAddress]; !ok {
			infosMap[info.ProtocolAddress] = []*types.RingSubmitInfo{}
		}
		infosMap[info.ProtocolAddress] = append(infosMap[info.ProtocolAddress], info)
	}
	for protocolAddr, infos := range infosMap {
		contractAddress := protocolAddr
		miners := []common.Address{}
		ringhashes := []common.Hash{}
		for _, info := range infos {
			miners = append(miners, info.RawRing.Miner)
			ringhashes = append(ringhashes, info.Ringhash)
		}
		ringhashRegistryAbi := submitter.Accessor.ProtocolImpls[contractAddress].RinghashRegistryAbi
		ringhashRegistryAddress := submitter.Accessor.ProtocolImpls[contractAddress].RinghashRegistryAddress

		if registryData, err := ringhashRegistryAbi.Pack("submitRinghash",
			miners,
			ringhashes); nil != err {
			return err
		} else {
			if gas, gasPrice, err1 := submitter.Accessor.EstimateGas(registryData, ringhashRegistryAddress); nil != err {
				return err1
			} else {
				if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.miner, ringhashRegistryAddress, gas, gasPrice, registryData); nil != err {
					return err
				} else {
					submitter.dbService.UpdateRingSubmitInfoRegistryTxHash(ringhashes, txHash)
				}
			}
		}

	}

	return nil
}

func (submitter *RingSubmitter) ringhashRegistry(ringState *types.RingSubmitInfo) error {
	contractAddress := ringState.ProtocolAddress
	ringhashRegistryAddress := submitter.Accessor.ProtocolImpls[contractAddress].RinghashRegistryAddress
	if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.miner, ringhashRegistryAddress, ringState.RegistryGas, ringState.RegistryGasPrice, ringState.RegistryData); nil != err {
		return err
	} else {
		ringState.RegistryTxHash = common.HexToHash(txHash)
		submitter.dbService.UpdateRingSubmitInfoRegistryTxHash([]common.Hash{ringState.Ringhash}, txHash)
	}
	return nil
}

func (submitter *RingSubmitter) submitRing(ringSate *types.RingSubmitInfo) error {
	if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.miner, ringSate.ProtocolAddress, ringSate.ProtocolGas, ringSate.ProtocolGasPrice, ringSate.ProtocolData); nil != err {
		return err
	} else {
		ringSate.SubmitTxHash = common.HexToHash(txHash)
		submitter.dbService.UpdateRingSubmitInfoSubmitTxHash(ringSate.Ringhash, txHash)
	}
	return nil
}

func (submitter *RingSubmitter) handleSubmitRingEvent(e eventemitter.EventData) error {
	if nil != e {
		event := e.(types.SubmitRingEvent)
		//excute ring failed
		//if nil == event {
		submitter.submitFailed(event.TxHash)
		//}
	}
	return nil
}

func (submitter *RingSubmitter) submitFailed(event common.Hash) {
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
		event := e.(types.RingHashRegistryEvent)
		//registry failed
		//if nil == event {
		//	submitter.submitFailed(event.TxHash)
		//} else {
		ringHash := event.RingHash
		println("ringHash.HexringHash.Hex", ringHash.Hex())
		//todo:change to dao
		ringData := []byte{}
		if nil != ringData {
			ringState := &types.RingSubmitInfo{}
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
		//}
	}

	return nil
}

func (submitter *RingSubmitter) GenerateRingSubmitInfo(ringState *types.Ring) (*types.RingSubmitInfo, error) {
	var err error
	protocolAddress := ringState.Orders[0].OrderState.RawOrder.Protocol
	protocolAbi := submitter.Accessor.ProtocolImpls[protocolAddress].ProtocolImplAbi

	ringForSubmit := &types.RingSubmitInfo{RawRing: ringState}
	if types.IsZeroHash(ringState.Hash) {
		ringState.Hash = ringState.GenerateHash()
	}
	ringForSubmit.Miner = ringState.Miner
	ringForSubmit.ProtocolAddress = protocolAddress
	ringForSubmit.OrdersCount = big.NewInt(int64(len(ringState.Orders)))
	ringForSubmit.Ringhash = ringState.Hash

	registryCost := big.NewInt(int64(0))
	if submitter.ifRegistryRingHash {
		ringhashRegistryAbi := submitter.Accessor.ProtocolImpls[protocolAddress].RinghashRegistryAbi
		ringhashRegistryAddress := submitter.Accessor.ProtocolImpls[protocolAddress].RinghashRegistryAddress

		ringForSubmit.RegistryData, err = ringhashRegistryAbi.Pack("submitRinghash",
			submitter.miner.Address,
			ringForSubmit.Ringhash)
		if nil != err {
			return nil, err
		}

		ringForSubmit.RegistryGas, ringForSubmit.RegistryGasPrice, err = submitter.Accessor.EstimateGas(ringForSubmit.RegistryData, ringhashRegistryAddress)
		if nil != err {
			return nil, err
		}
		registryCost.Mul(ringForSubmit.RegistryGas, ringForSubmit.RegistryGasPrice)
	}

	ringSubmitArgs := ringState.GenerateSubmitArgs(submitter.miner.Address, submitter.feeReceipt)
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

	protocolCost := new(big.Int).Mul(ringForSubmit.ProtocolGas, ringForSubmit.ProtocolGasPrice)
	cost := new(big.Rat).SetInt(new(big.Int).Add(protocolCost, registryCost))
	cost = cost.Mul(cost, submitter.marketCapProvider.GetEthCap())
	received := new(big.Rat).Sub(ringState.LegalFee, cost)
	ringForSubmit.Received = received
	if received.Cmp(big.NewRat(int64(0), int64(1))) <= 0 {
		return nil, errors.New("received can't be less than 0")
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
