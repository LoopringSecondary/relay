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
	GetOrderByHash(orderhash common.Hash) (*Order, error)
	GetOrdersByHash(orderhashs []string) (map[string]Order, error)
	MarkMinerOrders(filterOrderhashs []string, blockNumber int64) error
	UnMarkMinerOrders(blockNumber int64) error
	GetOrdersForMiner(tokenS, tokenB string, length int, filterStatus []types.OrderStatus) ([]*Order, error)
	GetOrdersWithBlockNumberRange(from, to int64) ([]Order, error)
	GetCutoffOrders(cutoffTime int64) ([]Order, error)
	SettleOrdersStatus(orderhashs []string, status types.OrderStatus) error
	CheckOrderCutoff(orderhash string, cutoff int64) bool
	GetOrderBook(protocol, tokenS, tokenB common.Address, length int) ([]Order, error)
	OrderPageQuery(query map[string]interface{}, pageIndex, pageSize int) (PageResult, error)
	UpdateBroadcastTimeByHash(hash string, bt int) error

	// block table
	FindBlockByHash(blockhash common.Hash) (*Block, error)
	FindBlockByParentHash(parenthash common.Hash) (*Block, error)
	FindLatestBlock() (*Block, error)
	FindForkBlock() (*Block, error)

	// fill event table
	FindFillEventByRinghashAndOrderhash(ringhash, orderhash common.Hash) (*FillEvent, error)
	FirstPreMarket(tokenS, tokenB string) (fill FillEvent, err error)
	QueryRecentFills(mkt, owner string, start int64, end int64) (fills []FillEvent, err error)
	RollBackFill(from, to int64) error
	FillsPageQuery(query map[string]interface{}, pageIndex, pageSize int) (res PageResult, err error)

	// cancel event table
	FindCancelEvent(orderhash common.Hash, cancelledAmount *big.Int) (*CancelEvent, error)
	RollBackCancel(from, to int64) error

	// cutoff event table
	FindCutoffEventByOwnerAddress(owner common.Address) (*CutOffEvent, error)
	FindValidCutoffEvents() ([]types.CutoffEvent, error)
	RollBackCutoff(from, to int64) error

	// trend table
	TrendPageQuery(query Trend, pageIndex, pageSize int) (pageResult PageResult, err error)
	TrendQueryByTime(market string, start, end int64) (trends []Trend, err error)

	// white list
	GetWhiteList() ([]WhiteList, error)

	//ringSubmitInfo
	UpdateRingSubmitInfoRegistryTxHash(ringhashs []common.Hash, txHash, err string) error
	UpdateRingSubmitInfoSubmitTxHash(ringhash common.Hash, txHash, err string) error
	UpdateRingSubmitInfoFailed(ringhashs []common.Hash, err string) error
	GetRingForSubmitByHash(ringhash common.Hash) (RingSubmitInfo, error)
	GetRingHashesByTxHash(txHash common.Hash) ([]common.Hash, error)
	RingMinedPageQuery(query map[string]interface{}, pageIndex, pageSize int) (res PageResult, err error)

	// token
	FindUnDeniedTokens() ([]Token, error)
	FindDeniedTokens() ([]Token, error)
	FindUnDeniedMarkets() ([]Token, error)
	FindDeniedMarkets() ([]Token, error)
}
