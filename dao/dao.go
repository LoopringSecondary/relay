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
	"github.com/Loopring/relay/types"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

type RdsService interface {
	// create tables
	Prepare()

	// base functions
	Add(item interface{}) error
	Del(item interface{}) error
	First(item interface{}) error
	Last(item interface{}) error
	Update(item interface{}) error
	FindAll(item interface{}) error

	// ring mined table
	FindRingMinedByRingHash(ringhash string) (*RingMined, error)
	RollBackRingMined(from, to int64) error

	// order table
	GetOrderByHash(orderhash types.Hash) (*Order, error)
	MarkMinerOrders(filterOrderhashs []string, blockNumber int64) error
	UnMarkMinerOrders(blockNumber int64) error
	GetOrdersForMiner(tokenS, tokenB string, filterStatus []uint8) ([]Order, error)
	GetOrdersWithBlockNumberRange(from, to int64) ([]Order, error)
	GetCutoffOrders(cutoffTime int64) ([]Order, error)
	SettleOrdersStatus(orderhashs []string, status types.OrderStatus) error
	CheckOrderCutoff(orderhash string, cutoff int64) bool

	// block table
	FindBlockByHash(blockhash types.Hash) (*Block, error)
	FindBlockByParentHash(parenthash types.Hash) (*Block, error)
	FindLatestBlock() (*Block, error)
	FindForkBlock() (*Block, error)

	// fill event table
	FindFillEventByRinghashAndOrderhash(ringhash, orderhash types.Hash) (*FillEvent, error)
	FirstPreMarket(tokenS, tokenB string) (fill FillEvent, err error)
	QueryRecentFills(tokenS string, tokenB string, start int64, end int64) (fills []FillEvent, err error)
	RollBackFill(from, to int64) error

	// cancel event table
	FindCancelEventByOrderhash(orderhash types.Hash) (*CancelEvent, error)
	RollBackCancel(from, to int64) error

	// cutoff event table
	FindCutoffEventByOwnerAddress(owner types.Address) (*CutOffEvent, error)
	RollBackCutoff(from, to int64) error

	// trend table
	TrendPageQuery(query Trend, pageIndex, pageSize int) (pageResult PageResult, err error)
	TrendQueryByTime(market string, start, end int64) (trends []Trend, err error)
}

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

	impl.db = db

	return impl
}

// create tables if not exists
func (s *RdsServiceImpl) Prepare() {
	var tables []interface{}

	tables = append(tables, &Order{})
	tables = append(tables, &Block{})
	tables = append(tables, &RingMined{})
	tables = append(tables, &FillEvent{})
	tables = append(tables, &CancelEvent{})
	tables = append(tables, &CutOffEvent{})
	tables = append(tables, &Trend{})

	for _, t := range tables {
		if ok := s.db.HasTable(t); !ok {
			if err := s.db.CreateTable(t).Error; err != nil {
				log.Errorf("create mysql table error:%s", err.Error())
			}
		}
	}
}
