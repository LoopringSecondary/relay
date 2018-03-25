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
	db             dao.RdsService
	accountmanager *market.AccountManager

	approveMethodWatcher *eventemitter.Watcher
	approveEventWatcher  *eventemitter.Watcher

	cancelOrderMethodWatcher   *eventemitter.Watcher
	orderCancelledEventWatcher *eventemitter.Watcher

	cutoffAllMethodWatcher *eventemitter.Watcher
	cutoffAllEventWatcher  *eventemitter.Watcher

	cutoffPairMethodWatcher *eventemitter.Watcher
	cutoffPairEventWatcher  *eventemitter.Watcher

	wethDepositMethodWatcher *eventemitter.Watcher
	wethDepositEventWatcher  *eventemitter.Watcher

	wethWithdrawalMethodWatcher *eventemitter.Watcher
	wethWithdrawalEventWatcher  *eventemitter.Watcher

	transferMethodWatcher *eventemitter.Watcher
	transferEventWatcher  *eventemitter.Watcher

	orderFilledEventWatcher *eventemitter.Watcher

	ethTransferEventWatcher *eventemitter.Watcher
}

func NewTxManager(db dao.RdsService, accountmanager *market.AccountManager) TransactionManager {
	var tm TransactionManager
	tm.db = db
	tm.accountmanager = accountmanager
	return tm
}

// Start start orderbook as a service
func (tm *TransactionManager) Start() {
	tm.approveMethodWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveApproveMethod}
	eventemitter.On(eventemitter.TxManagerApproveMethod, tm.approveMethodWatcher)
	tm.approveEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveApproveEvent}
	eventemitter.On(eventemitter.TxManagerApproveEvent, tm.approveEventWatcher)

	tm.cancelOrderMethodWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCancelOrderMethod}
	eventemitter.On(eventemitter.TxManagerCancelOrderMethod, tm.cancelOrderMethodWatcher)
	tm.orderCancelledEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveOrderCancelledEvent}
	eventemitter.On(eventemitter.TxManagerOrderCancelledEvent, tm.orderCancelledEventWatcher)

	tm.cutoffAllMethodWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCutoffAllMethod}
	eventemitter.On(eventemitter.TxManagerCutoffAllMethod, tm.cutoffAllMethodWatcher)
	tm.cutoffAllEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCutoffAllEvent}
	eventemitter.On(eventemitter.TxManagerCutoffAllEvent, tm.cutoffAllEventWatcher)

	tm.cutoffPairMethodWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCutoffPairMethod}
	eventemitter.On(eventemitter.TxManagerCutoffPairMethod, tm.cutoffPairMethodWatcher)
	tm.cutoffPairEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCutoffPairEvent}
	eventemitter.On(eventemitter.TxManagerCutoffPairEvent, tm.cutoffPairEventWatcher)

	tm.wethDepositMethodWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveWethDepositMethod}
	eventemitter.On(eventemitter.TxManagerWethDepositMethod, tm.wethDepositMethodWatcher)
	tm.wethDepositEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveWethDepositEvent}
	eventemitter.On(eventemitter.TxManagerWethDepositEvent, tm.wethDepositEventWatcher)

	tm.wethWithdrawalMethodWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveWethWithdrawalMethod}
	eventemitter.On(eventemitter.TxManagerWethWithdrawalMethod, tm.wethWithdrawalMethodWatcher)
	tm.wethWithdrawalEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveWethWithdrawalEvent}
	eventemitter.On(eventemitter.TxManagerWethWithdrawalEvent, tm.wethWithdrawalEventWatcher)

	tm.transferMethodWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveTransferMethod}
	eventemitter.On(eventemitter.TxManagerTransferMethod, tm.transferMethodWatcher)
	tm.transferEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveTransferEvent}
	eventemitter.On(eventemitter.TxManagerTransferEvent, tm.transferEventWatcher)

	tm.orderFilledEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveOrderFilledEvent}
	eventemitter.On(eventemitter.TxManagerOrderFilledEvent, tm.orderFilledEventWatcher)

	tm.ethTransferEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveEthTransferEvent}
	eventemitter.On(eventemitter.TxManagerEthTransferEvent, tm.ethTransferEventWatcher)
}

