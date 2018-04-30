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
	"github.com/Loopring/relay/types"
	"log"
	"math/big"
)

type forkDetector struct {
	db          dao.RdsService
	latestBlock *types.Block
}

func newForkDetector(db dao.RdsService, startBlockConfig *big.Int) *forkDetector {
	detector := &forkDetector{}
	detector.db = db
	detector.latestBlock = &types.Block{}

	if entity, err := detector.db.FindLatestBlock(); err == nil {
		entity.ConvertUp(detector.latestBlock)
		return detector
	}

	var block ethaccessor.Block
	if err := ethaccessor.GetBlockByNumber(&block, startBlockConfig, false); err != nil {
		log.Fatalf("extractor,fork detector can not find init block:%s", startBlockConfig.String())
	}

	detector.latestBlock.BlockNumber = block.Number.BigInt()
	detector.latestBlock.BlockHash = block.Hash
	detector.latestBlock.CreateTime = block.Timestamp.BigInt().Int64()
	detector.latestBlock.ParentHash = block.ParentHash

	model := &dao.Block{}
	model.ConvertDown(detector.latestBlock)
	detector.db.SaveBlock(model)

	return detector
}

func (detector *forkDetector) Detect(currentBlock *types.Block) (*types.ForkedEvent, error) {
	// filter invalid block
	if types.IsZeroHash(currentBlock.ParentHash) || types.IsZeroHash(currentBlock.BlockHash) {
		return nil, fmt.Errorf("extractor,fork detector find invalid block:%s", currentBlock.BlockNumber.String())
	}

	// no fork
	if detector.latestBlock.BlockHash == currentBlock.BlockHash || detector.latestBlock.BlockHash == currentBlock.ParentHash {
		detector.latestBlock = currentBlock
		return nil, nil
	}

	// find forked root block
	forkBlock, err := detector.getForkedBlock(currentBlock)
	if err != nil {
		return nil, fmt.Errorf("extractor,get forked block failed :%s,node should be shut down...", err.Error())
	}
	detector.latestBlock = forkBlock

	// set fork event
	var forkEvent types.ForkedEvent
	forkEvent.ForkHash = forkBlock.BlockHash
	forkEvent.ForkBlock = forkBlock.BlockNumber
	forkEvent.DetectedHash = currentBlock.BlockHash
	forkEvent.DetectedBlock = currentBlock.BlockNumber

	// mark fork block in database
	model := dao.Block{}
	model.ConvertDown(forkBlock)
	if err := detector.db.SetForkBlock(forkEvent.ForkBlock.Int64(), forkEvent.DetectedBlock.Int64()); err != nil {
		return nil, fmt.Errorf("extractor,fork detector mark fork block %s failed, you should mark it manual, err:%s", forkBlock.BlockHash.Hex(), err.Error())
	}

	return &forkEvent, nil
}

func (detector *forkDetector) getForkedBlock(block *types.Block) (*types.Block, error) {
	var (
		ethBlock    ethaccessor.Block
		parentBlock types.Block
	)

	// find parent block in database
	if parentBlockModel, err := detector.db.FindBlockByHash(block.ParentHash); err == nil {
		parentBlockModel.ConvertUp(&parentBlock)
		return &parentBlock, nil
	}

	// find parent block on chain
	if err := ethaccessor.GetBlockByHash(&ethBlock, block.ParentHash.Hex(), false); err != nil {
		return nil, err
	}

	preBlock := &types.Block{}
	preBlock.BlockNumber = ethBlock.Number.BigInt()
	preBlock.BlockHash = ethBlock.Hash
	preBlock.ParentHash = ethBlock.ParentHash

	return detector.getForkedBlock(preBlock)
}
