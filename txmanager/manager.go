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
	txtyp "github.com/Loopring/relay/txmanager/types"
	"github.com/Loopring/relay/types"
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
	ethTransferEventWatcher    *eventemitter.Watcher
	forkDetectedEventWatcher   *eventemitter.Watcher
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
	eventemitter.On(eventemitter.Approve, tm.approveEventWatcher)

	tm.orderCancelledEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveOrderCancelledEvent}
	eventemitter.On(eventemitter.CancelOrder, tm.orderCancelledEventWatcher)

	tm.cutoffAllEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCutoffAllEvent}
	eventemitter.On(eventemitter.CutoffAll, tm.cutoffAllEventWatcher)

	tm.cutoffPairEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveCutoffPairEvent}
	eventemitter.On(eventemitter.CutoffPair, tm.cutoffPairEventWatcher)

	tm.wethDepositEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveWethDepositEvent}
	eventemitter.On(eventemitter.WethDeposit, tm.wethDepositEventWatcher)

	tm.wethWithdrawalEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveWethWithdrawalEvent}
	eventemitter.On(eventemitter.WethWithdrawal, tm.wethWithdrawalEventWatcher)

	tm.transferEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveTransferEvent}
	eventemitter.On(eventemitter.Transfer, tm.transferEventWatcher)

	tm.ethTransferEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.SaveEthTransferEvent}
	eventemitter.On(eventemitter.EthTransferEvent, tm.ethTransferEventWatcher)

	tm.forkDetectedEventWatcher = &eventemitter.Watcher{Concurrent: false, Handle: tm.ForkProcess}
	eventemitter.On(eventemitter.ChainForkDetected, tm.forkDetectedEventWatcher)
}

func (tm *TransactionManager) Stop() {
	eventemitter.Un(eventemitter.Approve, tm.approveEventWatcher)
	eventemitter.Un(eventemitter.CancelOrder, tm.orderCancelledEventWatcher)
	eventemitter.Un(eventemitter.CutoffAll, tm.cutoffAllEventWatcher)
	eventemitter.Un(eventemitter.CutoffPair, tm.cutoffPairEventWatcher)
	eventemitter.Un(eventemitter.WethDeposit, tm.wethDepositEventWatcher)
	eventemitter.Un(eventemitter.WethWithdrawal, tm.wethWithdrawalEventWatcher)
	eventemitter.Un(eventemitter.Transfer, tm.transferEventWatcher)
	eventemitter.Un(eventemitter.EthTransferEvent, tm.ethTransferEventWatcher)
	eventemitter.Un(eventemitter.ChainForkDetected, tm.forkDetectedEventWatcher)
}

func (tm *TransactionManager) ForkProcess(input eventemitter.EventData) error {
	log.Debugf("txmanager,processing chain fork......")

	tm.Stop()
	forkEvent := input.(*types.ForkedEvent)
	from := forkEvent.ForkBlock.Int64()
	to := forkEvent.DetectedBlock.Int64()
	if err := tm.db.RollBackTxEntity(from, to); err != nil {
		log.Warnf("txmanager,process fork error:%s", err.Error())
	}
	if err := tm.db.RollBackTxView(from, to); err != nil {
		log.Warnf("txmanager,process fork error:%s", err.Error())
	}
	tm.Start()

	return nil
}

func (tm *TransactionManager) SaveApproveEvent(input eventemitter.EventData) error {
	event := input.(*types.ApprovalEvent)

	log.Debugf("transaction manager, Approve tx:%s, owner:%s, spender:%s", event.TxHash.Hex(), event.Owner.Hex(), event.Spender.Hex())

	var (
		entity txtyp.TransactionEntity
		list   []txtyp.TransactionView
	)

	entity.FromApproveEvent(event)
	view, err := txtyp.ApproveView(event)
	if err != nil {
		return err
	}
	list = append(list, view)

	return tm.saveTransaction(&entity, list)
}

func (tm *TransactionManager) SaveOrderCancelledEvent(input eventemitter.EventData) error {
	event := input.(*types.OrderCancelledEvent)

	log.Debugf("transaction manager, CancelOrder tx:%s, from:%s, orderhash:%s", event.TxHash.Hex(), event.From.Hex(), event.OrderHash.Hex())

	var (
		entity txtyp.TransactionEntity
		list   []txtyp.TransactionView
	)

	entity.FromCancelEvent(event)
	view := txtyp.CancelView(event)
	list = append(list, view)

	return tm.saveTransaction(&entity, list)
}

func (tm *TransactionManager) SaveCutoffAllEvent(input eventemitter.EventData) error {
	event := input.(*types.CutoffEvent)

	log.Debugf("transaction manager, Cutoff tx:%s, owner:%s, cutoff:%s", event.TxHash.Hex(), event.Owner.Hex(), event.Cutoff.String())

	var (
		entity txtyp.TransactionEntity
		list   []txtyp.TransactionView
	)
	entity.FromCutoffEvent(event)
	view := txtyp.CutoffView(event)
	list = append(list, view)

	return tm.saveTransaction(&entity, list)
}

func (tm *TransactionManager) SaveCutoffPairEvent(input eventemitter.EventData) error {
	event := input.(*types.CutoffPairEvent)

	log.Debugf("transaction manager, CutoffPair tx:%s, owner:%s, cutoff:%s, token1:%s, token2:%s", event.TxHash.Hex(), event.Owner.Hex(), event.Cutoff.String(), event.Token1.Hex(), event.Token2.Hex())

	var (
		entity txtyp.TransactionEntity
		list   []txtyp.TransactionView
	)

	entity.FromCutoffPairEvent(event)
	view := txtyp.CutoffPairView(event)
	list = append(list, view)

	return tm.saveTransaction(&entity, list)
}