func (tm *TransactionManager) Stop() {
	eventemitter.Un(eventemitter.TxManagerApproveMethod, tm.approveMethodWatcher)
	eventemitter.Un(eventemitter.TxManagerApproveEvent, tm.approveEventWatcher)

	eventemitter.Un(eventemitter.TxManagerCancelOrderMethod, tm.cancelOrderMethodWatcher)
	eventemitter.Un(eventemitter.TxManagerOrderCancelledEvent, tm.orderCancelledEventWatcher)

	eventemitter.Un(eventemitter.TxManagerCutoffAllMethod, tm.cutoffAllMethodWatcher)
	eventemitter.Un(eventemitter.TxManagerCutoffAllEvent, tm.cutoffAllEventWatcher)

	eventemitter.Un(eventemitter.TxManagerCutoffPairMethod, tm.cutoffPairMethodWatcher)
	eventemitter.Un(eventemitter.TxManagerCutoffPairEvent, tm.cutoffPairEventWatcher)

	eventemitter.Un(eventemitter.TxManagerWethDepositMethod, tm.wethDepositMethodWatcher)
	eventemitter.Un(eventemitter.TxManagerWethDepositEvent, tm.wethDepositEventWatcher)

	eventemitter.Un(eventemitter.TxManagerWethWithdrawalMethod, tm.wethWithdrawalMethodWatcher)
	eventemitter.Un(eventemitter.TxManagerWethWithdrawalEvent, tm.wethWithdrawalEventWatcher)

	eventemitter.Un(eventemitter.TxManagerTransferEvent, tm.transferEventWatcher)
	eventemitter.Un(eventemitter.TxManagerTransferMethod, tm.transferMethodWatcher)

	eventemitter.Un(eventemitter.TxManagerOrderFilledEvent, tm.orderFilledEventWatcher)

	eventemitter.Un(eventemitter.TxManagerEthTransferEvent, tm.ethTransferEventWatcher)
}

const ETH_SYMBOL = "ETH"

func (tm *TransactionManager) SaveApproveMethod(input eventemitter.EventData) error {
	evt := input.(*types.ApproveMethodEvent)
	var tx types.Transaction
	tx.FromApproveMethod(evt)
	tx.Symbol, _ = util.GetSymbolWithAddress(tx.Protocol)
	return tm.saveTransaction(&tx)
}

func (tm *TransactionManager) SaveApproveEvent(input eventemitter.EventData) error {
	evt := input.(*types.ApprovalEvent)
	var tx types.Transaction
	tx.FromApproveEvent(evt)
	tx.Symbol, _ = util.GetSymbolWithAddress(tx.Protocol)
	return tm.saveTransaction(&tx)
}

func (tm *TransactionManager) SaveCancelOrderMethod(input eventemitter.EventData) error {
	evt := input.(*types.OrderCancelledEvent)
	var tx types.Transaction
	tx.FromCancelMethod(evt)
	tx.Symbol = ETH_SYMBOL
	return tm.saveTransaction(&tx)
}

func (tm *TransactionManager) SaveOrderCancelledEvent(input eventemitter.EventData) error {
	evt := input.(*types.OrderCancelledEvent)
	var tx types.Transaction
	tx.FromCancelEvent(evt)
	tx.Symbol = ETH_SYMBOL
	return tm.saveTransaction(&tx)
}

