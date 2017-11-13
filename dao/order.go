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
	"errors"
	"github.com/Loopring/ringminer/types"
	"math/big"
)

// order amountS 上限1e30

type Order struct {
	ID                    int     `gorm:"column:id;primary_key;"`
	Protocol              string  `gorm:"column:protocol;type:varchar(42)"`
	Owner                 string  `gorm:"column:owner;type:varchar(42)"`
	OrderHash             string  `gorm:"column:order_hash;type:varchar(82);unique_index"`
	TokenS                string  `gorm:"column:token_s;type:varchar(42)"`
	TokenB                string  `gorm:"column:token_b;type:varchar(42)"`
	AmountS               []byte  `gorm:"column:amount_s;type:varchar(30)"`
	AmountB               []byte  `gorm:"column:amount_b;type:varchar(30)"`
	CreateTime            int64   `gorm:"column:create_time"`
	Ttl                   int64   `gorm:"column:ttl"`
	Salt                  int64   `gorm:"column:salt"`
	LrcFee                []byte  `gorm:"column:lrc_fee;type:varchar(128)"`
	BuyNoMoreThanAmountB  bool    `gorm:"column:buy_nomore_than_amountb"`
	MarginSplitPercentage uint8   `gorm:"column:margin_split_percentage;type:tinyint(4)"`
	V                     uint8   `gorm:"column:v;type:tinyint(4)"`
	R                     string  `gorm:"column:r;type:varchar(66)"`
	S                     string  `gorm:"column:s;type:varchar(66)"`
	Price                 float64 `gorm:"column:price;type:decimal(28,16);"`
}

func (o *Order) ConvertDown(state *types.OrderState) error {
	src := state.RawOrder

	var err error
	o.Price, _ = src.Price.Float64()
	if o.Price > 1e12 || o.Price < 0.0000000000000001 {
		return errors.New("price is out of range")
	}
	if o.AmountB, err = src.AmountB.MarshalText(); err != nil {
		return err
	}
	if o.AmountS, err = src.AmountS.MarshalText(); err != nil {
		return err
	}
	if o.LrcFee, err = src.LrcFee.MarshalText(); err != nil {
		return err
	}

	o.Protocol = src.Protocol.Hex()
	o.Owner = src.Owner.Hex()
	o.OrderHash = src.Hash.Hex()
	o.TokenB = src.TokenB.Hex()
	o.TokenS = src.TokenS.Hex()
	o.CreateTime = src.Timestamp.Int64()
	o.Ttl = src.Ttl.Int64()
	o.Salt = src.Salt.Int64()
	o.BuyNoMoreThanAmountB = src.BuyNoMoreThanAmountB
	o.MarginSplitPercentage = src.MarginSplitPercentage
	o.V = src.V
	o.S = src.S.Hex()
	o.R = src.R.Hex()

	return nil
}

func (o *Order) ConvertUp(state *types.OrderState) error {
	dst := state.RawOrder

	dst.AmountS = new(big.Int)
	if err := dst.AmountS.UnmarshalText(o.AmountS); err != nil {
		return err
	}
	dst.AmountB = new(big.Int)
	if err := dst.AmountB.UnmarshalText(o.AmountB); err != nil {
		return err
	}
	dst.LrcFee = new(big.Int)
	if err := dst.LrcFee.UnmarshalText(o.LrcFee); err != nil {
		return err
	}
	dst.GeneratePrice()

	dst.Protocol = types.HexToAddress(o.Protocol)
	dst.TokenS = types.HexToAddress(o.TokenS)
	dst.TokenB = types.HexToAddress(o.TokenB)
	dst.Timestamp = big.NewInt(o.CreateTime)
	dst.Ttl = big.NewInt(o.Ttl)
	dst.Salt = big.NewInt(o.Salt)
	dst.BuyNoMoreThanAmountB = o.BuyNoMoreThanAmountB
	dst.MarginSplitPercentage = o.MarginSplitPercentage
	dst.V = o.V
	dst.S = types.HexToSign(o.S)
	dst.R = types.HexToSign(o.R)
	dst.Owner = types.HexToAddress(o.Owner)

	dst.Hash = types.HexToHash(o.OrderHash)
	if dst.Hash != dst.GenerateHash() {
		return errors.New("dao order convert down generate hash error")
	}

	return nil
}

func (s *RdsServiceImpl) GetOrderByHash(orderhash types.Hash) (*Order, error) {
	order := &Order{}
	err := s.db.Where("order_hash = ?", orderhash.Hex()).First(order).Error
	return order, err
}

func (s *RdsServiceImpl) GetOrdersForMiner(orderhashList []types.Hash) ([]Order, error) {
	var (
		list        []Order
		filterhashs []string
		err         error
	)

	for _, v := range orderhashList {
		filterhashs = append(filterhashs, v.Hex())
	}

	if len(filterhashs) == 0 {
		err = s.db.Order("price desc").Find(&list).Error
	} else {
		err = s.db.Where("order_hash not in(?)", filterhashs).Order("price desc").Find(&list).Error
	}

	return list, err
}
