package extractor

import (
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// 这里无需考虑版本问题，对解析来说，不接受版本升级带来数据结构变化的可能性
func (l *ExtractorServiceImpl) loadContract() {
	l.events = make(map[string]ContractData)
	l.loadProtocolContract()

	// todo: get erc20 token address and former abi
	l.loadErc20Contract([]types.Address{})

}

type ContractData struct {
	Event       interface{}
	CAbi        *abi.ABI
	Id          string
	Name        string
	WatchName   string
	BlockNumber *types.Big
	Time        *types.Big
}

const (
	RINGMINED_EVT_NAME = "RingMined"
	CANCEL_EVT_NAME    = "OrderCancelled"
	CUTOFF_EVT_NAME    = "CutoffTimestampChanged"
	TRANSFER_EVT_NAME  = "Transfer"
	APPROVAL_EVT_NAME  = "Approval"
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
			contract.CAbi = impl.ProtocolImplAbi
			contract.Name = name
			contract.Id = event.Id().Hex()
			contract.WatchName = impl.ContractAddress.Hex() + "-" + contract.Name

			switch contract.Name {
			case RINGMINED_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleRingMinedEvent}
			case CANCEL_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderCancelledEvent}
			case CUTOFF_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleCutoffTimestampEvent}
			}

			eventemitter.On(contract.WatchName, watcher)
			l.events[contract.Id] = contract
		}
	}
}

func (l *ExtractorServiceImpl) loadErc20Contract(addrs []types.Address) {
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
			contract.CAbi = tokenabi
			contract.Name = name
			contract.Id = event.Id().Hex()
			contract.WatchName = addr.Hex() + "-" + contract.Name

			switch contract.Name {
			case TRANSFER_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleTransferEvent}
			case APPROVAL_EVT_NAME:
				watcher = &eventemitter.Watcher{Concurrent: false, Handle: l.handleApprovalEvent}
			}

			eventemitter.On(contract.WatchName, watcher)
			l.events[contract.Id] = contract
		}
	}
}
