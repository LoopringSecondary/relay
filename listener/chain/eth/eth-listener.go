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
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/orderbook"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
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
	options      config.ChainClientOptions
	commOpts     config.CommonOptions
	whisper      *Whisper
	ob           *orderbook.OrderBook
	stop         chan struct{}
	stopSubEvent chan struct{}
	stopSubBlock chan struct{}
	lock         sync.RWMutex
	filterIds    map[string]string
	ethClient    *eth.EthClient
}

func NewListener(options config.ChainClientOptions,
	commonOpts config.CommonOptions,
	whisper *Whisper,
	ethClient *eth.EthClient,
	ob *orderbook.OrderBook) *EthClientListener {
	var l EthClientListener

	l.options = options
	l.commOpts = commonOpts
	l.whisper = whisper
	l.ethClient = ethClient
	l.ob = ob

	return &l
}

// TODO(fukun): 这里调试调不通,应当返回channel
func (l *EthClientListener) Start() {
	go l.startSubEvents()
	go l.startSubBlocks()
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

func (l *EthClientListener) startSubEvents() {
	l.stopSubEvent = make(chan struct{})

	// 获取filterId
	filterId, err := l.newEventFilter()
	if err != nil {
		panic(err)
	}

	//获取blockNumber对应的所有logs
	var oldLogs []eth.Log
	if err := l.ethClient.GetFilterLogs(&oldLogs, filterId); err != nil {
		panic(err)
	}

	// 所有logs重新存一遍
	for _, v := range oldLogs {
		if err := l.doEvent(v); err != nil {
			log.Error("save event error", zap.String("content", err.Error()))
		}
	}

	// 监听新事件
	var newLogs []eth.Log
	for {
		err := l.ethClient.GetFilterChanges(&newLogs, filterId)
		if err != nil {
			panic(err)
		}

		for _, v := range newLogs {
			if err := l.doEvent(v); err != nil {
				log.Error("save event error", zap.String("content", err.Error()))
			}
		}
	}
}

func (l *EthClientListener) doEvent(v eth.Log) error {
	address := types.HexToAddress(v.Address)
	impl, ok := miner.LoopringInstance.LoopringImpls[address]
	if !ok {
		return error("contract address do not exsit")
	}

	topic := v.Topics[0]
	height := v.BlockNumber.Int()
	tx := types.HexToHash(v.TransactionHash)
	data := []byte(v.Data)

	switch topic {
	case impl.RingMined.Id():
		// TODO(fukun): 无需转换
		evt := chainclient.RingMinedEvent{}
		impl.RingMined.Unpack(&evt, data, v.Topics)
		if _, err := evt.MarshalJSON(); err != nil {
			return nil
		}

	case impl.OrderFilled.Id():
		evt := chainclient.OrderFilledEvent{}
		impl.OrderFilled.Unpack(&evt, data, v.Topics)
		bs, err := evt.MarshalJSON()
		if err != nil {
			return err
		}

		// todo: 如果ob中不存在该订单(其他形式传来的)，那么跳过该event，在doTransaction中解析该order
		hash := types.BytesToHash(evt.OrderHash)
		ord, err := l.ob.GetOrder(hash)
		if err != nil {
			return err
		}

		// 将event中相关数据装换为orderState
		evt.ConvertDown(ord)
		l.whisper.ChainOrderChan <- ord
		l.ob.SetTransaction(topic, height, tx, bs)

	case impl.OrderCancelled.Id():
		evt := chainclient.OrderCancelledEvent{}
		impl.OrderCancelled.Unpack(&evt, data, v.Topics)

		bs, err := evt.MarshalJSON()
		if err != nil {
			return err
		}

		hash := types.BytesToHash(evt.OrderHash)
		ord, err := l.ob.GetOrder(hash)
		if err != nil {
			return err
		}

		evt.ConvertDown(ord)
		l.whisper.ChainOrderChan <- ord
		l.ob.SetTransaction(topic, height, tx, bs)

	case impl.CutoffTimestampChanged.Id():
		evt := chainclient.CutoffTimestampChangedEvent{}
		impl.CutoffTimestampChanged.Unpack(&evt, data, v.Topics)
		if _, err := evt.MarshalJSON(); err != nil {
			return err
		}
		// todo(fukun)

	}

	return nil
}

func (l *EthClientListener) stopSubEvents() {
	l.lock.Lock()
	defer l.lock.Unlock()

	close(l.stopSubEvent)
}

// 监听blockhash
func (l *EthClientListener) startSubBlocks() {

}

func (l *EthClientListener) stopSubBlocks() {

}

func (l *EthClientListener) newEventFilter() (string, error) {
	var filterId string

	filter := eth.FilterQuery{}
	filter.FromBlock = types.Int2BlockNumHex(l.ob.GetBlockNumber())
	filter.ToBlock = "latest"
	filter.Address = l.getTokenAddress()

	err := l.ethClient.NewFilter(&filterId, &filter)
	if err != nil {
		return "", err
	}

	return filterId, nil
}

func (l *EthClientListener) getTokenAddress() []common.Address {
	var ret []common.Address
	for _, v := range l.commOpts.LoopringImpAddresses {
		ret = append(ret, common.HexToAddress(v))
	}
	return ret
}
