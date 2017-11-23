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
	"github.com/Loopring/relay/chainclient"
	"github.com/Loopring/relay/chainclient/eth"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"reflect"
	"strconv"
	"sync"
	"time"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/ethereum/bak/go-ethereum/accounts/abi"
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
	commOpts        config.CommonOptions
	accessor 		*ethaccessor.EthNodeAccessor
	dao             dao.RdsService
	stop            chan struct{}
	lock            sync.RWMutex
	abis 			map[types.Address]abi.ABI
}

func NewExtractorService(commonOpts config.CommonOptions, accessor *ethaccessor.EthNodeAccessor, rds dao.RdsService) *ExtractorServiceImpl {
	var l ExtractorServiceImpl

	l.commOpts = commonOpts
	l.accessor = accessor
	l.dao = rds

	submitRingMethodWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleSubmitRingMethod}
	ringhashSubmitEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleRinghashSubmitEvent}
	orderFilledEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderFilledEvent}
	orderCancelledEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleOrderCancelledEvent}
	cutoffTimestampEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleCutoffTimestampEvent}
	transferEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleTransferEvent}
	approvalEventWatcher := &eventemitter.Watcher{Concurrent: false, Handle: l.handleApprovalEvent}

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

	for _, impl := range miner.MinerInstance.Loopring.Tokens {
		transferEvt := impl.TransferEvt
		approvalEvt := impl.ApprovalEvt

		l.addContractEvent(transferEvt)
		l.addContractEvent(approvalEvt)

		eventemitter.On(transferEvt.WatcherTopic(), transferEventWatcher)
		eventemitter.On(approvalEvt.WatcherTopic(), approvalEventWatcher)
	}

	l.loadContract()
	l.startDetectFork()

	return &l
}

func (l *ExtractorServiceImpl) Start() {
	l.stop = make(chan struct{})

	log.Info("eth listener start...")
	start, end := l.getBlockNumberRange()
	iterator := l.accessor.BlockIterator(start, end, true, uint64(0))

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
		if err := l.accessor.Call(&receipt, "eth_getTransactionReceipt", tx.Hash); err != nil {
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
				Event:       dstEvt.Elem().Interface().(chainclient.AbiEvent),
				BlockNumber: &evtLog.BlockNumber,
				Time:        &block.Timestamp,
			}
			eventemitter.Emit(contractEvt.WatcherTopic(), event)

			txhashs = append(txhashs, txhash)
			// todo 是否需要存储transaction
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

func (l *ExtractorServiceImpl) handleRingMinedEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:ringMined")

	contractData := input.(chainclient.ContractData)
	contractEvent := contractData.Event.(chainclient.RingMinedEvent)
	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber
	evt.IsDeleted = false

	if l.commOpts.Develop {
		log.Debugf("extractor ring mined event ringhash -> %s", evt.Ringhash.Hex())
		log.Debugf("extractor ring mined event ringIndex -> %s", evt.RingIndex.BigInt().String())
		log.Debugf("extractor ring mined event miner -> %s", evt.Miner.Hex())
		log.Debugf("extractor ring mined event feeRecipient -> %s", evt.FeeRecipient.Hex())
		log.Debugf("extractor ring mined event isRinghashReserved -> %s", strconv.FormatBool(evt.IsRinghashReserved))
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorRingMined, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleOrderFilledEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:orderFilled")

	contractData := input.(chainclient.ContractData)
	contractEvent := contractData.Event.(chainclient.OrderFilledEvent)
	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber
	evt.IsDeleted = false

	if l.commOpts.Develop {
		log.Debugf("extractor order filled event ringhash -> %s", evt.Ringhash.Hex())
		log.Debugf("extractor order filled event amountS -> %s", evt.AmountS.BigInt().String())
		log.Debugf("extractor order filled event amountB -> %s", evt.AmountB.BigInt().String())
		log.Debugf("extractor order filled event orderhash -> %s", evt.OrderHash.Hex())
		log.Debugf("extractor order filled event blocknumber -> %s", evt.Blocknumber.BigInt().String())
		log.Debugf("extractor order filled event time -> %s", evt.Time.BigInt().String())
		log.Debugf("extractor order filled event lrcfee -> %s", evt.LrcFee.BigInt().String())
		log.Debugf("extractor order filled event lrcreward -> %s", evt.LrcReward.BigInt().String())
		log.Debugf("extractor order filled event nextorderhash -> %s", evt.NextOrderHash.Hex())
		log.Debugf("extractor order filled event preorderhash -> %s", evt.PreOrderHash.Hex())
		log.Debugf("extractor order filled event ringindex -> %s", evt.RingIndex.BigInt().String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorFill, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleOrderCancelledEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:orderCancelled")

	contractData := input.(chainclient.ContractData)
	contractEvent := contractData.Event.(chainclient.OrderCancelledEvent)
	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber
	evt.IsDeleted = false

	if l.commOpts.Develop {
		log.Debugf("extractor order cancelled event orderhash -> %s", evt.OrderHash.Hex())
		log.Debugf("extractor order cancelled event time -> %s", evt.Time.BigInt().String())
		log.Debugf("extractor order cancelled event block -> %s", evt.Blocknumber.BigInt().String())
		log.Debugf("extractor order cancelled event cancel amount -> %s", evt.AmountCancelled.BigInt().String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorCancel, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleCutoffTimestampEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:cutOffTimestampChanged")

	contractData := input.(chainclient.ContractData)
	contractEvent := contractData.Event.(chainclient.CutoffTimestampChangedEvent)
	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber
	evt.IsDeleted = false

	if l.commOpts.Develop {
		log.Debugf("extractor cutoffTimestampChanged event owner address -> %s", evt.Owner.Hex())
		log.Debugf("extractor cutoffTimestampChanged event time -> %s", evt.Time.BigInt().String())
		log.Debugf("extractor cutoffTimestampChanged event block -> %s", evt.Blocknumber.BigInt().String())
		log.Debugf("extractor cutoffTimestampChanged event cutoff time -> %s", evt.Cutoff.BigInt().String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorCutoff, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleRinghashSubmitEvent(input eventemitter.EventData) error {
	return nil
}

func (l *ExtractorServiceImpl) handleTransferEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:erc20 transfer event")

	contractData := input.(chainclient.ContractData)
	contractEvent := contractData.Event.(chainclient.TransferEvent)
	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor transfer event from -> %s", evt.From.Hex())
		log.Debugf("extractor transfer event to -> %s", evt.To.Hex())
		log.Debugf("extractor transfer event value -> %s", evt.Value.BigInt().String())
	}

	eventemitter.Emit(eventemitter.AccountTransfer, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleApprovalEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:erc20 approval event")

	contractData := input.(chainclient.ContractData)
	contractEvent := contractData.Event.(chainclient.ApprovalEvent)
	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor approval event owner -> %s", evt.Owner.Hex())
		log.Debugf("extractor approval event spender -> %s", evt.Spender.Hex())
		log.Debugf("extractor approval event value -> %s", evt.Value.BigInt().String())
	}

	eventemitter.Emit(eventemitter.AccountApproval, evt)

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
