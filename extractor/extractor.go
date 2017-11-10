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
	"errors"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/dao"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"reflect"
	"sync"
)

/**
区块链的listener, 得到order以及ring的事件，
*/

type ExtractorService interface {
	Start()
	Stop()
	Restart()
}

// TODO(fukun):不同的channel，应当交给orderbook统一进行后续处理，可以将channel作为函数返回值、全局变量、参数等方式
type ExtractorServiceImpl struct {
	options         config.ChainClientOptions
	commOpts        config.CommonOptions
	ethClient       *eth.EthClient
	dao             dao.RdsService
	rds             *Rds
	stop            chan struct{}
	lock            sync.RWMutex
	contractMethods map[types.Address]map[types.Hash]chainclient.AbiMethod
	contractEvents  map[types.Address]map[types.Hash]chainclient.AbiEvent
}

func NewExtractorService(options config.ChainClientOptions,
	commonOpts config.CommonOptions,
	ethClient *eth.EthClient,
	rds dao.RdsService,
	database db.Database) *ExtractorServiceImpl {
	var l ExtractorServiceImpl

	l.rds = NewRds(database, commonOpts)
	l.options = options
	l.commOpts = commonOpts
	l.ethClient = ethClient
	l.dao = rds

	l.loadContract()

	//l.StartForkDetect()
	return &l
}

func (l *ExtractorServiceImpl) loadContract() {
	l.contractEvents = make(map[types.Address]map[types.Hash]chainclient.AbiEvent)
	l.contractMethods = make(map[types.Address]map[types.Hash]chainclient.AbiMethod)

	submitRingMethodWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleSubmitRingMethod}
	ringhashSubmitEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleRinghashSubmitEvent}
	orderFilledEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderFilledEvent}
	orderCancelledEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderCancelledEvent}
	//cutoffTimestampEventWatcher := &eventemitter.Watcher{Concurrent:false, Handle: l.handleCutoffTimestampEvent}

	for _, impl := range miner.MinerInstance.Loopring.LoopringImpls {
		submitRingMtd := impl.SubmitRing
		ringhashSubmittedEvt := impl.RingHashRegistry.RinghashSubmittedEvent
		orderFilledEvt := impl.OrderFilledEvent
		orderCancelledEvt := impl.OrderCancelledEvent
		//cutoffTimestampEvt := impl.CutoffTimestampChangedEvent

		l.addContractMethod(submitRingMtd)
		l.addContractEvent(ringhashSubmittedEvt)
		l.addContractEvent(orderFilledEvt)
		l.addContractEvent(orderCancelledEvt)
		//l.addContractEvent(cutoffTimestampEvt)

		eventemitter.On(submitRingMtd.WatcherTopic(), submitRingMethodWatcher)
		eventemitter.On(ringhashSubmittedEvt.WatcherTopic(), ringhashSubmitEventWatcher)
		eventemitter.On(orderFilledEvt.WatcherTopic(), orderFilledEventWatcher)
		eventemitter.On(orderCancelledEvt.WatcherTopic(), orderCancelledEventWatcher)
		//eventemitter.On(cutoffTimestampEvt.WatcherTopic(), cutoffTimestampEventWatcher)
	}
}

func (l *ExtractorServiceImpl) Start() {
	l.stop = make(chan struct{})

	log.Info("eth listener start...")
	start, end := l.getBlockNumberRange()
	iterator := l.ethClient.BlockIterator(start, end, true, uint64(0))

	go func() {
		for {
			inter, err := iterator.Next()
			if err != nil {
				log.Fatalf("eth listener iterator next error:%s", err.Error())
			}

			block := inter.(*eth.BlockWithTxObject)
			log.Debugf("eth listener get block:%s->%s", block.Number.BigInt().String(), block.Hash.Hex())

			txcnt := len(block.Transactions)
			if txcnt < 1 {
				log.Debugf("eth listener get none block transaction")
				continue
			} else {
				log.Infof("eth listener get block transaction list length %d", txcnt)
			}

			if err := l.rds.SaveBlock(*block); err != nil {
				log.Errorf("eth listener save block hash error:%s", err.Error())
				continue
			}

			l.doBlock(*block)
		}
	}()
}

func (l *ExtractorServiceImpl) doBlock(block eth.BlockWithTxObject) {
	txhashs := []types.Hash{}

	for _, tx := range block.Transactions {
		log.Debugf("eth listener get transaction hash:%s", tx.Hash)
		log.Debugf("eth listener get transaction input:%s", tx.Input)

		// 解析method，获得ring内等orders并发送到orderbook保存
		l.doMethod(tx.Input)

		// 获取transaction内所有event logs
		var receipt eth.TransactionReceipt
		err := l.ethClient.GetTransactionReceipt(&receipt, tx.Hash)
		if err != nil {
			log.Errorf("eth listener get transaction receipt error:%s", err.Error())
			continue
		}

		if len(receipt.Logs) == 0 {
			// todo
		}

		log.Debugf("transaction receipt  event logs number:%d", len(receipt.Logs))

		contractAddr := types.HexToAddress(receipt.To)
		txhash := types.HexToHash(tx.Hash)

		for _, evtLog := range receipt.Logs {
			data := hexutil.MustDecode(evtLog.Data)

			// 寻找合约事件
			contractEvt, err := l.getContractEvent(contractAddr, types.HexToHash(evtLog.Topics[0]))
			if err != nil {
				log.Errorf("%s", err.Error())
				continue
			}

			// 解析事件
			dstEvt := reflect.New(reflect.TypeOf(contractEvt))
			if err := contractEvt.Unpack(dstEvt, data, evtLog.Topics); nil != err {
				log.Errorf("err :%s", err.Error())
				continue
			}

			// 处理事件
			event := chainclient.ContractData{
				Event: dstEvt.Elem().Interface().(chainclient.AbiEvent),
			}
			eventemitter.Emit(contractEvt.WatcherTopic(), event)

			// 最后存储
			txhashs = append(txhashs, txhash)
		}
	}

	// 存储block内所有transaction hash
	if err := l.rds.SaveTransactions(block.Hash, txhashs); err != nil {
		log.Errorf("eth listener save transactions error:%s", err.Error())
	}
}

