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

type CancelEvent struct {
	ID              int    `gorm:"column:id;primary_key;"`
	Protocol        string `gorm:"column:contract_address;type:varchar(42)"`
	OrderHash       string `gorm:"column:order_hash;varchar(82);unique_index"`
	TxHash          string `gorm:"column:tx_hash;type:varchar(82)"`
	BlockNumber     int64  `gorm:"column:block_number"`
	CreateTime      int64  `gorm:"column:create_time"`
	AmountCancelled string `gorm:"column:amount_cancelled;type:varchar(30)"`
}

// convert chainClient/orderCancelledEvent to dao/CancelEvent
func (e *CancelEvent) ConvertDown(src *types.OrderCancelledEvent) error {
	e.AmountCancelled = src.AmountCancelled.String()
	e.OrderHash = src.OrderHash.Hex()
	e.TxHash = src.TxHash.Hex()
	e.Protocol = src.ContractAddress.Hex()
	e.CreateTime = src.Time.Int64()
	e.BlockNumber = src.Blocknumber.Int64()

	return nil
}

func (s *RdsServiceImpl) FindCancelEvent(orderhash common.Hash, cancelledAmount *big.Int) (*CancelEvent, error) {
	var (
		model CancelEvent
		err   error
	)

	err = s.db.Where("order_hash = ? and amount_cancelled = ?", orderhash.Hex(), cancelledAmount.String()).First(&model).Error

	return &model, err
}

func (s *RdsServiceImpl) RollBackCancel(from, to int64) error {
	return s.db.Where("block_number > ? and block_number <= ?", from, to).Delete(&CancelEvent{}).Error
}
