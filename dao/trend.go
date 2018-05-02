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
	ID         int     `gorm:"column:id;primary_key;"`
	Market     string  `gorm:"column:market;type:varchar(42);unique_index:market_intervals_start"`
	Intervals  string  `gorm:"column:intervals;type:varchar(42);unique_index:market_intervals_start"`
	Vol        float64 `gorm:"column:vol;type:float"`
	Amount     float64 `gorm:"column:amount;type:float"`
	CreateTime int64   `gorm:"column:create_time;type:bigint"`
	UpdateTime int64   `gorm:"column:update_time;type:bigint"`
	Open       float64 `gorm:"column:open;type:float"`
	Close      float64 `gorm:"column:close;type:float"`
	High       float64 `gorm:"column:high;type:float"`
	Low        float64 `gorm:"column:low;type:float"`
	Start      int64   `gorm:"column:start;type:bigint;unique_index:market_intervals_start"`
	End        int64   `gorm:"column:end;type:bigint"`
}

func (s *RdsServiceImpl) TrendQueryLatest(query Trend, pageIndex, pageSize int) (trends []Trend, err error) {
	trends = make([]Trend, 0)

	if pageIndex <= 0 {
		pageIndex = 1
	}

	if pageSize <= 0 {
		pageSize = 50
	}

	err = s.db.Model(&Trend{}).Where(query).Order("start desc").Offset((pageIndex - 1) * pageSize).Limit(pageSize).Find(&trends).Error
	return
}

func (s *RdsServiceImpl) TrendQueryByTime(intervals, market string, start, end int64) (trends []Trend, err error) {
	err = s.db.Model(&Trend{}).Where("intervals = ? and market = ? and start = ? and end = ?", intervals, market, start, end).Order("start desc").Find(&trends).Error
	return
}

func (s *RdsServiceImpl) TrendQueryByInterval(intervals, market string, start, end int64) (trends []Trend, err error) {
	err = s.db.Model(&Trend{}).Where("intervals = ? and market = ? and start >= ? and end <= ?", intervals, market, start, end).Order("start").Find(&trends).Error
	return
}

func (s *RdsServiceImpl) TrendQueryForProof(mkt, interval string, start int64) (trends []Trend, err error) {
	trends = make([]Trend, 0)
	err = s.db.Model(&Trend{}).Where("intervals = ? and market = ? and start >= ?", interval, mkt, start).Find(&trends).Error
	return
}
