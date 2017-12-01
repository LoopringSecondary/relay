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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"strings"
)

// 这里无需考虑版本问题，对解析来说，不接受版本升级带来数据结构变化的可能性
func (l *ExtractorServiceImpl) loadContract() {
	l.events = make(map[string]ContractData)
	l.loadProtocolContract()
	l.loadTokenRegisterContract()
	l.loadRingHashRegisteredContract()
	l.loadTokenTransferDelegateProtocol()

	// todo: get erc20 token address and former abi
	l.loadErc20Contract([]common.Address{})
}

type ContractData struct {
	Event           interface{}
	ImplAddress     string // lrc合约入口地址
	ContractAddress string // 某个合约具体地址
	TxHash          string // transaction hash
	CAbi            *abi.ABI
	Id              string
	Name            string
	Key             string
	BlockNumber     *types.Big
	Time            *types.Big
	Topics          []string
}

func (c *ContractData) generateSymbol(id, name string) {
	c.Name = name
	c.Id = id
	c.Key = generateKey(c.ImplAddress, c.Id)
}

func generateKey(addr string, id string) string {
	return strings.ToLower(addr) + "-" + strings.ToLower(id)
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
	TEST_EVT_NAME                = "TestEvent"
)

func (l *ExtractorServiceImpl) loadProtocolContract() {
	for _, impl := range l.accessor.ProtocolImpls {
		for name, event := range impl.ProtocolImplAbi.Events {
			if name != RINGMINED_EVT_NAME && name != CANCEL_EVT_NAME && name != CUTOFF_EVT_NAME && name != TEST_EVT_NAME {
				continue
			}

			var (
				contract ContractData
				watcher  *eventemitter.Watcher
			)
			contract.ImplAddress = impl.ContractAddress.Hex()
			contract.CAbi = impl.ProtocolImplAbi
			contract.generateSymbol(event.Id().Hex(), name)

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
			case TEST_EVT_NAME:
				contract.Event = &ethaccessor.TestEvent{}
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTestEvent}
			}

			eventemitter.On(contract.Key, watcher)
			l.events[contract.Key] = contract
			log.Debugf("extracotr,contract event name %s, key %s", contract.Name, contract.Key)
		}
	}
}

func (l *ExtractorServiceImpl) loadErc20Contract(addrs []common.Address) {
	tokenabi := l.accessor.Erc20Abi
	for _, addr := range addrs {
		for name, event := range tokenabi.Events {
			if name != TRANSFER_EVT_NAME && name != APPROVAL_EVT_NAME {
				continue
			}

			var (
				contract ContractData
				watcher  *eventemitter.Watcher
			)
			contract.ImplAddress = addr.Hex()
			contract.CAbi = tokenabi
			contract.generateSymbol(event.Id().Hex(), name)

			switch contract.Name {
			case TRANSFER_EVT_NAME:
				contract.Event = &ethaccessor.TransferEvent{}
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTransferEvent}
			case APPROVAL_EVT_NAME:
				contract.Event = &ethaccessor.ApprovalEvent{}
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleApprovalEvent}
			}

			eventemitter.On(contract.Key, watcher)
			l.events[contract.Key] = contract
			log.Debugf("extracotr,contract event name %s -> key:%s", contract.Name, contract.Key)
		}
	}
}

func (l *ExtractorServiceImpl) loadTokenRegisterContract() {
	for _, impl := range l.accessor.ProtocolImpls {
		for name, event := range impl.TokenRegistryAbi.Events {
			if name != TOKENREGISTERED_EVT_NAME && name != TOKENUNREGISTERED_EVT_NAME {
				continue
			}

			var (
				contract ContractData
				watcher  *eventemitter.Watcher
			)
			contract.ImplAddress = impl.TokenRegistryAddress.Hex()
			contract.CAbi = impl.TokenRegistryAbi
			contract.generateSymbol(event.Id().Hex(), name)

			switch contract.Name {
			case TOKENREGISTERED_EVT_NAME:
				contract.Event = &ethaccessor.TokenRegisteredEvent{}
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTokenRegisteredEvent}
			case TOKENUNREGISTERED_EVT_NAME:
				contract.Event = &ethaccessor.TokenUnRegisteredEvent{}
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTokenUnRegisteredEvent}
			}

			eventemitter.On(contract.Key, watcher)
			l.events[contract.Key] = contract
			log.Debugf("extracotr,contract event name %s -> key:%s", contract.Name, contract.Key)
		}
	}
}

func (l *ExtractorServiceImpl) loadRingHashRegisteredContract() {
	for _, impl := range l.accessor.ProtocolImpls {
		for name, event := range impl.RinghashRegistryAbi.Events {
			if name != RINGHASHREGISTERED_EVT_NAME {
				continue
			}

			var (
				contract ContractData
				watcher  *eventemitter.Watcher
			)
			contract.ImplAddress = impl.RinghashRegistryAddress.Hex()
			contract.CAbi = impl.RinghashRegistryAbi
			contract.generateSymbol(event.Id().Hex(), name)
			contract.Event = &ethaccessor.RingHashSubmittedEvent{}

			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleRinghashSubmitEvent}
			eventemitter.On(contract.Key, watcher)

			l.events[contract.Key] = contract
			log.Debugf("extracotr,contract event name %s -> key:%s", contract.Name, contract.Key)
		}
	}
}

func (l *ExtractorServiceImpl) loadTokenTransferDelegateProtocol() {
	for _, impl := range l.accessor.ProtocolImpls {
		for name, event := range impl.DelegateAbi.Events {
			if name != ADDRESSAUTHORIZED_EVT_NAME && name != ADDRESSDEAUTHORIZED_EVT_NAME {
				continue
			}

			var (
				contract ContractData
				watcher  *eventemitter.Watcher
			)
			contract.ImplAddress = impl.DelegateAddress.Hex()
			contract.CAbi = impl.DelegateAbi
			contract.generateSymbol(event.Id().Hex(), name)

			switch contract.Name {
			case ADDRESSAUTHORIZED_EVT_NAME:
				contract.Event = &ethaccessor.AddressAuthorizedEvent{}
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleAddressAuthorizedEvent}
			case ADDRESSDEAUTHORIZED_EVT_NAME:
				contract.Event = &ethaccessor.AddressDeAuthorizedEvent{}
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleAddressDeAuthorizedEvent}
			}

			eventemitter.On(contract.Key, watcher)
			l.events[contract.Key] = contract
			log.Debugf("extracotr,contract event name %s -> key:%s", contract.Name, contract.Key)
		}
	}
}
