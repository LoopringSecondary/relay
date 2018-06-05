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
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// order amountS 上限1e30

type Order struct {
	ID                    int     `gorm:"column:id;primary_key;"`
	Protocol              string  `gorm:"column:protocol;type:varchar(42)"`
	DelegateAddress       string  `gorm:"column:delegate_address;type:varchar(42)"`
	Owner                 string  `gorm:"column:owner;type:varchar(42)"`
	AuthAddress           string  `gorm:"column:auth_address;type:varchar(42)"`
	PrivateKey            string  `gorm:"column:priv_key;type:varchar(128)"`
	WalletAddress         string  `gorm:"column:wallet_address;type:varchar(42)"`
	OrderHash             string  `gorm:"column:order_hash;type:varchar(82)"`
	TokenS                string  `gorm:"column:token_s;type:varchar(42)"`
	TokenB                string  `gorm:"column:token_b;type:varchar(42)"`
	AmountS               string  `gorm:"column:amount_s;type:varchar(40)"`
	AmountB               string  `gorm:"column:amount_b;type:varchar(40)"`
	CreateTime            int64   `gorm:"column:create_time;type:bigint"`
	ValidSince            int64   `gorm:"column:valid_since;type:bigint"`
	ValidUntil            int64   `gorm:"column:valid_until;type:bigint"`
	LrcFee                string  `gorm:"column:lrc_fee;type:varchar(40)"`
	BuyNoMoreThanAmountB  bool    `gorm:"column:buy_nomore_than_amountb"`
	MarginSplitPercentage uint8   `gorm:"column:margin_split_percentage;type:tinyint(4)"`
	V                     uint8   `gorm:"column:v;type:tinyint(4)"`
	R                     string  `gorm:"column:r;type:varchar(66)"`
	S                     string  `gorm:"column:s;type:varchar(66)"`
	PowNonce              uint64  `gorm:"column:pow_nonce;type:bigint"`
	Price                 float64 `gorm:"column:price;type:decimal(28,16);"`
	UpdatedBlock          int64   `gorm:"column:updated_block;type:bigint"`
	DealtAmountS          string  `gorm:"column:dealt_amount_s;type:varchar(40)"`
	DealtAmountB          string  `gorm:"column:dealt_amount_b;type:varchar(40)"`
	CancelledAmountS      string  `gorm:"column:cancelled_amount_s;type:varchar(40)"`
	CancelledAmountB      string  `gorm:"column:cancelled_amount_b;type:varchar(40)"`
	SplitAmountS          string  `gorm:"column:split_amount_s;type:varchar(40)"`
	SplitAmountB          string  `gorm:"column:split_amount_b;type:varchar(40)"`
	Status                uint8   `gorm:"column:status;type:tinyint(4)"`
	MinerBlockMark        int64   `gorm:"column:miner_block_mark;type:bigint"`
	BroadcastTime         int     `gorm:"column:broadcast_time;type:bigint"`
	Market                string  `gorm:"column:market;type:varchar(40)"`
	Side                  string  `gorm:"column:side;type:varchar(40)`
	OrderType             string  `gorm:"column:order_type;type:varchar(40)`
}

