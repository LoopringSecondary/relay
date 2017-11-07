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
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/eventemiter"
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
func (l *EthClientListener) StartForkDetect() error {

	detectedEventChan := make(chan chainclient.ForkedEvent)

	forkWatcher := &eventemitter.Watcher{Concurrent: true, Handle: func(eventData eventemitter.EventData) error {
		event := eventData.(chainclient.ForkedEvent)
		log.Debugf("forked:%s , checked:%s", event.ForkHash.Hex(), event.DetectedHash.Hex())
		detectedEventChan <- event
		return nil
	}}
	eventemitter.On(eventemitter.Fork, forkWatcher)

	go func() {
	L:
		go l.forkDetect(l.rds.db)
		for {
			select {
			case event := <-detectedEventChan:
				log.Debugf("forked:%s , checked:%s", event.ForkHash.Hex(), event.DetectedHash.Hex())
				goto L
			}
		}
	}()
	return nil
}

//todo:should be optimized， 启动点等需要重新考虑，获取分叉点等的问题
func (l *EthClientListener) forkDetect(database db.Database) error {
	detect := &forkDetect{}
	detect.hashStore = db.NewTable(database, "fork_")
	startedNumberBs, _ := detect.hashStore.Get([]byte("latest"))
	detect.startedNumber = new(big.Int).SetBytes(startedNumberBs)
	iterator := l.ethClient.BlockIterator(detect.startedNumber, nil, false, uint64(0))
	for {
		b, err := iterator.Next()
		if nil != err {
			log.Errorf("err:%s", err.Error())
			panic(err)
		} else {
			block := b.(*eth.BlockWithTxHash)
			if block.ParentHash == detect.parentHash || detect.parentHash.IsZero() {
				detect.hashStore.Put(block.Number.BigInt().Bytes(), block.Hash.Bytes())
				detect.parentHash = block.Hash
				detect.hashStore.Put([]byte("latest"), block.Number.BigInt().Bytes())
			} else {
				parentNumber := new(big.Int).Set(block.Number.BigInt())
				parentNumber.Sub(parentNumber, big.NewInt(1))
				if forkedNumber, forkedHash, err := l.getForkedBlock(parentNumber, detect.hashStore); nil != err {
					panic(err)
				} else {
					forkedEvent := chainclient.ForkedEvent{
						DetectedBlock: block.Number.BigInt(),
						DetectedHash:  block.Hash,
						ForkBlock:     forkedNumber,
						ForkHash:      forkedHash,
					}
					detect.hashStore.Put([]byte("latest"), forkedNumber.Bytes())
					eventemitter.Emit(eventemitter.Fork, forkedEvent)
					break
				}
			}
		}
	}
	return nil
}

func (l *EthClientListener) getForkedBlock(parentNumber *big.Int, hashStore db.Database) (*big.Int, types.Hash, error) {
	bs, _ := hashStore.Get(parentNumber.Bytes())
	parentStoredHash := types.BytesToHash(bs)
	if parentStoredHash.IsZero() {
		return nil, types.HexToHash("0x"), errors.New("detected fork ,but parent block not stored in database")
	} else if parentNumber.Cmp(big.NewInt(0)) < 0 {
		return nil, types.HexToHash("0x"), errors.New("detected fork ,but not found forked block")
	}
	var parentBlock eth.Block
	l.ethClient.GetBlockByNumber(&parentBlock, fmt.Sprintf("%#x", parentNumber), false)

	if parentBlock.Hash == parentStoredHash {
		return parentNumber, parentStoredHash, nil
	} else {
		return l.getForkedBlock(parentNumber.Sub(parentNumber, big.NewInt(1)), hashStore)
	}
}