func (tm *TransactionManager) SaveWethDepositEvent(input eventemitter.EventData) error {
	event := input.(*types.WethDepositEvent)

	log.Debugf("transaction manager, WethDeposit tx:%s, owner:%s, value:%s", event.TxHash.Hex(), event.Dst.Hex(), event.Amount.String())

	var entity txtyp.TransactionEntity
	entity.FromWethDepositEvent(event)
	list := txtyp.WethDepositView(event)

	return tm.saveTransaction(&entity, list)
}

func (tm *TransactionManager) SaveWethWithdrawalEvent(input eventemitter.EventData) error {
	event := input.(*types.WethWithdrawalEvent)

	log.Debugf("transaction manager, WethWithdrawal tx:%s, owner:%s, value:%s", event.TxHash.Hex(), event.Src.Hex(), event.Amount.String())

	var entity txtyp.TransactionEntity
	entity.FromWethWithdrawalEvent(event)
	list := txtyp.WethWithdrawalView(event)

	return tm.saveTransaction(&entity, list)
}

func (tm *TransactionManager) SaveTransferEvent(input eventemitter.EventData) error {
	event := input.(*types.TransferEvent)

	log.Debugf("transaction manager, Transfer tx:%s, sender:%s, receiver:%s", event.TxHash.Hex(), event.Sender.Hex(), event.Receiver.Hex())

	var entity txtyp.TransactionEntity
	entity.FromTransferEvent(event)
	list, err := txtyp.TransferView(event)
	if err != nil {
		return err
	}

	return tm.saveTransaction(&entity, list)
}

// 普通的transaction
// 当value大于0时认为是eth转账
// 当value等于0时认为是调用系统不支持的合约,默认使用fromTransferEvent/send type为unsupported_contract
func (tm *TransactionManager) SaveEthTransferEvent(input eventemitter.EventData) error {
	event := input.(*types.TransferEvent)

	var entity txtyp.TransactionEntity
	entity.FromEthTransferEvent(event)
	list := txtyp.EthTransferView(event)

	return tm.saveTransaction(&entity, list)
}

func (tm *TransactionManager) SaveOrderFilledEvent(input eventemitter.EventData) error {
	// todo

	return nil
}

func (tm *TransactionManager) saveTransaction(tx *txtyp.TransactionEntity, list []txtyp.TransactionView) error {
	// save entity
	if tx.Status == types.TX_STATUS_PENDING {
		tm.savePendingEntity(tx, list)
	} else {
		tm.saveMinedEntity(tx, list)
	}

	// save views
	for _, v := range list {
		if tx.Status == types.TX_STATUS_PENDING {
			tm.savePendingView(&v)
		} else {
			tm.saveMinedView(&v)
		}
	}

	return nil
}

func (tm *TransactionManager) savePendingEntity(tx *txtyp.TransactionEntity, viewList []txtyp.TransactionView) error {
	if !tm.validateEntity(viewList) {
		return nil
	}
	if list, _ := tm.db.FindPendingTxEntityByHash(tx.Hash.Hex()); len(list) > 0 {
		return nil
	}
	return tm.addEntity(tx)
}

func (tm *TransactionManager) saveMinedEntity(tx *txtyp.TransactionEntity, viewList []txtyp.TransactionView) error {
	if !tm.validateEntity(viewList) {
		return nil
	}
	if err := tm.db.DelPendingTxEntityByHash(tx.Hash.Hex()); err != nil {
		log.Errorf(err.Error())
	}
	if _, err := tm.db.FindTxEntityByHashAndLogIndex(tx.Hash.Hex(), tx.LogIndex); err == nil {
		return nil
	}
	return tm.addEntity(tx)
}

func (tm *TransactionManager) savePendingView(tx *txtyp.TransactionView) error {
	if !tm.validateView(tx) {
		return nil
	}
	if list, _ := tm.db.FindPendingTxViewByOwnerAndHash(tx.Symbol, tx.Owner.Hex(), tx.TxHash.Hex()); len(list) > 0 {
		return nil
	}
	return tm.addView(tx)
}

func (tm *TransactionManager) saveMinedView(tx *txtyp.TransactionView) error {
	if !tm.validateView(tx) {
		return nil
	}
	if err := tm.db.DelPendingTxViewByOwnerAndNonce(tx.Owner.Hex(), tx.Nonce.String()); err != nil {
		log.Errorf(err.Error())
	}
	if list, _ := tm.db.FindMinedTxViewByOwnerAndEvent(tx.Symbol, tx.Owner.Hex(), tx.TxHash.Hex(), tx.LogIndex); len(list) > 0 {
		return nil
	}
	return tm.addView(tx)
}

func (tm *TransactionManager) addEntity(tx *txtyp.TransactionEntity) error {
	var item dao.TransactionEntity
	item.ConvertDown(tx)
	return tm.db.Add(&item)
}

func (tm *TransactionManager) addView(tx *txtyp.TransactionView) error {
	var item dao.TransactionView

	item.ConvertDown(tx)
	if err := tm.db.Add(&item); err != nil {
		return err
	}

	// todo(fuk): emit to frontend and use new tx type
	eventemitter.Emit(eventemitter.TransactionEvent, &tx)
	return nil
}

func (tm *TransactionManager) validateEntity(viewList []txtyp.TransactionView) bool {
	owners := txtyp.RelatedOwners(viewList)

	unlocked := false
	for _, v := range owners {
		if ok, _ := tm.accountmanager.HasUnlocked(v.Hex()); ok {
			unlocked = true
			break
		}
	}

	return unlocked
}

func (tm *TransactionManager) validateView(view *txtyp.TransactionView) bool {
	unlocked, _ := tm.accountmanager.HasUnlocked(view.Owner.Hex())
	return unlocked
}
