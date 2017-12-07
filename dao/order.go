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
	"fmt"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"time"
)

// order amountS 上限1e30

type Order struct {
	ID                    int     `gorm:"column:id;primary_key;"`
	Protocol              string  `gorm:"column:protocol;type:varchar(42)"`
	Owner                 string  `gorm:"column:owner;type:varchar(42)"`
	OrderHash             string  `gorm:"column:order_hash;type:varchar(82);unique_index"`
	TokenS                string  `gorm:"column:token_s;type:varchar(42)"`
	TokenB                string  `gorm:"column:token_b;type:varchar(42)"`
	AmountS               string  `gorm:"column:amount_s;type:varchar(30)"`
	AmountB               string  `gorm:"column:amount_b;type:varchar(30)"`
	CreateTime            int64   `gorm:"column:create_time;type:bigint"`
	Ttl                   int64   `gorm:"column:ttl;type:bigint"`
	Salt                  int64   `gorm:"column:salt;type:bigint"`
	LrcFee                string  `gorm:"column:lrc_fee;type:varchar(30)"`
	BuyNoMoreThanAmountB  bool    `gorm:"column:buy_nomore_than_amountb"`
	MarginSplitPercentage uint8   `gorm:"column:margin_split_percentage;type:tinyint(4)"`
	V                     uint8   `gorm:"column:v;type:tinyint(4)"`
	R                     string  `gorm:"column:r;type:varchar(66)"`
	S                     string  `gorm:"column:s;type:varchar(66)"`
	Price                 float64 `gorm:"column:price;type:decimal(28,16);"`
	BlockNumber           int64   `gorm:"column:block_number;type:bigint"`
	DealtAmountS          string  `gorm:"column:dealt_amount_s;type:varchar(30)"`
	DealtAmountB          string  `gorm:"column:dealt_amount_b;type:varchar(30)"`
	CancelledAmountS      string  `gorm:"column:cancelled_amount_s;type:varchar(30)"`
	CancelledAmountB      string  `gorm:"column:cancelled_amount_b;type:varchar(30)"`
	Status                uint8   `gorm:"column:status;type:tinyint(4)"`
	MinerBlockMark        int64   `gorm:"column:miner_block_mark;type:bigint"`
	BroadcastTime         int     `gorm:"column:broadcast_time;type:bigint"`
	Market                string  `gorm:"column:market;type:varchar(40)"`
}

// convert types/orderState to dao/order
func (o *Order) ConvertDown(state *types.OrderState) error {
	src := state.RawOrder

	o.Price, _ = src.Price.Float64()
	if o.Price > 1e12 || o.Price < 0.0000000000000001 {
		return fmt.Errorf("dao order convert down,price out of range")
	}

	o.AmountS = src.AmountS.String()
	o.AmountB = src.AmountB.String()
	o.DealtAmountS = state.DealtAmountS.String()
	o.DealtAmountB = state.DealtAmountB.String()
	o.CancelledAmountS = state.CancelledAmountS.String()
	o.CancelledAmountB = state.CancelledAmountB.String()
	o.LrcFee = src.LrcFee.String()

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
	if state.BlockNumber != nil {
		o.BlockNumber = state.BlockNumber.Int64()
	}
	o.Status = uint8(state.Status)
	o.V = src.V
	o.S = src.S.Hex()
	o.R = src.R.Hex()
	o.BroadcastTime = state.BroadcastTime
	o.Market, _ = util.WrapMarketByAddress(o.TokenB, o.TokenS)
	return nil
}

// convert dao/order to types/orderState
func (o *Order) ConvertUp(state *types.OrderState) error {
	state.RawOrder.AmountS, _ = new(big.Int).SetString(o.AmountS, 0)
	state.RawOrder.AmountB, _ = new(big.Int).SetString(o.AmountB, 0)
	state.DealtAmountS, _ = new(big.Int).SetString(o.DealtAmountS, 0)
	state.DealtAmountB, _ = new(big.Int).SetString(o.DealtAmountB, 0)
	state.CancelledAmountS, _ = new(big.Int).SetString(o.CancelledAmountS, 0)
	state.CancelledAmountB, _ = new(big.Int).SetString(o.CancelledAmountB, 0)
	state.RawOrder.LrcFee, _ = new(big.Int).SetString(o.LrcFee, 0)

	state.RawOrder.GeneratePrice()
	state.RawOrder.Protocol = common.HexToAddress(o.Protocol)
	state.RawOrder.TokenS = common.HexToAddress(o.TokenS)
	state.RawOrder.TokenB = common.HexToAddress(o.TokenB)
	state.RawOrder.Timestamp = big.NewInt(o.CreateTime)
	state.RawOrder.Ttl = big.NewInt(o.Ttl)
	state.RawOrder.Salt = big.NewInt(o.Salt)
	state.RawOrder.BuyNoMoreThanAmountB = o.BuyNoMoreThanAmountB
	state.RawOrder.MarginSplitPercentage = o.MarginSplitPercentage
	state.RawOrder.V = o.V
	state.RawOrder.S = types.HexToBytes32(o.S)
	state.RawOrder.R = types.HexToBytes32(o.R)
	state.RawOrder.Owner = common.HexToAddress(o.Owner)
	state.RawOrder.Hash = common.HexToHash(o.OrderHash)

	if state.RawOrder.Hash != state.RawOrder.GenerateHash() {
		return fmt.Errorf("dao order convert down generate hash error")
	}

	state.BlockNumber = big.NewInt(o.BlockNumber)
	state.Status = types.OrderStatus(o.Status)
	state.BroadcastTime = o.BroadcastTime

	return nil
}

