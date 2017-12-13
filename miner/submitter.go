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
	"errors"
	"math/big"
	"sync"

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
)

//保存ring，并将ring发送到区块链，同样需要分为待完成和已完成
type RingSubmitter struct {
	Accessor           *ethaccessor.EthNodeAccessor
	miner              accounts.Account
	ks                 *keystore.KeyStore
	feeReceipt         common.Address //used to receive fee
	ifRegistryRingHash bool
	gasLimit           *big.Int
	stopChan           chan bool

	mtx *sync.RWMutex

	//todo:
	registeredRings map[common.Hash]types.RingSubmitInfo

	dbService         dao.RdsService
	marketCapProvider *marketcap.MarketCapProvider

	newRingWatcher             *eventemitter.Watcher
	ringhashSubmitWatcher      *eventemitter.Watcher
	registryMethodWatcher      *eventemitter.Watcher
	batchRegistryMethodWatcher *eventemitter.Watcher
	ringSubmitMethodWatcher    *eventemitter.Watcher
}

type RingSubmitFailed struct {
	RingState *types.Ring
	err       error
}

func NewSubmitter(options config.MinerOptions, accessor *ethaccessor.EthNodeAccessor, dbService dao.RdsService, marketCapProvider *marketcap.MarketCapProvider) *RingSubmitter {
	submitter := &RingSubmitter{}
	submitter.gasLimit = big.NewInt(options.GasLimit)
	submitter.dbService = dbService
	submitter.marketCapProvider = marketCapProvider
	submitter.Accessor = accessor
	submitter.mtx = &sync.RWMutex{}
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
			log.Errorf("miner submitter,insert new ring err:%s", err.Error())
			return err
		} else {
			for _, filledOrder := range info.RawRing.Orders {
				daoOrder := &dao.FilledOrder{}
				daoOrder.ConvertDown(filledOrder, info.Ringhash)
				if err1 := submitter.dbService.Add(daoOrder); nil != err1 {
					log.Errorf("miner submitter,insert filled Order err:%s", err1.Error())
					return err1
				}
			}
		}
	}

	if submitter.ifRegistryRingHash {
		if len(ringInfos) == 1 {
			if err := submitter.ringhashRegistry(ringInfos[0]); nil != err {
				submitter.dbService.UpdateRingSubmitInfoFailed([]common.Hash{ringInfos[0].Ringhash}, err.Error())
				return err
			}
		} else {
			infosMap := make(map[common.Address][]*types.RingSubmitInfo)
			for _, info := range ringInfos {
				if _, ok := infosMap[info.ProtocolAddress]; !ok {
					infosMap[info.ProtocolAddress] = []*types.RingSubmitInfo{}
				}
				infosMap[info.ProtocolAddress] = append(infosMap[info.ProtocolAddress], info)
			}
			for protocolAddr, infos := range infosMap {
				ringhashes := []common.Hash{}
				miners := []common.Address{}
				for _, info := range infos {
					miners = append(miners, info.RawRing.Miner)
					ringhashes = append(ringhashes, info.Ringhash)
				}
				if err := submitter.batchRinghashRegistry(protocolAddr, ringhashes, miners); nil != err {
					submitter.dbService.UpdateRingSubmitInfoFailed(ringhashes, err.Error())
				}
			}
		}
	} else {
		for _, ringState := range ringInfos {
			if err := submitter.submitRing(ringState); nil != err {
				//todo:index
				submitter.dbService.UpdateRingSubmitInfoFailed([]common.Hash{ringState.Ringhash}, err.Error())
				return err
			}
		}
	}
	return nil
}

//todo: 不在submit中的才会提交
func (submitter *RingSubmitter) canSubmit(ringState *types.RingSubmitInfo) error {
	return errors.New("had been processed")
}

func (submitter *RingSubmitter) batchRinghashRegistry(contractAddress common.Address, ringhashes []common.Hash, miners []common.Address) error {
	ringhashRegistryAbi := submitter.Accessor.RinghashRegistryAbi
	var ringhashRegistryAddress common.Address
	if implAddress, exists := submitter.Accessor.ProtocolAddresses[contractAddress]; !exists {
		return errors.New("does't contain this version")
	} else {
		ringhashRegistryAddress = implAddress.RinghashRegistryAddress
	}
	if registryData, err := ringhashRegistryAbi.Pack("submitRinghash",
		miners,
		ringhashes); nil != err {
		return err
	} else {
		if gas, gasPrice, err1 := submitter.Accessor.EstimateGas(registryData, ringhashRegistryAddress); nil != err {
			return err1
		} else {
			if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.miner, ringhashRegistryAddress, gas, gasPrice, nil, registryData); nil != err {
				return err
			} else {
				submitter.dbService.UpdateRingSubmitInfoRegistryTxHash(ringhashes, txHash)
			}
		}
	}
	return nil
}