func (l *ExtractorServiceImpl) doMethod(input string) error {
	return nil
}

// 只需要解析submitRing,cancel，cutoff这些方法在event里，如果方法不成功也不用执行后续逻辑
func (l *ExtractorServiceImpl) handleSubmitRingMethod(input eventemitter.EventData) error {
	println("doMethoddoMethoddoMethoddoMethoddoMethoddoMethod")
	//println(input.(string))
	// todo: unpack method
	// input := tx.Input
	// l.ethClient
	return nil
}

func (l *ExtractorServiceImpl) handleOrderFilledEvent(input eventemitter.EventData) error {
	log.Debugf("eth listener log event:orderFilled")

	evt := input.(chainclient.ContractData).Event.(chainclient.OrderFilledEvent)

	if l.commOpts.Develop {
		log.Debugf("eth listener order filled event ringhash -> %s", types.BytesToHash(evt.Ringhash).Hex())
		log.Debugf("eth listener order filled event amountS -> %s", evt.AmountS.String())
		log.Debugf("eth listener order filled event amountB -> %s", evt.AmountB.String())
		log.Debugf("eth listener order filled event orderhash -> %s", types.BytesToHash(evt.OrderHash).Hex())
		log.Debugf("eth listener order filled event blocknumber -> %s", evt.Blocknumber.String())
		log.Debugf("eth listener order filled event time -> %s", evt.Time.String())
		log.Debugf("eth listener order filled event lrcfee -> %s", evt.LrcFee.String())
		log.Debugf("eth listener order filled event lrcreward -> %s", evt.LrcReward.String())
		log.Debugf("eth listener order filled event nextorderhash -> %s", types.BytesToHash(evt.NextOrderHash).Hex())
		log.Debugf("eth listener order filled event preorderhash -> %s", types.BytesToHash(evt.PreOrderHash).Hex())
		log.Debugf("eth listener order filled event ringindex -> %s", evt.RingIndex.String())
	}

	hash := types.BytesToHash(evt.OrderHash)
	model, err := l.dao.GetOrderByHash(hash)
	if err != nil {
		return err
	}

	state := &types.OrderState{}
	if err := model.ConvertUp(state); err != nil {
		return err
	}

	eventemitter.Emit(eventemitter.OrderBookExtractor, state)

	return nil
}

func (l *ExtractorServiceImpl) handleOrderCancelledEvent(input eventemitter.EventData) error {
	log.Debugf("eth listener log event:orderCancelled")

	evt := input.(chainclient.ContractData).Event.(chainclient.OrderCancelledEvent)

	if l.commOpts.Develop {
		log.Debugf("eth listener order cancelled event orderhash -> %s", types.BytesToHash(evt.OrderHash).Hex())
		log.Debugf("eth listener order cancelled event time -> %s", evt.Time.String())
		log.Debugf("eth listener order cancelled event block -> %s", evt.Blocknumber.String())
		log.Debugf("eth listener order cancelled event cancel amount -> %s", evt.AmountCancelled.String())
	}

	hash := types.BytesToHash(evt.OrderHash)
	model, err := l.dao.GetOrderByHash(hash)
	if err != nil {
		return err
	}

	state := &types.OrderState{}
	if err := model.ConvertUp(state); err != nil {
		return err
	}

	eventemitter.Emit(eventemitter.OrderBookExtractor, state)

	return nil
}

func (l *ExtractorServiceImpl) handleCutoffTimestampEvent(input eventemitter.EventData) error {
	return nil
}

func (l *ExtractorServiceImpl) handleRinghashSubmitEvent(input eventemitter.EventData) error {
	return nil
}

func (l *ExtractorServiceImpl) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	close(l.stop)
}

// 重启(分叉)时先关停subscribeEvents，然后关
func (l *ExtractorServiceImpl) Restart() {

}

func (l *ExtractorServiceImpl) Name() string {
	return "eth-listener"
}

func (l *ExtractorServiceImpl) getBlockNumberRange() (*big.Int, *big.Int) {
	start := l.commOpts.DefaultBlockNumber
	end := l.commOpts.EndBlockNumber

	currentBlockNumber, err := l.rds.GetBlockNumber()
	if err != nil {
		return start, end
	}

	if currentBlockNumber.Cmp(start) == 1 {
		start = currentBlockNumber
	}

	log.Debugf("eth started block number :%s", start.String())

	return start, end
}

func (l *ExtractorServiceImpl) judgeContractAddress(addr string) bool {
	for _, v := range l.commOpts.LoopringImpAddresses {
		if addr == v {
			return true
		}
	}
	return false
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
		return nil, errors.New("eth listener getContractEvent cann't find contract impl:" + addr.Hex())
	}
	if event, ok = impl[id]; !ok {
		return nil, errors.New("eth listener getContractEvent cann't find contract event:" + id.Hex())
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
