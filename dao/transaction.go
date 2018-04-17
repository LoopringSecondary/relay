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
	Protocol    string `gorm:"column:protocol;type:varchar(42)"`
	From        string `gorm:"column:tx_from;type:varchar(42)"`
	To          string `gorm:"column:tx_to;type:varchar(42)"`
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

// convert types/transaction to dao/transaction
// todo(fuk): judge nil fields
func (tx *Transaction) ConvertDown(src *types.Transaction) error {
	tx.Protocol = src.Protocol.Hex()
	tx.From = src.From.Hex()
	tx.To = src.To.Hex()
	tx.TxHash = src.TxHash.Hex()
	tx.Content = string(src.Content)
	tx.BlockNumber = src.BlockNumber.Int64()
	tx.Value = src.Value.String()
	tx.Type = uint8(src.Type)
	tx.Status = uint8(src.Status)
	tx.TxIndex = src.TxIndex
	tx.LogIndex = src.LogIndex
	tx.CreateTime = src.CreateTime
	tx.UpdateTime = src.UpdateTime
	tx.GasLimit = src.GasLimit.String()
	tx.GasUsed = src.GasUsed.String()
	tx.GasPrice = src.GasPrice.String()
	tx.Nonce = src.Nonce.String()
	tx.Fork = false

	return nil
}

// convert dao/transaction to types/transaction
func (tx *Transaction) ConvertUp(dst *types.Transaction) error {
	dst.Protocol = common.HexToAddress(tx.Protocol)
	dst.From = common.HexToAddress(tx.From)
	dst.To = common.HexToAddress(tx.To)
	dst.TxHash = common.HexToHash(tx.TxHash)
	dst.Content = []byte(tx.Content)
	dst.BlockNumber = big.NewInt(tx.BlockNumber)
	dst.TxIndex = tx.TxIndex
	dst.LogIndex = tx.LogIndex
	dst.Value, _ = new(big.Int).SetString(tx.Value, 0)
	dst.Type = types.TxType(tx.Type)
	dst.Status = types.TxStatus(tx.Status)
	dst.CreateTime = tx.CreateTime
	dst.UpdateTime = tx.UpdateTime
	dst.GasLimit, _ = new(big.Int).SetString(tx.GasLimit, 0)
	dst.GasUsed, _ = new(big.Int).SetString(tx.GasUsed, 0)
	dst.GasPrice, _ = new(big.Int).SetString(tx.GasPrice, 0)
	dst.Nonce, _ = new(big.Int).SetString(tx.Nonce, 0)

	return nil
}

func (s *RdsServiceImpl) FindTransactionWithoutLogIndex(txhash string) (Transaction, error) {
	var (
		tx  Transaction
		err error
	)
	err = s.db.Where("tx_hash=?", txhash).First(&tx).Error
	return tx, err
}

func (s *RdsServiceImpl) FindTransactionWithLogIndex(txhash string, logIndex int64) (Transaction, error) {
	var (
		tx  Transaction
		err error
	)
	err = s.db.Where("tx_hash=? and tx_log_index=?", txhash, logIndex).First(&tx).Error
	return tx, err
}

func (s *RdsServiceImpl) GetTrxByHashes(hashes []string) ([]Transaction, error) {
	var trxs []Transaction
	err := s.db.Where("tx_hash in (?)", hashes).Where("fork=?", false).Find(&trxs).Error
	return trxs, err
}

func (s *RdsServiceImpl) GetPendingTransactions(owner string, status types.TxStatus) ([]Transaction, error) {
	var txs []Transaction
	err := s.db.Where("from = ? or to = ?", owner, owner).
		Where("status=?", status).
		Where("fork=?", false).
		Find(&txs).Error
	return txs, err
}

func (s *RdsServiceImpl) GetMinedTransactionCount(owner string, protocol string, status []types.TxStatus) (int, error) {
	var (
		number int
		err    error
	)

	err = s.db.Model(&Transaction{}).Select("distinct(tx_hash)").
		Where("tx_from=? or tx_to=?", owner, owner).
		Where("protocol=?", protocol).
		Where("status in (?)", status).
		Where("fork=?", false).
		Count(&number).Error

	return number, err
}

func (s *RdsServiceImpl) GetMinedTransactionHashs(owner string, protocol string, status []types.TxStatus, limit, offset int) ([]string, error) {
	var (
		hashs []string
		err   error
	)

	err = s.db.Model(&Transaction{}).Pluck("distinct(tx_hash)", &hashs).
		Where("from=? or to=?", owner, owner).
		Where("protocol=?", protocol).
		Where("status in (?)", status).
		Where("fork=?", false).
		Limit(limit).Offset(offset).Error

	return hashs, err
}

func (s *RdsServiceImpl) RollBackTransaction(from, to int64) error {
	return s.db.Model(&Transaction{}).Where("block_number > ? and block_number <= ?", from, to).Update("fork", true).Error
}
