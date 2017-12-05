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

import "fmt"

// order amountS 上限1e30
type Trend struct {
	ID         int     `gorm:"column:id;primary_key;"`
	Interval   string  `gorm:"column:interval;type:varchar(42)"`
	Market     string  `gorm:"column:market;type:varchar(42)"`
	Vol        float64 `gorm:"column:vol;type:float"`
	Amount     float64 `gorm:"column:amount;type:float"`
	CreateTime int64   `gorm:"column:create_time;type:bigint"`
	Open       float64 `gorm:"column:open;type:float"`
	Close      float64 `gorm:"column:close;type:float"`
	High       float64 `gorm:"column:high;type:float"`
	Low        float64 `gorm:"column:low;type:float"`
	Start      int64   `gorm:"column:start;type:bigint"`
	End        int64   `gorm:"column:end;type:bigint"`
}

func (s *RdsServiceImpl) TrendPageQuery(query Trend, pageIndex, pageSize int) (pageResult PageResult, err error) {

	var result PageResult

	fmt.Println("trend query is .......")
	fmt.Println(query)

	trends := make([]Trend,0)

	if pageIndex <= 0 {
		pageIndex = 1
	}

	if pageSize <= 0 {
		pageSize = 50
	}

	for t := range trends {
		pageResult.Data = append(pageResult.Data, t)
	}

	result.PageIndex = pageIndex
	result.PageSize = pageSize

	fmt.Println(query)
	if err = s.db.Model(&Trend{}).Where(query).Order("start desc").Offset((pageIndex - 1) * pageSize).Limit(pageSize).Find(&trends).Error; err != nil {
		return
	}

	err = s.db.Model(&Trend{}).Where(query).Count(&result.Total).Error
	return
}

func (s *RdsServiceImpl) TrendQueryByTime(market string, start, end int64) (trends []Trend, err error) {
	err = s.db.Where("market = ? and start > ? and end <= ?", market, start, end).Order("start desc").Find(&trends).Error
	return
}
