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

const (
	BLOCK_HASH_TABLE_NAME = "block_hash_table"
	TRANSACTION_HASH_TABLE_NAME = "transaction_hash_table"
)

type Whisper struct {
	ChainOrderChan chan *types.OrderState
}

// TODO(fukun):不同的channel，应当交给orderbook统一进行后续处理，可以将channel作为函数返回值、全局变量、参数等方式
type EthClientListener struct {
	options        config.ChainClientOptions
	commOpts       config.CommonOptions
	ethClient      *eth.EthClient
	ob             *orderbook.OrderBook
	db             db.Database
	blockhashTable db.Database
	txhashTable 	db.Database
	whisper        *Whisper
	stop           chan struct{}
	lock           sync.RWMutex
}

func NewListener(options config.ChainClientOptions,
	commonOpts config.CommonOptions,
	whisper *Whisper,
	ethClient *eth.EthClient,
	ob *orderbook.OrderBook,
	database db.Database) *EthClientListener {
	var l EthClientListener

	l.options = options
	l.commOpts = commonOpts
	l.whisper = whisper
	l.ethClient = ethClient
	l.ob = ob
	l.db = database
	l.blockhashTable = db.NewTable(l.db, BLOCK_HASH_TABLE_NAME)
	l.txhashTable = db.NewTable(l.db, TRANSACTION_HASH_TABLE_NAME)

	return &l
}

// TODO(fukun): 这里调试调不通,应当返回channel
func (l *EthClientListener) Start() {
	l.stop = make(chan struct{})

	start := l.commOpts.DefaultBlockNumber
	end := l.commOpts.EndBlockNumber

	for {
		blockData, err := l.ethClient.BlockIterator(start, end).Next()
		if err != nil {
			log.Errorf("get block hash error:%s", err.Error())
			continue
		}
		// todo:
		println(blockData)
	}

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
	// 获取filterId
	//filterId, err := l.newEventFilter()
	//if err != nil {
	//	panic(err)
	//}

	filterId := ""

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
		return errors.New("contract address do not exsit")
	}

	topic := v.Topics[0]
	//height := v.BlockNumber.Int()
	//tx := types.HexToHash(v.TransactionHash)
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
		//bs, err := evt.MarshalJSON()
		//if err != nil {
		//	return err
		//}

		// todo: 如果ob中不存在该订单(其他形式传来的)，那么跳过该event，在doTransaction中解析该order
		hash := types.BytesToHash(evt.OrderHash)
		ord, err := l.ob.GetOrder(hash)
		if err != nil {
			return err
		}

		// 将event中相关数据装换为orderState
		evt.ConvertDown(ord)
		l.whisper.ChainOrderChan <- ord

	case impl.OrderCancelled.Id():
		evt := chainclient.OrderCancelledEvent{}
		impl.OrderCancelled.Unpack(&evt, data, v.Topics)

		//bs, err := evt.MarshalJSON()
		//if err != nil {
		//	return err
		//}

		hash := types.BytesToHash(evt.OrderHash)
		ord, err := l.ob.GetOrder(hash)
		if err != nil {
			return err
		}

		evt.ConvertDown(ord)
		l.whisper.ChainOrderChan <- ord

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

//func (l *EthClientListener) newEventFilter() (string, error) {
//	var filterId string
//
//	filter := eth.FilterQuery{}
//	filter.FromBlock = types.Int2BlockNumHex(l.ob.GetBlockNumber())
//	filter.ToBlock = "latest"
//	filter.Address = l.getTokenAddress()
//
//	err := l.ethClient.NewFilter(&filterId, &filter)
//	if err != nil {
//		return "", err
//	}
//
//	return filterId, nil
//}

func (l *EthClientListener) getTokenAddress() []common.Address {
	var ret []common.Address
	for _, v := range l.commOpts.LoopringImpAddresses {
		ret = append(ret, common.HexToAddress(v))
	}
	return ret
}
