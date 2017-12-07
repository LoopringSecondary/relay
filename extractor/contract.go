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
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

// 这里无需考虑版本问题，对解析来说，不接受版本升级带来数据结构变化的可能性
func (l *ExtractorServiceImpl) loadContract() {
	l.events = make(map[common.Hash]ContractData)
	l.protocols = make(map[common.Address]string)

	l.loadErc20Contract()
	l.loadProtocolContract()
	l.loadTokenRegisterContract()
	l.loadRingHashRegisteredContract()
	l.loadTokenTransferDelegateProtocol()
}

func (l *ExtractorServiceImpl) loadProtocolAddress() {
	for _, v := range util.AllTokens {
		l.protocols[v.Protocol] = v.Symbol
	}
}

type ContractData struct {
	Event           interface{}
	ContractAddress string // 某个合约具体地址
	TxHash          string // transaction hash
	CAbi            *abi.ABI
	Id              common.Hash
	Name            string
	BlockNumber     *big.Int
	Time            *big.Int
	Topics          []string
}

const (
	RINGMINED_EVT_NAME           = "RingMined"
	CANCEL_EVT_NAME              = "OrderCancelled"
	CUTOFF_EVT_NAME              = "CutoffTimestampChanged"
	TRANSFER_EVT_NAME            = "Transfer"
	APPROVAL_EVT_NAME            = "Approval"
	TOKENREGISTERED_EVT_NAME     = "TokenRegistered"
	TOKENUNREGISTERED_EVT_NAME   = "TokenUnregistered"
	RINGHASHREGISTERED_EVT_NAME  = "RinghashSubmitted"
	ADDRESSAUTHORIZED_EVT_NAME   = "AddressAuthorized"
	ADDRESSDEAUTHORIZED_EVT_NAME = "AddressDeauthorized"
)

func newContractData(event *abi.Event, cabi *abi.ABI) ContractData {
	var c ContractData

	c.Id = event.Id()
	c.Name = event.Name
	c.CAbi = cabi

	return c
}

func (l *ExtractorServiceImpl) loadProtocolContract() {
	for name, event := range l.accessor.ProtocolImplAbi.Events {
		if name != RINGMINED_EVT_NAME && name != CANCEL_EVT_NAME && name != CUTOFF_EVT_NAME {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newContractData(&event, l.accessor.ProtocolImplAbi)

		switch contract.Name {
		case RINGMINED_EVT_NAME:
			contract.Event = &ethaccessor.RingMinedEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleRingMinedEvent}
		case CANCEL_EVT_NAME:
			contract.Event = &ethaccessor.OrderCancelledEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderCancelledEvent}
		case CUTOFF_EVT_NAME:
			contract.Event = &ethaccessor.CutoffTimestampChangedEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleCutoffTimestampEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		l.events[contract.Id] = contract
		log.Debugf("extracotr,contract event name %s, key %s", contract.Name, contract.Id.Hex())
	}
}

func (l *ExtractorServiceImpl) loadErc20Contract() {
	for name, event := range l.accessor.Erc20Abi.Events {
		if name != TRANSFER_EVT_NAME && name != APPROVAL_EVT_NAME {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newContractData(&event, l.accessor.Erc20Abi)

		switch contract.Name {
		case TRANSFER_EVT_NAME:
			contract.Event = &ethaccessor.TransferEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTransferEvent}
		case APPROVAL_EVT_NAME:
			contract.Event = &ethaccessor.ApprovalEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleApprovalEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		l.events[contract.Id] = contract
		log.Debugf("extracotr,contract event name %s -> key:%s", contract.Name, contract.Id.Hex())
	}
}

func (l *ExtractorServiceImpl) loadTokenRegisterContract() {
	for name, event := range l.accessor.TokenRegistryAbi.Events {
		if name != TOKENREGISTERED_EVT_NAME && name != TOKENUNREGISTERED_EVT_NAME {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newContractData(&event, l.accessor.TokenRegistryAbi)

		switch contract.Name {
		case TOKENREGISTERED_EVT_NAME:
			contract.Event = &ethaccessor.TokenRegisteredEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTokenRegisteredEvent}
		case TOKENUNREGISTERED_EVT_NAME:
			contract.Event = &ethaccessor.TokenUnRegisteredEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTokenUnRegisteredEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		l.events[contract.Id] = contract
		log.Debugf("extracotr,contract event name %s -> key:%s", contract.Name, contract.Id.Hex())
	}
}

func (l *ExtractorServiceImpl) loadRingHashRegisteredContract() {
	for name, event := range l.accessor.RinghashRegistryAbi.Events {
		if name != RINGHASHREGISTERED_EVT_NAME {
			continue
		}

		contract := newContractData(&event, l.accessor.RinghashRegistryAbi)
		contract.Event = &ethaccessor.RingHashSubmittedEvent{}

		watcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleRinghashSubmitEvent}
		eventemitter.On(contract.Id.Hex(), watcher)

		l.events[contract.Id] = contract
		log.Debugf("extracotr,contract event name %s -> key:%s", contract.Name, contract.Id.Hex())
	}
}

func (l *ExtractorServiceImpl) loadTokenTransferDelegateProtocol() {
	for name, event := range l.accessor.DelegateAbi.Events {
		if name != ADDRESSAUTHORIZED_EVT_NAME && name != ADDRESSDEAUTHORIZED_EVT_NAME {
			continue
		}

		watcher := &eventemitter.Watcher{}
		contract := newContractData(&event, l.accessor.DelegateAbi)

		switch contract.Name {
		case ADDRESSAUTHORIZED_EVT_NAME:
			contract.Event = &ethaccessor.AddressAuthorizedEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleAddressAuthorizedEvent}
		case ADDRESSDEAUTHORIZED_EVT_NAME:
			contract.Event = &ethaccessor.AddressDeAuthorizedEvent{}
			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleAddressDeAuthorizedEvent}
		}

		eventemitter.On(contract.Id.Hex(), watcher)
		l.events[contract.Id] = contract
		log.Debugf("extracotr,contract event name %s -> key:%s", contract.Name, contract.Id.Hex())
	}
}
