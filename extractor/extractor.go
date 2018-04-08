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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"sync"
	"time"
)

/**
区块链的listener, 得到order以及ring的事件，
*/

const defaultEndBlockNumber = 1000000000

type ExtractorService interface {
	Start()
	Stop()
	ForkProcess(block *types.Block)
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
	forktest         bool
}

func NewExtractorService(options config.ExtractorOptions,
	db dao.RdsService,
	ac *market.AccountManager) *ExtractorServiceImpl {
	var l ExtractorServiceImpl

	l.options = options
	l.dao = db
	l.processor = newAbiProcessor(db, ac)
	l.detector = newForkDetector(db, l.options.StartBlockNumber)
	l.stop = make(chan bool, 1)

	l.setBlockNumberRange()
	return &l
}

func (l *ExtractorServiceImpl) Start() {
	if !l.options.Open {
		return
	}

	log.Info("extractor start...")
	l.syncComplete = false

	l.pendingTxWatcher = &eventemitter.Watcher{Concurrent: false, Handle: l.WatchingPendingTransaction}
	eventemitter.On(eventemitter.PendingTransaction, l.pendingTxWatcher)

	l.iterator = ethaccessor.NewBlockIterator(l.startBlockNumber, l.endBlockNumber, true, l.options.ConfirmBlockNumber)
	go func() {
		for {
			select {
			case <-l.stop:
				return
			default:
				l.ProcessBlock()
			}
		}
	}()
}

func (l *ExtractorServiceImpl) Stop() {
	if !l.options.Open {
		return
	}

	eventemitter.Un(eventemitter.PendingTransaction, l.pendingTxWatcher)
	l.stop <- true
}

// 重启(分叉)时先关停subscribeEvents，然后关
func (l *ExtractorServiceImpl) ForkProcess(currentBlock *types.Block) {
	forkBlock := l.detector.Detect(currentBlock)
	if forkBlock == nil {
		return
	}

	// mark fork block in database
	model := dao.Block{}
	model.ConvertDown(forkBlock)
	if err := l.dao.SetForkBlock(forkBlock.BlockHash); err != nil {
		log.Fatalf("extractor,fork detector mark fork block %s failed, you should mark it manual, err:%s", forkBlock.BlockHash.Hex(), err.Error())
	}

	// emit fork event
	var forkEvent types.ForkedEvent
	forkEvent.ForkHash = forkBlock.BlockHash
	forkEvent.ForkBlock = forkBlock.BlockNumber
	forkEvent.DetectedHash = currentBlock.BlockHash
	forkEvent.DetectedBlock = currentBlock.BlockNumber

	log.Debugf("extractor,detected chain fork, from :%d to %d", forkEvent.ForkBlock.Int64(), forkEvent.DetectedBlock.Int64())
	eventemitter.Emit(eventemitter.ChainForkDetected, &forkEvent)

	l.Stop()
	l.startBlockNumber = new(big.Int).Add(forkBlock.BlockNumber, big.NewInt(1))
	l.Start()
}

