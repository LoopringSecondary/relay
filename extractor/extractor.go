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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"strconv"
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
	options  config.AccessorOptions
	commOpts config.CommonOptions
	accessor *ethaccessor.EthNodeAccessor
	dao      dao.RdsService
	stop     chan struct{}
	lock     sync.RWMutex
	events   map[string]ContractData
}

func NewExtractorService(options config.AccessorOptions,
	commonOpts config.CommonOptions,
	accessor *ethaccessor.EthNodeAccessor,
	rds dao.RdsService) *ExtractorServiceImpl {
	var l ExtractorServiceImpl

	l.commOpts = commonOpts
	l.accessor = accessor
	l.dao = rds

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

			block := inter.(*ethaccessor.BlockWithTxObject)
			log.Debugf("extractor get block:%s->%s", block.Number.BigInt().String(), block.Hash)

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

func (l *ExtractorServiceImpl) doBlock(block ethaccessor.BlockWithTxObject) {
	for _, tx := range block.Transactions {
		log.Debugf("extractor get transaction hash:%s", tx.Hash)
		log.Debugf("extractor get transaction input:%s", tx.Input)

		// 解析method，获得ring内等orders并发送到orderbook保存
		l.doMethod(tx.Input)

		// 获取transaction内所有event logs
		var receipt ethaccessor.TransactionReceipt
		if err := l.accessor.Call(&receipt, "eth_getTransactionReceipt", tx.Hash); err != nil {
			log.Errorf("extractor get transaction receipt error:%s", err.Error())
			continue
		}

		if len(receipt.Logs) == 0 {
			// todo
		}

		log.Debugf("transaction receipt  event logs number:%d", len(receipt.Logs))

		// todo 是否需要存储transaction

		for _, evtLog := range receipt.Logs {
			var (
				contract ContractData
				ok       bool
			)

			data := hexutil.MustDecode(evtLog.Data)
			id := evtLog.Topics[0]
			key := generateKey(receipt.To, id)
			if contract, ok = l.events[key]; !ok {
				log.Errorf("extractor: contract event id error:" + id)
				continue
			}

			// 解析事件
			if err := contract.CAbi.Unpack(contract.Event, contract.Name, data, abi.SEL_UNPACK_EVENT); nil != err {
				log.Errorf("err :%s", err.Error())
				continue
			}

			contract.Topics = evtLog.Topics
			contract.BlockNumber = &evtLog.BlockNumber
			contract.Time = &block.Timestamp
			eventemitter.Emit(contract.Key, contract)
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

	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return errors.New("extractor:ring mined event indexed fields number error")
	}

	contractEvent := contractData.Event.(ethaccessor.RingMinedEvent)
	contractEvent.RingHash = common.HexToHash(contractData.Topics[1])

	ringmined, fills, err := contractEvent.ConvertDown()
	if err != nil {
		return err
	}
	ringmined.Time = contractData.Time
	ringmined.Blocknumber = contractData.BlockNumber
	ringmined.IsDeleted = false

	if l.commOpts.Develop {
		log.Debugf("extractor ring mined event ringhash -> %s", ringmined.Ringhash.Hex())
		log.Debugf("extractor ring mined event ringIndex -> %s", ringmined.RingIndex.BigInt().String())
		log.Debugf("extractor ring mined event miner -> %s", ringmined.Miner.Hex())
		log.Debugf("extractor ring mined event feeRecipient -> %s", ringmined.FeeRecipient.Hex())
		log.Debugf("extractor ring mined event isRinghashReserved -> %s", strconv.FormatBool(ringmined.IsRinghashReserved))
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorRingMined, ringmined)

	for _, fill := range fills {
		fill.Time = contractData.Time
		fill.Blocknumber = contractData.BlockNumber
		fill.IsDeleted = false

		if l.commOpts.Develop {
			log.Debugf("extractor order filled event ringhash -> %s", fill.Ringhash.Hex())
			log.Debugf("extractor order filled event amountS -> %s", fill.AmountS.BigInt().String())
			log.Debugf("extractor order filled event amountB -> %s", fill.AmountB.BigInt().String())
			log.Debugf("extractor order filled event orderhash -> %s", fill.OrderHash.Hex())
			log.Debugf("extractor order filled event blocknumber -> %s", fill.Blocknumber.BigInt().String())
			log.Debugf("extractor order filled event time -> %s", fill.Time.BigInt().String())
			log.Debugf("extractor order filled event lrcfee -> %s", fill.LrcFee.BigInt().String())
			log.Debugf("extractor order filled event lrcreward -> %s", fill.LrcReward.BigInt().String())
			log.Debugf("extractor order filled event nextorderhash -> %s", fill.NextOrderHash.Hex())
			log.Debugf("extractor order filled event preorderhash -> %s", fill.PreOrderHash.Hex())
			log.Debugf("extractor order filled event ringindex -> %s", fill.RingIndex.BigInt().String())
		}

		eventemitter.Emit(eventemitter.OrderManagerExtractorFill, fill)
	}

	return nil
}

func (l *ExtractorServiceImpl) handleOrderCancelledEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:orderCancelled")

	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return errors.New("extractor:order cancelled event indexed fields number error")
	}

	contractEvent := contractData.Event.(ethaccessor.OrderCancelledEvent)
	contractEvent.OrderHash = common.HexToHash(contractData.Topics[1])

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

	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return errors.New("extractor:cutoff timestamp changed event indexed fields number error")
	}

	contractEvent := contractData.Event.(ethaccessor.CutoffTimestampChangedEvent)
	contractEvent.Owner = common.HexToAddress(contractData.Topics[1])

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

