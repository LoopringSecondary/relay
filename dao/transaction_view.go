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

type TransactionView struct {
	ID          int    `gorm:"column:id;primary_key;"`
	Symbol      string `gorm:"column:symbol;type:varchar(20)"`
	Owner       string `gorm:"column:owner;type:varchar(42)"`
	TxHash      string `gorm:"column:tx_hash;type:varchar(82)"`
	BlockNumber int64  `gorm:"column:block_number"`
	LogIndex    int64  `gorm:"column:tx_log_index"`
	Amount      string `gorm:"column:amount;type:varchar(40)"`
	Nonce       int64  `gorm:"column:nonce"`
	Type        uint8  `gorm:"column:tx_type"`
	Status      uint8  `gorm:"column:status"`
	CreateTime  int64  `gorm:"column:create_time"`
	UpdateTime  int64  `gorm:"column:update_time"`
	Fork        bool   `gorm:"column:fork"`
}

// convert types/transaction to dao/transaction
// todo(fuk): judge nil fields
func (tx *TransactionView) ConvertDown(src *txtyp.TransactionView) error {
	tx.Symbol = src.Symbol
	tx.Owner = src.Owner.Hex()
	tx.TxHash = src.TxHash.Hex()
	tx.BlockNumber = src.BlockNumber
	tx.LogIndex = src.LogIndex
	tx.Amount = src.Amount.String()
	tx.Nonce = src.Nonce.Int64()
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
	dst.BlockNumber = tx.BlockNumber
	dst.LogIndex = tx.LogIndex
	dst.Amount, _ = new(big.Int).SetString(tx.Amount, 0)
	dst.Nonce = big.NewInt(tx.Nonce)
	dst.Type = txtyp.TxType(tx.Type)
	dst.Status = types.TxStatus(tx.Status)
	dst.CreateTime = tx.CreateTime
	dst.UpdateTime = tx.UpdateTime

	return nil
}

//////////// write related

// 更新交易发起者相同nonce下tx为failed
func (s *RdsServiceImpl) SetPendingTxViewFailed(hashlist []string) error {
	err := s.db.Model(&TransactionView{}).
		Where("tx_hash in (?)", hashlist).
		Where("status=?", types.TX_STATUS_PENDING).
		Where("fork=?", false).
		Update("status", types.TX_STATUS_FAILED).Error

	return err
}

// 根据hash删除pending tx
func (s *RdsServiceImpl) DelPendingTxView(hash string) error {
	err := s.db.Where("tx_hash=?", hash).
		Where("status=?", types.TX_STATUS_PENDING).
		Where("fork=?", false).
		Delete(&TransactionView{}).Error
	return err
}

//////////// read related
func (s *RdsServiceImpl) GetTxViewByOwnerAndHashs(owner string, hashs []string) ([]TransactionView, error) {
	var txs []TransactionView

	err := s.db.Where("owner=?", owner).
		Where("tx_hash in (?)", hashs).
		Where("fork=?", false).
		Find(&txs).Error

	return txs, err
}

func (s *RdsServiceImpl) GetPendingTxViewByOwner(owner string) ([]TransactionView, error) {
	var txs []TransactionView

	err := s.db.Where("owner=?", owner).
		Where("status=?", types.TX_STATUS_PENDING).
		Where("fork=?", false).
		Order("update_time DESC").
		Find(&txs).Error

	return txs, err
}

func (s *RdsServiceImpl) GetTxViewCountByOwner(owner string, symbol string, status types.TxStatus, typ txtyp.TxType) (int, error) {
	var number int

	query := assembleTxViewQuery(owner, symbol, status, typ)

	err := s.db.Model(&TransactionView{}).Where(query).Count(&number).Error

	return number, err
}

func (s *RdsServiceImpl) GetTxViewByOwner(owner string, symbol string, status types.TxStatus, typ txtyp.TxType, limit, offset int) ([]TransactionView, error) {
	var txs []TransactionView

	query := assembleTxViewQuery(owner, symbol, status, typ)

	err := s.db.Where(query).Order("update_time DESC").Limit(limit).Offset(offset).Find(&txs).Error

	return txs, err
}

func (s *RdsServiceImpl) RollBackTxView(from, to int64) error {
	return s.db.Model(&TransactionView{}).Where("block_number > ? and block_number <= ?", from, to).Update("fork", true).Error
}

func assembleTxViewQuery(owner, symbol string, status types.TxStatus, typ txtyp.TxType) map[string]interface{} {
	query := make(map[string]interface{})
	query["owner"] = owner
	query["symbol"] = symbol
	query["fork"] = false
	if status != types.TX_STATUS_UNKNOWN {
		query["status"] = status
	}
	if typ != txtyp.TX_TYPE_UNKNOWN {
		query["tx_type"] = typ
	}

	return query
}
