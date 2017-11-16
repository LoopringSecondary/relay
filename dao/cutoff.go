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

type CutOffEvent struct {
	ID          int    `gorm:"column:id;primary_key;"`
	Owner       string `gorm:"column:owner;type:varchar(42);unique_index"`
	BlockNumber int64  `gorm:"column:block_number"`
	Cutoff      int64  `gorm:"column:cutoff"`
	CreateTime  int64  `gorm:"column:create_time"`
}

// convert chainClient/orderCancelledEvent to dao/CancelEvent
func (e *CutOffEvent) ConvertDown(src *types.CutoffEvent) error {
	e.Owner = src.Owner.Hex()
	e.Cutoff = src.Cutoff.Int64()
	e.BlockNumber = src.Blocknumber.Int64()
	e.CreateTime = src.Time.Int64()

	return nil
}

func (s *RdsServiceImpl) FindCutoffEventByOwnerAddress(owner types.Address) (*CutOffEvent, error) {
	var (
		model CutOffEvent
		err   error
	)

	err = s.db.Where("owner = ?", owner.Hex()).First(&model).Error

	return &model, err
}
