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
	"math/big"
)

type Block struct {
	ID          int    `gorm:"column:id;primary_key"`
	BlockNumber []byte `gorm:"column:block_number;type:varchar(30)"`
	BlockHash   string `gorm:"column:block_hash;type:varchar(82);unique_index"`
	ParentHash  string `gorm:"column:parent_hash;type:varchar(82);unique_index"`
	CreateTime  int64  `gorm:"column:create_time"`
	Fork        bool   `gorm:"column:fork;"`
}

// convert types/block to dao/block
func (b *Block) ConvertDown(src *types.Block) error {
	var err error
	b.BlockNumber, err = src.BlockNumber.MarshalText()
	if err != nil {
		return err
	}

	b.BlockHash = src.BlockHash.Hex()
	b.ParentHash = src.ParentHash.Hex()
	b.CreateTime = src.CreateTime
	b.Fork = false

	return nil
}

// convert dao/block to types/block
func (b *Block) ConvertUp(dst *types.Block) error {
	dst.BlockNumber = new(big.Int)
	if err := dst.BlockNumber.UnmarshalText(b.BlockNumber); err != nil {
		return err
	}

	dst.BlockHash = types.HexToHash(b.BlockHash)
	dst.ParentHash = types.HexToHash(b.ParentHash)
	dst.CreateTime = b.CreateTime

	return nil
}

func (s *RdsServiceImpl) FindBlockByHash(blockhash types.Hash) (*Block, error) {
	var (
		block Block
		err   error
	)

	if types.IsZeroHash(blockhash) {
		return nil, errors.New("block table findBlockByHash get an illegal hash")
	}

	err = s.db.Where("block_hash = ?", blockhash.Hex()).First(&block).Error

	return &block, err
}

func (s *RdsServiceImpl) FindBlockByParentHash(parenthash types.Hash) (*Block, error) {
	var (
		block Block
		err   error
	)

	if types.IsZeroHash(parenthash) {
		return nil, errors.New("block table findBlockByParentHash get an  illegal hash")
	}

	err = s.db.Where("parent_hash = ?", parenthash.Hex()).First(&block).Error

	return &block, err
}

func (s *RdsServiceImpl) FindLatestBlock() (*Block, error) {
	var (
		block Block
		err   error
	)

	err = s.db.Order("create_time, desc").First(&block).Error

	return &block, err
}

func (s *RdsServiceImpl) FindForkBlock() (*Block, error) {
	var (
		block Block
		err   error
	)

	err = s.db.Where("fork = ?", true).First(&block).Error

	return &block, err
}
