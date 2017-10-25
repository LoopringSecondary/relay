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
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

func (ringClient *RingClient) NewRing(ringState *types.RingState) {
	ringClient.mtx.Lock()
	defer ringClient.mtx.Unlock()

	if canSubmit(ringState) {
		if ringBytes, err := json.Marshal(ringState); err == nil {
			ringClient.unSubmitedRingsStore.Put(ringState.RawRing.Hash.Bytes(), ringBytes)
			ringData, _ := json.Marshal(ringState)
			log.Debugf("ringState:%s", string(ringData))
			if IfRegistryRingHash {
				ringClient.sendRinghashRegistry(ringState)
			} else {
				ringClient.submitRing(ringState)
			}
		} else {
			log.Errorf("error:%s", err.Error())
		}
	} else {
		//todo:save to store
	}
}

func isOrdersRemined(ring *types.RingState) bool {
	//todo:args validator
	return true
}
func canSubmit(ring *types.RingState) bool {
	return true
}

//send Fingerprint to block chain
func (ringClient *RingClient) sendRinghashRegistry(ringState *types.RingState) {
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
		log.Error(err.Error())
	} else {
		ringState.RegistryTxHash = types.HexToHash(txHash)
		if ringBytes, err := json.Marshal(ringState); nil != err {
			log.Error(err.Error())
		} else {
			ringClient.unSubmitedRingsStore.Put(ringState.RawRing.Hash.Bytes(), ringBytes)
		}
	}
}

//listen ringhash  accept by chain and then send Ring to block chain
func (ringClient *RingClient) listenRinghashRegistrySucessAndSendRing() {
	var filterId string
	filterReq := &eth.FilterQuery{}
	filterReq.Address = []types.Address{}
	filterReq.FromBlock = "latest"
	filterReq.ToBlock = "latest"
	topics := []types.Hash{}
	//todo:topics, eventId
	//todo:Registry，没有事件发生，无法判断执行情况
	for _, impl := range LoopringInstance.LoopringImpls {
		filterReq.Address = append(filterReq.Address, impl.RingHashRegistry.Address)
		topics = append(topics, types.HexToHash(impl.RingHashRegistry.SubmitRinghash.MethodId()))
	}

	if err := LoopringInstance.Client.NewFilter(&filterId, filterReq); nil != err {
		log.Errorf("error:%s", err.Error())
	} else {
		log.Debugf("ringhashRegistry filterId:%s", filterId)
	}

	//todo：Uninstall this filterId when stop
	defer func() {
		var a string
		LoopringInstance.Client.UninstallFilter(&a, filterId)
	}()

	logChan := make(chan []eth.Log)
	if err := LoopringInstance.Client.Subscribe(&logChan, filterId); nil != err {
		log.Errorf("error:%s", err.Error())
	} else {
		for {
			select {
			case logs := <-logChan:
				for _, log1 := range logs {
					evt := chainclient.RinghashSubmitted{}
					address := types.HexToAddress(log1.Address)
					var ringHashRegistry *chainclient.LoopringRinghashRegistry
					for _, implTmp := range LoopringInstance.LoopringImpls {
						if implTmp.RingHashRegistry.Address == address {
							ringHashRegistry = implTmp.RingHashRegistry
						}
					}
					if nil == ringHashRegistry {
						log.Errorf("no such ringhashRegistry with address :%s", log1.Address)
					}
					data := hexutil.MustDecode(log1.Data)
					if err := ringHashRegistry.RinghashSubmittedEvent.Unpack(&evt, data, log1.Topics); err != nil {
						log.Errorf("err :%s", err.Error())
					}
					ringHash := types.BytesToHash(evt.RingHash)
					log.Debugf("ringState txHash:%s, eventHash:%s", log1.TransactionHash, ringHash.Hex())
					ringData, _ := ringClient.unSubmitedRingsStore.Get(ringHash.Bytes())
					if nil != ringData {
						ring := &types.RingState{}
						if err := json.Unmarshal(ringData, ring); nil != err {
							log.Errorf("error:%s", err.Error())
						} else {
							log.Debugf("ringhashRegistry:%s", string(ringData))
							ringClient.submitRing(ring)
						}
					}
				}
			case stop := <-ringClient.stopChan:
				if stop {
					break
				}
			}
		}
	}
}

func (ringClient *RingClient) submitRing(ringSate *types.RingState) {
	ring := ringSate.RawRing
	contractAddress := ring.Orders[0].OrderState.RawOrder.Protocol
	ringSubmitArgs := ring.GenerateSubmitArgs(MinerPrivateKey)
	if txHash, err1 := LoopringInstance.LoopringImpls[contractAddress].SubmitRing.SendTransaction(types.HexToAddress("0x"),
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
	); nil != err1 {
		log.Errorf("error:%s", err1.Error())
	} else {
		//标记为已删除,迁移到已完成的列表中
		ringClient.unSubmitedRingsStore.Delete(ring.Hash.Bytes())
		ringSate.SubmitTxHash = types.HexToHash(txHash)
		if data, err := json.Marshal(ringSate); nil != err {
			log.Error(err.Error())
		} else {
			ringClient.submitedRingsStore.Put(ring.Hash.Bytes(), data)
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

func (ringClient *RingClient) Start() {
	//ringClient.recoverRing()
	if IfRegistryRingHash {
		go ringClient.listenRinghashRegistrySucessAndSendRing()
	}
}