func (tm *TransactionManager) SaveCutoffAllMethod(input eventemitter.EventData) error {
	evt := input.(*types.CutoffMethodEvent)
	var tx types.Transaction
	tx.FromCutoffMethodEvent(evt)
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

func (tm *TransactionManager) SaveCutoffPairMethod(input eventemitter.EventData) error {
	evt := input.(*types.CutoffPairMethodEvent)
	var tx types.Transaction
	tx.FromCutoffPairMethod(evt)
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

func (tm *TransactionManager) SaveWethDepositMethod(input eventemitter.EventData) error {
	evt := input.(*types.WethDepositMethodEvent)
	var tx1, tx2 types.Transaction

	// save weth
	tx1.FromWethDepositMethod(evt, true)
	tx1.Symbol, _ = util.GetSymbolWithAddress(tx1.Protocol)
	if err := tm.saveTransaction(&tx1); err != nil {
		return err
	}

	// save eth
	tx2.FromWethDepositMethod(evt, false)
	tx2.Protocol = types.NilAddress
	tx2.Symbol = "ETH"
	if err := tm.saveTransaction(&tx2); err != nil {
		return err
	}

	return nil
}

func (tm *TransactionManager) SaveWethDepositEvent(input eventemitter.EventData) error {
	evt := input.(*types.WethDepositEvent)
	var tx1, tx2 types.Transaction

	log.Debugf("extractor:tx:%s saveWethDepositEventAsTx", evt.TxHash.Hex())

	// save weth
	tx1.FromWethDepositEvent(evt, true)
	tx1.Symbol, _ = util.GetSymbolWithAddress(tx1.Protocol)
	if err := tm.saveTransaction(&tx1); err != nil {
		return err
	}

	// save eth
	tx2.FromWethDepositEvent(evt, false)
	tx2.Protocol = types.NilAddress
	tx2.Symbol = "ETH"
	if err := tm.saveTransaction(&tx2); err != nil {
		return err
	}

	return nil
}

func (tm *TransactionManager) SaveWethWithdrawalMethod(input eventemitter.EventData) error {
	evt := input.(*types.WethWithdrawalMethodEvent)
	var tx1, tx2 types.Transaction

	log.Debugf("extractor:tx:%s saveWethWithdrawalMethodAsTx", evt.TxHash.Hex())

	tx1.FromWethWithdrawalMethod(evt, false)
	tx1.Symbol, _ = util.GetSymbolWithAddress(tx1.Protocol)
	if err := tm.saveTransaction(&tx1); err != nil {
		return err
	}

	tx2.FromWethWithdrawalMethod(evt, true)
	tx2.Protocol = types.NilAddress
	tx2.Symbol = "ETH"
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
	tx2.Symbol = "ETH"
	if err := tm.saveTransaction(&tx2); err != nil {
		return err
	}

	return nil
}

func (tm *TransactionManager) SaveTransferMethod(input eventemitter.EventData) error {
	evt := input.(*types.TransferMethodEvent)
	var tx1, tx2 types.Transaction

	tx1.FromTransferMethodEvent(evt, types.TX_TYPE_SEND)
	tx1.Symbol, _ = util.GetSymbolWithAddress(tx1.Protocol)
	tx2.FromTransferMethodEvent(evt, types.TX_TYPE_RECEIVE)
	tx2.Symbol = tx1.Symbol
	if err := tm.saveTransaction(&tx1); err != nil {
		return err
	}
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
	tx1.Symbol = ""
	tm.saveTransaction(&tx1)

	tx2.FromFillEvent(evt, types.TX_TYPE_SELL)
	tx1.Symbol = ""
	tm.saveTransaction(&tx2)

	return nil
}

// 普通的transaction
// 当value大于0时认为是eth转账
// 当value等于0时认为是调用系统不支持的合约,默认使用fromTransferEvent/send type为unsupported_contract
func (tm *TransactionManager) SaveEthTransferEvent(input eventemitter.EventData) error {
	evt := input.(*types.TransferEvent)

	if evt.Value.Cmp(big.NewInt(0)) > 0 {
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
		tx.Protocol = types.NilAddress
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

	if unlocked, _ := tm.accountmanager.HasUnlocked(tx.Owner.Hex()); unlocked == true {
		return tm.db.SaveTransaction(&model)
	}

	return nil
}
