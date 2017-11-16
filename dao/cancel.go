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
)

type CancelEvent struct {
	ID              int    `gorm:"column:id;primary_key;"`
	OrderHash       string `gorm:"column:order_hash;varchar(82);unique_index"`
	BlockNumber     int64  `gorm:"column:block_number"`
	CreateTime      int64  `gorm:"column:create_time"`
	AmountCancelled []byte `gorm:"column:amount_cancelled;type:varchar(30)"`
}

// convert chainClient/orderCancelledEvent to dao/CancelEvent
func (e *CancelEvent) ConvertDown(src *types.OrderCancelledEvent) error {
	var err error
	e.AmountCancelled, err = src.AmountCancelled.MarshalText()
	if err != nil {
		return err
	}

	e.OrderHash = src.OrderHash.Hex()
	e.CreateTime = src.Time.Int64()
	e.BlockNumber = src.Blocknumber.Int64()

	return nil
}

func (s *RdsServiceImpl) FindCancelEventByOrderhash(orderhash types.Hash) (*CancelEvent, error) {
	var (
		model CancelEvent
		err   error
	)

	err = s.db.Where("order_hash = ?", orderhash.Hex()).First(&model).Error

	return &model, err
}
