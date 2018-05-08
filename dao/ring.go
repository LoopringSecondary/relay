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

type FilledOrder struct {
	ID               int    `gorm:"column:id;primary_key;"`
	RingHash         string `gorm:"column:ringhash;type:varchar(82)"`
	OrderHash        string `gorm:"column:orderhash;type:varchar(82)"`
	FeeSelection     uint8  `gorm:"column:fee_selection" json:"feeSelection"`
	RateAmountS      string `gorm:"column:rate_amount_s;type:text" json:"rateAmountS"`
	AvailableAmountS string `gorm:"column:available_amount_s;type:text"json:"availableAmountS"`
	AvailableAmountB string `gorm:"column:available_amount_b;type:text"`
	FillAmountS      string `gorm:"column:fill_amount_s;type:text" json:"fillAmountS"`
	FillAmountB      string `gorm:"column:fill_amount_b;type:text" json:"fillAmountB"`
	LrcReward        string `gorm:"column:lrc_reward;type:text" json:"lrcReward"`
	LrcFee           string `gorm:"column:lrc_fee;type:text" json:"lrcFee"`
	FeeS             string `gorm:"column:fee_s;type:text" json:"feeS"`
	LegalFee         string `gorm:"column:legal_fee;type:text" json:"legalFee"`
	SPrice           string `gorm:"column:s_price;type:text" json:"sPrice"`
	BPrice           string `gorm:"column:b_price;type:text" json:"sPrice"`
}

func getRatString(v *big.Rat) string {
	if nil == v {
		return ""
	} else {
		return v.String()
	}
}

func (daoFilledOrder *FilledOrder) ConvertDown(filledOrder *types.FilledOrder, ringhash common.Hash) error {
	daoFilledOrder.RingHash = ringhash.Hex()
	daoFilledOrder.OrderHash = filledOrder.OrderState.RawOrder.Hash.Hex()
	daoFilledOrder.FeeSelection = filledOrder.FeeSelection
	daoFilledOrder.RateAmountS = getRatString(filledOrder.RateAmountS)
	daoFilledOrder.AvailableAmountS = getRatString(filledOrder.AvailableAmountS)
	daoFilledOrder.AvailableAmountB = getRatString(filledOrder.AvailableAmountB)
	daoFilledOrder.FillAmountS = getRatString(filledOrder.FillAmountS)
	daoFilledOrder.FillAmountB = getRatString(filledOrder.FillAmountB)
	daoFilledOrder.LrcReward = getRatString(filledOrder.LrcReward)
	daoFilledOrder.LrcFee = getRatString(filledOrder.LrcFee)
	daoFilledOrder.FeeS = getRatString(filledOrder.FeeS)
	daoFilledOrder.LegalFee = getRatString(filledOrder.LegalFee)
	daoFilledOrder.SPrice = getRatString(filledOrder.SPrice)
	daoFilledOrder.BPrice = getRatString(filledOrder.BPrice)
	return nil
}

