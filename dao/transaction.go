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
	"time"
)

type Transaction struct {
	ID          int    `gorm:"column:id;primary_key;"`
	Protocol    string `gorm:"column:protocol;type:varchar(42)"`
	Owner       string `gorm:"column:owner;type:varchar(42)"`
	From        string `gorm:"column:tx_from;type:varchar(42)"`
	To          string `gorm:"column:tx_to;type:varchar(42)"`
	TxHash      string `gorm:"column:tx_hash;type:varchar(82)"`
	Content     string `gorm:"column:content;type:text"`
	BlockNumber int64  `gorm:"column:block_number"`
	LogIndex    int64  `gorm:"column:tx_log_index"`
	Value       string `gorm:"column:amount;type:varchar(30)"`
	Type        uint8  `gorm:"column:tx_type"`
	Status      uint8  `gorm:"column:status"`
	CreateTime  int64  `gorm:"column:create_time"`
	UpdateTime  int64  `gorm:"column:update_time"`
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
	tx.Type = src.Type
	tx.Status = src.Status
	tx.LogIndex = src.LogIndex
	tx.CreateTime = src.CreateTime
	tx.UpdateTime = src.UpdateTime

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
	dst.LogIndex = tx.LogIndex
	dst.Value, _ = new(big.Int).SetString(tx.Value, 0)
	dst.Type = tx.Type
	dst.Status = tx.Status
	dst.CreateTime = tx.CreateTime
	dst.UpdateTime = tx.UpdateTime

	return nil
}

// value,status可能会变更
func (s *RdsServiceImpl) SaveTransaction(latest *Transaction) error {
	var (
		current Transaction
		query   string
		args    []interface{}
	)

	switch latest.Type {
	case types.TX_TYPE_SELL, types.TX_TYPE_BUY:
		query = "tx_hash=? and tx_from=? and tx_to=? and tx_type=?"
		args = append(args, latest.TxHash, latest.From, latest.To, latest.Type)

	case types.TX_TYPE_CANCEL_ORDER:
		query = "tx_hash=? and tx_type=?"
		args = append(args, latest.TxHash, latest.Type)

	case types.TX_TYPE_WRAP, types.TX_TYPE_UNWRAP:
		query = "tx_hash=? and owner=? and tx_type=?"
		args = append(args, latest.TxHash, latest.Owner, latest.Type)

	case types.TX_TYPE_APPROVE:
		query = "tx_hash=? and tx_from=? and tx_to=? and tx_type=?"
		args = append(args, latest.TxHash, latest.From, latest.To, latest.Type)

	case types.TX_TYPE_SEND, types.TX_TYPE_RECEIVE:
		query = "tx_hash=? and tx_log_index=? and tx_type=?"
		args = append(args, latest.TxHash, latest.LogIndex, latest.Type)

	case types.TX_TYPE_CUTOFF:
		query = "tx_hash=?"
		args = append(args, latest.TxHash)
	}

	if len(query) == 0 || len(args) == 0 {
		return nil
	}

	err := s.db.Where(query, args...).Find(&current).Error
	if err != nil {
		return s.db.Create(latest).Error
	}

	if latest.Value != current.Value || latest.Status != current.Status {
		latest.ID = current.ID
		latest.UpdateTime = time.Now().Unix()
		if err := s.db.Save(latest).Error; err != nil {
			return err
		}
	}

	return nil
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

	if err = s.db.Where(query).Offset((pageIndex - 1) * pageSize).Order("create_time DESC").Limit(pageSize).Find(&trxs).Error; err != nil {
		return pageResult, err
	}

	for _, v := range trxs {
		data = append(data, v)
	}

	pageResult = PageResult{data, pageIndex, pageSize, 0}

	err = s.db.Model(&Transaction{}).Where(query).Count(&pageResult.Total).Error
	if err != nil {
		return pageResult, err
	}

	return pageResult, err
}

func (s *RdsServiceImpl) GetTrxByHashes(hashes []string) ([]Transaction, error) {
	var trxs []Transaction
	err := s.db.Where("tx_hash in (?)", hashes).Find(&trxs).Error
	return trxs, err
}
