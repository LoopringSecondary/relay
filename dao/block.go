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

package dao

import (
	"errors"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type Block struct {
	ID          int    `gorm:"column:id;primary_key"`
	BlockNumber int64  `gorm:"column:block_number;type:bigint"`
	BlockHash   string `gorm:"column:block_hash;type:varchar(82)"`
	ParentHash  string `gorm:"column:parent_hash;type:varchar(82)"`
	CreateTime  int64  `gorm:"column:create_time"`
	Fork        bool   `gorm:"column:fork;"`
}

// convert types/block to dao/block
func (b *Block) ConvertDown(src *types.Block) error {
	b.BlockNumber = src.BlockNumber.Int64()
	b.BlockHash = src.BlockHash.Hex()
	b.ParentHash = src.ParentHash.Hex()
	b.CreateTime = src.CreateTime
	b.Fork = false

	return nil
}

// convert dao/block to types/block
func (b *Block) ConvertUp(dst *types.Block) error {
	dst.BlockNumber = big.NewInt(b.BlockNumber)
	dst.BlockHash = common.HexToHash(b.BlockHash)
	dst.ParentHash = common.HexToHash(b.ParentHash)
	dst.CreateTime = b.CreateTime

	return nil
}

func (s *RdsServiceImpl) FindBlockByHash(blockhash common.Hash) (*Block, error) {
	var block Block
	if types.IsZeroHash(blockhash) {
		return nil, errors.New("block table findBlockByHash get an illegal hash")
	}

	err := s.db.Where("block_hash = ?", blockhash.Hex()).Where("fork = ?", false).First(&block).Error

	return &block, err
}

func (s *RdsServiceImpl) FindLatestBlock() (*Block, error) {
	var block Block
	err := s.db.Order("create_time desc").Where("fork = ?", false).First(&block).Error
	return &block, err
}

func (s *RdsServiceImpl) SetForkBlock(from, to int64) error {
	return s.db.Model(&Block{}).Where("block_number > ? and block_number <= ?", from, to).Update("fork", true).Error
}

func (s *RdsServiceImpl) SaveBlock(latest *Block) error {
	var current Block
	if err := s.db.Where("block_hash=?", latest.BlockHash).Find(&current).Error; err == nil {
		return nil
	}

	return s.db.Create(latest).Error
}
