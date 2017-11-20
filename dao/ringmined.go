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
	"math/big"
)

type RingMined struct {
	ID                 int    `gorm:"column:id;primary_key"`
	RingIndex          []byte `gorm:"column:ring_index;type:varchar(30)"`
	RingHash           string `gorm:"column:ring_hash;type:varchar(82);unique_index"`
	Miner              string `gorm:"column:miner;type:varchar(42);"`
	FeeRecipient       string `gorm:"column:fee_recipient;type:varchar(42)"`
	IsRinghashReserved bool   `gorm:"column:is_ring_hash_reserved;"`
	BlockNumber        int64  `gorm:"column:block_number;type:bigint"`
	Time               int64  `gorm:"column:time;type:bigint"`
}

func (r *RingMined) ConvertDown(event *types.RingMinedEvent) error {
	var err error
	r.RingIndex, err = event.RingIndex.MarshalText()
	if err != nil {
		return err
	}

	r.Miner = event.Miner.Hex()
	r.FeeRecipient = event.FeeRecipient.Hex()
	r.RingHash = event.Ringhash.Hex()
	r.IsRinghashReserved = event.IsRinghashReserved
	r.BlockNumber = event.Blocknumber.Int64()
	r.Time = event.Time.Int64()

	return nil
}

func (r *RingMined) ConvertUp(event *types.RingMinedEvent) error {
	event.RingIndex = new(types.Big)
	if err := event.RingIndex.UnmarshalText(r.RingIndex); err != nil {
		return err
	}

	event.Ringhash = types.HexToHash(r.RingHash)
	event.Miner = types.HexToAddress(r.Miner)
	event.FeeRecipient = types.HexToAddress(r.FeeRecipient)
	event.IsRinghashReserved = r.IsRinghashReserved
	event.Blocknumber = types.NewBigPtr(big.NewInt(r.BlockNumber))
	event.Time = types.NewBigPtr(big.NewInt(r.Time))

	return nil
}

func (s *RdsServiceImpl) FindRingMinedByRingHash(ringhash string) (*RingMined, error) {
	var (
		model RingMined
		err   error
	)

	err = s.db.Where("ring_hash = ?", ringhash).First(&model).Error

	return &model, err
}
