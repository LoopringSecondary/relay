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
	Hash             string `gorm:"column:hash;type:varchar(82)"`
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
}

func (info *RingSubmitInfo) ConvertDown(typesInfo *types.RingSubmitInfo) error {
	info.Hash = typesInfo.Ringhash.Hex()
	info.ProtocolAddress = typesInfo.ProtocolAddress.Hex()
	info.OrdersCount = typesInfo.OrdersCount.Int64()
	info.ProtocolData = common.ToHex(typesInfo.ProtocolData)
	info.ProtocolGas = typesInfo.ProtocolGas.Bytes()
	info.ProtocolGasPrice = typesInfo.ProtocolGasPrice.Bytes()
	info.RegistryData = common.ToHex(typesInfo.RegistryData)
	info.RegistryGas = typesInfo.RegistryGas.Bytes()
	info.RegistryGasPrice = typesInfo.RegistryGasPrice.Bytes()

	return nil
}

func (info *RingSubmitInfo) ConvertUp(typesInfo *types.RingSubmitInfo) error {
	typesInfo.Ringhash = common.HexToHash(info.Hash)
	typesInfo.ProtocolAddress = common.HexToAddress(info.ProtocolAddress)
	typesInfo.OrdersCount = big.NewInt(info.OrdersCount)
	typesInfo.ProtocolData = common.FromHex(info.ProtocolData)
	typesInfo.ProtocolGas = new(big.Int).SetBytes(info.ProtocolGas)
	typesInfo.ProtocolGasPrice = new(big.Int).SetBytes(info.ProtocolGasPrice)
	typesInfo.RegistryData = common.FromHex(info.RegistryData)
	typesInfo.RegistryGas = new(big.Int).SetBytes(info.RegistryGas)
	typesInfo.RegistryGasPrice = new(big.Int).SetBytes(info.RegistryGasPrice)
	typesInfo.SubmitTxHash = common.HexToHash(info.SubmitTxHash)
	typesInfo.RegistryTxHash = common.HexToHash(info.RegistryTxHash)
	return nil
}

func (s *RdsServiceImpl) UpdateRingSubmitInfoRegistryTxHash(ringhashs []common.Hash, txHash string) error {
	hashes := []string{}
	for _, h := range ringhashs {
		hashes = append(hashes, h.Hex())
	}
	return s.db.Model(&Ring{}).Where("hash in (?)", hashes).Update(" registry_tx_hash", txHash).Error
}

func (s *RdsServiceImpl) UpdateRingSubmitInfoSubmitTxHash(ringhash common.Hash, txHash string) error {
	return s.db.Model(&Ring{}).Where("hash = ?", ringhash.Hex()).Update(" submit_tx_hash", txHash).Error
}
