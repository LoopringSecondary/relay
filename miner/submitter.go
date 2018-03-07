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
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
)

//保存ring，并将ring发送到区块链，同样需要分为待完成和已完成
type RingSubmitter struct {
	minerAccountForSign accounts.Account
	miner               *types.NameRegistryInfo

	maxGasLimit *big.Int
	minGasLimit *big.Int

	normalMinerAddresses  []*NormalMinerAddress
	percentMinerAddresses []*SplitMinerAddress

	dbService         dao.RdsService
	marketCapProvider marketcap.MarketCapProvider
	matcher           Matcher

	stopFuncs []func()
}

type RingSubmitFailed struct {
	RingState *types.Ring
	err       error
}

func NewSubmitter(options config.MinerOptions, dbService dao.RdsService, marketCapProvider marketcap.MarketCapProvider) *RingSubmitter {
	submitter := &RingSubmitter{}
	submitter.maxGasLimit = big.NewInt(options.MaxGasLimit)
	submitter.minGasLimit = big.NewInt(options.MinGasLimit)
	for _, addr := range options.NormalMiners {
		//var nonce types.Big
		//if err := accessor.Call(&nonce, "eth_getTransactionCount", addr.Address, "pending"); nil != err {
		//	log.Errorf("err:%s", err.Error())
		//}
		miner := &NormalMinerAddress{}
		miner.Address = common.HexToAddress(addr.Address)
		miner.GasPriceLimit = big.NewInt(addr.GasPriceLimit)
		miner.MaxPendingCount = addr.MaxPendingCount
		miner.MaxPendingTtl = addr.MaxPendingTtl
		//miner.Nonce = nonce.BigInt()
		submitter.normalMinerAddresses = append(submitter.normalMinerAddresses, miner)
	}

	for _, addr := range options.PercentMiners {
		//var nonce types.Big
		//if err := accessor.Call(&nonce, "eth_getTransactionCount", addr.Address, "pending"); nil != err {
		//	log.Errorf("err:%s", err.Error())
		//}
		miner := &SplitMinerAddress{}
		//miner.Nonce = nonce.BigInt()
		miner.Address = common.HexToAddress(addr.Address)
		miner.FeePercent = addr.FeePercent
		miner.StartFee = addr.StartFee
		submitter.percentMinerAddresses = append(submitter.percentMinerAddresses, miner)
	}

	submitter.dbService = dbService
	submitter.marketCapProvider = marketCapProvider

	//todo:需要更改为nameregistryinfo的类型
	//if len(options.NormalMiners) > 0 {
	//	submitter.minerAccountForSign = accounts.Account{Address: common.HexToAddress(options.NormalMiners[0].Address)}
	//} else {
	//	submitter.minerAccountForSign = accounts.Account{Address: common.HexToAddress(options.PercentMiners[0].Address)}
	//}

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

					//if submitter.ifRegistryRingHash {
					//	if len(ringInfos) == 1 {
					//		if err := submitter.ringhashRegistry(ringInfos[0]); nil != err {
					//			submitter.dbService.UpdateRingSubmitInfoFailed([]common.Hash{ringInfos[0].Ringhash}, err.Error())
					//		}
					//	} else {
					//		infosMap := make(map[common.Address][]*types.RingSubmitInfo)
					//		for _, info := range ringInfos {
					//			if _, ok := infosMap[info.ProtocolAddress]; !ok {
					//				infosMap[info.ProtocolAddress] = []*types.RingSubmitInfo{}
					//			}
					//			infosMap[info.ProtocolAddress] = append(infosMap[info.ProtocolAddress], info)
					//		}
					//		for protocolAddr, infos := range infosMap {
					//			ringhashes := []common.Hash{}
					//			miners := []common.Address{}
					//			for _, info := range infos {
					//				miners = append(miners, info.Miner)
					//				ringhashes = append(ringhashes, info.Ringhash)
					//			}
					//			if err := submitter.batchRinghashRegistry(protocolAddr, ringhashes, miners); nil != err {
					//				submitter.dbService.UpdateRingSubmitInfoFailed(ringhashes, err.Error())
					//			}
					//		}
					//	}
					//} else {
					for _, ringState := range ringInfos {
						submitter.submitRing(ringState)
					}
					//}
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

//func (submitter *RingSubmitter) batchRinghashRegistry(contractAddress common.Address, ringhashes []common.Hash, miners []common.Address) error {
//	ringhashRegistryAbi := ethaccessor.RinghashRegistryAbi()
//	var ringhashRegistryAddress common.Address
//	if implAddress, exists := ethaccessor.ProtocolAddresses()[contractAddress]; !exists {
//		return errors.New("does't contain this version")
//	} else {
//		ringhashRegistryAddress = implAddress.RinghashRegistryAddress
//	}
//	if registryData, err := ringhashRegistryAbi.Pack("batchSubmitRinghash",
//		miners,
//		ringhashes); nil != err {
//		return err
//	} else {
//		if gas, gasPrice, err := ethaccessor.EstimateGas(registryData, ringhashRegistryAddress, "latest"); nil != err {
//			return err
//		} else {
//			if txHash, err := ethaccessor.SignAndSendTransaction(miners[0], ringhashRegistryAddress, gas, gasPrice, nil, registryData); nil != err {
//				return err
//			} else {
//				submitter.dbService.UpdateRingSubmitInfoRegistryTxHash(ringhashes, txHash)
//			}
//		}
//	}
//	return nil
//}

//func (submitter *RingSubmitter) ringhashRegistry(ringSubmitInfo *types.RingSubmitInfo) error {
//	contractAddress := ringSubmitInfo.ProtocolAddress
//	var ringhashRegistryAddress common.Address
//	if implAddress, exists := ethaccessor.ProtocolAddresses()[contractAddress]; !exists {
//		return errors.New("does't contains this version")
//	} else {
//		ringhashRegistryAddress = implAddress.RinghashRegistryAddress
//	}
//
//	if txHash, err := ethaccessor.SignAndSendTransaction(ringSubmitInfo.Miner, ringhashRegistryAddress, ringSubmitInfo.RegistryGas, ringSubmitInfo.RegistryGasPrice, nil, ringSubmitInfo.RegistryData); nil != err {
//		return err
//	} else {
//		ringSubmitInfo.RegistryTxHash = common.HexToHash(txHash)
//		submitter.dbService.UpdateRingSubmitInfoRegistryTxHash([]common.Hash{ringSubmitInfo.Ringhash}, txHash)
//	}
//	return nil
//}

func (submitter *RingSubmitter) submitRing(ringSubmitInfo *types.RingSubmitInfo) error {
	if txHash, err := ethaccessor.SignAndSendTransaction(ringSubmitInfo.Miner, ringSubmitInfo.ProtocolAddress, ringSubmitInfo.ProtocolGas, ringSubmitInfo.ProtocolGasPrice, nil, ringSubmitInfo.ProtocolData); nil != err {
		submitter.submitFailed([]common.Hash{ringSubmitInfo.Ringhash}, err)
		return err
	} else {
		ringSubmitInfo.SubmitTxHash = common.HexToHash(txHash)
		submitter.dbService.UpdateRingSubmitInfoProtocolTxHash(ringSubmitInfo.Ringhash, txHash)
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

//func (submitter *RingSubmitter) listenBatchSubmitRingMethodEvent() {
//	submitRingMethodChan := make(chan *types.BatchSubmitRingHashMethodEvent)
//	go func() {
//		for {
//			select {
//			case event := <-submitRingMethodChan:
//				if nil != event {
//					if nil != event.Err {
//						if ringhashes, err := submitter.dbService.GetRingHashesByTxHash(event.TxHash); nil != err {
//							log.Errorf("err:%s", err.Error())
//						} else {
//							submitter.submitFailed(ringhashes, errors.New("failed to execute ring"))
//						}
//					}
//					submitter.dbService.UpdateRingSubmitInfoRegistryUsedGas(event.TxHash.Hex(), event.UsedGas)
//				}
//			}
//		}
//	}()
//
//	watcher := &eventemitter.Watcher{
//		Concurrent: false,
//		Handle: func(eventData eventemitter.EventData) error {
//			e := eventData.(*types.BatchSubmitRingHashMethodEvent)
//			submitRingMethodChan <- e
//			return nil
//		},
//	}
//	eventemitter.On(eventemitter.Miner_BatchSubmitRingHash_Method, watcher)
//	submitter.stopFuncs = append(submitter.stopFuncs, func() {
//		close(submitRingMethodChan)
//		eventemitter.Un(eventemitter.Miner_BatchSubmitRingHash_Method, watcher)
//	})
//}

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

//func (submitter *RingSubmitter) listenRegistryMethodEvent() {
//	submitRingMethodChan := make(chan *types.RingHashSubmitMethodEvent)
//	go func() {
//		for {
//			select {
//			case event := <-submitRingMethodChan:
//				if nil != event {
//					if nil != event.Err {
//						if ringhashes, err := submitter.dbService.GetRingHashesByTxHash(event.TxHash); nil != err {
//							log.Errorf("err:%s", err.Error())
//						} else {
//							submitter.submitFailed(ringhashes, errors.New("failed to execute ringhash registry:"+event.Err.Error()))
//						}
//					} else {
//						submitter.dbService.UpdateRingSubmitInfoRegistryUsedGas(event.TxHash.Hex(), event.UsedGas)
//					}
//				}
//			}
//		}
//	}()
//
//	watcher := &eventemitter.Watcher{
//		Concurrent: false,
//		Handle: func(eventData eventemitter.EventData) error {
//			e := eventData.(*types.RingHashSubmitMethodEvent)
//			submitRingMethodChan <- e
//			return nil
//		},
//	}
//	eventemitter.On(eventemitter.Miner_SubmitRingHash_Method, watcher)
//	submitter.stopFuncs = append(submitter.stopFuncs, func() {
//		close(submitRingMethodChan)
//		eventemitter.Un(eventemitter.Miner_SubmitRingHash_Method, watcher)
//	})
//}
//
//func (submitter *RingSubmitter) listenRegistryEvent() {
//	registryChan := make(chan *types.RinghashSubmittedEvent)
//	go func() {
//		for {
//			select {
//			case event := <-registryChan:
//				if nil != event {
//					var (
//						err         error
//						implAddress *ethaccessor.ProtocolAddress
//						exists      bool
//					)
//					info := &types.RingSubmitInfo{}
//					daoInfo, _ := submitter.dbService.GetRingForSubmitByHash(event.RingHash)
//					daoInfo.ConvertUp(info)
//					if types.IsZeroHash(info.Ringhash) {
//						err = errors.New("ring hash is zero")
//					} else {
//						if implAddress, exists = ethaccessor.ProtocolAddresses()[info.ProtocolAddress]; !exists {
//							err = errors.New("doesn't contain this version of protocol:" + info.ProtocolAddress.Hex())
//						}
//						var canSubmit bool
//						canSubmit, err = ethaccessor.ProtocolCanSubmit(implAddress, info.Ringhash, info.Miner)
//						if nil != err {
//							log.Errorf("err:%s", err.Error())
//						} else {
//							if !canSubmit {
//								err = errors.New("failed to call method:canSubmit")
//							}
//						}
//					}
//
//					if nil == err {
//						submitter.submitRing(info)
//					}
//				}
//			}
//		}
//	}()
//
//	watcher := &eventemitter.Watcher{
//		Concurrent: false,
//		Handle: func(eventData eventemitter.EventData) error {
//			e := eventData.(*types.RinghashSubmittedEvent)
//			registryChan <- e
//			return nil
//		},
//	}
//	eventemitter.On(eventemitter.RingHashSubmitted, watcher)
//	submitter.stopFuncs = append(submitter.stopFuncs, func() {
//		close(registryChan)
//		eventemitter.Un(eventemitter.RingHashSubmitted, watcher)
//	})
//}

func (submitter *RingSubmitter) GenerateRingSubmitInfo(ringState *types.Ring) (*types.RingSubmitInfo, error) {
	protocolAddress := ringState.Orders[0].OrderState.RawOrder.Protocol
	var (
		//implAddress *ethaccessor.ProtocolAddress
		//exists      bool
		err error
	)
	//if implAddress, exists = ethaccessor.ProtocolAddresses()[protocolAddress]; !exists {
	//	return nil, errors.New("doesn't contain this version of protocol:" + protocolAddress.Hex())
	//}
	protocolAbi := ethaccessor.ProtocolImplAbi()

	ringSubmitInfo := &types.RingSubmitInfo{RawRing: ringState}
	if types.IsZeroHash(ringState.Hash) {
		ringState.Hash = ringState.GenerateHash(submitter.miner)
	}

	ringSubmitInfo.ProtocolAddress = protocolAddress
	ringSubmitInfo.OrdersCount = big.NewInt(int64(len(ringState.Orders)))
	ringSubmitInfo.Ringhash = ringState.Hash

	//if submitter.ifRegistryRingHash {
	//	ringhashRegistryAbi := ethaccessor.RinghashRegistryAbi()
	//	ringhashRegistryAddress := implAddress.RinghashRegistryAddress
	//	ringSubmitInfo.RegistryData, err = ringhashRegistryAbi.Pack("submitRinghash",
	//		submitter.minerAccountForSign,
	//		ringSubmitInfo.Ringhash)
	//	if nil != err {
	//		return nil, err
	//	}
	//
	//	log.Debugf("ringhashRegistryAddress", ringhashRegistryAddress.Hex())
	//	ringSubmitInfo.RegistryGas, ringSubmitInfo.RegistryGasPrice, err = ethaccessor.EstimateGas(ringSubmitInfo.RegistryData, ringhashRegistryAddress, "latest")
	//	if nil != err {
	//		return nil, err
	//	}
	//	if submitter.maxGasLimit.Sign() > 0 && ringSubmitInfo.RegistryGas.Cmp(submitter.maxGasLimit) > 0 {
	//		ringSubmitInfo.RegistryGas.Set(submitter.maxGasLimit)
	//	}
	//	if submitter.minGasLimit.Sign() > 0 && ringSubmitInfo.RegistryGas.Cmp(submitter.minGasLimit) < 0 {
	//		ringSubmitInfo.RegistryGas.Set(submitter.minGasLimit)
	//	}
	//
	//	ringSubmitInfo.RegistryGas.Add(ringSubmitInfo.RegistryGas, big.NewInt(1000))
	//}

	if ringSubmitArgs, err1 := ringState.GenerateSubmitArgs(submitter.miner); nil != err1 {
		return nil, err1
	} else {
		ringSubmitInfo.ProtocolData, err = protocolAbi.Pack("submitRing",
			ringSubmitArgs.AddressList,
			ringSubmitArgs.UintArgsList,
			ringSubmitArgs.Uint8ArgsList,
			ringSubmitArgs.BuyNoMoreThanAmountBList,
			ringSubmitArgs.VList,
			ringSubmitArgs.RList,
			ringSubmitArgs.SList,
			ringSubmitArgs.Miner.ParticipantId,
			ringSubmitArgs.FeeSelections,
		)
	}
	if nil != err {
		return nil, err
	}
	ringSubmitInfo.ProtocolGas, ringSubmitInfo.ProtocolGasPrice, err = ethaccessor.EstimateGas(ringSubmitInfo.ProtocolData, protocolAddress, "latest")
	if nil != err {
		return nil, err
	}
	if submitter.maxGasLimit.Sign() > 0 && ringSubmitInfo.ProtocolGas.Cmp(submitter.maxGasLimit) > 0 {
		ringSubmitInfo.ProtocolGas.Set(submitter.maxGasLimit)
	}
	if submitter.minGasLimit.Sign() > 0 && ringSubmitInfo.ProtocolGas.Cmp(submitter.minGasLimit) < 0 {
		ringSubmitInfo.ProtocolGas.Set(submitter.minGasLimit)
	}

	ringSubmitInfo.ProtocolGas.Add(ringSubmitInfo.ProtocolGas, big.NewInt(1000))

	submitter.computeReceivedAndSelectMiner(ringSubmitInfo)
	log.Debugf("miner,submitter generate ring info, legal cost:%s, legalFee:%s, received:%s", ringSubmitInfo.LegalCost.FloatString(2), ringState.LegalFee.FloatString(2), ringSubmitInfo.Received.FloatString(2))

	if ringSubmitInfo.Received.Sign() <= 0 {
		// todo: warning
		//return nil, errors.New("received can't be less than 0")
	}
	return ringSubmitInfo, nil
}

func (submitter *RingSubmitter) stop() {
	for _, stop := range submitter.stopFuncs {
		stop()
	}
}

func (submitter *RingSubmitter) start() {
	submitter.listenNewRings()
	//submitter.listenRegistryMethodEvent()
	//submitter.listenBatchSubmitRingMethodEvent()
	submitter.listenSubmitRingMethodEvent()
	//submitter.listenRegistryEvent()
}

func (submitter *RingSubmitter) availabeMinerAddress() []*NormalMinerAddress {
	minerAddresses := []*NormalMinerAddress{}
	for _, minerAddress := range submitter.normalMinerAddresses {
		var blockedTxCount, txCount types.Big
		ethaccessor.GetTransactionCount(&blockedTxCount, minerAddress.Address, "latest")
		ethaccessor.GetTransactionCount(&txCount, minerAddress.Address, "pending")
		//submitter.Accessor.Call("latest", &blockedTxCount, "eth_getTransactionCount", minerAddress.Address.Hex(), "latest")
		//submitter.Accessor.Call("latest", &txCount, "eth_getTransactionCount", minerAddress.Address.Hex(), "pending")

		pendingCount := big.NewInt(int64(0))
		pendingCount.Sub(txCount.BigInt(), blockedTxCount.BigInt())
		if pendingCount.Int64() <= minerAddress.MaxPendingCount {
			minerAddresses = append(minerAddresses, minerAddress)
		}
	}

	if len(minerAddresses) <= 0 {
		minerAddresses = append(minerAddresses, submitter.normalMinerAddresses[0])
	}
	return minerAddresses
}

func (submitter *RingSubmitter) computeReceivedAndSelectMiner(ringSubmitInfo *types.RingSubmitInfo) error {
	ringState := ringSubmitInfo.RawRing
	ringState.LegalFee = new(big.Rat).SetInt(big.NewInt(int64(0)))
	ethPrice, _ := submitter.marketCapProvider.GetEthCap()
	ethPrice = ethPrice.Quo(ethPrice, new(big.Rat).SetInt(util.AllTokens["WETH"].Decimals))
	lrcAddress := ethaccessor.ProtocolAddresses()[ringState.Orders[0].OrderState.RawOrder.Protocol].LrcTokenAddress
	useSplit := false
	//for _,splitMiner := range submitter.splitMinerAddresses {
	//	//todo:optimize it
	//	if lrcFee > splitMiner.StartFee || splitFee > splitMiner.StartFee || len(submitter.normalMinerAddresses) <= 0  {
	//		useSplit = true
	//		ringState.Miner = splitMiner.Address
	//		minerLrcBalance, _ := submitter.matcher.GetAccountAvailableAmount(splitMiner.Address, lrcAddress)
	//		//the lrcreward should be send to order.owner when miner selects MarginSplit as the selection of fee
	//		//be careful！！！ miner will received nothing, if miner set FeeSelection=1 and he doesn't have enough lrc
	//
	//
	//		if ringState.LrcLegalFee.Cmp(ringState.SplitLegalFee) < 0 && minerLrcBalance.Cmp(filledOrder.LrcFee) > 0 {
	//			filledOrder.FeeSelection = 1
	//			splitPer := new(big.Rat).SetInt64(int64(filledOrder.OrderState.RawOrder.MarginSplitPercentage))
	//			legalAmountOfSaving.Mul(legalAmountOfSaving, splitPer)
	//			filledOrder.LrcReward = legalAmountOfLrc
	//			legalAmountOfSaving.Sub(legalAmountOfSaving, legalAmountOfLrc)
	//			filledOrder.LegalFee = legalAmountOfSaving
	//
	//			minerLrcBalance.Sub(minerLrcBalance, filledOrder.LrcFee)
	//			//log.Debugf("Miner,lrcReward:%s  legalFee:%s", lrcReward.FloatString(10), filledOrder.LegalFee.FloatString(10))
	//		} else {
	//			filledOrder.FeeSelection = 0
	//			filledOrder.LegalFee = legalAmountOfLrc
	//		}
	//
	//		ringState.LegalFee.Add(ringState.LegalFee, filledOrder.LegalFee)
	//	}
	//}
	minerAddresses := submitter.availabeMinerAddress()
	if !useSplit {
		for _, normalMinerAddress := range minerAddresses {
			minerLrcBalance, _ := submitter.matcher.GetAccountAvailableAmount(normalMinerAddress.Address, lrcAddress)

			legalFee := new(big.Rat).SetInt(big.NewInt(int64(0)))
			feeSelections := []uint8{}
			legalFees := []*big.Rat{}
			lrcRewards := []*big.Rat{}
			for _, filledOrder := range ringState.Orders {
				lrcFee := new(big.Rat).SetInt(big.NewInt(int64(2)))
				lrcFee.Mul(lrcFee, filledOrder.LegalLrcFee)
				if lrcFee.Cmp(filledOrder.LegalFeeS) < 0 && minerLrcBalance.Cmp(filledOrder.LrcFee) > 0 {
					feeSelections = append(feeSelections, 1)
					fee := new(big.Rat).Set(filledOrder.LegalFeeS)
					fee.Sub(fee, filledOrder.LegalLrcFee)
					legalFees = append(legalFees, fee)
					lrcRewards = append(lrcRewards, filledOrder.LegalLrcFee)
					legalFee.Add(legalFee, fee)

					minerLrcBalance.Sub(minerLrcBalance, filledOrder.LrcFee)
					//log.Debugf("Miner,lrcReward:%s  legalFee:%s", lrcReward.FloatString(10), filledOrder.LegalFee.FloatString(10))
				} else {
					feeSelections = append(feeSelections, 0)
					legalFees = append(legalFees, filledOrder.LegalLrcFee)
					lrcRewards = append(lrcRewards, new(big.Rat).SetInt(big.NewInt(int64(0))))
					legalFee.Add(legalFee, filledOrder.LegalLrcFee)
				}
			}

			if ringState.LegalFee.Sign() == 0 || ringState.LegalFee.Cmp(legalFee) < 0 {
				ringState.LegalFee = legalFee
				ringSubmitInfo.Miner = normalMinerAddress.Address
				for idx, filledOrder := range ringState.Orders {
					filledOrder.FeeSelection = feeSelections[idx]
					filledOrder.LegalFee = legalFees[idx]
					filledOrder.LrcReward = lrcRewards[idx]
				}

				if nil == ringSubmitInfo.ProtocolGasPrice || ringSubmitInfo.ProtocolGasPrice.Cmp(normalMinerAddress.GasPriceLimit) > 0 {
					ringSubmitInfo.ProtocolGasPrice = normalMinerAddress.GasPriceLimit
				}
			}
		}
	}

	registryCost := big.NewInt(int64(0))
	//if submitter.ifRegistryRingHash {
	//	ringSubmitInfo.RegistryGasPrice = ringSubmitInfo.ProtocolGasPrice
	//	registryCost.Mul(ringSubmitInfo.RegistryGas, ringSubmitInfo.RegistryGasPrice)
	//}

	protocolCost := new(big.Int).Mul(ringSubmitInfo.ProtocolGas, ringSubmitInfo.ProtocolGasPrice)

	costEth := new(big.Rat).SetInt(new(big.Int).Add(protocolCost, registryCost))
	costLegal, _ := submitter.marketCapProvider.LegalCurrencyValueOfEth(costEth)
	ringSubmitInfo.LegalCost = costLegal
	received := new(big.Rat).Sub(ringState.LegalFee, costLegal)
	ringSubmitInfo.Received = received

	return nil
}

func (submitter *RingSubmitter) SetMatcher(matcher Matcher) {
	submitter.matcher = matcher
}
