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
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

//保存ring，并将ring发送到区块链，同样需要分为待完成和已完成
type RingClient struct {
	Chainclient *chainclient.Client
	store       db.Database

	submitedRingsStore db.Database

	unSubmitedRingsStore db.Database

	fingerprintChan chan *chainclient.FingerprintEvent

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

func (ringClient *RingClient) NewRing(ring *types.RingState) {
	ringClient.mtx.Lock()
	defer ringClient.mtx.Unlock()

	if canSubmit(ring) {
		//todo:save
		if ringBytes, err := json.Marshal(ring); err == nil {
			ringClient.unSubmitedRingsStore.Put(ring.Hash.Bytes(), ringBytes)
			log.Infof("ringHash:%s", ring.Hash.Hex())
			//todo:async send to block chain
			ringClient.sendRingFingerprint(ring)
		} else {
			log.Errorf("error:%s", err.Error())
		}
	}
}

func canSubmit(ring *types.RingState) bool {
	//todo:args validator
	return true
}

//send Fingerprint to block chain
func (ringClient *RingClient) sendRingFingerprint(ring *types.RingState) {
	//contractAddress := ring.RawRing.Orders[0].OrderState.RawOrder.Protocol
	//_, err := loopring.LoopringFingerprints[contractAddress].SubmitRingFingerprint.SendTransaction("",nil,nil,"")
	//if err != nil {
	//	println(err.Error())
	//}
}

//listen fingerprint  accept by chain and then send Ring to block chain
func (ringClient *RingClient) listenFingerprintSucessAndSendRing() {
	var filterId string
	addresses := []common.Address{}
	for _, fingerprint := range Loopring.LoopringFingerprints {
		addresses = append(addresses, common.HexToAddress(fingerprint.Address))
	}
	filterReq := &eth.FilterQuery{}
	filterReq.Address = addresses
	filterReq.FromBlock = "latest"
	filterReq.ToBlock = "latest"
	//todo:topics, eventId
	//filterReq.Topics =
	if err := Loopring.Client.NewFilter(&filterId, filterReq); nil != err {
		log.Errorf("error:%s", err.Error())
	} else {
		log.Infof("filterId:%s", filterId)
	}
	//todo：Uninstall this filterId when stop
	defer func() {
		var a string
		Loopring.Client.UninstallFilter(&a, filterId)
	}()

	logChan := make(chan []eth.Log)
	if err := Loopring.Client.Subscribe(&logChan, filterId); nil != err {
		log.Errorf("error:%s", err.Error())
	} else {
		for {
			select {
			case logs := <-logChan:
				for _, log1 := range logs {
					ringHash := []byte(log1.TransactionHash)
					if _, err := ringClient.store.Get(ringHash); err == nil {
						ring := &types.RingState{}
						contractAddress := ring.RawRing.Orders[0].OrderState.RawOrder.Protocol
						//todo:发送到区块链
						_, err1 := Loopring.LoopringImpls[contractAddress].SubmitRing.SendTransactionWithSpecificGas("", nil, nil, "")
						if err1 != nil {
							log.Errorf("error:%s", err1.Error())
						} else {
							//标记为已删除,迁移到已完成的列表中
							ringClient.unSubmitedRingsStore.Delete(ringHash)
							//submitedRingsStore.Put(ringHash, ring.MarshalJSON())
						}
					} else {
						log.Errorf("error:%s", err.Error())
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

//recover after restart
func (ringClient *RingClient) recoverRing() {

	//todo: Traversal the uncompelete rings
	iterator := ringClient.unSubmitedRingsStore.NewIterator(nil, nil)
	for iterator.Next() {
		dataBytes := iterator.Value()
		ring := &types.RingState{}
		if err := json.Unmarshal(dataBytes, ring); nil != err {
			log.Errorf("error:%s", err.Error())
		} else {
			contractAddress := ring.RawRing.Orders[0].OrderState.RawOrder.Protocol
			var isSubmitFingerprint bool
			var isSubmitRing bool
			if canSubmit(ring) {
				if err := Loopring.LoopringFingerprints[contractAddress].FingerprintFound.Call(&isSubmitFingerprint, "", ""); err == nil {
					if isSubmitFingerprint {
						//todo:sendTransaction, check have ring been submited.
						if err := Loopring.LoopringImpls[contractAddress].SettleRing.Call(&isSubmitRing, "", ""); err == nil {
							if !isSubmitRing && canSubmit(ring) {
								//loopring.LoopringImpls[contractAddress].SubmitRing.SendTransaction(contractAddress, "", "")
							}
						} else {
							log.Errorf("error:%s", err.Error())
						}
					} else {
						ringClient.sendRingFingerprint(ring)
					}
				} else {
					log.Errorf("error:%s", err.Error())
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

	ringClient.recoverRing()
	//go listenFingerprintSucessAndSendRing();

}