func (l *ExtractorServiceImpl) handleTransferEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:erc20 transfer event")

	contractData := input.(ContractData)
	if len(contractData.Topics) < 3 {
		return errors.New("extractor:token transfer event indexed fields number error")
	}

	contractEvent := contractData.Event.(ethaccessor.TransferEvent)
	contractEvent.From = common.HexToAddress(contractData.Topics[1])
	contractEvent.To = common.HexToAddress(contractData.Topics[2])

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

	contractData := input.(ContractData)
	if len(contractData.Topics) < 3 {
		return errors.New("extractor:token approval event indexed fields number error")
	}

	contractEvent := contractData.Event.(ethaccessor.ApprovalEvent)
	contractEvent.Owner = common.HexToAddress(contractData.Topics[1])
	contractEvent.Spender = common.HexToAddress(contractData.Topics[2])

	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor approval event owner -> %s", evt.Owner.Hex())
		log.Debugf("extractor approval event spender -> %s", evt.Spender.Hex())
		log.Debugf("extractor approval event value -> %s", evt.Value.BigInt().String())
	}

	eventemitter.Emit(eventemitter.TokenRegistered, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleTokenRegisteredEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:token registered event")

	contractData := input.(ContractData)
	contractEvent := contractData.Event.(ethaccessor.TokenRegisteredEvent)
	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor token registered event address -> %s", evt.Token.Hex())
		log.Debugf("extractor token registered event spender -> %s", evt.Symbol)
	}

	eventemitter.Emit(eventemitter.TokenRegistered, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleTokenUnRegisteredEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:token unregistered event")

	contractData := input.(ContractData)
	contractEvent := contractData.Event.(ethaccessor.TokenUnRegisteredEvent)
	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor token unregistered event address -> %s", evt.Token.Hex())
		log.Debugf("extractor token unregistered event spender -> %s", evt.Symbol)
	}

	eventemitter.Emit(eventemitter.TokenUnRegistered, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleRinghashSubmitEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:ringhash registered event")

	contractData := input.(ContractData)
	if len(contractData.Topics) < 3 {
		return errors.New("extractor:ringhash registered event indexed fields number error")
	}

	contractEvent := contractData.Event.(ethaccessor.RingHashSubmittedEvent)
	contractEvent.RingMiner = common.HexToAddress(contractData.Topics[1])
	contractEvent.RingHash = common.HexToHash(contractData.Topics[2])

	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor ringhash submit event ringhash -> %s", evt.RingHash.Hex())
		log.Debugf("extractor ringhash submit event ringminer -> %s", evt.RingMiner.Hex())
	}

	eventemitter.Emit(eventemitter.RingHashSubmitted, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleAddressAuthorizedEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:address authorized event")

	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return errors.New("extractor:address authorized event indexed fields number error")
	}

	contractEvent := contractData.Event.(ethaccessor.AddressAuthorizedEvent)
	contractEvent.ContractAddress = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor address authorized event address -> %s", evt.ContractAddress.Hex())
		log.Debugf("extractor address authorized event number -> %d", evt.Number)
	}

	eventemitter.Emit(eventemitter.AddressAuthorized, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleAddressDeAuthorizedEvent(input eventemitter.EventData) error {
	log.Debugf("extractor log event:address deauthorized event")

	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return errors.New("extractor:address deauthorized event indexed fields number error")
	}

	contractEvent := contractData.Event.(ethaccessor.AddressDeAuthorizedEvent)
	contractEvent.ContractAddress = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor address deauthorized event address -> %s", evt.ContractAddress.Hex())
		log.Debugf("extractor address deauthorized event number -> %d", evt.Number)
	}

	eventemitter.Emit(eventemitter.AddressAuthorized, evt)

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
