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
)

type CancelEvent struct {
	ID              int    `gorm:"column:id;primary_key;"`
	Protocol        string `gorm:"column:contract_address;type:varchar(42)"`
	OrderHash       string `gorm:"column:order_hash;varchar(82);unique_index"`
	TxHash          string `gorm:"column:tx_hash;type:varchar(82)"`
	BlockNumber     int64  `gorm:"column:block_number"`
	CreateTime      int64  `gorm:"column:create_time"`
	AmountCancelled []byte `gorm:"column:amount_cancelled;type:varchar(30)"`
	IsDeleted       bool   `gorm:"column:is_deleted"`
}

// convert chainClient/orderCancelledEvent to dao/CancelEvent
func (e *CancelEvent) ConvertDown(src *types.OrderCancelledEvent) error {
	var err error
	e.AmountCancelled, err = src.AmountCancelled.MarshalText()
	if err != nil {
		return err
	}

	e.OrderHash = src.OrderHash.Hex()
	e.TxHash = src.TxHash.Hex()
	e.Protocol = src.ContractAddress.Hex()
	e.CreateTime = src.Time.Int64()
	e.BlockNumber = src.Blocknumber.Int64()
	e.IsDeleted = src.IsDeleted

	return nil
}

func (s *RdsServiceImpl) FindCancelEventByOrderhash(orderhash common.Hash) (*CancelEvent, error) {
	var (
		model CancelEvent
		err   error
	)

	err = s.db.Where("order_hash = ? and is_deleted = false", orderhash.Hex()).First(&model).Error

	return &model, err
}

func (s *RdsServiceImpl) RollBackCancel(from, to int64) error {
	return s.db.Model(&CancelEvent{}).Where("block_number > ? and block_number <= ?", from, to).UpdateColumn("is_deleted", true).Error
}
