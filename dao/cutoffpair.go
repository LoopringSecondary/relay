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
	"fmt"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type CutOffPairEvent struct {
	ID          int    `gorm:"column:id;primary_key;"`
	Protocol    string `gorm:"column:contract_address;type:varchar(42)"`
	Owner       string `gorm:"column:owner;type:varchar(42)"`
	Token1      string `gorm:"column:token1;type:varchar(42)"`
	Token2      string `gorm:"column:token2;type:varchar(42)"`
	TxHash      string `gorm:"column:tx_hash;type:varchar(82)"`
	BlockNumber int64  `gorm:"column:block_number"`
	Cutoff      int64  `gorm:"column:cutoff"`
	CreateTime  int64  `gorm:"column:create_time"`
}

// convert types/cutoffEvent to dao/CancelEvent
func (e *CutOffPairEvent) ConvertDown(src *types.OrdersCancelledEvent) error {
	e.Owner = src.Owner.Hex()
	e.Protocol = src.Protocol.Hex()
	e.TxHash = src.TxHash.Hex()
	e.Token1 = src.Token1.Hex()
	e.Token2 = src.Token2.Hex()
	e.Cutoff = src.Cutoff.Int64()
	e.BlockNumber = src.BlockNumber.Int64()
	e.CreateTime = src.BlockTime

	return nil
}

// convert dao/cutoffEvent to types/cutoffEvent
func (e *CutOffPairEvent) ConvertUp(dst *types.OrdersCancelledEvent) error {
	dst.Owner = common.HexToAddress(e.Owner)
	dst.Protocol = common.HexToAddress(e.Protocol)
	dst.TxHash = common.HexToHash(e.TxHash)
	dst.Token1 = common.HexToAddress(e.Token1)
	dst.Token2 = common.HexToAddress(e.Token2)
	dst.BlockNumber = big.NewInt(e.BlockNumber)
	dst.Cutoff = big.NewInt(e.Cutoff)
	dst.BlockTime = e.CreateTime

	return nil
}

func (s *RdsServiceImpl) GetCutoffPairEvent(protocol, owner, token1, token2 common.Address) (*CutOffEvent, error) {
	var (
		model CutOffEvent
		err   error
	)

	if token1 == token2 {
		return nil, fmt.Errorf("dao cutoffpair tokens %s %s should not be the same", token1.Hex(), token2.Hex())
	}

	addresses := []string{token1.Hex(), token2.Hex()}

	err = s.db.Where("contract_address = ? and owner = ?", protocol.Hex(), owner.Hex()).
		Where("token1 in (?)", addresses).
		Where("token2 in (?)", addresses).
		First(&model).Error

	return &model, err
}

func (s *RdsServiceImpl) DelCutoffPairEvent(protocol, owner, token1, token2 common.Address) error {
	if token1 == token2 {
		return fmt.Errorf("dao cutoffpair tokens %s %s should not be the same", token1.Hex(), token2.Hex())
	}

	addresses := []string{token1.Hex(), token2.Hex()}

	return s.db.Delete(CutOffEvent{}, "contract_address = ? and owner = ?", protocol.Hex(), owner.Hex()).
		Where("token1 in (?)", addresses).
		Where("token2 in (?)", addresses).Error
}

func (s *RdsServiceImpl) RollBackCutoffPair(from, to int64) error {
	return s.db.Where("block_number > ? and block_number <= ?", from, to).Delete(&CutOffEvent{}).Error
}

func (s *RdsServiceImpl) UpdateCutoffPairEvent(protocol, owner, token1, token2 common.Address, txhash common.Hash, blockNumber, cutoff, createTime *big.Int) error {
	if token1 == token2 {
		return fmt.Errorf("dao cutoffpair tokens %s %s should not be the same", token1.Hex(), token2.Hex())
	}

	addresses := []string{token1.Hex(), token2.Hex()}

	item := map[string]interface{}{"tx_hash": txhash.Hex(), "block_number": blockNumber.Int64(), "cutoff": cutoff.Int64(), "create_time": createTime}

	return s.db.Model(&CutOffEvent{}).Where("contract_address = ? and owner = ?", protocol.Hex(), owner.Hex()).
		Where("token1 in (?)", addresses).
		Where("token2 in (?)", addresses).
		Update(item).Error
}
