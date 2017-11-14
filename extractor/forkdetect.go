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
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/dao"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/types"
	"math/big"
)

func (l *ExtractorServiceImpl) startDetectFork() {
	forkWatcher := &eventemitter.Watcher{Concurrent: true, Handle: l.processFork}
	eventemitter.On(eventemitter.Fork, forkWatcher)
}

func (l *ExtractorServiceImpl) detectFork(block *types.Block) error {
	var (
		latestBlock   types.Block
		newBlockModel dao.Block
		forkEvent     chainclient.ForkedEvent
	)

	latestBlockModel, err := l.dao.FindLatestBlock()
	if err != nil {
		return err
	}
	if err := latestBlockModel.ConvertUp(&latestBlock); err != nil {
		return err
	}

	// 重启时第一个块
	if block.BlockHash == latestBlock.BlockHash {
		return nil
	}

	// 没有分叉
	if block.ParentHash == latestBlock.BlockHash || latestBlock.ParentHash.IsZero() {
		if err := newBlockModel.ConvertUp(block); err != nil {
			return err
		}
		if err := l.dao.Add(newBlockModel); err != nil {
			return err
		}
	}

	// 已经分叉,寻找分叉块,出错则在下一个块继续检查
	forkBlock, err := l.getForkedBlock(block)
	if err != nil {
		return err
	}

	forkEvent.ForkHash = forkBlock.BlockHash
	forkEvent.ForkBlock = forkBlock.BlockNumber
	forkEvent.DetectedHash = block.BlockHash
	forkEvent.DetectedBlock = block.BlockNumber

	eventemitter.Emit(eventemitter.Fork, &forkEvent)
	return nil
}

func (l *ExtractorServiceImpl) getForkedBlock(block *types.Block) (*types.Block, error) {
	var (
		ethBlock    eth.Block
		parentBlock types.Block
	)

	// 如果数据库已存在,则该block即为分叉根节点
	if parentBlockModel, err := l.dao.FindBlockByParentHash(block.ParentHash); err == nil {
		if err := parentBlockModel.ConvertUp(&parentBlock); err != nil {
			return nil, err
		} else {
			return &parentBlock, nil
		}
	}

	// 如果不存在,则查询以太坊
	parentBlockNumber := block.BlockNumber.Sub(block.BlockNumber, big.NewInt(1))
	l.ethClient.GetBlockByNumber(ethBlock, fmt.Sprintf("%#x", parentBlockNumber), false)

	forkBlock := &types.Block{}
	forkBlock.BlockNumber = ethBlock.Number.BigInt()
	forkBlock.BlockHash = ethBlock.Hash
	forkBlock.ParentHash = ethBlock.ParentHash

	return l.getForkedBlock(forkBlock)
}

func (l *ExtractorServiceImpl) processFork(input eventemitter.EventData) error {
	l.Stop()

	return nil
}
