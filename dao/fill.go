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
)

type FillEvent struct {
	ID            int    `gorm:"column:id;primary_key;"`
	RingIndex     int64  `gorm:"column:ring_index;"`
	BlockNumber   int64  `gorm:"column:block_number"`
	CreateTime    int64  `gorm:"column:create_time"`
	RingHash      string `gorm:"column:ring_hash;varchar(82)"`
	PreOrderHash  string `gorm:"column:pre_order_hash;varchar(82)"`
	NextOrderHash string `gorm:"column:next_order_hash;varchar(82)"`
	OrderHash     string `gorm:"column:order_hash;type:varchar(82)"`
	AmountS       []byte `gorm:"column:amount_s;type:varchar(30)"`
	AmountB       []byte `gorm:"column:amount_b;type:varchar(30)"`
	TokenS        string  `gorm:"column:token_s;type:varchar(42)"`
	TokenB        string  `gorm:"column:token_b;type:varchar(42)"`
	LrcReward     []byte `gorm:"column:lrc_reward;type:varchar(30)"`
	LrcFee        []byte `gorm:"column:lrc_fee;type:varchar(30)"`
	IsDeleted     bool   `gorm:"column:is_deleted"`
}

// convert chainclient/orderFilledEvent to dao/fill
func (f *FillEvent) ConvertDown(src *types.OrderFilledEvent) error {
	var err error

	if f.AmountS, err = src.AmountS.MarshalText(); err != nil {
		return err
	}
	if f.AmountB, err = src.AmountB.MarshalText(); err != nil {
		return err
	}
	if f.LrcReward, err = src.LrcReward.MarshalText(); err != nil {
		return err
	}
	if f.LrcFee, err = src.LrcFee.MarshalText(); err != nil {
		return err
	}

	f.RingIndex = src.RingIndex.Int64()
	f.BlockNumber = src.Blocknumber.Int64()
	f.CreateTime = src.Time.Int64()
	f.RingHash = src.Ringhash.Hex()
	f.PreOrderHash = src.PreOrderHash.Hex()
	f.NextOrderHash = src.NextOrderHash.Hex()
	f.OrderHash = src.OrderHash.Hex()
	f.IsDeleted = src.IsDeleted

	return nil
}

func (s *RdsServiceImpl) FindFillEventByRinghashAndOrderhash(ringhash, orderhash types.Hash) (*FillEvent, error) {
	var (
		fill FillEvent
		err  error
	)
	err = s.db.Where("ring_hash = ? and order_hash = ? and is_deleted = false", ringhash.Hex(), orderhash.Hex()).First(&fill).Error

	return &fill, err
}

func (s *RdsServiceImpl) FirstPreMarket(tokenS string, tokenB string) (fill FillEvent, err error) {
	err = s.db.First(&fill).Error
	return
}

func (s *RdsServiceImpl) QueryRecentFills(tokenS string, tokenB string, start int64, end int64) (fills [] FillEvent, err error) {
	if end != 0 {
		err = s.db.Where("token_s = ? and token_b = ? and create_time > ? and create_time <= ?", tokenS, tokenB, start, end).Order("create_time desc").Limit(100).Find(&fills).Error
	} else {
		err = s.db.Where("token_s = ? and token_b = ? and create_time > ?", tokenS, tokenB, start).Order("create_time desc").Limit(100).Find(&fills).Error
	}
	return
}

func (s *RdsServiceImpl) RollBackFill(from, to int64) error {
	return s.db.Where("block_number > ? and block_number <= ?", from, to).UpdateColumn("is_deleted = ?", true).Error
}
