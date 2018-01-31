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

type Transaction struct {
	ID          int    `gorm:"column:id;primary_key;"`
	From        string `gorm:"column:from;type:varchar(42)"`
	To          string `gorm:"column:to;type:varchar(42)"`
	Hash        string `gorm:"column:hash;type:varchar(82)"`
	BlockNumber int64  `gorm:"column:block_number"`
	Value       string `gorm:"column:value;type:varchar(30)"`
	Type        uint8  `gorm:"column:tx_type"`
	Status      uint8  `gorm:"column:status"`
	CreateTime  int64  `gorm:"column:create_time"`
	UpdateTime  int64  `gorm:"column:update_time"`
}

// convert types/transaction to dao/transaction
func (tx *Transaction) ConvertDown(src *types.Transaction) error {
	tx.From = src.From.Hex()
	tx.To = src.To.Hex()
	tx.Hash = src.Hash.Hex()
	tx.BlockNumber = src.BlockNumber.Int64()
	tx.Value = src.Value.String()
	tx.Type = src.Type
	tx.Status = src.Status
	tx.CreateTime = src.CreateTime
	tx.UpdateTime = src.UpdateTime

	return nil
}

// convert dao/transaction to types/transaction
func (tx *Transaction) ConvertUp(dst *types.Transaction) error {
	dst.From = common.HexToAddress(tx.From)
	dst.To = common.HexToAddress(tx.To)
	dst.Hash = common.HexToHash(tx.Hash)
	dst.BlockNumber = big.NewInt(tx.BlockNumber)
	dst.Value, _ = new(big.Int).SetString(tx.Value, 0)
	dst.Type = tx.Type
	dst.Status = tx.Status
	dst.CreateTime = tx.CreateTime
	dst.UpdateTime = tx.UpdateTime

	return nil
}
