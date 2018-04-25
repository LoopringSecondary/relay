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

type TransactionView struct {
	ID         int    `gorm:"column:id;primary_key;"`
	Symbol     string `gorm:"column:symbol;type:varchar(20)"`
	Owner      string `gorm:"column:owner;type:varchar(42)"`
	TxHash     string `gorm:"column:tx_hash;type:varchar(82)"`
	LogIndex   int64  `gorm:"column:tx_log_index"`
	Type       uint8  `gorm:"column:tx_type"`
	Status     uint8  `gorm:"column:status"`
	CreateTime int64  `gorm:"column:create_time"`
	UpdateTime int64  `gorm:"column:update_time"`
	Fork       bool   `gorm:"column:fork"`
}

// convert types/transaction to dao/transaction
// todo(fuk): judge nil fields
func (tx *TransactionView) ConvertDown(src *types.Transaction) error {

	return nil
}

// convert dao/transaction to types/transaction
func (tx *TransactionView) ConvertUp(dst *types.Transaction) error {

	return nil
}
