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
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"sort"
	"sync"
	"time"
)

/**
区块链的listener, 得到order以及ring的事件，
*/

const (
	defaultEndBlockNumber  = 1000000000
	defaultForkWaitingTime = 10
)

type ExtractorService interface {
	Start()
	Stop()
	ForkProcess(block *types.Block) error
}

// TODO(fukun):不同的channel，应当交给orderbook统一进行后续处理，可以将channel作为函数返回值、全局变量、参数等方式
type ExtractorServiceImpl struct {
	options          config.ExtractorOptions
	detector         *forkDetector
	processor        *AbiProcessor
	dao              dao.RdsService
	stop             chan bool
	lock             sync.RWMutex
	startBlockNumber *big.Int
	endBlockNumber   *big.Int
	iterator         *ethaccessor.BlockIterator
	pendingTxWatcher *eventemitter.Watcher
	syncComplete     bool
	forkComplete     bool
}

func NewExtractorService(options config.ExtractorOptions, db dao.RdsService) *ExtractorServiceImpl {
	var l ExtractorServiceImpl

	if options.ForkWaitingTime <= 0 {
		options.ForkWaitingTime = defaultForkWaitingTime
	}

	l.options = options
	l.dao = db
	l.processor = newAbiProcessor(db, &options)
	l.detector = newForkDetector(db, l.options.StartBlockNumber)
	l.stop = make(chan bool, 1)
	l.setBlockNumberRange()

	l.pendingTxWatcher = &eventemitter.Watcher{Concurrent: false, Handle: l.WatchingPendingTransaction}
	eventemitter.On(eventemitter.PendingTransaction, l.pendingTxWatcher)

	return &l
}

func (l *ExtractorServiceImpl) Start() {
	if !l.options.Open {
		return
	}

	log.Infof("extractor start from block:%s...", l.startBlockNumber.String())
	l.syncComplete = false

	l.iterator = ethaccessor.NewBlockIterator(l.startBlockNumber, l.endBlockNumber, true, l.options.ConfirmBlockNumber)
	go func() {
		for {
			select {
			case <-l.stop:
				return
			default:
				if err := l.ProcessBlock(); nil != err {
					log.Error(err.Error())
					time.Sleep(1 * time.Second)
				}
			}
		}
	}()
}

func (l *ExtractorServiceImpl) Stop() {
	if !l.options.Open {
		return
	}

	l.stop <- true
}

// 重启(分叉)时先关停subscribeEvents，然后关
func (l *ExtractorServiceImpl) ForkProcess(currentBlock *types.Block) error {
	forkEvent, err := l.detector.Detect(currentBlock)
	if err != nil {
		l.Warning(err)
		return err
	}

	if forkEvent == nil {
		return nil
	}

	log.Debugf("extractor,detected chain fork, from :%d to %d", forkEvent.ForkBlock.Int64(), forkEvent.DetectedBlock.Int64())

	l.Stop()

	// emit event
	eventemitter.Emit(eventemitter.ChainForkDetected, forkEvent)

	// reset start blockNumber
	l.startBlockNumber = new(big.Int).Add(forkEvent.ForkBlock, big.NewInt(1))

	// waiting for the eth node catch up
	time.Sleep(time.Duration(l.options.ForkWaitingTime) * time.Second)

	l.Start()

	return fmt.Errorf("extractor,detected chain fork")
}

func (l *ExtractorServiceImpl) Sync(blockNumber *big.Int) {
	var syncBlock types.Big
	if err := ethaccessor.BlockNumber(&syncBlock); err != nil {
		l.Warning(fmt.Errorf("extractor,Sync chain block,get ethereum node current block number error:%s", err.Error()))
	}
	currentBlockNumber := new(big.Int).Add(blockNumber, big.NewInt(int64(l.options.ConfirmBlockNumber)))
	if syncBlock.BigInt().Cmp(currentBlockNumber) <= 0 {
		eventemitter.Emit(eventemitter.SyncChainComplete, syncBlock)
		l.syncComplete = true
		log.Info("extractor,Sync chain block complete!")
	} else {
		log.Debugf("extractor,chain block syncing... ")
	}
}

// Warning 当发生严重错误时需关停extractor，并通知其他模块
func (l *ExtractorServiceImpl) Warning(err error) {
	l.Stop()
	log.Warnf("extractor, warning:%s", err.Error())
	var event types.ExtractorWarningEvent
	eventemitter.Emit(eventemitter.ExtractorWarning, &event)
}

func (l *ExtractorServiceImpl) WatchingPendingTransaction(input eventemitter.EventData) error {
	tx := input.(*ethaccessor.Transaction)
	return l.ProcessPendingTransaction(tx)
}