func (daoFilledOrder *FilledOrder) ConvertUp(filledOrder *types.FilledOrder, rds RdsService) error {
	if nil != rds {
		daoOrderState, err := rds.GetOrderByHash(common.HexToHash(daoFilledOrder.OrderHash))
		if nil != err {
			return err
		}
		orderState := &types.OrderState{}
		daoOrderState.ConvertUp(orderState)
		filledOrder.OrderState = *orderState
	}
	filledOrder.FeeSelection = daoFilledOrder.FeeSelection
	filledOrder.RateAmountS = new(big.Rat)
	filledOrder.RateAmountS.SetString(daoFilledOrder.RateAmountS)
	filledOrder.AvailableAmountS = new(big.Rat)
	filledOrder.AvailableAmountB = new(big.Rat)
	filledOrder.AvailableAmountS.SetString(daoFilledOrder.AvailableAmountS)
	filledOrder.AvailableAmountB.SetString(daoFilledOrder.AvailableAmountB)
	filledOrder.FillAmountS = new(big.Rat)
	filledOrder.FillAmountB = new(big.Rat)
	filledOrder.FillAmountS.SetString(daoFilledOrder.FillAmountS)
	filledOrder.FillAmountB.SetString(daoFilledOrder.FillAmountB)
	filledOrder.LrcReward = new(big.Rat)
	filledOrder.LrcFee = new(big.Rat)
	filledOrder.LrcReward.SetString(daoFilledOrder.LrcReward)
	filledOrder.LrcFee.SetString(daoFilledOrder.LrcFee)
	filledOrder.FeeS = new(big.Rat)
	filledOrder.FeeS.SetString(daoFilledOrder.FeeS)
	filledOrder.LegalFee = new(big.Rat)
	filledOrder.LegalFee.SetString(daoFilledOrder.LegalFee)
	filledOrder.SPrice = new(big.Rat)
	filledOrder.SPrice.SetString(daoFilledOrder.SPrice)
	filledOrder.BPrice = new(big.Rat)
	filledOrder.BPrice.SetString(daoFilledOrder.BPrice)
	return nil
}

func (s *RdsServiceImpl) GetFilledOrderByRinghash(ringhash common.Hash) ([]*FilledOrder, error) {
	var (
		filledOrders []*FilledOrder
		err          error
	)

	err = s.db.Where("ringhash = ?", ringhash.Hex()).
		Find(&filledOrders).
		Error

	return filledOrders, err
}

type RingSubmitInfo struct {
	ID               int    `gorm:"column:id;primary_key;"`
	RingHash         string `gorm:"column:ringhash;type:varchar(82)"`
	UniqueId         string `gorm:"column:unique_id;type:varchar(82)"`
	ProtocolAddress  string `gorm:"column:protocol_address;type:varchar(42)"`
	OrdersCount      int64  `gorm:"column:order_count;type:bigint"`
	ProtocolData     string `gorm:"column:protocol_data;type:text"`
	ProtocolGas      string `gorm:"column:protocol_gas;type:varchar(50)"`
	ProtocolGasPrice string `gorm:"column:protocol_gas_price;type:varchar(50)"`
	ProtocolUsedGas  string `gorm:"column:protocol_used_gas;type:varchar(50)"`
	ProtocolTxHash   string `gorm:"column:protocol_tx_hash;type:varchar(82)"`

	Status      int       `gorm:"column:status;type:int"`
	RingIndex   string    `gorm:"column:ring_index;type:varchar(50)"`
	BlockNumber string    `gorm:"column:block_number;type:varchar(50)"`
	Miner       string    `gorm:"column:miner;type:varchar(42)"`
	Err         string    `gorm:"column:err;type:text"`
	CreateTime  time.Time `gorm:"column:create_time;type:TIMESTAMP;default:CURRENT_TIMESTAMP"`
}

func getBigIntString(v *big.Int) string {
	if nil == v {
		return ""
	} else {
		return v.String()
	}
}

func (info *RingSubmitInfo) ConvertDown(typesInfo *types.RingSubmitInfo, err error) error {
	info.RingHash = typesInfo.Ringhash.Hex()
	info.UniqueId = typesInfo.RawRing.GenerateUniqueId().Hex()
	info.ProtocolAddress = typesInfo.ProtocolAddress.Hex()
	info.OrdersCount = typesInfo.OrdersCount.Int64()
	info.ProtocolData = common.ToHex(typesInfo.ProtocolData)
	info.ProtocolGas = getBigIntString(typesInfo.ProtocolGas)
	info.ProtocolUsedGas = getBigIntString(typesInfo.ProtocolUsedGas)
	info.ProtocolGasPrice = getBigIntString(typesInfo.ProtocolGasPrice)
	info.Miner = typesInfo.Miner.Hex()
	info.ProtocolTxHash = typesInfo.SubmitTxHash.Hex()
	if nil != err {
		info.Err = err.Error()
	}
	return nil
}