// convert types/orderState to dao/order
func (o *Order) ConvertDown(state *types.OrderState) error {
	src := state.RawOrder

	o.Price, _ = src.Price.Float64()
	o.AmountS = src.AmountS.String()
	o.AmountB = src.AmountB.String()
	o.DealtAmountS = state.DealtAmountS.String()
	o.DealtAmountB = state.DealtAmountB.String()
	o.SplitAmountS = state.SplitAmountS.String()
	o.SplitAmountB = state.SplitAmountB.String()
	o.CancelledAmountS = state.CancelledAmountS.String()
	o.CancelledAmountB = state.CancelledAmountB.String()
	o.LrcFee = src.LrcFee.String()

	o.Protocol = src.Protocol.Hex()
	o.DelegateAddress = src.DelegateAddress.Hex()
	o.Owner = src.Owner.Hex()

	auth, _ := src.AuthPrivateKey.MarshalText()
	o.PrivateKey = string(auth)
	o.AuthAddress = src.AuthAddr.Hex()
	o.WalletAddress = src.WalletAddress.Hex()

	o.OrderHash = src.Hash.Hex()
	o.TokenB = src.TokenB.Hex()
	o.TokenS = src.TokenS.Hex()
	o.CreateTime = time.Now().Unix()
	o.ValidSince = src.ValidSince.Int64()
	o.ValidUntil = src.ValidUntil.Int64()

	o.BuyNoMoreThanAmountB = src.BuyNoMoreThanAmountB
	o.MarginSplitPercentage = src.MarginSplitPercentage
	if state.UpdatedBlock != nil {
		o.UpdatedBlock = state.UpdatedBlock.Int64()
	}
	o.Status = uint8(state.Status)
	o.V = src.V
	o.S = src.S.Hex()
	o.R = src.R.Hex()
	o.PowNonce = src.PowNonce
	o.BroadcastTime = state.BroadcastTime
	o.Side = state.RawOrder.Side
	o.OrderType = state.RawOrder.OrderType

	return nil
}

// convert dao/order to types/orderState
func (o *Order) ConvertUp(state *types.OrderState) error {
	state.RawOrder.AmountS, _ = new(big.Int).SetString(o.AmountS, 0)
	state.RawOrder.AmountB, _ = new(big.Int).SetString(o.AmountB, 0)
	state.DealtAmountS, _ = new(big.Int).SetString(o.DealtAmountS, 0)
	state.DealtAmountB, _ = new(big.Int).SetString(o.DealtAmountB, 0)
	state.SplitAmountS, _ = new(big.Int).SetString(o.SplitAmountS, 0)
	state.SplitAmountB, _ = new(big.Int).SetString(o.SplitAmountB, 0)
	state.CancelledAmountS, _ = new(big.Int).SetString(o.CancelledAmountS, 0)
	state.CancelledAmountB, _ = new(big.Int).SetString(o.CancelledAmountB, 0)
	state.RawOrder.LrcFee, _ = new(big.Int).SetString(o.LrcFee, 0)

	state.RawOrder.Price = new(big.Rat).SetFloat64(o.Price)
	state.RawOrder.Protocol = common.HexToAddress(o.Protocol)
	state.RawOrder.DelegateAddress = common.HexToAddress(o.DelegateAddress)
	state.RawOrder.TokenS = common.HexToAddress(o.TokenS)
	state.RawOrder.TokenB = common.HexToAddress(o.TokenB)
	state.RawOrder.ValidSince = big.NewInt(o.ValidSince)
	state.RawOrder.ValidUntil = big.NewInt(o.ValidUntil)

	if len(o.AuthAddress) > 0 {
		state.RawOrder.AuthAddr = common.HexToAddress(o.AuthAddress)
	}
	if len(o.PrivateKey) > 0 {
		authPrivateKey, err := crypto.NewPrivateKeyCrypto(false, o.PrivateKey)
		if err == nil {
			state.RawOrder.AuthPrivateKey = authPrivateKey
		}
	}
	state.RawOrder.WalletAddress = common.HexToAddress(o.WalletAddress)

	state.RawOrder.BuyNoMoreThanAmountB = o.BuyNoMoreThanAmountB
	state.RawOrder.MarginSplitPercentage = o.MarginSplitPercentage
	state.RawOrder.V = o.V
	state.RawOrder.S = types.HexToBytes32(o.S)
	state.RawOrder.R = types.HexToBytes32(o.R)
	state.RawOrder.PowNonce = o.PowNonce
	state.RawOrder.Owner = common.HexToAddress(o.Owner)
	state.RawOrder.Hash = common.HexToHash(o.OrderHash)

	if state.RawOrder.Hash != state.RawOrder.GenerateHash() {
		log.Debug("different order hash found......")
		log.Debug(state.RawOrder.Hash.Hex())
		log.Debug(state.RawOrder.GenerateHash().Hex())
		return fmt.Errorf("dao order convert down generate hash error")
	}

	state.UpdatedBlock = big.NewInt(o.UpdatedBlock)
	state.Status = types.OrderStatus(o.Status)
	state.BroadcastTime = o.BroadcastTime
	state.RawOrder.Market = o.Market
	state.RawOrder.CreateTime = o.CreateTime
	if o.Side == "" {
		state.RawOrder.Side = util.GetSide(o.TokenS, o.TokenB)
	} else {
		state.RawOrder.Side = o.Side
	}
	state.RawOrder.OrderType = o.OrderType
	return nil
}

