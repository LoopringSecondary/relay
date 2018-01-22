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

package extractor

import (
	"fmt"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"math/big"
)

type forkDetector struct {
	db          dao.RdsService
	accessor    *ethaccessor.EthNodeAccessor
	latestBlock *types.Block
}

func newForkDetector(db dao.RdsService, accessor *ethaccessor.EthNodeAccessor) *forkDetector {
	detector := &forkDetector{}
	detector.accessor = accessor
	detector.db = db
	detector.latestBlock = nil

	return detector
}

func (detector *forkDetector) Detect(currentBlock *types.Block) bool {
	var (
		forkEvent types.ForkedEvent
	)

	// filter invalid block
	if types.IsZeroHash(currentBlock.ParentHash) || types.IsZeroHash(currentBlock.BlockHash) {
		log.Debugf("extractor,fork detector find invalid block:%s", currentBlock.BlockNumber.String())
		return false
	}

	// initialize latest block
	if detector.latestBlock == nil {
		entity, err := detector.db.FindLatestBlock()
		if err != nil {
			detector.latestBlock = currentBlock
			log.Debugf("extractor,fork detector started at first time")
			return false
		} else {
			detector.latestBlock = new(types.Block)
			entity.ConvertUp(detector.latestBlock)
		}
	}

	// no fork
	if detector.latestBlock.BlockHash == currentBlock.BlockHash || detector.latestBlock.BlockHash == currentBlock.ParentHash {
		detector.latestBlock = currentBlock
		return false
	}

	// find forked root block
	forkBlock, err := detector.getForkedBlock(currentBlock)
	if err != nil {
		log.Fatalf("extractor,get forked block failed :%s,node should be shut down...", err.Error())
	}
	detector.latestBlock = forkBlock

	// mark fork block in database
	model := dao.Block{}
	if err := model.ConvertDown(forkBlock); err == nil {
		if err := detector.db.SetForkBlock(forkBlock.BlockHash); err != nil {
			log.Fatalf("extractor,fork detector mark fork block %s failed, you should mark it manual, err:%s", forkBlock.BlockHash.Hex(), err.Error())
		}
	}

	// emit fork event
	forkEvent.ForkHash = forkBlock.BlockHash
	forkEvent.ForkBlock = forkBlock.BlockNumber
	forkEvent.DetectedHash = currentBlock.BlockHash
	forkEvent.DetectedBlock = currentBlock.BlockNumber

	log.Debugf("extractor,detected chain fork, from :%d to %d", forkEvent.ForkBlock.Int64(), forkEvent.DetectedBlock.Int64())
	eventemitter.Emit(eventemitter.ChainForkDetected, &forkEvent)

	return true
}

func (detector *forkDetector) getForkedBlock(block *types.Block) (*types.Block, error) {
	var (
		ethBlock    ethaccessor.Block
		parentBlock types.Block
	)

	// find parent block in database
	if parentBlockModel, err := detector.db.FindBlockByParentHash(block.ParentHash); err == nil {
		parentBlockModel.ConvertUp(&parentBlock)
		return &parentBlock, nil
	}

	// find parent block on chain
	// todo if jsonrpc failed
	// 如果不存在,则查询以太坊
	parentBlockNumber := block.BlockNumber.Sub(block.BlockNumber, big.NewInt(1))
	if err := detector.accessor.RetryCall(parentBlockNumber.String(), 2, &ethBlock, "eth_getBlockByNumber", fmt.Sprintf("%#x", parentBlockNumber), false); err != nil {
		return nil, err
	}

	preBlock := &types.Block{}
	preBlock.BlockNumber = ethBlock.Number.BigInt()
	preBlock.BlockHash = ethBlock.Hash
	preBlock.ParentHash = ethBlock.ParentHash

	return detector.getForkedBlock(preBlock)
}
