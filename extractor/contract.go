package extractor

import (
	"errors"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
)

func (l *ExtractorServiceImpl) loadContract() {
	l.contractEvents = make(map[types.Address]map[types.Hash]chainclient.AbiEvent)
	l.contractMethods = make(map[types.Address]map[types.Hash]chainclient.AbiMethod)

	submitRingMethodWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleSubmitRingMethod}
	ringhashSubmitEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleRinghashSubmitEvent}
	orderFilledEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderFilledEvent}
	orderCancelledEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderCancelledEvent}
	cutoffTimestampEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleCutoffTimestampEvent}

	for _, impl := range miner.MinerInstance.Loopring.LoopringImpls {
		submitRingMtd := impl.SubmitRing
		ringhashSubmittedEvt := impl.RingHashRegistry.RinghashSubmittedEvent
		orderFilledEvt := impl.OrderFilledEvent
		orderCancelledEvt := impl.OrderCancelledEvent
		cutoffTimestampEvt := impl.CutoffTimestampChangedEvent

		l.addContractMethod(submitRingMtd)
		l.addContractEvent(ringhashSubmittedEvt)
		l.addContractEvent(orderFilledEvt)
		l.addContractEvent(orderCancelledEvt)
		l.addContractEvent(cutoffTimestampEvt)

		eventemitter.On(submitRingMtd.WatcherTopic(), submitRingMethodWatcher)
		eventemitter.On(ringhashSubmittedEvt.WatcherTopic(), ringhashSubmitEventWatcher)
		eventemitter.On(orderFilledEvt.WatcherTopic(), orderFilledEventWatcher)
		eventemitter.On(orderCancelledEvt.WatcherTopic(), orderCancelledEventWatcher)
		eventemitter.On(cutoffTimestampEvt.WatcherTopic(), cutoffTimestampEventWatcher)
	}
}

func (l *ExtractorServiceImpl) addContractEvent(event chainclient.AbiEvent) {
	id := types.HexToHash(event.Id())
	addr := event.Address()

	log.Infof("addContractEvent address:%s", addr.Hex())
	if _, ok := l.contractEvents[addr]; !ok {
		l.contractEvents[addr] = make(map[types.Hash]chainclient.AbiEvent)
	}

	log.Infof("addContractEvent id:%s", id.Hex())
	l.contractEvents[addr][id] = event
}

func (l *ExtractorServiceImpl) addContractMethod(method chainclient.AbiMethod) {
	id := types.HexToHash(method.MethodId())
	addr := method.Address()

	if _, ok := l.contractMethods[addr]; !ok {
		l.contractMethods[addr] = make(map[types.Hash]chainclient.AbiMethod)
	}

	l.contractMethods[addr][id] = method
}

func (l *ExtractorServiceImpl) getContractEvent(addr types.Address, id types.Hash) (chainclient.AbiEvent, error) {
	var (
		impl  map[types.Hash]chainclient.AbiEvent
		event chainclient.AbiEvent
		ok    bool
	)
	if impl, ok = l.contractEvents[addr]; !ok {
		return nil, errors.New("extractor getContractEvent cann't find contract impl:" + addr.Hex())
	}
	if event, ok = impl[id]; !ok {
		return nil, errors.New("extractor getContractEvent cann't find contract event:" + id.Hex())
	}

	return event, nil
}

func (l *ExtractorServiceImpl) getContractMethod(addr types.Address, id types.Hash) (chainclient.AbiMethod, error) {
	var (
		impl   map[types.Hash]chainclient.AbiMethod
		method chainclient.AbiMethod
		ok     bool
	)

	if impl, ok = l.contractMethods[addr]; !ok {
		return nil, errors.New("eth listener getContractMethod cann't find contract impl")
	}
	if method, ok = impl[id]; !ok {
		return nil, errors.New("eth listener getContractMethod cann't find contract method")
	}

	return method, nil
}
