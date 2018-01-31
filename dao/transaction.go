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

type Transaction struct {
	ID          int    `gorm:"column:id;primary_key;"`
	From        string `gorm:"column:from;type:varchar(42)"`
	To          string `gorm:"column:to;type:varchar(42)"`
	Hash        string `gorm:"column:hash;type:varchar(82)"`
	BlockNumber int64  `gorm:"column:block_number"`
	Value       int64  `gorm:"column:value"`
	Type        uint8  `gorm:"column:type"`
	Status      uint8  `gorm:"column:status"`
	CreateTime  int64  `gorm:"column:create_time"`
	UpdateTime  int64  `gorm:"column:update_time"`
}
