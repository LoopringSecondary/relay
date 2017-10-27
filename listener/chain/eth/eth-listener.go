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

package eth

import (
	"errors"
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/orderbook"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"reflect"
	"sync"
)

/**
区块链的listener, 得到order以及ring的事件，
*/

type Whisper struct {
	ChainOrderChan chan *types.OrderState
}

// TODO(fukun):不同的channel，应当交给orderbook统一进行后续处理，可以将channel作为函数返回值、全局变量、参数等方式
type EthClientListener struct {
	options         config.ChainClientOptions
	commOpts        config.CommonOptions
	ethClient       *eth.EthClient
	ob              *orderbook.OrderBook
	whisper         *Whisper
	rds             *Rds
	stop            chan struct{}
	lock            sync.RWMutex
	contractMethods map[types.Address]map[types.Hash]chainclient.AbiMethod
	contractEvents  map[types.Address]map[types.Hash]chainclient.AbiEvent
}

func NewListener(options config.ChainClientOptions,
	commonOpts config.CommonOptions,
	whisper *Whisper,
	ethClient *eth.EthClient,
	ob *orderbook.OrderBook,
	database db.Database) *EthClientListener {
	var l EthClientListener

	l.rds = NewRds(database, commonOpts)
	l.options = options
	l.commOpts = commonOpts
	l.whisper = whisper
	l.ethClient = ethClient
	l.ob = ob

	l.loadContract()
	return &l
}

func (l *EthClientListener) loadContract() {
	l.contractEvents = make(map[types.Address]map[types.Hash]chainclient.AbiEvent)
	l.contractMethods = make(map[types.Address]map[types.Hash]chainclient.AbiMethod)

	submitRingMethodWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleSubmitRingMethod}
	ringhashSubmitEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleRinghashSubmitEvent}
	orderFilledEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderFilledEvent}
	orderCancelledEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderCancelledEvent}
	//cutoffTimestampEventWatcher := &eventemitter.Watcher{Concurrent:false, Handle: l.handleCutoffTimestampEvent}

	for _, impl := range miner.LoopringInstance.LoopringImpls {
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

func (l *EthClientListener) Start() {
	l.stop = make(chan struct{})

	log.Info("eth listener start...")
	start, end := l.getBlockNumberRange()
	iterator := l.ethClient.BlockIterator(start, end, true)

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

func (l *EthClientListener) doBlock(block eth.BlockWithTxObject) {
	txhashs := []types.Hash{}

	for _, tx := range block.Transactions {
		// 判断合约地址是否合法
		if !l.judgeContractAddress(tx.To) {
			log.Errorf("eth listener received order contract address %s invalid", tx.To)
			continue
		}

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

func (l *EthClientListener) doMethod(input string) error {
	return nil
}

// 只需要解析submitRing,cancel，cutoff这些方法在event里，如果方法不成功也不用执行后续逻辑
func (l *EthClientListener) handleSubmitRingMethod(input eventemitter.EventData) error {
	println("doMethoddoMethoddoMethoddoMethoddoMethoddoMethod")
	//println(input.(string))
	// todo: unpack method
	// input := tx.Input
	// l.ethClient
	return nil
}

func (l *EthClientListener) handleOrderFilledEvent(input eventemitter.EventData) error {
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
	ord, err := l.ob.GetOrder(hash)
	if err != nil {
		return err
	}
	if err := evt.ConvertDown(ord); err != nil {
		return err
	}

	eventemitter.Emit(eventemitter.OrderBookChain.Name(), evt)

	return nil
}

func (l *EthClientListener) handleOrderCancelledEvent(input eventemitter.EventData) error {
	log.Debugf("eth listener log event:orderCancelled")

	evt := input.(chainclient.ContractData).Event.(chainclient.OrderCancelledEvent)

	if l.commOpts.Develop {
		log.Debugf("eth listener order cancelled event orderhash -> %s", types.BytesToHash(evt.OrderHash).Hex())
		log.Debugf("eth listener order cancelled event time -> %s", evt.Time.String())
		log.Debugf("eth listener order cancelled event block -> %s", evt.Blocknumber.String())
		log.Debugf("eth listener order cancelled event cancel amount -> %s", evt.AmountCancelled.String())
	}

	hash := types.BytesToHash(evt.OrderHash)
	ord, err := l.ob.GetOrder(hash)
	if err != nil {
		return err
	}

	evt.ConvertDown(ord)
	l.whisper.ChainOrderChan <- ord

	return nil
}

func (l *EthClientListener) handleCutoffTimestampEvent(input eventemitter.EventData) error {
	return nil
}

func (l *EthClientListener) handleRinghashSubmitEvent(input eventemitter.EventData) error {
	return nil
}

func (l *EthClientListener) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	close(l.stop)
}

// 重启(分叉)时先关停subscribeEvents，然后关
func (l *EthClientListener) Restart() {

}

func (l *EthClientListener) Name() string {
	return "eth-listener"
}

func (l *EthClientListener) getBlockNumberRange() (*big.Int, *big.Int) {
	start := l.commOpts.DefaultBlockNumber
	end := l.commOpts.EndBlockNumber

	// todo: free comment
	//currentBlockNumber, err:= l.getBlockNumber()
	//if err != nil {
	//	panic(err)
	//} else {
	//	log.Debugf("eth block number :%s", currentBlockNumber.String())
	//}
	//start = currentBlockNumber

	return start, end
}

func (l *EthClientListener) judgeContractAddress(addr string) bool {
	for _, v := range l.commOpts.LoopringImpAddresses {
		if addr == v {
			return true
		}
	}
	return false
}

func (l *EthClientListener) addContractEvent(event chainclient.AbiEvent) {
	id := types.HexToHash(event.Id())
	addr := event.Address()

	log.Infof("addContractEvent address:%s", addr.Hex())
	if _, ok := l.contractEvents[addr]; !ok {
		l.contractEvents[addr] = make(map[types.Hash]chainclient.AbiEvent)
	}

	log.Infof("addContractEvent id:%s", id.Hex())
	l.contractEvents[addr][id] = event
}

func (l *EthClientListener) addContractMethod(method chainclient.AbiMethod) {
	id := types.HexToHash(method.MethodId())
	addr := method.Address()

	if _, ok := l.contractMethods[addr]; !ok {
		l.contractMethods[addr] = make(map[types.Hash]chainclient.AbiMethod)
	}

	l.contractMethods[addr][id] = method
}

func (l *EthClientListener) getContractEvent(addr types.Address, id types.Hash) (chainclient.AbiEvent, error) {
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

func (l *EthClientListener) getContractMethod(addr types.Address, id types.Hash) (chainclient.AbiMethod, error) {
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