func (s *RdsServiceImpl) GetOrderByHash(orderhash common.Hash) (*Order, error) {
	order := &Order{}
	err := s.db.Where("order_hash = ?", orderhash.Hex()).First(order).Error
	return order, err
}

func (s *RdsServiceImpl) MarkMinerOrders(filterOrderhashs []string, blockNumber int64) error {
	err := s.db.Model(&Order{}).
		Where("order_hash in (?)", filterOrderhashs).
		Update("miner_block_mark", blockNumber).Error

	return err
}

func (s *RdsServiceImpl) GetOrdersForMiner(protocol, tokenS, tokenB string, length int, filterStatus []types.OrderStatus, markBlockNumber int64) ([]*Order, error) {
	var (
		list []*Order
		err  error
	)

	if len(filterStatus) < 1 {
		return list, errors.New("should filter cutoff and finished orders")
	}

	nowtime := time.Now().Unix()
	err = s.db.Where("protocol = ? and token_s = ? and token_b = ?", protocol, tokenS, tokenB).
		Where("create_time + ttl > ? ", nowtime).
		Where("status not in (?) ", filterStatus).
		Where("miner_block_mark = ? or miner_block_mark <= ?", 0, markBlockNumber).
		Order("price desc").
		Limit(length).
		Find(&list).
		Error

	return list, err
}

func (s *RdsServiceImpl) GetOrdersByHash(orderhashs []string) (map[string]Order, error) {
	var (
		list []Order
		err  error
	)

	ret := make(map[string]Order)
	if err = s.db.Where("order_hash in (?)", orderhashs).Find(&list).Error; err != nil {
		return ret, err
	}

	for _, v := range list {
		ret[v.OrderHash] = v
	}

	return ret, err
}

func (s *RdsServiceImpl) GetOrdersWithBlockNumberRange(from, to int64) ([]Order, error) {
	var (
		list []Order
		err  error
	)

	if from < to {
		return list, fmt.Errorf("dao/order GetOrdersWithBlockNumberRange invalid block number")
	}

	err = s.db.Where("block_number between ? and ?", from, to).Find(&list).Error

	return list, err
}

func (s *RdsServiceImpl) GetCutoffOrders(cutoffTime int64) ([]Order, error) {
	var (
		list []Order
		err  error
	)

	err = s.db.Where("create_time < ?", cutoffTime).Find(&list).Error

	return list, err
}

func (s *RdsServiceImpl) CheckOrderCutoff(orderhash string, cutoff int64) bool {
	model := Order{}
	err := s.db.Where("order_hash = ? and create_time < ?").Find(&model).Error
	if err != nil {
		return false
	}

	return true
}

func (s *RdsServiceImpl) SettleOrdersCutoffStatus(owner common.Address, cutoffTime *big.Int) error {
	err := s.db.Model(&Order{}).Where("create_time < (?)", cutoffTime.Int64()).
		Where("owner = (?)", owner.Hex()).
		Update("status", types.ORDER_CUTOFF).Error
	return err
}

func (s *RdsServiceImpl) GetOrderBook(protocol, tokenS, tokenB common.Address, length int) ([]Order, error) {
	var (
		list []Order
		err  error
	)

	err = s.db.Where("protocol = ? and token_s = ? and token_b = ?", protocol.Hex(), tokenS.Hex(), tokenB.Hex()).
		Order("price desc").Limit(length).Find(&list).Error

	return list, err
}

func (s *RdsServiceImpl) OrderPageQuery(query map[string]interface{}, pageIndex, pageSize int) (PageResult, error) {
	var (
		orders     []Order
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

	if err = s.db.Where(query).Offset((pageIndex - 1) * pageSize).Limit(pageSize).Find(&orders).Error; err != nil {
		return pageResult, err
	}
	for _, v := range orders {
		data = append(data, v)
	}

	pageResult = PageResult{data, pageIndex, pageSize, 0}

	err = s.db.Model(&Order{}).Where(query).Count(&pageResult.Total).Error
	if err != nil {
		return pageResult, err
	}

	return pageResult, err
}

func (s *RdsServiceImpl) UpdateBroadcastTimeByHash(hash string, bt int) error {
	return s.db.Model(&Order{}).Where("order_hash = ?", hash).Update("broadcast_time", bt).Error
}

func (s *RdsServiceImpl) UpdateOrderWhileFill(hash common.Hash, status types.OrderStatus, dealtAmountS, dealtAmountB, blockNumber *big.Int) error {
	items := map[string]interface{}{
		"status":         uint8(status),
		"dealt_amount_s": dealtAmountS.String(),
		"dealt_amount_b": dealtAmountB.String(),
		"block_number":   blockNumber.String(),
	}
	return s.db.Model(&Order{}).Where("order_hash = ?", hash.Hex()).Update(items).Error
}

func (s *RdsServiceImpl) UpdateOrderWhileCancel(hash common.Hash, status types.OrderStatus, cancelledAmountS, cancelledAmountB, blockNumber *big.Int) error {
	items := map[string]interface{}{
		"status":             uint8(status),
		"cancelled_amount_s": cancelledAmountS.String(),
		"cancelled_amount_b": cancelledAmountB.String(),
		"block_number":       blockNumber.String(),
	}
	return s.db.Model(&Order{}).Where("order_hash = ?", hash.Hex()).Update(items).Error
}