func (l *ExtractorServiceImpl) ProcessBlock() error {
	inter, err := l.iterator.Next()
	if err != nil {
		return fmt.Errorf("extractor,iterator next error:%s", err.Error())
	}

	// get current block
	block := inter.(*ethaccessor.BlockWithTxAndReceipt)
	log.Infof("extractor,get block:%s->%s, transaction number:%d", block.Number.BigInt().String(), block.Hash.Hex(), len(block.Transactions))

	currentBlock := &types.Block{}
	currentBlock.BlockNumber = block.Number.BigInt()
	currentBlock.ParentHash = block.ParentHash
	currentBlock.BlockHash = block.Hash
	currentBlock.CreateTime = block.Timestamp.Int64()

	// convert and save block
	var entity dao.Block
	entity.ConvertDown(currentBlock)
	l.dao.SaveBlock(&entity)

	// sync block on chain
	if l.syncComplete == false {
		l.Sync(block.Number.BigInt())
	}

	// detect chain fork
	if err := l.ForkProcess(currentBlock); err != nil {
		return err
	}

	// emit new block
	blockEvent := &types.BlockEvent{}
	blockEvent.BlockNumber = block.Number.BigInt()
	blockEvent.BlockHash = block.Hash
	blockEvent.BlockTime = block.Timestamp.Int64()
	eventemitter.Emit(eventemitter.Block_New, blockEvent)

	if len(block.Transactions) > 0 {
		for idx, transaction := range block.Transactions {
			receipt := block.Receipts[idx]
			l.debug("extractor,tx:%s", transaction.Hash)
			l.ProcessMinedTransaction(&transaction, &receipt, block.Timestamp.BigInt())
		}
	}

	eventemitter.Emit(eventemitter.Block_End, blockEvent)
	return nil
}

func (l *ExtractorServiceImpl) ProcessPendingTransaction(tx *ethaccessor.Transaction) error {
	log.Debugf("extractor,process pending transaction %s", tx.Hash)

	blockTime := big.NewInt(time.Now().Unix())

	if l.processor.SupportedMethod(tx) {
		return l.ProcessMethod(tx, nil, blockTime)
	}

	return l.processor.handleEthTransfer(tx, nil, blockTime)
}

func (l *ExtractorServiceImpl) ProcessMinedTransaction(tx *ethaccessor.Transaction, receipt *ethaccessor.TransactionReceipt, blockTime *big.Int) error {
	l.debug("extractor,process mined transaction,tx:%s status :%s,logs:%d", tx.Hash, receipt.Status.BigInt().String(), len(receipt.Logs))

	if l.processor.SupportedEvents(receipt) {
		return l.ProcessEvent(tx, receipt, blockTime)
	}

	if l.processor.SupportedMethod(tx) {
		return l.ProcessMethod(tx, receipt, blockTime)
	}

	return l.processor.handleEthTransfer(tx, receipt, blockTime)
}

func (l *ExtractorServiceImpl) ProcessMethod(tx *ethaccessor.Transaction, receipt *ethaccessor.TransactionReceipt, blockTime *big.Int) error {
	method, ok := l.processor.GetMethod(tx)
	if !ok {
		l.debug("extractor,process method,tx:%s,unsupported contract method", tx.Hash)
		return nil
	}

	gas, status := l.processor.getGasAndStatus(tx, receipt)
	method.FullFilled(tx, gas, blockTime, status, method.Name)
	eventemitter.Emit(method.Id, method)

	return nil
}

func (l *ExtractorServiceImpl) ProcessEvent(tx *ethaccessor.Transaction, receipt *ethaccessor.TransactionReceipt, blockTime *big.Int) error {
	methodName := l.processor.GetMethodName(tx)

	// 如果是submitRing的相关事件，必须保证fill在前，transfer在后
	if ethaccessor.TxIsSubmitRing(methodName) && len(receipt.Logs) > 1 {
		sort.SliceStable(receipt.Logs, func(i, j int) bool {
			cmpEventName := ethaccessor.EVENT_RING_MINED

			evti, _ := l.processor.GetEvent(receipt.Logs[i])

			if evti.Name == cmpEventName {
				return true
			}
			return false
		})
	}

	for _, evtLog := range receipt.Logs {
		event, ok := l.processor.GetEvent(evtLog)
		if !ok {
			l.debug("extractor,process event,tx:%s,unsupported contract event", tx.Hash)
			continue
		}

		data := hexutil.MustDecode(evtLog.Data)
		if nil != data && len(data) > 0 {
			if err := event.CAbi.Unpack(event.Event, event.Name, data, abi.SEL_UNPACK_EVENT); nil != err {
				log.Errorf("extractor,process event,tx:%s unpack event error:%s", tx.Hash, err.Error())
				continue
			}
		}

		event.FullFilled(tx, &evtLog, receipt.GasUsed.BigInt(), blockTime, methodName)
		eventemitter.Emit(event.Id.Hex(), event)
	}

	return nil
}

func (l *ExtractorServiceImpl) setBlockNumberRange() {
	l.startBlockNumber = l.options.StartBlockNumber
	l.endBlockNumber = l.options.EndBlockNumber
	if l.endBlockNumber.Cmp(big.NewInt(0)) == 0 {
		l.endBlockNumber = big.NewInt(defaultEndBlockNumber)
	}

	// 寻找最新块
	var ret types.Block
	latestBlock, err := l.dao.FindLatestBlock()
	if err != nil {
		log.Debugf("extractor,get latest block number error:%s", err.Error())
		return
	}
	latestBlock.ConvertUp(&ret)
	l.startBlockNumber = ret.BlockNumber

	log.Debugf("extractor,configStartBlockNumber:%s latestBlockNumber:%s", l.options.StartBlockNumber.String(), l.startBlockNumber.String())
}

func (l *ExtractorServiceImpl) debug(template string, args ...interface{}) {
	if l.options.Debug {
		log.Debugf(template, args...)
	}
}
