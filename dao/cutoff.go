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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// todo(fuk): rename table
type CutOffEvent struct {
	ID          int    `gorm:"column:id;primary_key;"`
	Protocol    string `gorm:"column:contract_address;type:varchar(42)"`
	Owner       string `gorm:"column:owner;type:varchar(42)"`
	TxHash      string `gorm:"column:tx_hash;type:varchar(82)"`
	BlockNumber int64  `gorm:"column:block_number"`
	Cutoff      int64  `gorm:"column:cutoff"`
	CreateTime  int64  `gorm:"column:create_time"`
}

// convert types/cutoffEvent to dao/CancelEvent
func (e *CutOffEvent) ConvertDown(src *types.AllOrdersCancelledEvent) error {
	e.Owner = src.Owner.Hex()
	e.Protocol = src.Protocol.Hex()
	e.TxHash = src.TxHash.Hex()
	e.Cutoff = src.Cutoff.Int64()
	e.BlockNumber = src.BlockNumber.Int64()
	e.CreateTime = src.BlockTime

	return nil
}

// convert dao/cutoffEvent to types/cutoffEvent
func (e *CutOffEvent) ConvertUp(dst *types.AllOrdersCancelledEvent) error {
	dst.Owner = common.HexToAddress(e.Owner)
	dst.Protocol = common.HexToAddress(e.Protocol)
	dst.TxHash = common.HexToHash(e.TxHash)
	dst.BlockNumber = big.NewInt(e.BlockNumber)
	dst.Cutoff = big.NewInt(e.Cutoff)
	dst.BlockTime = e.CreateTime

	return nil
}

func (s *RdsServiceImpl) GetCutoffEvent(protocol, owner common.Address) (*CutOffEvent, error) {
	var (
		model CutOffEvent
		err   error
	)

	err = s.db.Where("contract_address = ? and owner = ?", protocol.Hex(), owner.Hex()).First(&model).Error

	return &model, err
}

func (s *RdsServiceImpl) DelCutoffEvent(protocol, owner common.Address) error {
	return s.db.Delete(CutOffEvent{}, "contract_address = ? and owner = ?", protocol.Hex(), owner.Hex()).Error
}

func (s *RdsServiceImpl) RollBackCutoff(from, to int64) error {
	return s.db.Where("block_number > ? and block_number <= ?", from, to).Delete(&CutOffEvent{}).Error
}

func (s *RdsServiceImpl) UpdateCutoffByProtocolAndOwner(protocol, owner common.Address, txhash common.Hash, blockNumber, cutoff, createTime *big.Int) error {
	item := map[string]interface{}{"tx_hash": txhash.Hex(), "block_number": blockNumber.Int64(), "cutoff": cutoff.Int64(), "create_time": createTime}
	return s.db.Model(&CutOffEvent{}).Where("contract_address = ? and owner = ?", protocol.Hex(), owner.Hex()).Update(item).Error
}
