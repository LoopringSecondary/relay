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

package extractor

import (
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
)

type EventData struct {
	types.TxInfo
	Event  interface{}
	CAbi   *abi.ABI
	Id     common.Hash
	Name   string
	Topics []string
}

func newEventData(event *abi.Event, cabi *abi.ABI) EventData {
	var c EventData

	c.Id = event.Id()
	c.Name = event.Name
	c.CAbi = cabi

	return c
}

func (event *EventData) FullFilled(tx *ethaccessor.Transaction, evtLog *ethaccessor.Log, gasUsed, blockTime *big.Int, methodName string) {
	event.TxInfo = setTxInfo(tx, gasUsed, blockTime, methodName)
	event.Topics = evtLog.Topics
	event.Protocol = common.HexToAddress(evtLog.Address)
	event.TxLogIndex = evtLog.LogIndex.Int64()
	event.Status = types.TX_STATUS_SUCCESS
}

type MethodData struct {
	types.TxInfo
	Method interface{}
	CAbi   *abi.ABI
	Id     string
	Name   string
	Input  string
}

func newMethodData(method *abi.Method, cabi *abi.ABI) MethodData {
	var c MethodData

	c.Id = common.ToHex(method.Id())
	c.Name = method.Name
	c.CAbi = cabi

	return c
}

func (method *MethodData) FullFilled(tx *ethaccessor.Transaction, gasUsed, blockTime *big.Int, status types.TxStatus, methodName string) {
	method.TxInfo = setTxInfo(tx, gasUsed, blockTime, methodName)
	method.Input = tx.Input
	method.TxLogIndex = 0
	method.Status = status
}

func setTxInfo(tx *ethaccessor.Transaction, gasUsed, blockTime *big.Int, methodName string) types.TxInfo {
	var txinfo types.TxInfo

	txinfo.BlockNumber = tx.BlockNumber.BigInt()
	txinfo.BlockTime = blockTime.Int64()
	txinfo.BlockHash = common.HexToHash(tx.BlockHash)
	txinfo.TxHash = common.HexToHash(tx.Hash)
	txinfo.TxIndex = tx.TransactionIndex.Int64()
	txinfo.Protocol = common.HexToAddress(tx.To)
	txinfo.From = common.HexToAddress(tx.From)
	txinfo.To = common.HexToAddress(tx.To)
	txinfo.GasLimit = tx.Gas.BigInt()
	txinfo.GasUsed = gasUsed
	txinfo.GasPrice = tx.GasPrice.BigInt()
	txinfo.Nonce = tx.Nonce.BigInt()
	txinfo.Value = tx.Value.BigInt()

	if impl, ok := ethaccessor.ProtocolAddresses()[txinfo.To]; ok {
		txinfo.DelegateAddress = impl.DelegateAddress
	} else {
		txinfo.DelegateAddress = types.NilAddress
	}

	txinfo.Identify = methodName

	return txinfo
}

type AbiProcessor struct {
	events      map[common.Hash]EventData
	methods     map[string]MethodData
	erc20Events map[common.Hash]bool
	protocols   map[common.Address]string
	delegates   map[common.Address]string
	db          dao.RdsService
	options     *config.ExtractorOptions
}

// 这里无需考虑版本问题，对解析来说，不接受版本升级带来数据结构变化的可能性
func newAbiProcessor(db dao.RdsService, option *config.ExtractorOptions) *AbiProcessor {
	processor := &AbiProcessor{}

	processor.events = make(map[common.Hash]EventData)
	processor.erc20Events = make(map[common.Hash]bool)
	processor.methods = make(map[string]MethodData)
	processor.protocols = make(map[common.Address]string)
	processor.delegates = make(map[common.Address]string)
	processor.db = db

	processor.options = option

	processor.loadProtocolAddress()
	processor.loadErc20Contract()
	processor.loadWethContract()
	processor.loadProtocolContract()
	//processor.loadTokenRegisterContract()
	//processor.loadTokenTransferDelegateProtocol()

	return processor
}

// GetEvent get EventData with id hash
func (processor *AbiProcessor) GetEvent(evtLog ethaccessor.Log) (EventData, bool) {
	var (
		event EventData
		ok    bool
	)

	id := evtLog.EventId()
	if id == types.NilHash {
		return event, false
	}

	event, ok = processor.events[id]
	return event, ok
}

