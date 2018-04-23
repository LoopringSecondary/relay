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
	Symbol      string `gorm:"column:symbol;type:varchar(20)"`
	Owner       string `gorm:"column:owner;type:varchar(42)"`
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
	tx.Owner = src.Owner.Hex()
	tx.From = src.From.Hex()
	tx.To = src.To.Hex()
	tx.TxHash = src.TxHash.Hex()
	tx.Content = string(src.Content)
	tx.BlockNumber = src.BlockNumber.Int64()
	tx.Value = src.Value.String()
	tx.Type = src.TypeValue()
	tx.Status = src.StatusValue()
	tx.TxIndex = src.TxIndex
	tx.LogIndex = src.LogIndex
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
	dst.Symbol = tx.Symbol
	dst.GasLimit, _ = new(big.Int).SetString(tx.GasLimit, 0)
	dst.GasUsed, _ = new(big.Int).SetString(tx.GasUsed, 0)
	dst.GasPrice, _ = new(big.Int).SetString(tx.GasPrice, 0)
	dst.Nonce, _ = new(big.Int).SetString(tx.Nonce, 0)

	return nil
}

// value,status可能会变更
func (s *RdsServiceImpl) SaveTransaction(latest *Transaction) error {
	var (
		current Transaction
		query   string
		args    []interface{}
	)

	switch types.TxType(latest.Type) {
	case types.TX_TYPE_SELL, types.TX_TYPE_BUY:
		query = "tx_hash=? and tx_from=? and tx_to=? and tx_type=?"
		args = append(args, latest.TxHash, latest.From, latest.To, latest.Type)

	case types.TX_TYPE_CANCEL_ORDER:
		query = "tx_hash=? and tx_type=?"
		args = append(args, latest.TxHash, latest.Type)

	case types.TX_TYPE_CONVERT_INCOME, types.TX_TYPE_CONVERT_OUTCOME:
		query = "tx_hash=? and tx_type=?"
		args = append(args, latest.TxHash, latest.Type)

	case types.TX_TYPE_APPROVE:
		query = "tx_hash=? and tx_from=? and tx_to=? and tx_type=?"
		args = append(args, latest.TxHash, latest.From, latest.To, latest.Type)

	case types.TX_TYPE_SEND, types.TX_TYPE_RECEIVE:
		return s.processTransfer(latest)

	case types.TX_TYPE_CUTOFF:
		query = "tx_hash=? and tx_type=?"
		args = append(args, latest.TxHash, latest.Type)

	case types.TX_TYPE_CUTOFF_PAIR:
		query = "tx_hash=? and tx_type=?"
		args = append(args, latest.TxHash, latest.Type)

	case types.TX_TYPE_UNSUPPORTED_CONTRACT:
		query = "tx_hash=? and tx_type=?"
		args = append(args, latest.TxHash, latest.Type)
	}

	if len(query) == 0 || len(args) == 0 {
		return nil
	}

	err := s.db.Where(query, args...).Where("fork=?", false).Find(&current).Error
	if err != nil {
		return s.db.Create(latest).Error
	}

	if latest.Value != current.Value || latest.Status != current.Status {
		latest.ID = current.ID
		latest.CreateTime = current.CreateTime
		if err := s.db.Save(latest).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *RdsServiceImpl) processTransfer(latest *Transaction) error {
	var (
		current Transaction
	)

	// delete pending then create new item
	if err := s.db.Where("tx_hash=? and owner=? and `status`=?", latest.TxHash, latest.Owner, types.TX_STATUS_PENDING).Find(&current).Error; err == nil {
		s.db.Delete(&current)
		return s.db.Create(latest).Error
	}

	// select mined transaction then create or update
	err := s.db.Where("tx_hash=? and owner=? and tx_log_index=?", latest.TxHash, latest.Owner, latest.LogIndex).Find(&current).Error
	if err == nil {
		latest.ID = current.ID
		return s.db.Save(latest).Error
	}
	return s.db.Create(latest).Error
}

func (s *RdsServiceImpl) TransactionPageQuery(query map[string]interface{}, pageIndex, pageSize int) (PageResult, error) {
	var (
		trxs       []Transaction
		err        error
		data       = make([]interface{}, 0)
		pageResult PageResult
	)

	if pageIndex <= 0 {
		pageIndex = 1
	}

	if pageSize <= 0 {
		pageSize = 20
	}

	if err = s.db.Where(query).Where("fork=?", false).Offset((pageIndex - 1) * pageSize).Order("create_time DESC").Limit(pageSize).Find(&trxs).Error; err != nil {
		return pageResult, err
	}

	for _, v := range trxs {
		data = append(data, v)
	}

	pageResult = PageResult{data, pageIndex, pageSize, 0}

	err = s.db.Model(&Transaction{}).Where("fork=?", false).Where(query).Count(&pageResult.Total).Error
	if err != nil {
		return pageResult, err
	}

	return pageResult, err
}

func (s *RdsServiceImpl) GetTrxByHashes(hashes []string) ([]Transaction, error) {
	var trxs []Transaction
	err := s.db.Where("tx_hash in (?)", hashes).Where("fork=?", false).Find(&trxs).Error
	return trxs, err
}

func (s *RdsServiceImpl) PendingTransactions(query map[string]interface{}) ([]Transaction, error) {
	var txs []Transaction
	err := s.db.Where(query).Where("fork=?", false).Find(&txs).Error
	return txs, err
}

func (s *RdsServiceImpl) UpdatePendingTransactionsByOwner(owner common.Address, status uint8) error {
	return s.db.Model(&Transaction{}).Where("owner=?", owner.Hex()).Where("status=?", uint8(types.TX_STATUS_PENDING)).
		Where("fork=?", false).
		Update("status", status).Error
}

func (s *RdsServiceImpl) RollBackTransaction(from, to int64) error {
	return s.db.Model(&Transaction{}).Where("block_number > ? and block_number <= ?", from, to).Update("fork", true).Error
}
