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
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/types"
	"math/big"
)

func (l *ExtractorServiceImpl) startDetectFork() {
	forkWatcher := &eventemitter.Watcher{Concurrent: true, Handle: l.processFork}
	eventemitter.On(eventemitter.ExtractorFork, forkWatcher)
}

func (l *ExtractorServiceImpl) detectFork(block *types.Block) error {
	var (
		latestBlock   types.Block
		newBlockModel dao.Block
		forkEvent     types.ForkedEvent
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
	if block.ParentHash == latestBlock.BlockHash || types.IsZeroHash(latestBlock.ParentHash) {
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

	// 更新block分叉标记
	model := dao.Block{}
	if err := model.ConvertDown(forkBlock); err == nil {
		l.dao.SetForkBlock(forkBlock.BlockHash)
	}

	// 发送分叉事件
	forkEvent.ForkHash = forkBlock.BlockHash
	forkEvent.ForkBlock = forkBlock.BlockNumber
	forkEvent.DetectedHash = block.BlockHash
	forkEvent.DetectedBlock = block.BlockNumber

	eventemitter.Emit(eventemitter.ExtractorFork, &forkEvent)
	eventemitter.Emit(eventemitter.OrderManagerFork, &forkEvent)

	return nil
}

func (l *ExtractorServiceImpl) getForkedBlock(block *types.Block) (*types.Block, error) {
	var (
		ethBlock    ethaccessor.Block
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
	if err := l.accessor.Call(&ethBlock, "eth_getBlockByNumber", parentBlockNumber, false); err != nil {
		return nil, err
	}

	preBlock := &types.Block{}
	preBlock.BlockNumber = ethBlock.Number.BigInt()
	preBlock.BlockHash = ethBlock.Hash
	preBlock.ParentHash = ethBlock.ParentHash

	return l.getForkedBlock(preBlock)
}

func (l *ExtractorServiceImpl) processFork(input eventemitter.EventData) error {
	l.Restart()
	return nil
}