// GetMethod get MethodData with method id
func (processor *AbiProcessor) GetMethod(tx *ethaccessor.Transaction) (MethodData, bool) {
	var (
		method MethodData
		ok     bool
	)

	id := tx.MethodId()
	if id == "" {
		return method, false
	}

	method, ok = processor.methods[id]
	return method, ok
}

// GetMethodName
func (processor *AbiProcessor) GetMethodName(tx *ethaccessor.Transaction) string {
	if method, ok := processor.GetMethod(tx); ok {
		return method.Name
	}
	return ethaccessor.METHOD_UNKNOWN
}

// SupportedContract judge protocol have ever been load
func (processor *AbiProcessor) SupportedContract(protocol common.Address) bool {
	_, ok := processor.protocols[protocol]
	return ok
}

// SupportedEvents supported contract events and unsupported erc20 events
func (processor *AbiProcessor) SupportedEvents(receipt *ethaccessor.TransactionReceipt) bool {
	if receipt == nil || len(receipt.Logs) == 0 {
		return false
	}

	for _, evtlog := range receipt.Logs {
		id := evtlog.EventId()
		if id == types.NilHash {
			continue
		}
		// supported contracts event
		if _, ok := processor.events[id]; ok {
			return true
		}
		// unsupported erc20 contracts event
		if _, ok := processor.erc20Events[id]; ok {
			return true
		}
	}

	return false
}

// SupportedMethod only supported contracts method
func (processor *AbiProcessor) SupportedMethod(tx *ethaccessor.Transaction) bool {
	if !processor.SupportedContract(common.HexToAddress(tx.To)) {
		return false
	}
	id := tx.MethodId()
	if id == "" {
		return false
	}
	_, ok := processor.methods[id]
	return ok
}

// HasSpender check approve spender address have ever been load
func (processor *AbiProcessor) HasSpender(spender common.Address) bool {
	_, ok := processor.delegates[spender]
	return ok
}

func (processor *AbiProcessor) loadProtocolAddress() {
	for _, v := range util.AllTokens {
		processor.protocols[v.Protocol] = v.Symbol
		log.Infof("extractor,contract protocol %s->%s", v.Symbol, v.Protocol.Hex())
	}

	for _, v := range ethaccessor.ProtocolAddresses() {
		protocolSymbol := "loopring"
		delegateSymbol := "transfer_delegate"
		tokenRegisterSymbol := "token_register"

		processor.protocols[v.ContractAddress] = protocolSymbol
		processor.protocols[v.TokenRegistryAddress] = tokenRegisterSymbol
		processor.protocols[v.DelegateAddress] = delegateSymbol

		log.Infof("extractor,contract protocol %s->%s", protocolSymbol, v.ContractAddress.Hex())
		log.Infof("extractor,contract protocol %s->%s", tokenRegisterSymbol, v.TokenRegistryAddress.Hex())
		log.Infof("extractor,contract protocol %s->%s", delegateSymbol, v.DelegateAddress.Hex())
	}
}

func (processor *AbiProcessor) loadProtocolContract() {
	for name, event := range ethaccessor.ProtocolImplAbi().Events {
		if name != ethaccessor.EVENT_RING_MINED && name != ethaccessor.EVENT_ORDER_CANCELLED && name != ethaccessor.EVENT_CUTOFF_ALL && name != ethaccessor.EVENT_CUTOFF_PAIR {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newEventData(&event, ethaccessor.ProtocolImplAbi())

		switch contract.Name {
		case ethaccessor.EVENT_RING_MINED:
			contract.Event = &ethaccessor.RingMinedEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleRingMinedEvent}
		case ethaccessor.EVENT_ORDER_CANCELLED:
			contract.Event = &ethaccessor.OrderCancelledEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleOrderCancelledEvent}
		case ethaccessor.EVENT_CUTOFF_ALL:
			contract.Event = &ethaccessor.CutoffEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleCutoffEvent}
		case ethaccessor.EVENT_CUTOFF_PAIR:
			contract.Event = &ethaccessor.CutoffPairEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleCutoffPairEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		processor.events[contract.Id] = contract
		log.Infof("extractor,contract event name:%s -> key:%s", contract.Name, contract.Id.Hex())
	}

	for name, method := range ethaccessor.ProtocolImplAbi().Methods {
		if name != ethaccessor.METHOD_SUBMIT_RING && name != ethaccessor.METHOD_CANCEL_ORDER && name != ethaccessor.METHOD_CUTOFF_ALL && name != ethaccessor.METHOD_CUTOFF_PAIR {
			continue
		}

		contract := newMethodData(&method, ethaccessor.ProtocolImplAbi())
		watcher := &eventemitter.Watcher{}

		switch contract.Name {
		case ethaccessor.METHOD_SUBMIT_RING:
			contract.Method = &ethaccessor.SubmitRingMethodInputs{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleSubmitRingMethod}
		case ethaccessor.METHOD_CANCEL_ORDER:
			contract.Method = &ethaccessor.CancelOrderMethod{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleCancelOrderMethod}
		case ethaccessor.METHOD_CUTOFF_ALL:
			contract.Method = &ethaccessor.CutoffMethod{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleCutoffMethod}
		case ethaccessor.METHOD_CUTOFF_PAIR:
			contract.Method = &ethaccessor.CutoffPairMethod{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleCutoffPairMethod}
		}

		eventemitter.On(contract.Id, watcher)
		processor.methods[contract.Id] = contract
		log.Infof("extractor,contract method name:%s -> key:%s", contract.Name, contract.Id)
	}
}

