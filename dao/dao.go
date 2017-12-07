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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type PageResult struct {
	Data      []interface{}
	PageIndex int
	PageSize  int
	Total     int
}

type RdsServiceImpl struct {
	options config.MysqlOptions
	db      *gorm.DB
}

func NewRdsService(options config.MysqlOptions) *RdsServiceImpl {
	impl := &RdsServiceImpl{}
	impl.options = options

	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return options.TablePrefix + defaultTableName
	}

	url := options.User + ":" + options.Password + "@/" + options.DbName + "?charset=utf8&parseTime=True&loc=" + options.Loc
	db, err := gorm.Open("mysql", url)
	if err != nil {
		log.Fatalf("mysql connection error:%s", err.Error())
	}

	db.LogMode(options.Debug)

	impl.db = db

	return impl
}

// create tables if not exists
func (s *RdsServiceImpl) Prepare() {
	var tables []interface{}

	tables = append(tables, &Order{})
	tables = append(tables, &Block{})
	tables = append(tables, &RingMinedEvent{})
	tables = append(tables, &FillEvent{})
	tables = append(tables, &CancelEvent{})
	tables = append(tables, &CutOffEvent{})
	tables = append(tables, &Trend{})
	tables = append(tables, &WhiteList{})
	tables = append(tables, &RingSubmitInfo{})
	tables = append(tables, &Token{})
	tables = append(tables, &EventLog{})

	for _, t := range tables {
		if ok := s.db.HasTable(t); !ok {
			if err := s.db.CreateTable(t).Error; err != nil {
				log.Fatalf("create mysql table error:%s", err.Error())
			}
		}
	}
}
