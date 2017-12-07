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
)

type Token struct {
	ID         int    `gorm:"column:id;primary_key"`
	Protocol   string `gorm:"column:protocol;type:varchar(42);unique_index"`
	Symbol     string `gorm:"column:symbol;type:varchar(10)"`
	Source     string `gorm:"column:source;type:varchar(200)"`
	CreateTime int64  `gorm:"column:create_time"`
	Deny       bool   `gorm:"column:deny"`
	IsMarket   bool   `gorm:"column:is_market"`
}

// convert types/token to dao/token
func (t *Token) ConvertDown(src *types.Token) error {
	t.Protocol = src.Protocol.Hex()
	t.Symbol = src.Symbol
	t.Source = src.Source
	t.CreateTime = src.Time
	t.Deny = src.Deny
	t.IsMarket = src.IsMarket

	return nil
}

// convert dao/token to types/token
func (t *Token) ConvertUp(dst *types.Token) error {
	dst.Protocol = common.HexToAddress(t.Protocol)
	dst.Symbol = t.Symbol
	dst.Source = t.Source
	dst.Time = t.CreateTime
	dst.Deny = t.Deny
	dst.IsMarket = t.IsMarket

	return nil
}

func (s *RdsServiceImpl) FindUnDeniedTokens() ([]Token, error) {
	var list []Token
	err := s.db.Where("deny = ? and is_market = ?", false, false).Find(&list).Error
	return list, err
}

func (s *RdsServiceImpl) FindDeniedTokens() ([]Token, error) {
	var list []Token
	err := s.db.Where("deny = ? and is_market = ?", true, false).Find(&list).Error
	return list, err
}

func (s *RdsServiceImpl) FindUnDeniedMarkets() ([]Token, error) {
	var list []Token
	err := s.db.Where("deny = ? and is_market = ?", false, true).Find(&list).Error
	return list, err
}

func (s *RdsServiceImpl) FindDeniedMarkets() ([]Token, error) {
	var list []Token
	err := s.db.Where("deny = ? and is_market = ?", true, true).Find(&list).Error
	return list, err
}
