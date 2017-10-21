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

package eth

import (
	"errors"
	"fmt"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"math/big"
)

type forkDetect struct {
	observers     []chan chainclient.ForkedEvent
	hashStore     db.Database
	parentHash    types.Hash
	startedNumber *big.Int
}

//fork detect
func (ethClient *EthClient) startForkDetect(observers []chan chainclient.ForkedEvent, database db.Database) error {
	if nil == observers || len(observers) == 0 {
		panic(errors.New("observers can't be empty when start ForkDetect"))
	} else {
		detectedEventChan := make(chan chainclient.ForkedEvent)
		observers = append(observers, detectedEventChan)
		go func() {
		L:
			go ethClient.forkDetect(observers, database)
			for {
				select {
				case event := <-detectedEventChan:
					log.Debugf("forked:%s , checked:%s", event.ForkHash.Hex(), event.DetectedHash.Hex())
					goto L
				}
			}
		}()
	}
	return nil
}

func (ethClient *EthClient) forkDetect(observers []chan chainclient.ForkedEvent, database db.Database) error {
	detect := &forkDetect{}
	detect.hashStore = db.NewTable(database, "fork_")
	detect.observers = observers
	startedNumberBs, _ := detect.hashStore.Get([]byte("latest"))
	detect.startedNumber = new(big.Int).SetBytes(startedNumberBs)
	iterator := ethClient.BlockIterator(detect.startedNumber, nil)
	for {
		b, err := iterator.Next()
		if nil != err {
			log.Errorf("err:%s", err.Error())
			panic(err)
		} else {
			block := b.(Block)
			if block.ParentHash == detect.parentHash || detect.parentHash.IsZero() {
				detect.hashStore.Put(block.Number.BigInt().Bytes(), block.Hash.Bytes())
				detect.parentHash = block.Hash
				detect.hashStore.Put([]byte("latest"), block.Number.BigInt().Bytes())
			} else {
				parentNumber := new(big.Int).Set(block.Number.BigInt())
				parentNumber.Sub(parentNumber, big.NewInt(1))
				if forkedNumber, forkedHash, err := getForkedBlock(parentNumber, detect.hashStore, ethClient); nil != err {
					panic(err)
				} else {
					forkedEvent := &chainclient.ForkedEvent{
						DetectedBlock: block.Number.BigInt(),
						DetectedHash:  block.Hash,
						ForkBlock:     forkedNumber,
						ForkHash:      forkedHash,
					}
					detect.hashStore.Put([]byte("latest"), forkedNumber.Bytes())
					for _, observer := range observers {
						go func() {
							observer <- *forkedEvent
						}()
					}
					break
				}
			}
		}
	}
	return nil
}

func getForkedBlock(parentNumber *big.Int, hashStore db.Database, ethClient *EthClient) (*big.Int, types.Hash, error) {
	bs, _ := hashStore.Get(parentNumber.Bytes())
	parentStoredHash := types.BytesToHash(bs)
	if parentStoredHash.IsZero() {
		return nil, types.HexToHash("0x"), errors.New("detected fork ,but parent block not stored in database")
	} else if parentNumber.Cmp(big.NewInt(0)) < 0 {
		return nil, types.HexToHash("0x"), errors.New("detected fork ,but not found forked block")
	}
	var parentBlock Block
	ethClient.GetBlockByNumber(&parentBlock, fmt.Sprintf("%#x", parentNumber), false)

	if parentBlock.Hash == parentStoredHash {
		return parentNumber, parentStoredHash, nil
	} else {
		return getForkedBlock(parentNumber.Sub(parentNumber, big.NewInt(1)), hashStore, ethClient)
	}
}