func (s *RdsServiceImpl) GetOrderByHash(orderhash common.Hash) (*Order, error) {
	order := &Order{}
	err := s.db.Where("order_hash = ?", orderhash.Hex()).First(order).Error
	return order, err
}

func (s *RdsServiceImpl) MarkMinerOrders(filterOrderhashs []string, blockNumber int64) error {
	if len(filterOrderhashs) == 0 {
		return nil
	}

	err := s.db.Model(&Order{}).
		Where("order_hash in (?)", filterOrderhashs).
		Update("miner_block_mark", blockNumber).Error

	return err
}

func (s *RdsServiceImpl) GetOrdersForMiner(protocol, tokenS, tokenB string, length int, filterStatus []types.OrderStatus, reservedTime, startBlockNumber, endBlockNumber int64) ([]*Order, error) {
	var (
		list []*Order
		err  error
	)

	if len(filterStatus) < 1 {
		return list, errors.New("should filter cutoff and finished orders")
	}

	nowtime := time.Now().Unix()
	sinceTime := nowtime
	untilTime := nowtime + reservedTime
	err = s.db.Where("delegate_address = ? and token_s = ? and token_b = ?", protocol, tokenS, tokenB).
		Where("valid_since < ?", sinceTime).
		Where("valid_until >= ? ", untilTime).
		Where("status not in (?) ", filterStatus).
		Where("order_type = ? ", types.ORDER_TYPE_MARKET).
		Where("miner_block_mark between ? and ?", startBlockNumber, endBlockNumber).
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

func (s *RdsServiceImpl) GetCutoffOrders(owner common.Address, cutoffTime *big.Int) ([]Order, error) {
	var (
		list []Order
		err  error
	)

	filterStatus := []types.OrderStatus{types.ORDER_PARTIAL, types.ORDER_NEW}
	err = s.db.Where("valid_since < ? and owner = ? and status in (?)", cutoffTime.Int64(), owner.Hex(), filterStatus).Find(&list).Error
	return list, err
}

func (s *RdsServiceImpl) GetCutoffPairOrders(owner, token1, token2 common.Address, cutoffTime *big.Int) ([]Order, error) {
	var (
		list []Order
		err  error
	)

	filterStatus := []types.OrderStatus{types.ORDER_PARTIAL, types.ORDER_NEW}
	tokens := []string{token1.Hex(), token2.Hex()}
	err = s.db.Model(&Order{}).Where("valid_since < ? and owner = ? and status in (?)", cutoffTime.Int64(), owner.Hex(), filterStatus).
		Where("token_s in (?)", tokens).
		Where("token_b in (?)", tokens).
		Find(&list).Error

	return list, err
}

func (s *RdsServiceImpl) SetCutOffOrders(orderHashList []common.Hash, blockNumber *big.Int) error {
	var list []string

	items := map[string]interface{}{
		"status":        uint8(types.ORDER_CUTOFF),
		"updated_block": blockNumber.Int64(),
	}

	for _, v := range orderHashList {
		list = append(list, v.Hex())
	}
	err := s.db.Model(&Order{}).Where("order_hash in (?)", list).Update(items).Error
	return err
}

func (s *RdsServiceImpl) GetOrderBook(delegate, tokenS, tokenB common.Address, length int) ([]Order, error) {
	var (
		list []Order
		err  error
	)

	filterStatus := []types.OrderStatus{types.ORDER_NEW, types.ORDER_PARTIAL}
	nowtime := time.Now().Unix()
	err = s.db.Where("delegate_address = ?", delegate.Hex()).
		Where("token_s = ? and token_b = ?", tokenS.Hex(), tokenB.Hex()).
		Where("status in (?)", filterStatus).
		Where("order_type = ? ", types.ORDER_TYPE_MARKET).
		Where("valid_since < ?", nowtime).
		Where("valid_until >= ? ", nowtime).
		Order("price desc").
		Limit(length).
		Find(&list).Error

	return list, err
}

func (s *RdsServiceImpl) OrderPageQuery(query map[string]interface{}, statusList []int, pageIndex, pageSize int) (PageResult, error) {
	var (
		orders        []Order
		err           error
		data          = make([]interface{}, 0)
		pageResult    PageResult
		statusStrList = make([]string, 0)
	)

	if pageIndex <= 0 {
		pageIndex = 1
	}

	if pageSize <= 0 {
		pageSize = 20
	}

	pageResult = PageResult{data, pageIndex, pageSize, 0}

	openedStatus := []types.OrderStatus{types.ORDER_NEW, types.ORDER_PARTIAL}
	now := time.Now().Unix()

	if len(statusList) == 1 {
		if statusList[0] == 6 {
			if err = s.db.Where(query).
				Where("valid_until < ?", now).
				Where("status in (?)", openedStatus).
				Offset((pageIndex - 1) * pageSize).Order("create_time DESC").Limit(pageSize).Find(&orders).Error; err != nil {
				return pageResult, err
			}

			err = s.db.Model(&Order{}).Where(query).
				Where("valid_until < ?", now).
				Where("status in (?)", openedStatus).Count(&pageResult.Total).Error

			if err != nil {
				return pageResult, err
			}

		} else {
			query["status"] = statusList[0]
			if err = s.db.Where(query).Offset((pageIndex - 1) * pageSize).Order("create_time DESC").Limit(pageSize).Find(&orders).Error; err != nil {
				return pageResult, err
			}

			err = s.db.Model(&Order{}).Where(query).Count(&pageResult.Total).Error
			if err != nil {
				return pageResult, err
			}
		}

	} else if len(statusList) > 1 {
		for _, s := range statusList {
			statusStrList = append(statusStrList, strconv.Itoa(s))
		}

		queryOpened := allContain(statusList, openedStatus)
		if queryOpened {
			if err = s.db.Where(query).
				Where("status in (?)", statusStrList).
				Where("valid_since < ?", now).
				Where("valid_until >= ? ", now).
				Offset((pageIndex - 1) * pageSize).Order("create_time DESC").Limit(pageSize).Find(&orders).Error; err != nil {
				return pageResult, err
			}

			err = s.db.Model(&Order{}).Where(query).
				Where("valid_since < ?", now).
				Where("valid_until >= ? ", now).
				Where("status in (?)", openedStatus).Count(&pageResult.Total).Error

			if err != nil {
				return pageResult, err
			}

		} else {
			if err = s.db.Where(query).Where("status in (?)", statusStrList).Offset((pageIndex - 1) * pageSize).Order("create_time DESC").Limit(pageSize).Find(&orders).Error; err != nil {
				return pageResult, err
			}

			err = s.db.Model(&Order{}).Where(query).
				Where("status in (?)", openedStatus).Count(&pageResult.Total).Error

			if err != nil {
				return pageResult, err
			}
		}

	} else {
		if err = s.db.Where(query).Offset((pageIndex - 1) * pageSize).Order("create_time DESC").Limit(pageSize).Find(&orders).Error; err != nil {
			return pageResult, err
		}

		err = s.db.Model(&Order{}).Where(query).Count(&pageResult.Total).Error
		if err != nil {
			return pageResult, err
		}
	}

	for _, v := range orders {
		data = append(data, v)
	}

	pageResult.Data = data

	return pageResult, err
}

func containStatus(status int, statusList []types.OrderStatus) bool {
	if len(statusList) == 0 {
		return false
	}

	for _, s := range statusList {
		if status == int(s) {
			return true
		}
	}
	return false
}

func allContain(left []int, right []types.OrderStatus) bool {

	for _, l := range left {
		if !containStatus(l, right) {
			return false
		}
	}

	return true
}

func (s *RdsServiceImpl) UpdateBroadcastTimeByHash(hash string, bt int) error {
	return s.db.Model(&Order{}).Where("order_hash = ?", hash).Update("broadcast_time", bt).Error
}

func (s *RdsServiceImpl) UpdateOrderWhileFill(hash common.Hash, status types.OrderStatus, dealtAmountS, dealtAmountB, splitAmountS, splitAmountB, blockNumber *big.Int) error {
	items := map[string]interface{}{
		"status":         uint8(status),
		"dealt_amount_s": dealtAmountS.String(),
		"dealt_amount_b": dealtAmountB.String(),
		"split_amount_s": splitAmountS.String(),
		"split_amount_b": splitAmountB.String(),
		"updated_block":  blockNumber.Int64(),
	}
	return s.db.Model(&Order{}).Where("order_hash = ?", hash.Hex()).Update(items).Error
}

func (s *RdsServiceImpl) UpdateOrderWhileCancel(hash common.Hash, status types.OrderStatus, cancelledAmountS, cancelledAmountB, blockNumber *big.Int) error {
	items := map[string]interface{}{
		"status":             uint8(status),
		"cancelled_amount_s": cancelledAmountS.String(),
		"cancelled_amount_b": cancelledAmountB.String(),
		"updated_block":      blockNumber.Int64(),
	}
	return s.db.Model(&Order{}).Where("order_hash = ?", hash.Hex()).Update(items).Error
}

func (s *RdsServiceImpl) UpdateOrderWhileRollbackCutoff(orderhash common.Hash, status types.OrderStatus, blockNumber *big.Int) error {
	items := map[string]interface{}{
		"status":        uint8(status),
		"updated_block": blockNumber.Int64(),
	}
	return s.db.Model(&Order{}).Where("order_hash = ?", orderhash.Hex()).Update(items).Error
}

func (s *RdsServiceImpl) GetFrozenAmount(owner common.Address, token common.Address, statusSet []types.OrderStatus, delegateAddress common.Address) ([]Order, error) {
	var (
		list []Order
		err  error
	)
	now := time.Now().Unix()
	err = s.db.Model(&Order{}).
		Where("token_s = ? and owner = ? and delegate_address = ? and status in "+buildStatusInSet(statusSet), token.Hex(), owner.Hex(), delegateAddress.Hex()).
		Where("valid_since < ?", now).
		Where("valid_until >= ? ", now).
		Find(&list).Error
	return list, err
}

func (s *RdsServiceImpl) GetFrozenLrcFee(owner common.Address, statusSet []types.OrderStatus) ([]Order, error) {
	var (
		list []Order
		err  error
	)

	now := time.Now().Unix()
	err = s.db.Model(&Order{}).
		Where("lrc_fee > 0 and owner = ? and status in "+buildStatusInSet(statusSet), owner.Hex()).
		Where("valid_since < ?", now).
		Where("valid_until >= ? ", now).
		Find(&list).Error
	return list, err
}

func buildStatusInSet(statusSet []types.OrderStatus) string {
	if len(statusSet) == 0 {
		return ""
	}
	result := "("
	strSet := make([]string, 0)
	for _, s := range statusSet {
		strSet = append(strSet, strconv.Itoa(int(s)))
	}
	result += strings.Join(strSet, ",")
	result += ")"
	return result
}