func (processor *AbiProcessor) loadErc20Contract() {
	for name, event := range ethaccessor.Erc20Abi().Events {
		if name != ethaccessor.EVENT_TRANSFER && name != ethaccessor.EVENT_APPROVAL {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newEventData(&event, ethaccessor.Erc20Abi())

		switch contract.Name {
		case ethaccessor.EVENT_TRANSFER:
			contract.Event = &ethaccessor.TransferEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleTransferEvent}
		case ethaccessor.EVENT_APPROVAL:
			contract.Event = &ethaccessor.ApprovalEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleApprovalEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		processor.events[contract.Id] = contract
		processor.erc20Events[contract.Id] = true
		log.Infof("extractor,contract event name:%s -> key:%s", contract.Name, contract.Id.Hex())
	}

	for name, method := range ethaccessor.Erc20Abi().Methods {
		if name != ethaccessor.METHOD_TRANSFER && name != ethaccessor.METHOD_APPROVE {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newMethodData(&method, ethaccessor.Erc20Abi())

		switch contract.Name {
		case ethaccessor.METHOD_TRANSFER:
			contract.Method = &ethaccessor.TransferMethod{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleTransferMethod}
		case ethaccessor.METHOD_APPROVE:
			contract.Method = &ethaccessor.ApproveMethod{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleApproveMethod}
		}

		eventemitter.On(contract.Id, watcher)
		processor.methods[contract.Id] = contract
		log.Infof("extractor,contract method name:%s -> key:%s", contract.Name, contract.Id)
	}
}

func (processor *AbiProcessor) loadWethContract() {
	for name, method := range ethaccessor.WethAbi().Methods {
		if name != ethaccessor.METHOD_WETH_DEPOSIT && name != ethaccessor.METHOD_WETH_WITHDRAWAL {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newMethodData(&method, ethaccessor.WethAbi())

		switch contract.Name {
		case ethaccessor.METHOD_WETH_DEPOSIT:
			// weth deposit without any inputs,use transaction.value as input
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleWethDepositMethod}
		case ethaccessor.METHOD_WETH_WITHDRAWAL:
			contract.Method = &ethaccessor.WethWithdrawalMethod{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleWethWithdrawalMethod}
		}

		eventemitter.On(contract.Id, watcher)
		processor.methods[contract.Id] = contract
		log.Infof("extractor,contract method name:%s -> key:%s", contract.Name, contract.Id)
	}

	for name, event := range ethaccessor.WethAbi().Events {
		if name != ethaccessor.EVENT_WETH_DEPOSIT && name != ethaccessor.EVENT_WETH_WITHDRAWAL {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newEventData(&event, ethaccessor.WethAbi())

		switch contract.Name {
		case ethaccessor.EVENT_WETH_DEPOSIT:
			contract.Event = &ethaccessor.WethDepositEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleWethDepositEvent}
		case ethaccessor.EVENT_WETH_WITHDRAWAL:
			contract.Event = &ethaccessor.WethWithdrawalEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleWethWithdrawalEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		processor.events[contract.Id] = contract
		log.Infof("extractor,contract event name:%s -> key:%s", contract.Name, contract.Id.Hex())
	}
}

func (processor *AbiProcessor) loadTokenRegisterContract() {
	for name, event := range ethaccessor.TokenRegistryAbi().Events {
		if name != ethaccessor.EVENT_TOKEN_REGISTERED && name != ethaccessor.EVENT_TOKEN_UNREGISTERED {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newEventData(&event, ethaccessor.TokenRegistryAbi())

		switch contract.Name {
		case ethaccessor.EVENT_TOKEN_REGISTERED:
			contract.Event = &ethaccessor.TokenRegisteredEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleTokenRegisteredEvent}
		case ethaccessor.EVENT_TOKEN_UNREGISTERED:
			contract.Event = &ethaccessor.TokenUnRegisteredEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleTokenUnRegisteredEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		processor.events[contract.Id] = contract
		log.Infof("extractor,contract event name:%s -> key:%s", contract.Name, contract.Id.Hex())
	}
}

func (processor *AbiProcessor) loadTokenTransferDelegateProtocol() {
	for name, event := range ethaccessor.DelegateAbi().Events {
		if name != ethaccessor.EVENT_ADDRESS_AUTHORIZED && name != ethaccessor.EVENT_ADDRESS_DEAUTHORIZED {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newEventData(&event, ethaccessor.DelegateAbi())

		switch contract.Name {
		case ethaccessor.EVENT_ADDRESS_AUTHORIZED:
			contract.Event = &ethaccessor.AddressAuthorizedEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleAddressAuthorizedEvent}
		case ethaccessor.EVENT_ADDRESS_DEAUTHORIZED:
			contract.Event = &ethaccessor.AddressDeAuthorizedEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: processor.handleAddressDeAuthorizedEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		processor.events[contract.Id] = contract
		log.Infof("extractor,contract event name:%s -> key:%s", contract.Name, contract.Id.Hex())
	}
}

// 只需要解析submitRing,cancel，cutoff这些方法在event里，如果方法不成功也不用执行后续逻辑
func (processor *AbiProcessor) handleSubmitRingMethod(input eventemitter.EventData) error {
	contract := input.(MethodData)

	// unpack submit ring method
	ring := contract.Method.(*ethaccessor.SubmitRingMethodInputs)
	ring.Protocol = contract.To
	data := hexutil.MustDecode("0x" + contract.Input[10:])
	if err := contract.CAbi.UnpackMethodInput(ring, contract.Name, data); err != nil {
		log.Errorf("extractor,tx:%s submitRing method, unpack error:%s", contract.TxHash.Hex(), err.Error())
		return nil
	}

	// convert data struct
	event, err := ring.ConvertDown()
	if err != nil {
		log.Errorf("extractor,tx:%s submitRing method convert order data error:%s", contract.TxHash.Hex(), err.Error())
		return nil
	}

	// set txinfo for event
	event.TxInfo = contract.TxInfo
	if event.Status == types.TX_STATUS_FAILED {
		event.Err = fmt.Errorf("method %s transaction failed", contract.Name)
	}

	// 不需要发送订单到gateway
	//for _, v := range event.OrderList {
	//	v.Hash = v.GenerateHash()
	//	log.Debugf("extractor,tx:%s submitRing method orderHash:%s,owner:%s,tokenS:%s,tokenB:%s,amountS:%s,amountB:%s", event.TxHash.Hex(), v.Hash.Hex(), v.Owner.Hex(), v.TokenS.Hex(), v.TokenB.Hex(), v.AmountS.String(), v.AmountB.String())
	//	eventemitter.Emit(eventemitter.GatewayNewOrder, v)
	//}

	log.Debugf("extractor,tx:%s submitRing method gas:%s, gasprice:%s, status:%s", event.TxHash.Hex(), event.GasUsed.String(), event.GasPrice.String(), types.StatusStr(event.Status))

	eventemitter.Emit(eventemitter.Miner_SubmitRing_Method, event)

	return nil
}

func (processor *AbiProcessor) handleCancelOrderMethod(input eventemitter.EventData) error {
	contract := input.(MethodData)
	contractEvent := contract.Method.(*ethaccessor.CancelOrderMethod)

	if contract.DelegateAddress == types.NilAddress {
		log.Errorf("extractor,tx:%s cancelOrder method cann't get delegate address", contract.TxHash.Hex())
		return nil
	}

	data := hexutil.MustDecode("0x" + contract.Input[10:])
	if err := contract.CAbi.UnpackMethodInput(contractEvent, contract.Name, data); err != nil {
		log.Errorf("extractor,tx:%s cancelOrder method unpack error:%s", contract.TxHash.Hex(), err.Error())
		return nil
	}

	order, cancelAmount, _ := contractEvent.ConvertDown()
	order.Protocol = contract.Protocol
	order.DelegateAddress = contract.DelegateAddress
	order.Hash = order.GenerateHash()
	log.Debugf("extractor,tx:%s cancelOrder method order tokenS:%s,tokenB:%s,amountS:%s,amountB:%s", contract.TxHash.Hex(), order.TokenS.Hex(), order.TokenB.Hex(), order.AmountS.String(), order.AmountB.String())

	// 发送到txmanager
	tmCancelEvent := &types.OrderCancelledEvent{}
	tmCancelEvent.TxInfo = contract.TxInfo
	tmCancelEvent.OrderHash = order.Hash
	tmCancelEvent.AmountCancelled = cancelAmount
	eventemitter.Emit(eventemitter.CancelOrder, tmCancelEvent)

	return nil
}

func (processor *AbiProcessor) handleCutoffMethod(input eventemitter.EventData) error {
	contract := input.(MethodData)
	contractMethod := contract.Method.(*ethaccessor.CutoffMethod)

	data := hexutil.MustDecode("0x" + contract.Input[10:])
	if err := contract.CAbi.UnpackMethodInput(&contractMethod.Cutoff, contract.Name, data); err != nil {
		log.Errorf("extractor,tx:%s cutoff method unpack error:%s", contract.TxHash.Hex(), err.Error())
		return nil
	}

	cutoff := contractMethod.ConvertDown()
	cutoff.TxInfo = contract.TxInfo
	cutoff.Owner = cutoff.From
	log.Debugf("extractor,tx:%s cutoff method owner:%s, cutoff:%d, status:%d", contract.TxHash.Hex(), cutoff.Owner.Hex(), cutoff.Cutoff.Int64(), cutoff.Status)

	eventemitter.Emit(eventemitter.CutoffAll, cutoff)

	return nil
}

func (processor *AbiProcessor) handleCutoffPairMethod(input eventemitter.EventData) error {
	contract := input.(MethodData)
	contractMethod := contract.Method.(*ethaccessor.CutoffPairMethod)

	data := hexutil.MustDecode("0x" + contract.Input[10:])
	if err := contract.CAbi.UnpackMethodInput(contractMethod, contract.Name, data); err != nil {
		log.Errorf("extractor,tx:%s cutoffpair method unpack error:%s", contract.TxHash.Hex(), err.Error())
		return nil
	}

	cutoffpair := contractMethod.ConvertDown()
	cutoffpair.TxInfo = contract.TxInfo
	cutoffpair.Owner = cutoffpair.From

	log.Debugf("extractor,tx:%s cutoffpair method owenr:%s, token1:%s, token2:%s, cutoff:%d", contract.TxHash.Hex(), cutoffpair.Owner.Hex(), cutoffpair.Token1.Hex(), cutoffpair.Token2.Hex(), cutoffpair.Cutoff.Int64())

	eventemitter.Emit(eventemitter.CutoffPair, cutoffpair)

	return nil
}

func (processor *AbiProcessor) handleApproveMethod(input eventemitter.EventData) error {
	contractData := input.(MethodData)
	contractMethod := contractData.Method.(*ethaccessor.ApproveMethod)

	data := hexutil.MustDecode("0x" + contractData.Input[10:])
	if err := contractData.CAbi.UnpackMethodInput(contractMethod, contractData.Name, data); err != nil {
		log.Errorf("extractor,tx:%s approve method unpack error:%s", contractData.TxHash.Hex(), err.Error())
		return nil
	}

	approve := contractMethod.ConvertDown()
	approve.Owner = contractData.From
	approve.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s approve method owner:%s, spender:%s, value:%s", contractData.TxHash.Hex(), approve.Owner.Hex(), approve.Spender.Hex(), approve.Amount.String())

	eventemitter.Emit(eventemitter.Approve, approve)

	return nil
}

func (processor *AbiProcessor) handleTransferMethod(input eventemitter.EventData) error {
	contractData := input.(MethodData)
	contractMethod := contractData.Method.(*ethaccessor.TransferMethod)

	data := hexutil.MustDecode("0x" + contractData.Input[10:])
	if err := contractData.CAbi.UnpackMethodInput(contractMethod, contractData.Name, data); err != nil {
		log.Errorf("extractor,tx:%s transfer method unpack error:%s", contractData.TxHash.Hex(), err.Error())
		return nil
	}

	transfer := contractMethod.ConvertDown()
	transfer.Sender = contractData.From
	transfer.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s transfer method sender:%s, receiver:%s, value:%s", transfer.TxHash.Hex(), transfer.Sender.Hex(), transfer.Receiver.Hex(), transfer.Amount.String())

	eventemitter.Emit(eventemitter.Transfer, transfer)
	return nil
}

func (processor *AbiProcessor) handleWethDepositMethod(input eventemitter.EventData) error {
	contractData := input.(MethodData)

	var deposit types.WethDepositEvent
	deposit.Dst = contractData.From
	deposit.Amount = contractData.Value
	deposit.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s wethDeposit method from:%s, to:%s, value:%s", contractData.TxHash.Hex(), deposit.From.Hex(), deposit.To.Hex(), deposit.Amount.String())

	eventemitter.Emit(eventemitter.WethDeposit, &deposit)

	return nil
}

func (processor *AbiProcessor) handleWethWithdrawalMethod(input eventemitter.EventData) error {
	contractData := input.(MethodData)
	contractMethod := contractData.Method.(*ethaccessor.WethWithdrawalMethod)

	data := hexutil.MustDecode("0x" + contractData.Input[10:])
	if err := contractData.CAbi.UnpackMethodInput(&contractMethod.Value, contractData.Name, data); err != nil {
		log.Errorf("extractor,tx:%s wethWithdrawal method unpack error:%s", contractData.TxHash.Hex(), err.Error())
		return nil
	}

	withdrawal := contractMethod.ConvertDown()
	withdrawal.Src = contractData.From
	withdrawal.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s wethWithdrawal method from:%s, to:%s, value:%s", contractData.TxHash.Hex(), withdrawal.From.Hex(), withdrawal.To.Hex(), withdrawal.Amount.String())

	eventemitter.Emit(eventemitter.WethWithdrawal, withdrawal)

	return nil
}

func (processor *AbiProcessor) handleRingMinedEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 2 {
		log.Errorf("extractor,tx:%s ringMined event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	// emit to miner
	//var evt types.SubmitRingMethodEvent
	//evt.TxInfo = contractData.TxInfo
	//evt.Err = nil
	//eventemitter.Emit(eventemitter.Miner_SubmitRing_Method, &evt)

	// process ringmined to fills
	contractEvent := contractData.Event.(*ethaccessor.RingMinedEvent)
	contractEvent.RingHash = common.HexToHash(contractData.Topics[1])

	ringmined, fills, err := contractEvent.ConvertDown()
	if err != nil {
		log.Errorf("extractor,tx:%s ringMined event convert down error:%s", contractData.TxHash.Hex(), err.Error())
		return nil
	}
	ringmined.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s ringMined event logIndex:%d delegate:%s, ringhash:%s, ringIndex:%s",
		contractData.TxHash.Hex(),
		ringmined.TxLogIndex,
		ringmined.DelegateAddress.Hex(),
		ringmined.Ringhash.Hex(),
		ringmined.RingIndex.String())

	eventemitter.Emit(eventemitter.RingMined, ringmined)

	var (
		fillList      []*types.OrderFilledEvent
		orderhashList []string
	)
	for _, fill := range fills {
		fill.TxInfo = contractData.TxInfo

		log.Debugf("extractor,tx:%s orderFilled event methodName:%s, delegate:%s, ringhash:%s, amountS:%s, amountB:%s, "+
			"orderhash:%s, nextOrderhash:%s, preOrderhash:%s, ringIndex:%s, splitS:%s, splitB:%s, lrcFee:%s, lrcReward:%s",
			contractData.TxHash.Hex(),
			fill.Identify,
			fill.DelegateAddress.Hex(),
			fill.Ringhash.Hex(),
			fill.AmountS.String(),
			fill.AmountB.String(),
			fill.OrderHash.Hex(),
			fill.NextOrderHash.Hex(),
			fill.PreOrderHash.Hex(),
			fill.RingIndex.String(),
			fill.SplitS.String(),
			fill.SplitB.String(),
			fill.LrcFee.String(),
			fill.LrcReward.String(),
		)

		fillList = append(fillList, fill)
		orderhashList = append(orderhashList, fill.OrderHash.Hex())
	}

	ordermap, err := processor.db.GetOrdersByHash(orderhashList)
	if err != nil {
		log.Errorf("extractor,tx:%s ringMined event getOrdersByHash error:%s", contractData.TxHash.Hex(), err.Error())
		return nil
	}

	length := len(fillList)
	for i := 0; i < length; i++ {
		fill := fillList[i]

		ord, ok := ordermap[fill.OrderHash.Hex()]
		if !ok {
			continue
		}

		fill.TokenS = common.HexToAddress(ord.TokenS)
		fill.TokenB = common.HexToAddress(ord.TokenB)
		fill.Owner = common.HexToAddress(ord.Owner)
		fill.Market, _ = util.WrapMarketByAddress(fill.TokenB.Hex(), fill.TokenS.Hex())

		if i == length-1 {
			fill.SellTo = fillList[0].Owner
		} else {
			fill.SellTo = fillList[i+1].Owner
		}
		if i == 0 {
			fill.BuyFrom = fillList[length-1].Owner
		} else {
			fill.BuyFrom = fillList[i-1].Owner
		}

		log.Debugf("extractor,tx:%s orderFilled event match fillIndex:%d and order:%s", contractData.TxHash.Hex(), fill.FillIndex.Int64(), ord.OrderHash)

		eventemitter.Emit(eventemitter.OrderFilled, fill)
	}
	return nil
}

func (processor *AbiProcessor) handleOrderCancelledEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 2 {
		log.Errorf("extractor,tx:%s orderCancelled event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.OrderCancelledEvent)
	contractEvent.OrderHash = common.HexToHash(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s orderCancelled event delegate:%s, orderhash:%s, cancelAmount:%s", contractData.TxHash.Hex(), evt.DelegateAddress.Hex(), evt.OrderHash.Hex(), evt.AmountCancelled.String())

	eventemitter.Emit(eventemitter.CancelOrder, evt)

	return nil
}

func (processor *AbiProcessor) handleCutoffEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 2 {
		log.Errorf("extractor,tx:%s cutoffTimestampChanged event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.CutoffEvent)
	contractEvent.Owner = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s cutoffTimestampChanged event delegate:%s, ownerAddress:%s, cutOffTime:%s, status:%d", contractData.TxHash.Hex(), evt.DelegateAddress.Hex(), evt.Owner.Hex(), evt.Cutoff.String(), evt.Status)

	eventemitter.Emit(eventemitter.CutoffAll, evt)

	return nil
}

func (processor *AbiProcessor) handleCutoffPairEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 2 {
		log.Errorf("extractor,tx:%s cutoffPair event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.CutoffPairEvent)
	contractEvent.Owner = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s cutoffPair event delegate:%s, ownerAddress:%s, token1:%s, token2:%s, cutOffTime:%s", contractData.TxHash.Hex(), evt.DelegateAddress.Hex(), evt.Owner.Hex(), evt.Token1.Hex(), evt.Token2.Hex(), evt.Cutoff.String())

	eventemitter.Emit(eventemitter.CutoffPair, evt)

	return nil
}

func (processor *AbiProcessor) handleTransferEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)

	if len(contractData.Topics) < 3 {
		log.Errorf("extractor,tx:%s tokenTransfer event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.TransferEvent)
	contractEvent.Sender = common.HexToAddress(contractData.Topics[1])
	contractEvent.Receiver = common.HexToAddress(contractData.Topics[2])

	transfer := contractEvent.ConvertDown()
	transfer.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s tokenTransfer event, methodName:%s, logIndex:%d, from:%s, to:%s, value:%s", contractData.TxHash.Hex(), transfer.Identify, transfer.TxLogIndex, transfer.Sender.Hex(), transfer.Receiver.Hex(), transfer.Amount.String())

	eventemitter.Emit(eventemitter.Transfer, transfer)

	return nil
}

func (processor *AbiProcessor) handleApprovalEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 3 {
		log.Errorf("extractor,tx:%s approval event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.ApprovalEvent)
	contractEvent.Owner = common.HexToAddress(contractData.Topics[1])
	contractEvent.Spender = common.HexToAddress(contractData.Topics[2])

	approve := contractEvent.ConvertDown()
	approve.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s approval event owner:%s, spender:%s, value:%s", contractData.TxHash.Hex(), approve.Owner.Hex(), approve.Spender.Hex(), approve.Amount.String())

	eventemitter.Emit(eventemitter.Approve, approve)

	return nil
}

func (processor *AbiProcessor) handleTokenRegisteredEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	contractEvent := contractData.Event.(*ethaccessor.TokenRegisteredEvent)

	evt := contractEvent.ConvertDown()
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s tokenRegistered event address:%s, symbol:%s", contractData.TxHash.Hex(), evt.Token.Hex(), evt.Symbol)

	eventemitter.Emit(eventemitter.TokenRegistered, evt)

	return nil
}

func (processor *AbiProcessor) handleTokenUnRegisteredEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	contractEvent := contractData.Event.(*ethaccessor.TokenUnRegisteredEvent)

	evt := contractEvent.ConvertDown()
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s tokenUnregistered event address:%s, symbol:%s", contractData.TxHash.Hex(), evt.Token.Hex(), evt.Symbol)

	eventemitter.Emit(eventemitter.TokenUnRegistered, evt)

	return nil
}

func (processor *AbiProcessor) handleAddressAuthorizedEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 2 {
		log.Errorf("extractor,tx:%s addressAuthorized event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.AddressAuthorizedEvent)
	contractEvent.ContractAddress = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s addressAuthorized event address:%s, number:%d", contractData.TxHash.Hex(), evt.Protocol.Hex(), evt.Number)

	eventemitter.Emit(eventemitter.AddressAuthorized, evt)

	return nil
}

func (processor *AbiProcessor) handleAddressDeAuthorizedEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 2 {
		log.Errorf("extractor,tx:%s addressDeAuthorized event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.AddressDeAuthorizedEvent)
	contractEvent.ContractAddress = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s addressDeAuthorized event address:%s, number:%d", contractData.TxHash.Hex(), evt.Protocol.Hex(), evt.Number)

	eventemitter.Emit(eventemitter.AddressAuthorized, evt)

	return nil
}

func (processor *AbiProcessor) handleWethDepositEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 2 {
		log.Errorf("extractor,tx:%s wethDeposit event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.WethDepositEvent)
	evt := contractEvent.ConvertDown()
	evt.Dst = common.HexToAddress(contractData.Topics[1])
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s wethDeposit event deposit to:%s, number:%s", contractData.TxHash.Hex(), evt.Dst.Hex(), evt.Amount.String())

	eventemitter.Emit(eventemitter.WethDeposit, evt)

	return nil
}

func (processor *AbiProcessor) handleWethWithdrawalEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
	if len(contractData.Topics) < 2 {
		log.Errorf("extractor,tx:%s wethWithdrawal event indexed fields number error", contractData.TxHash.Hex())
		return nil
	}

	contractEvent := contractData.Event.(*ethaccessor.WethWithdrawalEvent)

	evt := contractEvent.ConvertDown()
	evt.Src = common.HexToAddress(contractData.Topics[1])
	evt.TxInfo = contractData.TxInfo

	log.Debugf("extractor,tx:%s wethWithdrawal event withdrawal to:%s, number:%s", contractData.TxHash.Hex(), evt.Src.Hex(), evt.Amount.String())

	eventemitter.Emit(eventemitter.WethWithdrawal, evt)

	return nil
}

func (processor *AbiProcessor) handleEthTransfer(tx *ethaccessor.Transaction, receipt *ethaccessor.TransactionReceipt, time *big.Int) error {
	var dst types.TransferEvent

	dst.From = common.HexToAddress(tx.From)
	dst.To = common.HexToAddress(tx.To)
	dst.TxHash = common.HexToHash(tx.Hash)
	dst.Amount = tx.Value.BigInt()
	dst.Value = tx.Value.BigInt()
	dst.TxLogIndex = 0
	dst.BlockNumber = tx.BlockNumber.BigInt()
	dst.BlockTime = time.Int64()

	dst.GasLimit = tx.Gas.BigInt()
	dst.GasPrice = tx.GasPrice.BigInt()
	dst.Nonce = tx.Nonce.BigInt()

	dst.Sender = common.HexToAddress(tx.From)
	dst.Receiver = common.HexToAddress(tx.To)
	dst.GasUsed, dst.Status = processor.getGasAndStatus(tx, receipt)

	log.Debugf("extractor,tx:%s handleEthTransfer from:%s, to:%s, value:%s, gasUsed:%s, status:%d", tx.Hash, tx.From, tx.To, tx.Value.BigInt().String(), dst.GasUsed.String(), dst.Status)

	eventemitter.Emit(eventemitter.EthTransferEvent, &dst)

	return nil
}

func (processor *AbiProcessor) getGasAndStatus(tx *ethaccessor.Transaction, receipt *ethaccessor.TransactionReceipt) (*big.Int, types.TxStatus) {
	var (
		gasUsed *big.Int
		status  types.TxStatus
	)
	if receipt == nil {
		gasUsed = big.NewInt(0)
		status = types.TX_STATUS_PENDING
	} else if receipt.Failed(tx) {
		gasUsed = receipt.GasUsed.BigInt()
		status = types.TX_STATUS_FAILED
	} else {
		gasUsed = receipt.GasUsed.BigInt()
		status = types.TX_STATUS_SUCCESS
	}

	return gasUsed, status
}
