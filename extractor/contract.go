package extractor

import (
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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
	Event       interface{}
	Address     string
	CAbi        *abi.ABI
	Id          string
	Name        string
	Key         string
	BlockNumber *types.Big
	Time        *types.Big
	Topics      []string
}

func (c ContractData) generateSymbol(id, name string) {
	c.Name = name
	c.Id = id
	c.Key = generateKey(c.Address, c.Id)
}

func generateKey(addr string, id string) string {
	return addr + "-" + id
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

func (l *ExtractorServiceImpl) loadProtocolContract() {
	for _, impl := range l.accessor.ProtocolImpls {
		for name, event := range impl.ProtocolImplAbi.Events {
			if name != RINGMINED_EVT_NAME && name != CANCEL_EVT_NAME && name != CUTOFF_EVT_NAME {
				continue
			}

			var (
				contract ContractData
				watcher  *eventemitter.Watcher
			)
			contract.Address = impl.ContractAddress.Hex()
			contract.CAbi = impl.ProtocolImplAbi
			contract.generateSymbol(event.Id().Hex(), name)

			switch contract.Name {
			case RINGMINED_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleRingMinedEvent}
			case CANCEL_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderCancelledEvent}
			case CUTOFF_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleCutoffTimestampEvent}
			}

			eventemitter.On(contract.Key, watcher)
			l.events[contract.Key] = contract
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
			contract.Address = addr.Hex()
			contract.CAbi = tokenabi
			contract.generateSymbol(event.Id().Hex(), name)

			switch contract.Name {
			case TRANSFER_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTransferEvent}
			case APPROVAL_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleApprovalEvent}
			}

			eventemitter.On(contract.Key, watcher)
			l.events[contract.Key] = contract
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
			contract.Address = impl.TokenRegistryAddress.Hex()
			contract.CAbi = impl.TokenRegistryAbi
			contract.generateSymbol(event.Id().Hex(), name)

			switch contract.Name {
			case TOKENREGISTERED_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTokenRegisteredEvent}
			case TOKENUNREGISTERED_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTokenUnRegisteredEvent}
			}

			eventemitter.On(contract.Key, watcher)
			l.events[contract.Key] = contract
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
			contract.Address = impl.RinghashRegistryAddress.Hex()
			contract.CAbi = impl.RinghashRegistryAbi
			contract.generateSymbol(event.Id().Hex(), name)

			watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleRinghashSubmitEvent}
			eventemitter.On(contract.Key, watcher)

			l.events[contract.Key] = contract
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
			contract.Address = impl.DelegateAddress.Hex()
			contract.CAbi = impl.DelegateAbi
			contract.generateSymbol(event.Id().Hex(), name)

			switch contract.Name {
			case ADDRESSAUTHORIZED_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleAddressAuthorizedEvent}
			case ADDRESSDEAUTHORIZED_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleAddressDeAuthorizedEvent}
			}

			eventemitter.On(contract.Key, watcher)
			l.events[contract.Key] = contract
		}
	}
}
