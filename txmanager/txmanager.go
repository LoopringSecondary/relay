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

package txmanager

import (
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"math/big"
)

type TransactionManager struct {
	db                         dao.RdsService
	accountmanager             *market.AccountManager
	approveEventWatcher        *eventemitter.Watcher
	orderCancelledEventWatcher *eventemitter.Watcher
	cutoffAllEventWatcher      *eventemitter.Watcher
	cutoffPairEventWatcher     *eventemitter.Watcher
	wethDepositEventWatcher    *eventemitter.Watcher
	wethWithdrawalEventWatcher *eventemitter.Watcher
	transferEventWatcher       *eventemitter.Watcher
	orderFilledEventWatcher    *eventemitter.Watcher
	ethTransferEventWatcher    *eventemitter.Watcher
}

func NewTxManager(db dao.RdsService, accountmanager *market.AccountManager) TransactionManager {
	var tm TransactionManager
	tm.db = db
	tm.accountmanager = accountmanager
	return tm
}

// Start start orderbook as a service
func (tm *TransactionManager) Start() {
	tm.approveEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveApproveEvent}
	eventemitter.On(eventemitter.ApprovalEvent, tm.approveEventWatcher)

	tm.orderCancelledEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveOrderCancelledEvent}
	eventemitter.On(eventemitter.OrderCancelledEvent, tm.orderCancelledEventWatcher)

	tm.cutoffAllEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCutoffAllEvent}
	eventemitter.On(eventemitter.CutoffAllEvent, tm.cutoffAllEventWatcher)

	tm.cutoffPairEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCutoffPairEvent}
	eventemitter.On(eventemitter.CutoffPairEvent, tm.cutoffPairEventWatcher)

	tm.wethDepositEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveWethDepositEvent}
	eventemitter.On(eventemitter.WethDepositEvent, tm.wethDepositEventWatcher)

	tm.wethWithdrawalEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveWethWithdrawalEvent}
	eventemitter.On(eventemitter.WethWithdrawalEvent, tm.wethWithdrawalEventWatcher)

	tm.transferEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveTransferEvent}
	eventemitter.On(eventemitter.TransferEvent, tm.transferEventWatcher)

	tm.orderFilledEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveOrderFilledEvent}
	eventemitter.On(eventemitter.OrderFilledEvent, tm.orderFilledEventWatcher)

	tm.ethTransferEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveEthTransferEvent}
	eventemitter.On(eventemitter.EthTransferEvent, tm.ethTransferEventWatcher)
}

func (tm *TransactionManager) Stop() {
	eventemitter.Un(eventemitter.ApprovalEvent, tm.approveEventWatcher)

	eventemitter.Un(eventemitter.OrderCancelledEvent, tm.orderCancelledEventWatcher)

	eventemitter.Un(eventemitter.CutoffAllEvent, tm.cutoffAllEventWatcher)

	eventemitter.Un(eventemitter.CutoffPairEvent, tm.cutoffPairEventWatcher)

	eventemitter.Un(eventemitter.WethDepositEvent, tm.wethDepositEventWatcher)

	eventemitter.Un(eventemitter.WethWithdrawalEvent, tm.wethWithdrawalEventWatcher)

	eventemitter.Un(eventemitter.TransferEvent, tm.transferEventWatcher)

	eventemitter.Un(eventemitter.OrderFilledEvent, tm.orderFilledEventWatcher)

	eventemitter.Un(eventemitter.EthTransferEvent, tm.ethTransferEventWatcher)
}

const (
	ETH_SYMBOL = "ETH"
	NIL_SYMBOL = ""
)

func (tm *TransactionManager) SaveApproveEvent(input eventemitter.EventData) error {
	evt := input.(*types.ApprovalEvent)
	var tx types.Transaction
	tx.FromApproveEvent(evt)
	tx.Symbol, _ = util.GetSymbolWithAddress(tx.Protocol)
	return tm.saveTransaction(&tx)
}

func (tm *TransactionManager) SaveOrderCancelledEvent(input eventemitter.EventData) error {
	evt := input.(*types.OrderCancelledEvent)
	var tx types.Transaction
	tx.FromCancelEvent(evt)
	tx.Symbol = ETH_SYMBOL
	return tm.saveTransaction(&tx)
}

func (tm *TransactionManager) SaveCutoffAllEvent(input eventemitter.EventData) error {
	evt := input.(*types.CutoffEvent)
	var tx types.Transaction
	tx.FromCutoffEvent(evt)
	tx.Symbol = ETH_SYMBOL
	return tm.saveTransaction(&tx)
}

func (tm *TransactionManager) SaveCutoffPairEvent(input eventemitter.EventData) error {
	evt := input.(*types.CutoffPairEvent)
	var tx types.Transaction
	tx.FromCutoffPairEvent(evt)
	tx.Symbol = ETH_SYMBOL
	return tm.saveTransaction(&tx)
}

