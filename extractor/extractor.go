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
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/dao"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"reflect"
	"sync"
	"time"
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
	stop            chan struct{}
	lock            sync.RWMutex
	contractMethods map[types.Address]map[types.Hash]chainclient.AbiMethod
	contractEvents  map[types.Address]map[types.Hash]chainclient.AbiEvent
}

func NewExtractorService(options config.ChainClientOptions,
	commonOpts config.CommonOptions,
	ethClient *eth.EthClient,
	rds dao.RdsService) *ExtractorServiceImpl {
	var l ExtractorServiceImpl

	l.options = options
	l.commOpts = commonOpts
	l.ethClient = ethClient
	l.dao = rds

	l.loadContract()
	l.startDetectFork()

	return &l
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
				log.Fatalf("extractor iterator next error:%s", err.Error())
			}

			block := inter.(*eth.BlockWithTxObject)
			log.Debugf("extractor get block:%s->%s", block.Number.BigInt().String(), block.Hash.Hex())

			txcnt := len(block.Transactions)
			if txcnt < 1 {
				log.Debugf("extractor get none block transaction")
				continue
			} else {
				log.Infof("extractor get block transaction list length %d", txcnt)
			}

			checkForkBlock := types.Block{}
			checkForkBlock.BlockNumber = block.Number.BigInt()
			checkForkBlock.ParentHash = block.ParentHash
			checkForkBlock.BlockHash = block.Hash
			checkForkBlock.CreateTime = block.Timestamp.Int64()
			if err := l.detectFork(&checkForkBlock); err != nil {
				log.Debugf("extractor detect fork error:%s", err.Error())
			}

			l.doBlock(*block)
		}
	}()
}

func (l *ExtractorServiceImpl) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	close(l.stop)
}

// 重启(分叉)时先关停subscribeEvents，然后关
func (l *ExtractorServiceImpl) Restart() {
	l.Stop()
	time.Sleep(1 * time.Second)
	l.Start()
}

func (l *ExtractorServiceImpl) doBlock(block eth.BlockWithTxObject) {
	txhashs := []types.Hash{}

	for _, tx := range block.Transactions {
		log.Debugf("extractor get transaction hash:%s", tx.Hash)
		log.Debugf("extractor get transaction input:%s", tx.Input)

		// 解析method，获得ring内等orders并发送到orderbook保存
		l.doMethod(tx.Input)

		// 获取transaction内所有event logs
		var receipt eth.TransactionReceipt
		err := l.ethClient.GetTransactionReceipt(&receipt, tx.Hash)
		if err != nil {
			log.Errorf("extractor get transaction receipt error:%s", err.Error())
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
		log.Debugf("extractor order filled event ringhash -> %s", types.BytesToHash(evt.Ringhash).Hex())
		log.Debugf("extractor order filled event amountS -> %s", evt.AmountS.String())
		log.Debugf("extractor order filled event amountB -> %s", evt.AmountB.String())
		log.Debugf("extractor order filled event orderhash -> %s", types.BytesToHash(evt.OrderHash).Hex())
		log.Debugf("extractor order filled event blocknumber -> %s", evt.Blocknumber.String())
		log.Debugf("extractor order filled event time -> %s", evt.Time.String())
		log.Debugf("extractor order filled event lrcfee -> %s", evt.LrcFee.String())
		log.Debugf("extractor order filled event lrcreward -> %s", evt.LrcReward.String())
		log.Debugf("extractor order filled event nextorderhash -> %s", types.BytesToHash(evt.NextOrderHash).Hex())
		log.Debugf("extractor order filled event preorderhash -> %s", types.BytesToHash(evt.PreOrderHash).Hex())
		log.Debugf("extractor order filled event ringindex -> %s", evt.RingIndex.String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorFill, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleOrderCancelledEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:orderCancelled")

	evt := input.(chainclient.ContractData).Event.(chainclient.OrderCancelledEvent)

	if l.commOpts.Develop {
		log.Debugf("extractor order cancelled event orderhash -> %s", types.BytesToHash(evt.OrderHash).Hex())
		log.Debugf("extractor order cancelled event time -> %s", evt.Time.String())
		log.Debugf("extractor order cancelled event block -> %s", evt.Blocknumber.String())
		log.Debugf("extractor order cancelled event cancel amount -> %s", evt.AmountCancelled.String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorCancel, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleCutoffTimestampEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:cutOffTimestampChanged")

	evt := input.(chainclient.ContractData).Event.(chainclient.CutoffTimestampChangedEvent)

	if l.commOpts.Develop {
		log.Debugf("extractor cutoffTimestampChanged event owner address -> %s", evt.Owner.Hex())
		log.Debugf("extractor cutoffTimestampChanged event time -> %s", evt.Time.String())
		log.Debugf("extractor cutoffTimestampChanged event block -> %s", evt.Blocknumber.String())
		log.Debugf("extractor cutoffTimestampChanged event cutoff time -> %s", evt.Cutoff.String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorCutoff, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleRinghashSubmitEvent(input eventemitter.EventData) error {
	return nil
}

// todo: modify
func (l *ExtractorServiceImpl) getBlockNumberRange() (*big.Int, *big.Int) {
	var tForkBlock types.Block

	start := l.commOpts.DefaultBlockNumber
	end := l.commOpts.EndBlockNumber

	// 寻找分叉块，并归零分叉标记
	forkBlock, err := l.dao.FindForkBlock()
	if err == nil {
		forkBlock.ConvertUp(&tForkBlock)
		forkBlock.Fork = false
		l.dao.Update(forkBlock)
		return tForkBlock.BlockNumber, end
	}

	// 寻找最新块
	latestBlock, err := l.dao.FindLatestBlock()
	if err != nil {
		return start, end
	}
	latestBlock.ConvertUp(&tForkBlock)

	return tForkBlock.BlockNumber, end
}
