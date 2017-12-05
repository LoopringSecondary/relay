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
)

type Ring struct {
	ID   int    `gorm:"column:id;primary_key;"`
	Hash string `gorm:"column:hash;type:varchar(82)"`
}

type RingSubmitInfo struct {
	ID               int    `gorm:"column:id;primary_key;"`
	Hash             string `gorm:"column:ringhash;type:varchar(82)"`
	ProtocolAddress  string `gorm:"column:protocol_address;type:varchar(42)"`
	OrdersCount      int64  `gorm:"column:order_count;type:bigint"`
	ProtocolData     string `gorm:"column:protocol_data;type:text"`
	ProtocolGas      []byte `gorm:"column:protocol_gas;type:varchar(30)"`
	ProtocolGasPrice []byte `gorm:"column:protocol_gas_price;type:varchar(30)"`
	RegistryData     string `gorm:"column:registry_data;type:text"`
	RegistryGas      []byte `gorm:"column:registry_gas;type:varchar(30)"`
	RegistryGasPrice []byte `gorm:"column:registry_gas_price;type:varchar(30)"`
	SubmitTxHash     string `gorm:"column:submit_tx_hash;type:varchar(82)"`
	RegistryTxHash   string `gorm:"column:registry_tx_hash;type:varchar(82)"`
	Miner            string `gorm:"column:miner;type:varchar(42)"`
	Err              string `gorm:"column:err;type:text"`
}

func (info *RingSubmitInfo) ConvertDown(typesInfo *types.RingSubmitInfo) error {
	info.Hash = typesInfo.Ringhash.Hex()
	info.ProtocolAddress = typesInfo.ProtocolAddress.Hex()
	info.OrdersCount = typesInfo.OrdersCount.Int64()
	info.ProtocolData = common.ToHex(typesInfo.ProtocolData)
	info.ProtocolGas, _ = typesInfo.ProtocolGas.MarshalText()
	info.ProtocolGasPrice, _ = typesInfo.ProtocolGasPrice.MarshalText()
	info.RegistryData = common.ToHex(typesInfo.RegistryData)
	info.RegistryGas, _ = typesInfo.RegistryGas.MarshalText()
	info.RegistryGasPrice, _ = typesInfo.RegistryGasPrice.MarshalText()
	info.Miner = typesInfo.Miner.Hex()
	return nil
}

func (info *RingSubmitInfo) ConvertUp(typesInfo *types.RingSubmitInfo) error {
	typesInfo.Ringhash = common.HexToHash(info.Hash)
	typesInfo.ProtocolAddress = common.HexToAddress(info.ProtocolAddress)
	typesInfo.OrdersCount = big.NewInt(info.OrdersCount)
	typesInfo.ProtocolData = common.FromHex(info.ProtocolData)
	typesInfo.ProtocolGas = new(big.Int)
	typesInfo.ProtocolGas.UnmarshalText(info.ProtocolGas)
	typesInfo.ProtocolGasPrice = new(big.Int)
	typesInfo.ProtocolGasPrice.UnmarshalText(info.ProtocolGasPrice)
	typesInfo.RegistryData = common.FromHex(info.RegistryData)
	typesInfo.RegistryGas = new(big.Int)
	typesInfo.RegistryGas.UnmarshalText(info.RegistryGas)
	typesInfo.RegistryGasPrice = new(big.Int)
	typesInfo.RegistryGasPrice.UnmarshalText(info.RegistryGasPrice)
	typesInfo.SubmitTxHash = common.HexToHash(info.SubmitTxHash)
	typesInfo.RegistryTxHash = common.HexToHash(info.RegistryTxHash)
	typesInfo.Miner = common.HexToAddress(info.Miner)
	return nil
}

func (s *RdsServiceImpl) UpdateRingSubmitInfoRegistryTxHash(ringhashs []common.Hash, txHash, err string) error {
	hashes := []string{}
	for _, h := range ringhashs {
		hashes = append(hashes, h.Hex())
	}
	dbForUpdate := s.db.Model(&RingSubmitInfo{}).Where("ringhash in (?)", hashes)
	return dbForUpdate.Update("registry_tx_hash", txHash).Update("registry_tx_err", err).Error
}

func (s *RdsServiceImpl) UpdateRingSubmitInfoFailed(ringhashs []common.Hash, err string) error {
	hashes := []string{}
	for _, h := range ringhashs {
		hashes = append(hashes, h.Hex())
	}
	dbForUpdate := s.db.Model(&RingSubmitInfo{}).Where("ringhash in (?) ", hashes)
	return dbForUpdate.Update("err", err).Error
}

func (s *RdsServiceImpl) UpdateRingSubmitInfoSubmitTxHash(ringhash common.Hash, txHash, err string) error {
	dbForUpdate := s.db.Model(&RingSubmitInfo{}).Where("ringhash = ?", ringhash.Hex())
	return dbForUpdate.Update("submit_tx_hash", txHash).Update("submit_tx_err", err).Error
}

func (s *RdsServiceImpl) GetRingForSubmitByHash(ringhash common.Hash) (ringForSubmit RingSubmitInfo, err error) {
	err = s.db.Where("ringhash = ? ", ringhash.Hex()).First(&ringForSubmit).Error
	return
}

func (s *RdsServiceImpl) GetRingHashesByTxHash(txHash common.Hash) ([]common.Hash, error) {
	var (
		err       error
		hashes    []common.Hash
		hashesStr []string
	)

	err = s.db.Model(&RingSubmitInfo{}).Where("registry_tx_hash = ? or submit_tx_hash = ? ", txHash.Hex(), txHash.Hex()).Pluck("ringhash", hashesStr).Error
	for _, h := range hashesStr {
		hashes = append(hashes, common.HexToHash(h))
	}
	return hashes, err
}