func (submitter *RingSubmitter) ringhashRegistry(ringState *types.RingSubmitInfo) error {
	contractAddress := ringState.ProtocolAddress
	var ringhashRegistryAddress common.Address
	if implAddress, exists := submitter.Accessor.ProtocolAddresses[contractAddress]; !exists {
		return errors.New("does't contains this version")
	} else {
		ringhashRegistryAddress = implAddress.RinghashRegistryAddress
	}

	if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.miner, ringhashRegistryAddress, ringState.RegistryGas, ringState.RegistryGasPrice, nil, ringState.RegistryData); nil != err {
		return err
	} else {
		ringState.RegistryTxHash = common.HexToHash(txHash)
		submitter.dbService.UpdateRingSubmitInfoRegistryTxHash([]common.Hash{ringState.Ringhash}, txHash)
	}
	return nil
}

func (submitter *RingSubmitter) submitRing(ringSate *types.RingSubmitInfo) error {
	if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.miner, ringSate.ProtocolAddress, ringSate.ProtocolGas, ringSate.ProtocolGasPrice, nil, ringSate.ProtocolData); nil != err {
		return err
	} else {
		ringSate.SubmitTxHash = common.HexToHash(txHash)
		submitter.dbService.UpdateRingSubmitInfoProtocolTxHash(ringSate.Ringhash, txHash)
	}
	return nil
}

func (submitter *RingSubmitter) handleSubmitRingMethodEvent(e eventemitter.EventData) error {
	if nil != e {
		event := e.(types.SubmitRingMethodEvent)
		if nil != event.Err {
			if ringhashes, err := submitter.dbService.GetRingHashesByTxHash(event.TxHash); nil != err {
				log.Errorf("err:%s", err.Error())
				return err
			} else {
				submitter.submitFailed(ringhashes, errors.New("failed to execute ring"))
			}
		}
		return submitter.dbService.UpdateRingSubmitInfoRegistryUsedGas(event.TxHash.Hex(), event.UsedGas)
	}
	return nil
}

func (submitter *RingSubmitter) handleBatchSubmitRingMethodEvent(e eventemitter.EventData) error {
	if nil != e {
		event := e.(types.BatchSubmitRingHashMethodEvent)
		if nil != event.Err {
			if ringhashes, err := submitter.dbService.GetRingHashesByTxHash(event.TxHash); nil != err {
				log.Errorf("err:%s", err.Error())
				return err
			} else {
				submitter.submitFailed(ringhashes, errors.New("failed to execute ring"))
			}
		}
		return submitter.dbService.UpdateRingSubmitInfoRegistryUsedGas(event.TxHash.Hex(), event.UsedGas)
	}
	return nil
}

//提交错误，执行错误
func (submitter *RingSubmitter) submitFailed(ringhashes []common.Hash, err error) {
	if err := submitter.dbService.UpdateRingSubmitInfoFailed(ringhashes, err.Error()); nil != err {
		log.Errorf("err:%s", err.Error())
	} else {
		for _, ringhash := range ringhashes {
			failedEvent := &types.RingSubmitFailedEvent{RingHash: ringhash}
			eventemitter.Emit(eventemitter.Miner_RingSubmitFailed, failedEvent)
		}
	}
}

func (submitter *RingSubmitter) handleRegistryMethodEvent(e eventemitter.EventData) error {
	if nil != e {
		event := e.(*types.RingHashSubmitMethodEvent)
		if nil != event.Err {
			if ringhashes, err := submitter.dbService.GetRingHashesByTxHash(event.TxHash); nil != err {
				log.Errorf("err:%s", err.Error())
			} else {
				submitter.submitFailed(ringhashes, errors.New("failed to execute ringhash registry:"+event.Err.Error()))
			}
		} else {
			submitter.dbService.UpdateRingSubmitInfoRegistryUsedGas(event.TxHash.Hex(), event.UsedGas)
		}
	}
	return nil
}

func (submitter *RingSubmitter) handleRegistryEvent(e eventemitter.EventData) error {
	if nil != e {
		event := e.(*types.RinghashSubmittedEvent)
		var (
			err         error
			implAddress *ethaccessor.ProtocolAddress
			exists      bool
		)
		info := &types.RingSubmitInfo{}
		daoInfo, _ := submitter.dbService.GetRingForSubmitByHash(event.RingHash)
		daoInfo.ConvertUp(info)
		if types.IsZeroHash(info.Ringhash) {
			err = errors.New("ring hash is zero")
		} else {
			if implAddress, exists = submitter.Accessor.ProtocolAddresses[info.ProtocolAddress]; !exists {
				err = errors.New("doesn't contain this version of protocol:" + info.ProtocolAddress.Hex())
			}
			callMethod := submitter.Accessor.ContractCallMethod(submitter.Accessor.RinghashRegistryAbi, implAddress.RinghashRegistryAddress)
			var canSubmit types.Big
			if err = callMethod(&canSubmit, "canSubmit", "latest", info.Ringhash, info.Miner); nil != err {
				log.Errorf("err:%s", err.Error())
			} else {
				if canSubmit.Int() <= 0 {
					err = errors.New("failed to call method:canSubmit")
				}
			}
		}

		if nil == err {
			if err = submitter.submitRing(info); nil != err {
				log.Errorf("error:%s", err.Error())
				submitter.dbService.UpdateRingSubmitInfoFailed([]common.Hash{info.Ringhash}, err.Error())
			}
		}
	}
	return nil
}

