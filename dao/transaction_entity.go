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
	txtyp "github.com/Loopring/relay/txmanager/types"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// txhash&logIndex unique，
// fork should be marked and never used it again
type TransactionEntity struct {
	ID          int    `gorm:"column:id;primary_key;"`
	From        string `gorm:"column:tx_from;type:varchar(42)"`
	To          string `gorm:"column:tx_to;type:varchar(42)"`
	BlockNumber int64  `gorm:"column:block_number"`
	TxHash      string `gorm:"column:tx_hash;type:varchar(82)"`
	LogIndex    int64  `gorm:"column:tx_log_index"`
	Value       string `gorm:"column:amount;type:varchar(64)"`
	Content     string `gorm:"column:content;type:text"`
	Status      uint8  `gorm:"column:status"`
	GasLimit    string `gorm:"column:gas_limit;type:varchar(40)"`
	GasUsed     string `gorm:"column:gas_used;type:varchar(40)"`
	GasPrice    string `gorm:"column:gas_price;type:varchar(40)"`
	Nonce       string `gorm:"column:nonce;type:varchar(40)"`
	BlockTime  int64  `gorm:"column:block_time"`
	Fork        bool   `gorm:"column:fork"`
}

// convert to txmanager/types/transactionEntity to dao/transactionEntity
// todo(fuk): judge nil fields
func (tx *TransactionEntity) ConvertDown(src *txtyp.TransactionEntity) error {
	tx.From = src.From.Hex()
	tx.To = src.To.Hex()
	tx.BlockNumber = src.BlockNumber
	tx.TxHash = src.Hash.Hex()
	tx.LogIndex = src.LogIndex
	tx.Value = src.Value.String()
	tx.Content = src.Content
	tx.Status = uint8(src.Status)
	tx.GasLimit = src.GasLimit.String()
	tx.GasUsed = src.GasUsed.String()
	tx.GasPrice = src.GasPrice.String()
	tx.Nonce = src.Nonce.String()
	tx.BlockTime = src.BlockTime
	tx.Fork = false

	return nil
}

// convert dao/transactionEntity to txmanager/types/transactionEntity
func (tx *TransactionEntity) ConvertUp(dst *txtyp.TransactionEntity) error {
	dst.From = common.HexToAddress(tx.From)
	dst.To = common.HexToAddress(tx.To)
	dst.BlockNumber = tx.BlockNumber
	dst.Hash = common.HexToHash(tx.TxHash)
	dst.LogIndex = tx.LogIndex
	dst.Value, _ = new(big.Int).SetString(tx.Value, 0)
	dst.Content = tx.Content
	dst.Status = types.TxStatus(tx.Status)
	dst.GasLimit, _ = new(big.Int).SetString(tx.GasLimit, 0)
	dst.GasUsed, _ = new(big.Int).SetString(tx.GasUsed, 0)
	dst.GasPrice, _ = new(big.Int).SetString(tx.GasPrice, 0)
	dst.Nonce, _ = new(big.Int).SetString(tx.Nonce, 0)
	dst.BlockTime = tx.BlockTime

	return nil
}

// entity不处理pending数据
func (s *RdsServiceImpl) FindTxEntityByHashAndLogIndex(txhash string, logIndex int64) (TransactionEntity, error) {
	var tx TransactionEntity

	err := s.db.Where("tx_hash=?", txhash).
		Where("tx_log_index=?", logIndex).
		Where("fork=?", false).
		Find(&tx).Error

	return tx, err
}
