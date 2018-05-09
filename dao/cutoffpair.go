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
	"encoding/json"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type CutOffPairEvent struct {
	ID              int    `gorm:"column:id;primary_key;"`
	Protocol        string `gorm:"column:contract_address;type:varchar(42)"`
	DelegateAddress string `gorm:"column:delegate_address;type:varchar(42)"`
	Owner           string `gorm:"column:owner;type:varchar(42)"`
	Token1          string `gorm:"column:token1;type:varchar(42)"`
	Token2          string `gorm:"column:token2;type:varchar(42)"`
	TxHash          string `gorm:"column:tx_hash;type:varchar(82)"`
	OrderHashList   string `gorm:"column:order_hash_list;type:text"`
	BlockNumber     int64  `gorm:"column:block_number"`
	LogIndex        int64  `gorm:"column:log_index"`
	Cutoff          int64  `gorm:"column:cutoff"`
	CreateTime      int64  `gorm:"column:create_time"`
	Fork            bool   `gorm:"column:fork"`
}

// convert types/cutoffPairEvent to dao/cutoffPairEvent
func (e *CutOffPairEvent) ConvertDown(src *types.CutoffPairEvent) error {
	e.Owner = src.Owner.Hex()
	e.Protocol = src.Protocol.Hex()
	e.DelegateAddress = src.DelegateAddress.Hex()
	e.TxHash = src.TxHash.Hex()
	e.Token1 = src.Token1.Hex()
	e.Token2 = src.Token2.Hex()
	e.Cutoff = src.Cutoff.Int64()
	e.LogIndex = src.TxLogIndex
	e.BlockNumber = src.BlockNumber.Int64()
	e.CreateTime = src.BlockTime

	list := []string{}
	for _, v := range src.OrderHashList {
		list = append(list, v.Hex())
	}
	bs, _ := json.Marshal(list)
	e.OrderHashList = string(bs)

	return nil
}

// convert dao/cutoffPairEvent to types/cutoffPairEvent
func (e *CutOffPairEvent) ConvertUp(dst *types.CutoffPairEvent) error {
	dst.Owner = common.HexToAddress(e.Owner)
	dst.Protocol = common.HexToAddress(e.Protocol)
	dst.DelegateAddress = common.HexToAddress(e.DelegateAddress)
	dst.TxHash = common.HexToHash(e.TxHash)
	dst.Token1 = common.HexToAddress(e.Token1)
	dst.Token2 = common.HexToAddress(e.Token2)
	dst.BlockNumber = big.NewInt(e.BlockNumber)
	dst.Cutoff = big.NewInt(e.Cutoff)
	dst.TxLogIndex = e.LogIndex
	dst.BlockTime = e.CreateTime
	dst.OrderHashList = []common.Hash{}

	list := []string{}
	json.Unmarshal([]byte(e.OrderHashList), &list)
	for _, v := range list {
		dst.OrderHashList = append(dst.OrderHashList, common.HexToHash(v))
	}

	return nil
}

func (s *RdsServiceImpl) GetCutoffPairEvent(txhash common.Hash) (CutOffPairEvent, error) {
	var event CutOffPairEvent
	err := s.db.Where("tx_hash=?", txhash.Hex()).Where("fork=?", false).First(&event).Error
	return event, err
}

func (s *RdsServiceImpl) GetCutoffPairForkEvents(from, to int64) ([]CutOffPairEvent, error) {
	var (
		list []CutOffPairEvent
		err  error
	)

	err = s.db.Where("block_number > ? and block_number <= ?", from, to).
		Where("fork = ?", false).
		Find(&list).Error

	return list, err
}

func (s *RdsServiceImpl) RollBackCutoffPair(from, to int64) error {
	return s.db.Model(&CutOffPairEvent{}).Where("block_number > ? and block_number <= ?", from, to).Update("fork", true).Error
}
