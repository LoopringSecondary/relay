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
	Miner              accounts.Account
	ks                 *keystore.KeyStore
	feeReceipt         common.Address //used to receive fee
	ifRegistryRingHash bool
	gasLimit           *big.Int

	//todo:
	registeredRings map[common.Hash]types.RingSubmitInfo

	dbService         dao.RdsService
	marketCapProvider marketcap.MarketCapProvider

	stopFuncs []func()
}

type RingSubmitFailed struct {
	RingState *types.Ring
	err       error
}

func NewSubmitter(options config.MinerOptions, accessor *ethaccessor.EthNodeAccessor, dbService dao.RdsService, marketCapProvider marketcap.MarketCapProvider) *RingSubmitter {
	submitter := &RingSubmitter{}
	submitter.gasLimit = big.NewInt(options.GasLimit)
	submitter.dbService = dbService
	submitter.marketCapProvider = marketCapProvider
	submitter.Accessor = accessor
	submitter.Miner = accounts.Account{Address: common.HexToAddress(options.Miner)}

	submitter.feeReceipt = common.HexToAddress(options.FeeRecepient)
	submitter.ifRegistryRingHash = options.IfRegistryRingHash

	submitter.registeredRings = make(map[common.Hash]types.RingSubmitInfo)
	submitter.stopFuncs = []func(){}
	return submitter
}

