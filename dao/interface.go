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
	Save(item interface{}) error
	FindAll(item interface{}) error

	// ring mined table
	FindRingMinedByRingHash(ringhash string) (*RingMinedEvent, error)
	RollBackRingMined(from, to int64) error

	// order table
	GetOrderByHash(orderhash common.Hash) (*Order, error)
	GetOrdersByHash(orderhashs []string) (map[string]Order, error)
	MarkMinerOrders(filterOrderhashs []string, blockNumber int64) error
	GetOrdersForMiner(protocol, tokenS, tokenB string, length int, filterStatus []types.OrderStatus, currentBlockNumber int64) ([]*Order, error)
	GetOrdersWithBlockNumberRange(from, to int64) ([]Order, error)
	GetCutoffOrders(cutoffTime int64) ([]Order, error)
	SetCutOff(owner common.Address, cutoffTime *big.Int) error
	CheckOrderCutoff(orderhash string, cutoff int64) bool
	GetOrderBook(protocol, tokenS, tokenB common.Address, length int) ([]Order, error)
	OrderPageQuery(query map[string]interface{}, pageIndex, pageSize int) (PageResult, error)
	UpdateBroadcastTimeByHash(hash string, bt int) error
	UpdateOrderWhileFill(hash common.Hash, status types.OrderStatus, dealtAmountS, dealtAmountB, splitAmountS, splitAmountB, blockNumber *big.Int) error
	UpdateOrderWhileCancel(hash common.Hash, status types.OrderStatus, cancelledAmountS, cancelledAmountB, blockNumber *big.Int) error
	GetFrozenAmount(owner common.Address, token common.Address, statusSet []types.OrderStatus) ([]Order, error)

	// block table
	FindBlockByHash(blockhash common.Hash) (*Block, error)
	FindBlockByParentHash(parenthash common.Hash) (*Block, error)
	FindLatestBlock() (*Block, error)
	FindForkBlock() (*Block, error)
	SetForkBlock(blockhash common.Hash) error

	// fill event table
	FindFillEventByRinghashAndOrderhash(ringhash, orderhash common.Hash) (*FillEvent, error)
	QueryRecentFills(mkt, owner string, start int64, end int64) (fills []FillEvent, err error)
	RollBackFill(from, to int64) error
	FillsPageQuery(query map[string]interface{}, pageIndex, pageSize int) (res PageResult, err error)

	// cancel event table
	FindCancelEvent(orderhash, txhash common.Hash) (*CancelEvent, error)
	RollBackCancel(from, to int64) error

	// cutoff event table
	GetCutoffEvent(protocol, owner common.Address) (*CutOffEvent, error)
	DelCutoffEvent(protocol, owner common.Address) error
	UpdateCutoffByProtocolAndOwner(protocol, owner common.Address, txhash common.Hash, blockNumber, cutoff, createTime *big.Int) error
	RollBackCutoff(from, to int64) error

	// trend table
	TrendPageQuery(query Trend, pageIndex, pageSize int) (pageResult PageResult, err error)
	TrendQueryByTime(intervals, market string, start, end int64) (trends []Trend, err error)

	// white list
	GetWhiteList() ([]WhiteList, error)
	FindWhiteListUserByAddress(address common.Address) (*WhiteList, error)

	//ringSubmitInfo
	UpdateRingSubmitInfoRegistryTxHash(ringhashs []common.Hash, txHash string) error
	UpdateRingSubmitInfoProtocolTxHash(ringhash common.Hash, txHash string) error
	UpdateRingSubmitInfoRegistryUsedGas(txHash string, usedGas *big.Int) error
	UpdateRingSubmitInfoSubmitUsedGas(txHash string, usedGas *big.Int) error
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
