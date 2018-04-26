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
	txtyp "github.com/Loopring/relay/txmanager/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type TransactionView struct {
	ID         int    `gorm:"column:id;primary_key;"`
	Symbol     string `gorm:"column:symbol;type:varchar(20)"`
	Owner      string `gorm:"column:owner;type:varchar(42)"`
	TxHash     string `gorm:"column:tx_hash;type:varchar(82)"`
	LogIndex   int64  `gorm:"column:tx_log_index"`
	Amount 		string `gorm:"column:amount;type:varchar(40)"`
	Type       uint8  `gorm:"column:tx_type"`
	Status     uint8  `gorm:"column:status"`
	CreateTime int64  `gorm:"column:create_time"`
	UpdateTime int64  `gorm:"column:update_time"`
	Fork       bool   `gorm:"column:fork"`
}

// convert types/transaction to dao/transaction
// todo(fuk): judge nil fields
func (tx *TransactionView) ConvertDown(src *txtyp.TransactionView) error {
	tx.Symbol = src.Symbol
	tx.Owner = src.Owner.Hex()
	tx.TxHash = src.TxHash.Hex()
	tx.LogIndex = src.LogIndex
	tx.Amount = src.Amount.String()
	tx.Type = uint8(src.Type)
	tx.Status = uint8(src.Status)
	tx.CreateTime = src.CreateTime
	tx.UpdateTime = src.UpdateTime
	tx.Fork = false

	return nil
}

// convert dao/transaction to types/transaction
func (tx *TransactionView) ConvertUp(dst *txtyp.TransactionView) error {
	dst.Symbol = tx.Symbol
	dst.Owner = common.HexToAddress(tx.Owner)
	dst.TxHash = common.HexToHash(tx.TxHash)
	dst.LogIndex = tx.LogIndex
	dst.Amount, _ = new(big.Int).SetString(tx.Amount, 0)
	dst.Type = txtyp.TxType(tx.Type)
	dst.Status = types.TxStatus(tx.Status)
	dst.CreateTime = tx.CreateTime
	dst.UpdateTime = tx.UpdateTime

	return nil
}

// entity不处理pending数据
func (s *RdsServiceImpl) FindPendingTxViewByHashAndLogIndex(txhash string, logIndex int64) (TransactionEntity, error) {
	var tx TransactionEntity

	err := s.db.Where("tx_hash=?", txhash).
		Where("tx_log_index=?", logIndex).
		Where("fork=?", false).
		Find(&tx).Error

	return tx, err
}