func (submitter *RingSubmitter) listenNewRings() {
	ringSubmitInfoChan := make(chan []*types.RingSubmitInfo)
	go func() {
		for {
			select {
			case ringInfos := <-ringSubmitInfoChan:
				if nil != ringInfos {
					for _, info := range ringInfos {
						daoInfo := &dao.RingSubmitInfo{}
						daoInfo.ConvertDown(info)
						if err := submitter.dbService.Add(daoInfo); nil != err {
							log.Errorf("Miner submitter,insert new ring err:%s", err.Error())
						} else {
							for _, filledOrder := range info.RawRing.Orders {
								daoOrder := &dao.FilledOrder{}
								daoOrder.ConvertDown(filledOrder, info.Ringhash)
								if err1 := submitter.dbService.Add(daoOrder); nil != err1 {
									log.Errorf("Miner submitter,insert filled Order err:%s", err1.Error())
								}
							}
						}
					}

					if submitter.ifRegistryRingHash {
						if len(ringInfos) == 1 {
							if err := submitter.ringhashRegistry(ringInfos[0]); nil != err {
								submitter.dbService.UpdateRingSubmitInfoFailed([]common.Hash{ringInfos[0].Ringhash}, err.Error())
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
							}
						}
					}
				}
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			e := eventData.([]*types.RingSubmitInfo)
			log.Debugf("received ringstates length:%d", len(e))
			ringSubmitInfoChan <- e
			return nil
		},
	}
	eventemitter.On(eventemitter.Miner_NewRing, watcher)
	submitter.stopFuncs = append(submitter.stopFuncs, func() {
		close(ringSubmitInfoChan)
		eventemitter.Un(eventemitter.Miner_NewRing, watcher)
	})
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
	if registryData, err := ringhashRegistryAbi.Pack("batchSubmitRinghash",
		miners,
		ringhashes); nil != err {
		return err
	} else {
		if gas, gasPrice, err1 := submitter.Accessor.EstimateGas(registryData, ringhashRegistryAddress); nil != err {
			return err1
		} else {
			if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.Miner, ringhashRegistryAddress, gas, gasPrice, nil, registryData); nil != err {
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

	if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.Miner, ringhashRegistryAddress, ringState.RegistryGas, ringState.RegistryGasPrice, nil, ringState.RegistryData); nil != err {
		return err
	} else {
		ringState.RegistryTxHash = common.HexToHash(txHash)
		submitter.dbService.UpdateRingSubmitInfoRegistryTxHash([]common.Hash{ringState.Ringhash}, txHash)
	}
	return nil
}

func (submitter *RingSubmitter) submitRing(ringSate *types.RingSubmitInfo) error {
	if txHash, err := submitter.Accessor.ContractSendTransactionByData(submitter.Miner, ringSate.ProtocolAddress, ringSate.ProtocolGas, ringSate.ProtocolGasPrice, nil, ringSate.ProtocolData); nil != err {
		return err
	} else {
		ringSate.SubmitTxHash = common.HexToHash(txHash)
		submitter.dbService.UpdateRingSubmitInfoProtocolTxHash(ringSate.Ringhash, txHash)
	}
	return nil
}

func (submitter *RingSubmitter) listenSubmitRingMethodEvent() {
	submitRingMethodChan := make(chan *types.SubmitRingMethodEvent)
	go func() {
		for {
			select {
			case event := <-submitRingMethodChan:
				if nil != event {
					if nil != event.Err {
						if ringhashes, err := submitter.dbService.GetRingHashesByTxHash(event.TxHash); nil != err {
							log.Errorf("err:%s", err.Error())
						} else {
							submitter.submitFailed(ringhashes, errors.New("failed to execute ring"))
						}
					}
					submitter.dbService.UpdateRingSubmitInfoRegistryUsedGas(event.TxHash.Hex(), event.UsedGas)
				}
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			e := eventData.(*types.SubmitRingMethodEvent)
			submitRingMethodChan <- e
			return nil
		},
	}
	eventemitter.On(eventemitter.Miner_SubmitRing_Method, watcher)
	submitter.stopFuncs = append(submitter.stopFuncs, func() {
		close(submitRingMethodChan)
		eventemitter.Un(eventemitter.Miner_SubmitRing_Method, watcher)
	})
}

func (submitter *RingSubmitter) listenBatchSubmitRingMethodEvent() {
	submitRingMethodChan := make(chan *types.BatchSubmitRingHashMethodEvent)
	go func() {
		for {
			select {
			case event := <-submitRingMethodChan:
				if nil != event {
					if nil != event.Err {
						if ringhashes, err := submitter.dbService.GetRingHashesByTxHash(event.TxHash); nil != err {
							log.Errorf("err:%s", err.Error())
						} else {
							submitter.submitFailed(ringhashes, errors.New("failed to execute ring"))
						}
					}
					submitter.dbService.UpdateRingSubmitInfoRegistryUsedGas(event.TxHash.Hex(), event.UsedGas)
				}
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			e := eventData.(*types.BatchSubmitRingHashMethodEvent)
			submitRingMethodChan <- e
			return nil
		},
	}
	eventemitter.On(eventemitter.Miner_BatchSubmitRingHash_Method, watcher)
	submitter.stopFuncs = append(submitter.stopFuncs, func() {
		close(submitRingMethodChan)
		eventemitter.Un(eventemitter.Miner_BatchSubmitRingHash_Method, watcher)
	})
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

func (submitter *RingSubmitter) listenRegistryMethodEvent() {
	submitRingMethodChan := make(chan *types.RingHashSubmitMethodEvent)
	go func() {
		for {
			select {
			case event := <-submitRingMethodChan:
				if nil != event {
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
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			e := eventData.(*types.RingHashSubmitMethodEvent)
			submitRingMethodChan <- e
			return nil
		},
	}
	eventemitter.On(eventemitter.Miner_SubmitRingHash_Method, watcher)
	submitter.stopFuncs = append(submitter.stopFuncs, func() {
		close(submitRingMethodChan)
		eventemitter.Un(eventemitter.Miner_SubmitRingHash_Method, watcher)
	})
}

func (submitter *RingSubmitter) listenRegistryEvent() {
	registryChan := make(chan *types.RinghashSubmittedEvent)
	go func() {
		for {
			select {
			case event := <-registryChan:
				if nil != event {
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
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			e := eventData.(*types.RinghashSubmittedEvent)
			registryChan <- e
			return nil
		},
	}
	eventemitter.On(eventemitter.RingHashSubmitted, watcher)
	submitter.stopFuncs = append(submitter.stopFuncs, func() {
		close(registryChan)
		eventemitter.Un(eventemitter.RingHashSubmitted, watcher)
	})
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
	ringForSubmit.Miner = submitter.Miner.Address

	ringForSubmit.ProtocolAddress = protocolAddress
	ringForSubmit.OrdersCount = big.NewInt(int64(len(ringState.Orders)))
	ringForSubmit.Ringhash = ringState.Hash

	registryCost := big.NewInt(int64(0))

	if submitter.ifRegistryRingHash {
		ringhashRegistryAbi := submitter.Accessor.RinghashRegistryAbi
		ringhashRegistryAddress := implAddress.RinghashRegistryAddress
		ringForSubmit.RegistryData, err = ringhashRegistryAbi.Pack("submitRinghash",
			submitter.Miner.Address,
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

	ringSubmitArgs := ringState.GenerateSubmitArgs(submitter.Miner.Address, submitter.feeReceipt)
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
	c, _ := submitter.marketCapProvider.LegalCurrencyValueOfEth(cost)
	ringForSubmit.LegalCost = c
	received := new(big.Rat).Sub(ringState.LegalFee, c)
	ringForSubmit.Received = received

	log.Debugf("miner,submitter generate ring info, legal cost:%s, legalFee:%s, received:%s", ringForSubmit.LegalCost.FloatString(2), ringState.LegalFee.FloatString(2), received.FloatString(2))

	if received.Sign() <= 0 {
		// todo: warning
		//return nil, errors.New("received can't be less than 0")
	}
	return ringForSubmit, nil
}

func (submitter *RingSubmitter) stop() {
	for _, stop := range submitter.stopFuncs {
		stop()
	}
}

func (submitter *RingSubmitter) start() {
	submitter.listenNewRings()
	submitter.listenRegistryMethodEvent()
	submitter.listenBatchSubmitRingMethodEvent()
	submitter.listenSubmitRingMethodEvent()
	submitter.listenRegistryEvent()
}
