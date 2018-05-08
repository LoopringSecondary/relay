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
	txtyp "github.com/Loopring/relay/txmanager/types"
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
	FindRingMined(txhash string) (*RingMinedEvent, error)
	RollBackRingMined(from, to int64) error

	// order table
	GetOrderByHash(orderhash common.Hash) (*Order, error)
	GetOrdersByHash(orderhashs []string) (map[string]Order, error)
	MarkMinerOrders(filterOrderhashs []string, blockNumber int64) error
	GetOrdersForMiner(protocol, tokenS, tokenB string, length int, filterStatus []types.OrderStatus, reservedTime, startBlockNumber, endBlockNumber int64) ([]*Order, error)
	GetCutoffOrders(owner common.Address, cutoffTime *big.Int) ([]Order, error)
	GetCutoffPairOrders(owner, token1, token2 common.Address, cutoffTime *big.Int) ([]Order, error)
	SetCutOffOrders(orderHashList []common.Hash, blockNumber *big.Int) error
	GetOrderBook(protocol, tokenS, tokenB common.Address, length int) ([]Order, error)
	OrderPageQuery(query map[string]interface{}, statusList []int, pageIndex, pageSize int) (PageResult, error)
	UpdateBroadcastTimeByHash(hash string, bt int) error
	UpdateOrderWhileRollbackCutoff(orderhash common.Hash, status types.OrderStatus, blockNumber *big.Int) error
	UpdateOrderWhileFill(hash common.Hash, status types.OrderStatus, dealtAmountS, dealtAmountB, splitAmountS, splitAmountB, blockNumber *big.Int) error
	UpdateOrderWhileCancel(hash common.Hash, status types.OrderStatus, cancelledAmountS, cancelledAmountB, blockNumber *big.Int) error
	GetFrozenAmount(owner common.Address, token common.Address, statusSet []types.OrderStatus, delegateAddress common.Address) ([]Order, error)
	GetFrozenLrcFee(owner common.Address, statusSet []types.OrderStatus) ([]Order, error)

	// block table
	FindBlockByHash(blockhash common.Hash) (*Block, error)
	FindLatestBlock() (*Block, error)
	SetForkBlock(from, to int64) error
	SaveBlock(latest *Block) error

	// fill event table
	FindFillEvent(txhash string, FillIndex int64) (*FillEvent, error)
	QueryRecentFills(mkt, owner string, start int64, end int64) (fills []FillEvent, err error)
	GetFillForkEvents(from, to int64) ([]FillEvent, error)
	RollBackFill(from, to int64) error
	FillsPageQuery(query map[string]interface{}, pageIndex, pageSize int) (res PageResult, err error)
	GetLatestFills(query map[string]interface{}, limit int) (res []FillEvent, err error)
	FindFillsByRingHash(ringHash common.Hash) ([]FillEvent, error)

	// cancel event table
	GetCancelEvent(txhash common.Hash) (CancelEvent, error)
	RollBackCancel(from, to int64) error
	GetCancelForkEvents(from, to int64) ([]CancelEvent, error)

	// cutoff event table
	GetCutoffEvent(txhash common.Hash) (CutOffEvent, error)
	GetCutoffForkEvents(from, to int64) ([]CutOffEvent, error)
	RollBackCutoff(from, to int64) error

	// cutoffpair event table
	GetCutoffPairEvent(txhash common.Hash) (CutOffPairEvent, error)
	GetCutoffPairForkEvents(from, to int64) ([]CutOffPairEvent, error)
	RollBackCutoffPair(from, to int64) error

	// trend table
	TrendQueryLatest(query Trend, pageIndex, pageSize int) (trends []Trend, err error)
	TrendQueryByTime(intervals, market string, start, end int64) (trends []Trend, err error)
	TrendQueryByInterval(intervals, market string, start, end int64) (trends []Trend, err error)
	TrendQueryForProof(mkt string, interval string, start int64) (trends []Trend, err error)

	// white list
	GetWhiteList() ([]WhiteList, error)
	FindWhiteListUserByAddress(address common.Address) (*WhiteList, error)

	//ringSubmitInfo
	//UpdateRingSubmitInfoProtocolTxHash(ringhash common.Hash, txHash string) error
	//UpdateRingSubmitInfoSubmitUsedGas(txHash string, usedGas *big.Int) error
	//UpdateRingSubmitInfoFailed(ringhashs []common.Hash, err string) error

	UpdateRingSubmitInfoResult(submitResult *types.RingSubmitResultEvent) error
	GetRingForSubmitByHash(ringhash common.Hash) (RingSubmitInfo, error)
	GetRingHashesByTxHash(txHash common.Hash) ([]*RingSubmitInfo, error)
	RingMinedPageQuery(query map[string]interface{}, pageIndex, pageSize int) (res PageResult, err error)
	GetRingminedMethods(lastId int, limit int) ([]RingMinedEvent, error)
	GetFilledOrderByRinghash(ringhash common.Hash) ([]*FilledOrder, error)

	// transactions
	GetTransactionById(id int) (Transaction, error)

	// transactionEntity
	FindPendingTxEntity(hash string) (TransactionEntity, error)
	FindTxEntity(txhash string, logIndex int64) (TransactionEntity, error)
	GetTxEntity(hashlist []string) ([]TransactionEntity, error)
	GetPendingTxEntity(from string, nonce int64) ([]TransactionEntity, error)
	SetPendingTxEntityFailed(hashlist []string) error
	DelPendingTxEntity(hash string) error
	RollBackTxEntity(from, to int64) error

	// transactionView
	DelPendingTxView(hash string) error
	SetPendingTxViewFailed(hashlist []string) error
	GetTxViewByOwnerAndHashs(owner string, hashs []string) ([]TransactionView, error)
	GetPendingTxViewByOwner(owner string) ([]TransactionView, error)
	GetTxViewCountByOwner(owner string, symbol string, status types.TxStatus, typ txtyp.TxType) (int, error)
	GetTxViewByOwner(owner string, symbol string, status types.TxStatus, typ txtyp.TxType, limit, offset int) ([]TransactionView, error)
	RollBackTxView(from, to int64) error

	// checkpoint
	QueryCheckPointByType(businessType string) (point CheckPoint, err error)
}
