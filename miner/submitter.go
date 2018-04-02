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

	"encoding/json"
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
	"strings"
)

//保存ring，并将ring发送到区块链，同样需要分为待完成和已完成
type RingSubmitter struct {
	minerAccountForSign accounts.Account
	minerNameInfos      map[common.Address][]*types.NameRegistryInfo

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

func NewSubmitter(options config.MinerOptions, dbService dao.RdsService, marketCapProvider marketcap.MarketCapProvider) (*RingSubmitter, error) {
	submitter := &RingSubmitter{}
	submitter.maxGasLimit = big.NewInt(options.MaxGasLimit)
	submitter.minGasLimit = big.NewInt(options.MinGasLimit)
	for _, addr := range options.NormalMiners {
		var nonce types.Big
		normalAddr := common.HexToAddress(addr.Address)
		if err := ethaccessor.GetTransactionCount(&nonce, normalAddr, "pending"); nil != err {
			log.Errorf("err:%s", err.Error())
		}
		miner := &NormalMinerAddress{}
		miner.Address = normalAddr
		miner.GasPriceLimit = big.NewInt(addr.GasPriceLimit)
		miner.MaxPendingCount = addr.MaxPendingCount
		miner.MaxPendingTtl = addr.MaxPendingTtl
		miner.Nonce = nonce.BigInt()
		submitter.normalMinerAddresses = append(submitter.normalMinerAddresses, miner)
	}

	for _, addr := range options.PercentMiners {
		var nonce types.Big
		normalAddr := common.HexToAddress(addr.Address)
		if err := ethaccessor.GetTransactionCount(&nonce, normalAddr, "pending"); nil != err {
			log.Errorf("err:%s", err.Error())
		}
		miner := &SplitMinerAddress{}
		miner.Nonce = nonce.BigInt()
		miner.Address = normalAddr
		miner.FeePercent = addr.FeePercent
		miner.StartFee = addr.StartFee
		submitter.percentMinerAddresses = append(submitter.percentMinerAddresses, miner)
	}

	submitter.dbService = dbService
	submitter.marketCapProvider = marketCapProvider

	submitter.minerNameInfos = make(map[common.Address][]*types.NameRegistryInfo)
	//获取signer与feerecipient
	for addr, protocolAddr := range ethaccessor.ProtocolAddresses() {
		var resHex string
		callMethod := ethaccessor.ContractCallMethod(ethaccessor.NameRegistryAbi(), protocolAddr.NameRegistryAddress)
		err := callMethod(&resHex, "getParticipantIds", "latest", options.Name, big.NewInt(int64(0)), big.NewInt(int64(1000)))
		if nil != err {
			return nil, err
		} else {
			participantIds := []*big.Int{}
			err1 := ethaccessor.NameRegistryAbi().Unpack(&participantIds, "getParticipantIds", common.Hex2Bytes(strings.TrimPrefix(resHex, "0x")), 1)
			if nil != err1 {
				return nil, err1
			} else if len(participantIds) <= 0 {
				return nil, errors.New("miner hasn't been registerd. you can use `relay nameRegistry` to register it first.")
			}
			nameInfos := []*types.NameRegistryInfo{}
			for _, id := range participantIds {
				var nameRegistryHex string
				err := callMethod(&nameRegistryHex, "getParticipantById", "latest", id)

				if nil == err {
					nameInfo := &types.NameRegistryInfo{}
					err2 := ethaccessor.NameRegistryAbi().Unpack(nameInfo, "getParticipantById", common.Hex2Bytes(strings.TrimPrefix(nameRegistryHex, "0x")), 1)
					if nil == err2 {
						nameInfo.ParticipantId = new(big.Int)
						nameInfo.ParticipantId.Set(id)
						nameInfos = append(nameInfos, nameInfo)

						//if crypto.IsKSAccountUnlocked(nameInfo.Signer) {
						//	nameInfos = append(nameInfos, nameInfo)
						//} else {
						//	log.Errorf("the signer address: %s participantId: %s hasn't been unlocked.", nameInfo.Signer.Hex(), id.String())
						//}
					} else {
						log.Errorf("init submitter----unpack method:getParticipantById error:%s", err.Error())
					}
				} else {
					log.Errorf("init submitter error:%s", err.Error())
				}
			}
			if len(nameInfos) <= 0 {
				return nil, errors.New("there isn't useable nameinfo.")
			} else {
				submitter.minerNameInfos[addr] = nameInfos
			}
		}
	}

	submitter.stopFuncs = []func(){}
	return submitter, nil
}

func (submitter *RingSubmitter) listenNewRings() {
	ringSubmitInfoChan := make(chan []*types.RingSubmitInfo)
	go func() {
		for {
			select {
			case ringInfos := <-ringSubmitInfoChan:
				if nil != ringInfos {
					for _, ringState := range ringInfos {
						txHash, err := submitter.submitRing(ringState)
						ringState.SubmitTxHash = txHash

						daoInfo := &dao.RingSubmitInfo{}
						daoInfo.ConvertDown(ringState, err)
						if err := submitter.dbService.Add(daoInfo); nil != err {
							log.Errorf("Miner submitter,insert new ring err:%s", err.Error())
						} else {
							for _, filledOrder := range ringState.RawRing.Orders {
								daoOrder := &dao.FilledOrder{}
								daoOrder.ConvertDown(filledOrder, ringState.Ringhash)
								if err1 := submitter.dbService.Add(daoOrder); nil != err1 {
									log.Errorf("Miner submitter,insert filled Order err:%s", err1.Error())
								}
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

func (submitter *RingSubmitter) submitRing(ringSubmitInfo *types.RingSubmitInfo) (common.Hash, error) {
	status := types.TX_STATUS_SUCCESS
	ordersStr, _ := json.Marshal(ringSubmitInfo.RawRing.Orders)
	log.Debugf("submitring hash:%s, orders:%s", ringSubmitInfo.Ringhash.Hex(), string(ordersStr))

	txHashStr, err := ethaccessor.SignAndSendTransaction(ringSubmitInfo.Miner, ringSubmitInfo.ProtocolAddress, ringSubmitInfo.ProtocolGas, ringSubmitInfo.ProtocolGasPrice, nil, ringSubmitInfo.ProtocolData)
	if nil != err {
		log.Errorf("submitring hash:%s, err:%s", ringSubmitInfo.Ringhash.Hex(), err.Error())
		status = types.TX_STATUS_FAILED
	}
	txHash := common.HexToHash(txHashStr)
	submitter.submitResult(ringSubmitInfo.Ringhash, txHash, uint8(status), big.NewInt(0), big.NewInt(0), big.NewInt(0), err)
	return txHash, err
}

func (submitter *RingSubmitter) listenSubmitRingMethodEvent() {
	submitRingMethodChan := make(chan *types.RingMinedEvent)
	go func() {
		for {
			select {
			case event := <-submitRingMethodChan:
				if nil != event {

					if ringhashes, err := submitter.dbService.GetRingHashesByTxHash(event.TxHash); nil != err {
						log.Errorf("err:%s", err.Error())
					} else {
						var err1 error
						if event.Status == types.TX_STATUS_FAILED {
							err1 = errors.New("failed to execute ring")
						} else {
							err1 = errors.New("")
						}

						for _, ringhash := range ringhashes {
							submitter.submitResult(ringhash, event.TxHash, event.Status, big.NewInt(0), event.BlockNumber, event.GasUsed, err1)
						}
					}
					//}
				}
			}
		}
	}()

	watcher := &eventemitter.Watcher{
		Concurrent: false,
		Handle: func(eventData eventemitter.EventData) error {
			e := eventData.(*types.RingMinedEvent)
			submitRingMethodChan <- e
			return nil
		},
	}
	eventemitter.On(eventemitter.RingMined, watcher)
	submitter.stopFuncs = append(submitter.stopFuncs, func() {
		close(submitRingMethodChan)
		eventemitter.Un(eventemitter.RingMined, watcher)
	})
}

func (submitter *RingSubmitter) submitResult(ringhash, txhash common.Hash, status uint8, ringIndex, blockNumber, usedGas *big.Int, err error) {
	resultEvt := &types.RingSubmitResultEvent{
		RingHash:    ringhash,
		TxHash:      txhash,
		Status:      status,
		Err:         err,
		RingIndex:   ringIndex,
		BlockNumber: blockNumber,
		UsedGas:     usedGas,
	}
	if err := submitter.dbService.UpdateRingSubmitInfoResult(resultEvt); nil != err {
		log.Errorf("err:%s", err.Error())
	}
	eventemitter.Emit(eventemitter.Miner_RingSubmitResult, resultEvt)
}

////提交错误，执行错误
//func (submitter *RingSubmitter) submitFailed(ringhashes []common.Hash, err error) {
//	if err := submitter.dbService.UpdateRingSubmitInfoFailed(ringhashes, err.Error()); nil != err {
//		log.Errorf("err:%s", err.Error())
//	} else {
//		for _, ringhash := range ringhashes {
//			failedEvent := &types.RingSubmitResultEvent{RingHash: ringhash, Status:types.TX_STATUS_FAILED}
//			eventemitter.Emit(eventemitter.Miner_RingSubmitResult, failedEvent)
//		}
//	}
//}

func (submitter *RingSubmitter) GenerateRingSubmitInfo(ringState *types.Ring, gas, gasPrice *big.Int, received *big.Rat) (*types.RingSubmitInfo, error) {
	protocolAddress := ringState.Orders[0].OrderState.RawOrder.Protocol
	var (
		signer *types.NameRegistryInfo
		err    error
	)

	ringSubmitInfo := &types.RingSubmitInfo{RawRing: ringState, Received: received, ProtocolGasPrice: gasPrice, ProtocolGas: gas}
	if types.IsZeroHash(ringState.Hash) {
		if signers, exists := submitter.minerNameInfos[protocolAddress]; exists {
			if len(signers) > 0 {
				//todo:use the first one
				signer = signers[0]
				ringState.Hash = ringState.GenerateHash(signer)
			} else {
				return nil, errors.New("err:there isn't a address to sign")
			}
		} else {
			return nil, errors.New("err:there isn't a address to sign.")
		}
	}

	ringSubmitInfo.ProtocolAddress = protocolAddress
	ringSubmitInfo.OrdersCount = big.NewInt(int64(len(ringState.Orders)))
	ringSubmitInfo.Ringhash = ringState.Hash

	protocolAbi := ethaccessor.ProtocolImplAbi()
	if ringSubmitArgs, err1 := ringState.GenerateSubmitArgs(signer); nil != err1 {
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
			uint16(ringSubmitArgs.FeeSelections.Uint64()),
		)
	}
	if nil != err {
		return nil, err
	}
	//预先判断是否会提交成功
	//ringSubmitInfo.ProtocolGas, ringSubmitInfo.ProtocolGasPrice, err = ethaccessor.EstimateGas(ringSubmitInfo.ProtocolData, protocolAddress, "latest")
	submitter.computeReceivedAndSelectMiner(ringSubmitInfo)

	if nil != err {
		return nil, err
	}
	if submitter.maxGasLimit.Sign() > 0 && ringSubmitInfo.ProtocolGas.Cmp(submitter.maxGasLimit) > 0 {
		ringSubmitInfo.ProtocolGas.Set(submitter.maxGasLimit)
	}
	if submitter.minGasLimit.Sign() > 0 && ringSubmitInfo.ProtocolGas.Cmp(submitter.minGasLimit) < 0 {
		ringSubmitInfo.ProtocolGas.Set(submitter.minGasLimit)
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
	submitter.listenSubmitRingMethodEvent()
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
	protocolCost := new(big.Int).Mul(ringSubmitInfo.ProtocolGas, ringSubmitInfo.ProtocolGasPrice)

	costEth := new(big.Rat).SetInt(protocolCost)
	costLegal, _ := submitter.marketCapProvider.LegalCurrencyValueOfEth(costEth)
	ringSubmitInfo.LegalCost = costLegal
	received := new(big.Rat).Sub(ringState.LegalFee, costLegal)
	ringSubmitInfo.Received = received

	return nil
}

func (submitter *RingSubmitter) SetMatcher(matcher Matcher) {
	submitter.matcher = matcher
}

// TODO(hongyu): 初始化submmitter之前需要确认是否nameRegistry&addParticipant
