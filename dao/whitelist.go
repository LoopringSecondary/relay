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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
)

type WhiteList struct {
	ID         int    `gorm:"column:id;primary_key;"`
	Owner      string `gorm:"column:owner;varchar(42);unique_index"`
	CreateTime int64  `gorm:"column:create_time"`
	IsDeleted  bool   `gorm:"column:is_deleted"`
}

func (s *RdsServiceImpl) GetWhiteList() ([]WhiteList, error) {
	var (
		list []WhiteList
		err  error
	)

	err = s.db.Where("is_deleted = false").Find(&list).Error

	return list, err
}

func (s *RdsServiceImpl) FindWhiteListUserByAddress(address common.Address) (*WhiteList, error) {
	var (
		user WhiteList
		err  error
	)

	err = s.db.Where("owner = ? and is_deleted = ?", address.Hex(), false).First(&user).Error

	return &user, err
}

func (w *WhiteList) ConvertDown(src *types.WhiteListUser) error {
	w.Owner = src.Owner.Hex()
	w.CreateTime = src.CreateTime
	w.IsDeleted = false

	return nil
}

func (w *WhiteList) ConvertUp(dst *types.WhiteListUser) error {
	if w.IsDeleted == true {
		return errors.New("white list user " + w.Owner + " has deleted")
	}

	dst.Owner = common.HexToAddress(w.Owner)
	dst.CreateTime = w.CreateTime

	return nil
}
