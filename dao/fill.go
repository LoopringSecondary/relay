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
	"github.com/Loopring/relay/market"
)

type FillEvent struct {
	ID                    int    `gorm:"column:id;primary_key;"`
	Protocol              string `gorm:"column:contract_address;type:varchar(42)"`
	Owner                 string `gorm:"column:owner;type:varchar(42)"`
	RingIndex             int64  `gorm:"column:ring_index;"`
	BlockNumber           int64  `gorm:"column:block_number"`
	CreateTime            int64  `gorm:"column:create_time"`
	RingHash              string `gorm:"column:ring_hash;varchar(82)"`
	TxHash                string `gorm:"column:tx_hash;type:varchar(82)"`
	PreOrderHash          string `gorm:"column:pre_order_hash;varchar(82)"`
	NextOrderHash         string `gorm:"column:next_order_hash;varchar(82)"`
	OrderHash             string `gorm:"column:order_hash;type:varchar(82)"`
	AmountS               []byte `gorm:"column:amount_s;type:varchar(30)"`
	AmountB               []byte `gorm:"column:amount_b;type:varchar(30)"`
	TokenS                string `gorm:"column:token_s;type:varchar(42)"`
	TokenB                string `gorm:"column:token_b;type:varchar(42)"`
	LrcReward             []byte `gorm:"column:lrc_reward;type:varchar(30)"`
	LrcFee                []byte `gorm:"column:lrc_fee;type:varchar(30)"`
	MarginSplitPercentage int    `gorm:"column:margin_split_percentage"`
	IsDeleted             bool   `gorm:"column:is_deleted"`
	Market        string `gorm:"column:market;type:varchar(42)"`
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

	f.Protocol = src.ContractAddress.Hex()
	f.RingIndex = src.RingIndex.Int64()
	f.BlockNumber = src.Blocknumber.Int64()
	f.CreateTime = src.Time.Int64()
	f.RingHash = src.Ringhash.Hex()
	f.TxHash = src.TxHash.Hex()
	f.PreOrderHash = src.PreOrderHash.Hex()
	f.NextOrderHash = src.NextOrderHash.Hex()
	f.OrderHash = src.OrderHash.Hex()
	f.TokenS = src.TokenS.Hex()
	f.TokenB = src.TokenB.Hex()
	f.Owner = src.Owner.Hex()
	f.MarginSplitPercentage = src.MarginSplitPercentage
	f.IsDeleted = src.IsDeleted
	f.Market, _ = market.WrapMarketByAddress(src.TokenS.String(), src.TokenB.String())

	return nil
}

func (s *RdsServiceImpl) FindFillEventByRinghashAndOrderhash(ringhash, orderhash common.Hash) (*FillEvent, error) {
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

func (s *RdsServiceImpl) QueryRecentFills(tokenS, tokenB, owner string, start int64, end int64) (fills []FillEvent, err error) {

	if tokenS != "" {
		s.db = s.db.Where("token_s = ", tokenS)
	}

	if tokenB != "" {
		s.db = s.db.Where("token_b = ", tokenB)
	}

	if owner != "" {
		s.db = s.db.Where("owner = ", owner)
	}

	if start != 0 {
		s.db = s.db.Where("create_time > ", start)
	}

	err = s.db.Order("create_time desc").Limit(100).Find(&fills).Error
	return
}

func (s *RdsServiceImpl) RollBackFill(from, to int64) error {
	return s.db.Model(&FillEvent{}).Where("block_number > ? and block_number <= ?", from, to).UpdateColumn("is_deleted", true).Error
}
