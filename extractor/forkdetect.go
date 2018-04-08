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
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"math/big"
)

type forkDetector struct {
	db          dao.RdsService
	latestBlock *types.Block
}

func newForkDetector(db dao.RdsService, startBlockConfig *big.Int) *forkDetector {
	detector := &forkDetector{}
	detector.db = db

	entity, err := detector.db.FindLatestBlock()
	if err == nil {
		detector.latestBlock = new(types.Block)
		entity.ConvertUp(detector.latestBlock)
		return detector
	}

	if err := ethaccessor.GetBlockByNumber(detector.latestBlock, startBlockConfig, false); err != nil {
		log.Fatalf("extractor,fork detector can not find init block:%s", startBlockConfig.String())
	}

	return detector
}

func (detector *forkDetector) Detect(currentBlock *types.Block) *types.Block {
	// filter invalid block
	if types.IsZeroHash(currentBlock.ParentHash) || types.IsZeroHash(currentBlock.BlockHash) {
		log.Debugf("extractor,fork detector find invalid block:%s", currentBlock.BlockNumber.String())
		return nil
	}

	// no fork
	if detector.latestBlock.BlockHash == currentBlock.BlockHash || detector.latestBlock.BlockHash == currentBlock.ParentHash {
		detector.latestBlock = currentBlock
		return nil
	}

	// find forked root block
	forkBlock, err := detector.getForkedBlock(currentBlock)
	if err != nil {
		log.Fatalf("extractor,get forked block failed :%s,node should be shut down...", err.Error())
	}
	detector.latestBlock = forkBlock

	return forkBlock
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
	if err := ethaccessor.GetBlockByHash(&ethBlock, block.ParentHash.Hex(), false); err != nil {
		return nil, err
	}

	preBlock := &types.Block{}
	preBlock.BlockNumber = ethBlock.Number.BigInt()
	preBlock.BlockHash = ethBlock.Hash
	preBlock.ParentHash = ethBlock.ParentHash

	return detector.getForkedBlock(preBlock)
}
