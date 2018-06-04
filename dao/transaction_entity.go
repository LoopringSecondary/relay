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
	ID          int    `gorm:"column:id;primary_key;" json:"id"`
	Protocol    string `gorm:"column:protocol;type:varchar(42)" json:"protocol"`
	From        string `gorm:"column:tx_from;type:varchar(42)" json:"from"`
	To          string `gorm:"column:tx_to;type:varchar(42)" json:"to"`
	BlockNumber int64  `gorm:"column:block_number" json:"block_number"`
	TxHash      string `gorm:"column:tx_hash;type:varchar(82)" json:"tx_hash"`
	LogIndex    int64  `gorm:"column:tx_log_index" json:"log_index"`
	Value       string `gorm:"column:amount;type:varchar(64)" json:"value"`
	Content     string `gorm:"column:content;type:text" json:"content"`
	Status      uint8  `gorm:"column:status" json:"status"`
	GasLimit    string `gorm:"column:gas_limit;type:varchar(40)" json:"gas_limit"`
	GasUsed     string `gorm:"column:gas_used;type:varchar(40)" json:"gas_used"`
	GasPrice    string `gorm:"column:gas_price;type:varchar(40)" json:"gas_price"`
	Nonce       int64  `gorm:"column:nonce" json:"nonce"`
	BlockTime   int64  `gorm:"column:block_time" json:"block_time"`
	Fork        bool   `gorm:"column:fork" json:"fork"`
}

// convert to txmanager/types/transactionEntity to dao/transactionEntity
// todo(fuk): judge nil fields
func (tx *TransactionEntity) ConvertDown(src *txtyp.TransactionEntity) error {
	tx.Protocol = src.Protocol.Hex()
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
	tx.Nonce = src.Nonce.Int64()
	tx.BlockTime = src.BlockTime
	tx.Fork = false

	return nil
}

// convert dao/transactionEntity to txmanager/types/transactionEntity
func (tx *TransactionEntity) ConvertUp(dst *txtyp.TransactionEntity) error {
	dst.Protocol = common.HexToAddress(tx.Protocol)
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
	dst.Nonce = big.NewInt(tx.Nonce)
	dst.BlockTime = tx.BlockTime

	return nil
}

// 根据hash查询pending tx
func (s *RdsServiceImpl) FindPendingTxEntity(hash string) (TransactionEntity, error) {
	var tx TransactionEntity

	err := s.db.Where("tx_hash=?", hash).
		Where("status=?", types.TX_STATUS_PENDING).
		Where("fork=?", false).
		First(&tx).Error

	return tx, err
}

func (s *RdsServiceImpl) GetTxEntity(hashlist []string) ([]TransactionEntity, error) {
	var txs []TransactionEntity

	err := s.db.Where("tx_hash in (?)", hashlist).
		Where("fork=?", false).
		Find(&txs).Error

	return txs, err
}

// 根据交易发起者from地址及nonce获取pending tx
func (s *RdsServiceImpl) GetPendingTxEntity(from string, nonce int64) ([]TransactionEntity, error) {
	var txs []TransactionEntity

	err := s.db.Where("tx_from=?", from).
		Where("nonce<=?", nonce).
		Where("status=?", types.TX_STATUS_PENDING).
		Where("fork=?", false).
		Find(&txs).Error

	return txs, err
}

// 根据hash&status删除pending tx
func (s *RdsServiceImpl) DelPendingTxEntity(hash string) error {
	err := s.db.Where("tx_hash=?", hash).
		Where("status=?", types.TX_STATUS_PENDING).
		Where("fork=?", false).
		Delete(&TransactionEntity{}).Error
	return err
}

func (s *RdsServiceImpl) SetPendingTxEntityFailed(hashlist []string) error {
	err := s.db.Model(&TransactionEntity{}).
		Where("tx_hash in (?)", hashlist).
		Where("status=?", types.TX_STATUS_PENDING).
		Where("fork=?", false).
		Update("status", types.TX_STATUS_FAILED).Error

	return err
}

// 根据hash&logIndex查找唯一tx
func (s *RdsServiceImpl) FindTxEntity(txhash string, logIndex int64) (TransactionEntity, error) {
	var tx TransactionEntity

	err := s.db.Where("tx_hash=?", txhash).
		Where("tx_log_index=?", logIndex).
		Where("fork=?", false).
		First(&tx).Error

	return tx, err
}

func (s *RdsServiceImpl) RollBackTxEntity(from, to int64) error {
	return s.db.Model(&TransactionEntity{}).Where("block_number > ? and block_number <= ?", from, to).Update("fork", true).Error
}