func (tm *TransactionManager) SaveWethDepositEvent(input eventemitter.EventData) error {
	evt := input.(*types.WethDepositEvent)
	var tx1, tx2 types.Transaction

	// save weth
	tx1.FromWethDepositEvent(evt, true)
	tx1.Symbol, _ = util.GetSymbolWithAddress(tx1.Protocol)
	if err := tm.saveTransaction(&tx1); err != nil {
		return err
	}

	// save eth
	tx2.FromWethDepositEvent(evt, false)
	tx2.Protocol = types.NilAddress
	tx2.Symbol = ETH_SYMBOL
	if err := tm.saveTransaction(&tx2); err != nil {
		return err
	}

	return nil
}

func (tm *TransactionManager) SaveWethWithdrawalEvent(input eventemitter.EventData) error {
	evt := input.(*types.WethWithdrawalEvent)
	var tx1, tx2 types.Transaction

	log.Debugf("extractor:tx:%s saveWethWithdrawalEventAsTx", evt.TxHash.Hex())

	// save weth
	tx1.FromWethWithdrawalEvent(evt, false)
	tx1.Symbol, _ = util.GetSymbolWithAddress(tx1.Protocol)
	if err := tm.saveTransaction(&tx1); err != nil {
		return err
	}

	// save eth
	tx2.FromWethWithdrawalEvent(evt, true)
	tx2.Protocol = types.NilAddress
	tx2.Symbol = ETH_SYMBOL
	if err := tm.saveTransaction(&tx2); err != nil {
		return err
	}

	return nil
}

func (tm *TransactionManager) SaveTransferEvent(input eventemitter.EventData) error {
	evt := input.(*types.TransferEvent)
	var tx1, tx2 types.Transaction

	log.Debugf("extractor:tx:%s saveTransferAsTx", evt.TxHash.Hex())

	tx1.FromTransferEvent(evt, types.TX_TYPE_SEND)
	tx1.Symbol, _ = util.GetSymbolWithAddress(tx1.Protocol)
	tx2.FromTransferEvent(evt, types.TX_TYPE_RECEIVE)
	tx2.Symbol = tx1.Symbol
	if err := tm.saveTransaction(&tx1); err != nil {
		return err
	}
	if err := tm.saveTransaction(&tx2); err != nil {
		return err
	}

	return nil
}

func (tm *TransactionManager) SaveOrderFilledEvent(input eventemitter.EventData) error {
	evt := input.(*types.OrderFilledEvent)
	var tx1, tx2 types.Transaction

	tx1.FromFillEvent(evt, types.TX_TYPE_BUY)
	tx1.Symbol = NIL_SYMBOL
	tm.saveTransaction(&tx1)

	tx2.FromFillEvent(evt, types.TX_TYPE_SELL)
	tx1.Symbol = NIL_SYMBOL
	tm.saveTransaction(&tx2)

	return nil
}

// 普通的transaction
// 当value大于0时认为是eth转账
// 当value等于0时认为是调用系统不支持的合约,默认使用fromTransferEvent/send type为unsupported_contract
func (tm *TransactionManager) SaveEthTransferEvent(input eventemitter.EventData) error {
	evt := input.(*types.TransferEvent)

	if evt.Amount.Cmp(big.NewInt(0)) > 0 {
		var tx1, tx2 types.Transaction

		tx1.FromTransferEvent(evt, types.TX_TYPE_SEND)
		tx1.Protocol = types.NilAddress
		tx1.Symbol = ETH_SYMBOL
		if err := tm.saveTransaction(&tx1); err != nil {
			return err
		}

		tx2.FromTransferEvent(evt, types.TX_TYPE_RECEIVE)
		tx2.Protocol = types.NilAddress
		tx2.Symbol = ETH_SYMBOL
		if err := tm.saveTransaction(&tx2); err != nil {
			return err
		}
	} else {
		var tx types.Transaction
		tx.FromTransferEvent(evt, types.TX_TYPE_SEND)
		tx.Type = types.TX_TYPE_UNSUPPORTED_CONTRACT
		tx.Protocol = tx.To
		tx.Symbol = ETH_SYMBOL
		if err := tm.saveTransaction(&tx); err != nil {
			return err
		}
	}

	return nil
}

func (tm *TransactionManager) saveTransaction(tx *types.Transaction) error {
	var model dao.Transaction

	tx.CreateTime = tx.BlockTime
	tx.UpdateTime = tx.UpdateTime

	model.ConvertDown(tx)

	if unlocked, _ := tm.accountmanager.HasUnlocked(tx.Owner.Hex()); unlocked {
		return tm.db.SaveTransaction(&model)
	}

	return nil
}
