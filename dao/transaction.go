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
	Protocol    string `gorm:"column:protocol;type:varchar(42)"`
	Symbol      string `gorm:"column:symbol;type:varchar(20)"`
	Owner       string `gorm:"column:owner;type:varchar(42)"`
	From        string `gorm:"column:tx_from;type:varchar(42)"`
	To          string `gorm:"column:tx_to;type:varchar(42)"`
	RawFrom     string `gorm:"column:raw_from;type:varchar(42)"`
	RawTo       string `gorm:"column:raw_to;type:varchar(42)"`
	TxHash      string `gorm:"column:tx_hash;type:varchar(82)"`
	Content     string `gorm:"column:content;type:text"`
	BlockNumber int64  `gorm:"column:block_number"`
	TxIndex     int64  `gorm:"column:tx_index"`
	LogIndex    int64  `gorm:"column:tx_log_index"`
	Value       string `gorm:"column:amount;type:varchar(64)"`
	Type        uint8  `gorm:"column:tx_type"`
	Status      uint8  `gorm:"column:status"`
	GasLimit    string `gorm:"column:gas_limit;type:varchar(40)"`
	GasUsed     string `gorm:"column:gas_used;type:varchar(40)"`
	GasPrice    string `gorm:"column:gas_price;type:varchar(40)"`
	Nonce       string `gorm:"column:nonce;type:varchar(40)"`
	CreateTime  int64  `gorm:"column:create_time"`
	UpdateTime  int64  `gorm:"column:update_time"`
	Fork        bool   `gorm:"column:fork"`
}

func (s *RdsServiceImpl) GetTransactionById(id int) (Transaction, error) {
	var tx Transaction
	err := s.db.Where("id=?", id).First(&tx).Error
	return tx, err
}