func (info *RingSubmitInfo) ConvertUp(typesInfo *types.RingSubmitInfo) error {
	typesInfo.Ringhash = common.HexToHash(info.RingHash)
	typesInfo.ProtocolAddress = common.HexToAddress(info.ProtocolAddress)
	typesInfo.OrdersCount = big.NewInt(info.OrdersCount)
	typesInfo.ProtocolData = common.FromHex(info.ProtocolData)
	typesInfo.ProtocolGas = new(big.Int)
	typesInfo.ProtocolGas.SetString(info.ProtocolGas, 0)
	typesInfo.ProtocolUsedGas = new(big.Int)
	typesInfo.ProtocolUsedGas.SetString(info.ProtocolUsedGas, 0)
	typesInfo.ProtocolGasPrice = new(big.Int)
	typesInfo.ProtocolGasPrice.SetString(info.ProtocolGasPrice, 0)
	typesInfo.SubmitTxHash = common.HexToHash(info.ProtocolTxHash)
	typesInfo.Miner = common.HexToAddress(info.Miner)
	return nil
}

//func (s *RdsServiceImpl) UpdateRingSubmitInfoRegistryTxHash(ringhashs []common.Hash, txHash string) error {
//	hashes := []string{}
//	for _, h := range ringhashs {
//		hashes = append(hashes, h.Hex())
//	}
//	dbForUpdate := s.db.Model(&RingSubmitInfo{}).Where("ringhash in (?)", hashes)
//	return dbForUpdate.Update("registry_tx_hash", txHash).Error
//}

//func (s *RdsServiceImpl) UpdateRingSubmitInfoFailed(ringhashs []common.Hash, err string) error {
//	hashes := []string{}
//	for _, h := range ringhashs {
//		hashes = append(hashes, h.Hex())
//	}
//	dbForUpdate := s.db.Model(&RingSubmitInfo{}).Where("ringhash in (?) ", hashes)
//	return dbForUpdate.Update("err", err).Error
//}

func (s *RdsServiceImpl) UpdateRingSubmitInfoResult(submitResult *types.RingSubmitResultEvent) error {
	items := map[string]interface{}{
		"status":            uint8(submitResult.Status),
		"ring_index":        getBigIntString(submitResult.RingIndex),
		"block_number":      getBigIntString(submitResult.BlockNumber),
		"protocol_used_gas": getBigIntString(submitResult.UsedGas),
	}
	if nil != submitResult.Err {
		items["err"] = submitResult.Err.Error()
	}
	dbForUpdate := s.db.Model(&RingSubmitInfo{}).Where("ringhash = ? and protocol_tx_hash = ? ", submitResult.RingHash.Hex(), submitResult.TxHash.Hex())
	return dbForUpdate.Update(items).Error
}

//func (s *RdsServiceImpl) UpdateRingSubmitInfoProtocolTxHash(ringhash common.Hash, txHash string) error {
//	dbForUpdate := s.db.Model(&RingSubmitInfo{}).Where("ringhash = ?", ringhash.Hex())
//	return dbForUpdate.Update("protocol_tx_hash", txHash).Error
//}

func (s *RdsServiceImpl) GetRingForSubmitByHash(ringhash common.Hash) (ringForSubmit RingSubmitInfo, err error) {
	err = s.db.Where("ringhash = ? ", ringhash.Hex()).First(&ringForSubmit).Error
	return
}

func (s *RdsServiceImpl) GetRingHashesByTxHash(txHash common.Hash) ([]*RingSubmitInfo, error) {
	var (
		err   error
		infos []*RingSubmitInfo
	)

	err = s.db.Where("protocol_tx_hash = ? ", txHash.Hex()).
		Find(&infos).
		Error

	return infos, err
}

func (s *RdsServiceImpl) UpdateRingSubmitInfoSubmitUsedGas(txHash string, usedGas *big.Int) error {
	dbForUpdate := s.db.Model(&RingSubmitInfo{}).Where("protocol_tx_hash = ?", txHash)
	return dbForUpdate.Update("protocol_used_gas", getBigIntString(usedGas)).Error
}
