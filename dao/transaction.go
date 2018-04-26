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

/*
package dao

import (
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

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

// convert types/transaction to dao/transaction
// todo(fuk): judge nil fields
func (tx *Transaction) ConvertDown(src *types.Transaction) error {
	tx.Protocol = src.Protocol.Hex()
	tx.Owner = src.Owner.Hex()
	tx.From = src.From.Hex()
	tx.To = src.To.Hex()
	tx.RawFrom = src.RawFrom.Hex()
	tx.RawTo = src.RawTo.Hex()
	tx.TxHash = src.TxHash.Hex()
	tx.Content = string(src.Content)
	tx.BlockNumber = src.BlockNumber.Int64()
	tx.Value = src.Value.String()
	tx.Type = src.TypeValue()
	tx.Status = src.StatusValue()
	tx.TxIndex = src.TxIndex
	tx.LogIndex = src.TxLogIndex
	tx.CreateTime = src.CreateTime
	tx.UpdateTime = src.UpdateTime
	tx.Symbol = src.Symbol
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
	dst.Owner = common.HexToAddress(tx.Owner)
	dst.From = common.HexToAddress(tx.From)
	dst.To = common.HexToAddress(tx.To)
	dst.RawFrom = common.HexToAddress(tx.RawFrom)
	dst.RawTo = common.HexToAddress(tx.RawTo)
	dst.TxHash = common.HexToHash(tx.TxHash)
	dst.Content = []byte(tx.Content)
	dst.BlockNumber = big.NewInt(tx.BlockNumber)
	dst.TxIndex = tx.TxIndex
	dst.TxLogIndex = tx.LogIndex
	dst.Value, _ = new(big.Int).SetString(tx.Value, 0)
	dst.Type = types.TxType(tx.Type)
	dst.Status = types.TxStatus(tx.Status)
	dst.CreateTime = tx.CreateTime
	dst.UpdateTime = tx.UpdateTime
	dst.Symbol = tx.Symbol
	dst.GasLimit, _ = new(big.Int).SetString(tx.GasLimit, 0)
	dst.GasUsed, _ = new(big.Int).SetString(tx.GasUsed, 0)
	dst.GasPrice, _ = new(big.Int).SetString(tx.GasPrice, 0)
	dst.Nonce, _ = new(big.Int).SetString(tx.Nonce, 0)

	return nil
}

func (s *RdsServiceImpl) GetPendingTransactionsByOwner(owner string) ([]Transaction, error) {
	var txs []Transaction

	err := s.db.Where("tx_from = ? or tx_to = ?", owner, owner).
		Where("status=?", uint8(types.TX_STATUS_PENDING)).
		Where("fork=?", false).
		Find(&txs).Error

	return txs, err
}

func (s *RdsServiceImpl) GetTransactionCount(owner string, symbol string, status []types.TxStatus, typs []types.TxType) (int, error) {
	var (
		number int
		err    error
	)

	query := make(map[string]interface{})
	query["symbol"] = symbol
	query = combineTypeAndStatus(query, status, typs)

	err = s.db.Model(&Transaction{}).
		Where("tx_from=? or tx_to=?", owner, owner).
		Where(query).
		Where("fork=?", false).
		Select("count(distinct(tx_hash))").
		Count(&number).Error

	return number, err
}

func (s *RdsServiceImpl) GetTransactionHashs(owner string, symbol string, status []types.TxStatus, typs []types.TxType, limit, offset int) ([]string, error) {
	var (
		hashs []string
		err   error
	)

	query := make(map[string]interface{})
	query["symbol"] = symbol
	query = combineTypeAndStatus(query, status, typs)

	err = s.db.Model(&Transaction{}).
		Where("tx_from=? or tx_to=?", owner, owner).
		Where(query).
		Where("fork=?", false).
		Order("create_time DESC").
		Limit(limit).Offset(offset).Pluck("distinct(tx_hash)", &hashs).Error

	return hashs, err
}

func combineTypeAndStatus(query map[string]interface{}, status []types.TxStatus, typs []types.TxType) map[string]interface{} {
	if len(status) == 1 {
		query["status"] = status[0]
	} else if len(status) > 1 {
		query["status in (?)"] = status
	}

	if len(typs) == 1 {
		query["tx_type"] = typs[0]
	} else if len(typs) > 1 {
		query["tx_type in (?)"] = status
	}

	return query
}

func (s *RdsServiceImpl) GetPendingTransaction(hash common.Hash, rawFrom common.Address, nonce *big.Int) (Transaction, error) {
	var tx Transaction

	err := s.db.Where("tx_hash=?", hash.Hex()).
		Where("raw_from=?", rawFrom.Hex()).
		Where("nonce=?", nonce.String()).
		Where("status=?", uint8(types.TX_STATUS_PENDING)).
		Where("fork=?", false).
		First(&tx).Error

	return tx, err
}

func (s *RdsServiceImpl) GetTransactionsBySenderNonce(rawFrom common.Address, nonce *big.Int) ([]Transaction, error) {
	var txs []Transaction

	err := s.db.Where("raw_from=?", rawFrom.Hex()).
		Where("nonce=?", nonce.String()).
		Where("fork=?", false).
		Find(&txs).Error

	return txs, err
}

func (s *RdsServiceImpl) DeletePendingTransaction(hash common.Hash, rawFrom common.Address, nonce *big.Int) error {
	return s.db.Where("tx_hash=?", hash.Hex()).
		Where("raw_from=?", rawFrom.Hex()).
		Where("nonce=?", nonce.String()).
		Where("status=?", uint8(types.TX_STATUS_PENDING)).
		Where("fork=?", false).
		Delete(&Transaction{}).Error
}

func (s *RdsServiceImpl) DeletePendingTransactions(rawFrom common.Address, nonce *big.Int) error {
	return s.db.Where("raw_from=?", rawFrom.Hex()).
		Where("nonce=?", nonce.String()).
		Where("status=?", uint8(types.TX_STATUS_PENDING)).
		Where("fork=?", false).
		Delete(&Transaction{}).Error
}

func (s *RdsServiceImpl) GetTrxByHashes(hashes []string) ([]Transaction, error) {
	var trxs []Transaction
	err := s.db.Where("tx_hash in (?)", hashes).Order("create_time DESC").Where("fork=?", false).Find(&trxs).Error
	return trxs, err
}

func (s *RdsServiceImpl) PendingTransactions(query map[string]interface{}) ([]Transaction, error) {
	var txs []Transaction
	err := s.db.Where(query).Where("fork=?", false).Find(&txs).Error
	return txs, err
}

func (s *RdsServiceImpl) RollBackTransaction(from, to int64) error {
	return s.db.Model(&Transaction{}).Where("block_number > ? and block_number <= ?", from, to).Update("fork", true).Error
}