func (l *ExtractorServiceImpl) Sync(blockNumber *big.Int) {
	var syncBlock types.Big
	if err := ethaccessor.BlockNumber(&syncBlock); err != nil {
		log.Fatalf("extractor,Sync chain block,get ethereum node current block number error:%s", err.Error())
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

func (l *ExtractorServiceImpl) WatchingPendingTransaction(input eventemitter.EventData) error {
	tx := input.(*ethaccessor.Transaction)
	return l.ProcessPendingTransaction(tx)
}

func (l *ExtractorServiceImpl) ProcessBlock() {
	inter, err := l.iterator.Next()
	if err != nil {
		log.Fatalf("extractor,iterator next error:%s", err.Error())
	}

	// get current block
	block := inter.(*ethaccessor.BlockWithTxAndReceipt)
	log.Infof("extractor,get block:%s->%s, transaction number:%d", block.Number.BigInt().String(), block.Hash.Hex(), len(block.Transactions))

	currentBlock := &types.Block{}
	currentBlock.BlockNumber = block.Number.BigInt()
	currentBlock.ParentHash = block.ParentHash
	currentBlock.BlockHash = block.Hash
	currentBlock.CreateTime = block.Timestamp.Int64()

	// Sync blocks on chain
	if l.syncComplete == false {
		l.Sync(block.Number.BigInt())
	}

	// detect chain fork
	l.ForkProcess(currentBlock)

	// convert block to dao entity
	var entity dao.Block
	entity.ConvertDown(currentBlock)
	l.dao.SaveBlock(&entity)

	// emit new block
	blockEvent := &types.BlockEvent{}
	blockEvent.BlockNumber = block.Number.BigInt()
	blockEvent.BlockHash = block.Hash
	eventemitter.Emit(eventemitter.Block_New, blockEvent)

	var txcnt types.Big
	if err := ethaccessor.GetBlockTransactionCountByHash(&txcnt, block.Hash.Hex(), block.Number.BigInt().String()); err != nil {
		log.Fatalf("extractor,getBlockTransactionCountByHash error:%s", err.Error())
	}
	txcntinblock := len(block.Transactions)
	if txcntinblock < 1 {
		return
	}
	if txcnt.Int() != txcntinblock {
		log.Fatalf("extractor,transaction number %d != len(block.transactions) %d", txcnt.Int(), txcntinblock)
	}

	for idx, transaction := range block.Transactions {
		receipt := block.Receipts[idx]

		l.debug("extractor,tx:%s", transaction.Hash)
		l.ProcessMinedTransaction(&transaction, &receipt, block.Timestamp.BigInt())
	}
}

func (l *ExtractorServiceImpl) ProcessMinedTransaction(tx *ethaccessor.Transaction, receipt *ethaccessor.TransactionReceipt, blockTime *big.Int) error {
	l.debug("extractor,process mined transaction,tx:%s status :%s,logs:%d", tx.Hash, receipt.Status.BigInt().String(), len(receipt.Logs))

	if l.processor.HasContract(common.HexToAddress(tx.To)) {
		l.ProcessEvent(tx, receipt, blockTime)
		if receipt.IsFailed() {
			l.ProcessMethod(tx, receipt, blockTime)
		}
		return nil
	} else {
		return l.processor.handleEthTransfer(tx, receipt.GasUsed.BigInt(), blockTime, uint8(types.TX_STATUS_SUCCESS))
	}
}

func (l *ExtractorServiceImpl) ProcessPendingTransaction(tx *ethaccessor.Transaction) error {
	log.Debugf("extractor,process pending transaction %s", tx.Hash)

	blockTime := big.NewInt(time.Now().Unix())

	if l.processor.HasContract(common.HexToAddress(tx.To)) {
		return l.ProcessMethod(tx, nil, blockTime)
	} else {
		return l.processor.handleEthTransfer(tx, big.NewInt(0), blockTime, types.TX_STATUS_PENDING)
	}
}

func (l *ExtractorServiceImpl) ProcessMethod(tx *ethaccessor.Transaction, receipt *ethaccessor.TransactionReceipt, blockTime *big.Int) error {
	// filter method input
	input := common.FromHex(tx.Input)
	if len(input) < 4 || len(tx.Input) < 10 {
		l.debug("extractor,tx:%s contract method id %s length invalid", tx.Hash, tx.Input)
		return nil
	}

	// filter method id
	id := common.ToHex(input[0:4])
	method, ok := l.processor.GetMethod(id)
	if !ok {
		l.debug("extractor,tx:%s contract method id error:%s", tx.Hash, id)
		return nil
	}

	var (
		status  uint
		gasUsed *big.Int
	)
	if receipt == nil {
		status = types.TX_STATUS_PENDING
		gasUsed = big.NewInt(0)
	} else if receipt.IsFailed() {
		status = types.TX_STATUS_FAILED
		gasUsed = receipt.GasUsed.BigInt()
	} else {
		status = types.TX_STATUS_SUCCESS
		gasUsed = receipt.GasUsed.BigInt()
	}

	method.FullFilled(tx, gasUsed, blockTime, uint8(status))
	eventemitter.Emit(method.Id, method)

	return nil
}

func (l *ExtractorServiceImpl) ProcessEvent(tx *ethaccessor.Transaction, receipt *ethaccessor.TransactionReceipt, blockTime *big.Int) error {
	// filter logs
	if len(receipt.Logs) <= 0 {
		l.debug("extractor,tx:%s contract method unsupported protocol %s", tx.Hash, tx.To)
		return nil
	}

	txhash := receipt.TransactionHash
	for _, evtLog := range receipt.Logs {
		var (
			event EventData
			ok    bool
		)

		// 过滤合约
		protocolAddr := common.HexToAddress(evtLog.Address)
		if ok := l.processor.HasContract(protocolAddr); !ok {
			l.debug("extractor,tx:%s contract event unsupported protocol %s", txhash, protocolAddr.Hex())
			continue
		}

		// 过滤事件
		data := hexutil.MustDecode(evtLog.Data)
		id := common.HexToHash(evtLog.Topics[0])
		if event, ok = l.processor.GetEvent(id); !ok {
			l.debug("extractor,tx:%s contract event id error:%s", txhash, id.Hex())
			continue
		}

		if nil != data && len(data) > 0 {
			// 解析事件
			if err := event.CAbi.Unpack(event.Event, event.Name, data, abi.SEL_UNPACK_EVENT); nil != err {
				log.Errorf("extractor,tx:%s unpack event error:%s", txhash, err.Error())
				continue
			}
		}

		// full filled event and emit to abi processor
		event.FullFilled(tx, &evtLog, receipt.GasUsed.BigInt(), blockTime)
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
