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

// order amountS 上限1e30

type Trend struct {
	ID                    int     `gorm:"column:id;primary_key;"`
	Interval              string  `gorm:"column:interval;type:varchar(42)"`
	Market     string  `gorm:"column:market;type:varchar(42)"`
	AmountS    []byte  `gorm:"column:amount_s;type:varchar(30)"`
	AmountB    []byte  `gorm:"column:amount_b;type:varchar(30)"`
	CreateTime int64   `gorm:"column:create_time";type:bigint`
	Open       float64 `gorm:"column:open;type:decimal(28,16);"`
	Close      float64 `gorm:"column:close;type:decimal(28,16);"`
	High       float64 `gorm:"column:high;type:decimal(28,16);"`
	Low        float64 `gorm:"column:low;type:decimal(28,16);"`
	Start      int64   `gorm:"column:start";type:bigint`
	End        int64   `gorm:"column:end";type:bigint`

}

func (s *RdsServiceImpl) create(trend Trend) error {
	return s.db.Create(trend).Error
}

func (s *RdsServiceImpl) TrendPageQuery(query Trend, pageIndex, pageSize int) (pageResult PageResult, err error) {

	var result PageResult

	if pageIndex <= 0 {
		pageIndex = 1
	}

	if pageSize <= 0 {
		pageSize = 50
	}

	result.PageIndex = pageIndex
	result.PageSize = pageSize

	if err = s.db.Where(query).Order("start desc").Offset(pageIndex * pageSize).Limit(pageSize).Find(&result.Data).Error; err != nil {
		return
	}

	err = s.db.Where(query).Count(&result.Total).Error
	return
}