func (submitter *RingSubmitter) GenerateRingSubmitInfo(ringState *types.Ring) (*types.RingSubmitInfo, error) {
	protocolAddress := ringState.Orders[0].OrderState.RawOrder.Protocol
	var (
		implAddress *ethaccessor.ProtocolAddress
		exists      bool
		err         error
	)
	if implAddress, exists = submitter.Accessor.ProtocolAddresses[protocolAddress]; !exists {
		return nil, errors.New("doesn't contain this version of protocol:" + protocolAddress.Hex())
	}
	protocolAbi := submitter.Accessor.ProtocolImplAbi
	ringForSubmit := &types.RingSubmitInfo{RawRing: ringState}
	if types.IsZeroHash(ringState.Hash) {
		ringState.Hash = ringState.GenerateHash()
	}
	ringForSubmit.Miner = submitter.miner.Address

	ringForSubmit.ProtocolAddress = protocolAddress
	ringForSubmit.OrdersCount = big.NewInt(int64(len(ringState.Orders)))
	ringForSubmit.Ringhash = ringState.Hash

	registryCost := big.NewInt(int64(0))

	if submitter.ifRegistryRingHash {
		ringhashRegistryAbi := submitter.Accessor.RinghashRegistryAbi
		ringhashRegistryAddress := implAddress.RinghashRegistryAddress
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
		if ringForSubmit.RegistryGas.Cmp(submitter.gasLimit) > 0 {
			ringForSubmit.RegistryGas.Set(submitter.gasLimit)
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
	if ringForSubmit.ProtocolGas.Cmp(submitter.gasLimit) > 0 {
		ringForSubmit.ProtocolGas.Set(submitter.gasLimit)
	}
	protocolCost := new(big.Int).Mul(ringForSubmit.ProtocolGas, ringForSubmit.ProtocolGasPrice)

	cost := new(big.Rat).SetInt(new(big.Int).Add(protocolCost, registryCost))
	cost = cost.Mul(cost, submitter.marketCapProvider.GetEthCap())
	received := new(big.Rat).Sub(ringState.LegalFee, cost)
	ringForSubmit.Received = received

	if received.Cmp(big.NewRat(int64(0), int64(1))) <= 0 {
		// todo: warning
		//return nil, errors.New("received can't be less than 0")
	}
	return ringForSubmit, nil
}

func (submitter *RingSubmitter) stop() {
	eventemitter.Un(eventemitter.Miner_NewRing, submitter.newRingWatcher)
	eventemitter.Un(eventemitter.RingHashSubmitted, submitter.ringhashSubmitWatcher)
	eventemitter.Un(eventemitter.Miner_SubmitRingHash_Method, submitter.registryMethodWatcher)
	eventemitter.Un(eventemitter.Miner_BatchSubmitRingHash_Method, submitter.batchRegistryMethodWatcher)
	eventemitter.Un(eventemitter.Miner_SubmitRing_Method, submitter.ringSubmitMethodWatcher)
}

func (submitter *RingSubmitter) start() {
	newRingWatcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.newRings}
	submitter.newRingWatcher = newRingWatcher
	eventemitter.On(eventemitter.Miner_NewRing, submitter.newRingWatcher)

	registryMethodWatcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.handleRegistryMethodEvent}
	submitter.ringhashSubmitWatcher = registryMethodWatcher
	eventemitter.On(eventemitter.Miner_SubmitRingHash_Method, submitter.registryMethodWatcher)

	ringSubmitMethodWatcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.handleSubmitRingMethodEvent}
	submitter.ringSubmitMethodWatcher = ringSubmitMethodWatcher
	eventemitter.On(eventemitter.Miner_SubmitRing_Method, submitter.ringSubmitMethodWatcher)

	//Miner_BatchSubmitRingHash_Method
	batchRegistryMethodWatcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.handleBatchSubmitRingMethodEvent}
	submitter.batchRegistryMethodWatcher = batchRegistryMethodWatcher
	eventemitter.On(eventemitter.Miner_BatchSubmitRingHash_Method, submitter.batchRegistryMethodWatcher)

	watcher := &eventemitter.Watcher{Concurrent: false, Handle: submitter.handleRegistryEvent}
	submitter.ringhashSubmitWatcher = watcher
	eventemitter.On(eventemitter.RingHashSubmitted, submitter.ringhashSubmitWatcher)

}
