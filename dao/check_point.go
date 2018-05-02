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

import "qiniupkg.com/x/errors.v7"

const (
	TrendUpdateType = "last_trend__proof_time"
)

// common check point table
type CheckPoint struct {
	ID           int    `gorm:"column:id;primary_key;"`
	BusinessType string `gorm:"column:business_type;type:varchar(42);unique_index"`
	CheckPoint   int64  `gorm:"column:check_point;type:bigint"`
	CreateTime   int64  `gorm:"column:create_time;type:bigint"`
	ModifyTime   int64  `gorm:"column:modify_time;type:bigint"`
}

func (s *RdsServiceImpl) QueryCheckPointByType(businessType string) (point CheckPoint, err error) {
	points := make([]CheckPoint, 0)

	err = s.db.Model(&CheckPoint{}).Where("business_type = ? ", businessType).Find(&points).Error
	if err != nil || len(points) == 0 {
		return point, errors.New("can't found default check point for " + point.BusinessType)
	} else {
		return points[0], nil
	}
}
