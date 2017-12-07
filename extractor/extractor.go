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
	"encoding/json"
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
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

type ExtractorService interface {
	Start()
	Stop()
	Restart()
}

// TODO(fukun):不同的channel，应当交给orderbook统一进行后续处理，可以将channel作为函数返回值、全局变量、参数等方式
type ExtractorServiceImpl struct {
	options   config.AccessorOptions
	commOpts  config.CommonOptions
	accessor  *ethaccessor.EthNodeAccessor
	dao       dao.RdsService
	stop      chan struct{}
	lock      sync.RWMutex
	events    map[common.Hash]ContractData
	protocols map[common.Address]string
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

	log.Info("extractor start...")
	start, end := l.getBlockNumberRange()
	iterator := l.accessor.BlockIterator(start, end, true, uint64(0))

	go func() {
		for {
			inter, err := iterator.Next()
			if err != nil {
				log.Fatalf("extractor,iterator next error:%s", err.Error())
			}

			block := inter.(*ethaccessor.BlockWithTxObject)
			log.Debugf("extractor,get block:%s->%s", block.Number.BigInt().String(), block.Hash.Hex())

			currentBlock := &types.Block{}
			currentBlock.BlockNumber = block.Number.BigInt()
			currentBlock.ParentHash = block.ParentHash
			currentBlock.BlockHash = block.Hash
			currentBlock.CreateTime = block.Timestamp.Int64()

			blockEvent := &types.BlockEvent{}
			blockEvent.BlockNumber = block.Number.BigInt()
			blockEvent.BlockHash = block.Hash
			eventemitter.Emit(eventemitter.Block_New, blockEvent)

			var entity dao.Block
			if err := entity.ConvertDown(currentBlock); err != nil {
				log.Debugf("extractor,convert block to dao/entity error:%s", err.Error())
			} else {
				l.dao.Add(&entity)
			}

			txcnt := len(block.Transactions)
			if txcnt < 1 {
				log.Infof("extractor,get none block transaction")
				continue
			} else {
				log.Infof("extractor,get block transaction list length %d", txcnt)
			}

			if err := l.detectFork(currentBlock); err != nil {
				log.Debugf("extractor,detect fork error:%s", err.Error())
				continue
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
		log.Debugf("extractor,get transaction hash:%s", tx.Hash)

		// 解析method，获得ring内等orders并发送到orderbook保存
		l.doMethod(tx.Input)

		// 获取transaction内所有event logs
		var receipt ethaccessor.TransactionReceipt
		if err := l.accessor.Call(&receipt, "eth_getTransactionReceipt", tx.Hash); err != nil {
			log.Errorf("extractor,get transaction receipt error:%s", err.Error())
			continue
		}

		// todo:判断transactionRecipient.to地址是否是合约地址
		for _, evtLog := range receipt.Logs {
			var (
				contract ContractData
				ok       bool
			)

			// 过滤合约
			protocolAddr := common.HexToAddress(evtLog.Address)
			if _, ok := l.protocols[protocolAddr]; !ok {
				log.Debugf("extractor, unsupported protocol %s", protocolAddr.Hex())
				continue
			}

			// 过滤事件
			data := hexutil.MustDecode(evtLog.Data)
			id := common.HexToHash(evtLog.Topics[0])
			if contract, ok = l.events[id]; !ok {
				log.Debugf("extractor,contract event id error:%s", id)
				continue
			}

			if l.commOpts.SaveEventLog {
				if bs, err := json.Marshal(evtLog); err != nil {
					el := &dao.EventLog{}
					el.Protocol = evtLog.Address
					el.TxHash = tx.Hash
					el.BlockNumber = evtLog.BlockNumber.Int64()
					el.CreateTime = block.Timestamp.Int64()
					el.Data = bs
					l.dao.Add(el)
				}
			}

			if nil != data && len(data) > 0 {
				// 解析事件
				if err := contract.CAbi.Unpack(contract.Event, contract.Name, data, abi.SEL_UNPACK_EVENT); nil != err {
					log.Errorf("extractor,unpack event error:%s", err.Error())
					continue
				}
			}

			contract.Topics = evtLog.Topics
			contract.BlockNumber = evtLog.BlockNumber.BigInt()
			contract.Time = block.Timestamp.BigInt()
			contract.ContractAddress = evtLog.Address
			contract.TxHash = tx.Hash

			eventemitter.Emit(contract.Id.Hex(), contract)
		}
	}
}

func (l *ExtractorServiceImpl) doMethod(input string) error {
	return nil
}

// 只需要解析submitRing,cancel，cutoff这些方法在event里，如果方法不成功也不用执行后续逻辑
func (l *ExtractorServiceImpl) handleSubmitRingMethod(input eventemitter.EventData) error {
	// todo: unpack method
	return nil
}

func (l *ExtractorServiceImpl) handleRingMinedEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return fmt.Errorf("extractor,ring mined event indexed fields number error")
	}

	contractEvent := contractData.Event.(*ethaccessor.RingMinedEvent)
	contractEvent.RingHash = common.HexToHash(contractData.Topics[1])

	ringmined, fills, err := contractEvent.ConvertDown()
	if err != nil {
		return err
	}
	ringmined.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	ringmined.TxHash = common.HexToHash(contractData.TxHash)
	ringmined.Time = contractData.Time
	ringmined.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,ring mined event,ringhash:%s, ringIndex:%s, miner:%s, feeRecipient:%s,isRinghashReserved:%t",
			ringmined.Ringhash.Hex(),
			ringmined.RingIndex.String(),
			ringmined.Miner.Hex(),
			ringmined.FeeRecipient.Hex(),
			ringmined.IsRinghashReserved)
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorRingMined, ringmined)

	var (
		fillList      []*types.OrderFilledEvent
		orderhashList []string
	)
	for _, fill := range fills {
		fill.TxHash = common.HexToHash(contractData.TxHash)
		fill.ContractAddress = common.HexToAddress(contractData.ContractAddress)
		fill.Time = contractData.Time
		fill.Blocknumber = contractData.BlockNumber

		if l.commOpts.Develop {
			log.Debugf("extractor,order filled event,ringhash:%s, amountS:%s, amountB:%s, orderhash:%s, lrcFee:%s, lrcReward:%s, nextOrderhash:%s, preOrderhash:%s, ringIndex:%s",
				fill.Ringhash.Hex(),
				fill.AmountS.String(),
				fill.AmountB.String(),
				fill.OrderHash.Hex(),
				fill.LrcFee.String(),
				fill.LrcReward.String(),
				fill.NextOrderHash.Hex(),
				fill.PreOrderHash.Hex(),
				fill.RingIndex.String(),
			)
		}

		fillList = append(fillList, fill)
		orderhashList = append(orderhashList, fill.OrderHash.Hex())
	}

	ordermap, err := l.dao.GetOrdersByHash(orderhashList)
	if err != nil {
		return err
	}
	for _, v := range fillList {
		if ord, ok := ordermap[v.OrderHash.Hex()]; ok {
			v.TokenS = common.HexToAddress(ord.TokenS)
			v.TokenB = common.HexToAddress(ord.TokenB)
			v.Owner = common.HexToAddress(ord.Owner)
			v.Market, _ = util.WrapMarketByAddress(v.TokenS.Hex(), v.TokenB.Hex())

			eventemitter.Emit(eventemitter.OrderManagerExtractorFill, v)
		} else {
			log.Debugf("extractor,order filled event cann't match order %s", ord.OrderHash)
		}
	}

	return nil
}

func (l *ExtractorServiceImpl) handleOrderCancelledEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return fmt.Errorf("extractor,order cancelled event indexed fields number error")
	}

	contractEvent := contractData.Event.(*ethaccessor.OrderCancelledEvent)
	contractEvent.OrderHash = common.HexToHash(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.TxHash = common.HexToHash(contractData.TxHash)
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,order cancelled event,orderhash:%s, cancelAmount:%s", evt.OrderHash.Hex(), evt.AmountCancelled.String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorCancel, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleCutoffTimestampEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return fmt.Errorf("extractor,cutoff timestamp changed event indexed fields number error")
	}

	contractEvent := contractData.Event.(*ethaccessor.CutoffTimestampChangedEvent)
	contractEvent.Owner = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.TxHash = common.HexToHash(contractData.TxHash)
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,cutoffTimestampChanged event,ownerAddress:%s, cutOffTime:%s -> %s", evt.Owner.Hex(), evt.Cutoff.String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorCutoff, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleTransferEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	if len(contractData.Topics) < 3 {
		return fmt.Errorf("extractor,token transfer event indexed fields number error")
	}

	contractEvent := contractData.Event.(*ethaccessor.TransferEvent)
	contractEvent.From = common.HexToAddress(contractData.Topics[1])
	contractEvent.To = common.HexToAddress(contractData.Topics[2])

	evt := contractEvent.ConvertDown()
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,transfer event,from:%s, to:%s, value:%s", evt.From.Hex(), evt.To.Hex(), evt.Value.String())
	}

	eventemitter.Emit(eventemitter.AccountTransfer, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleApprovalEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	if len(contractData.Topics) < 3 {
		return fmt.Errorf("extractor,token approval event indexed fields number error")
	}

	contractEvent := contractData.Event.(*ethaccessor.ApprovalEvent)
	contractEvent.Owner = common.HexToAddress(contractData.Topics[1])
	contractEvent.Spender = common.HexToAddress(contractData.Topics[2])

	evt := contractEvent.ConvertDown()
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,approval event,owner:%s, spender:%s, value:%s", evt.Owner.Hex(), evt.Spender.Hex(), evt.Value.String())
	}

	eventemitter.Emit(eventemitter.AccountApproval, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleTokenRegisteredEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	contractEvent := contractData.Event.(*ethaccessor.TokenRegisteredEvent)

	evt := contractEvent.ConvertDown()
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,token registered event,address:%s, symbol:%s", evt.Token.Hex(), evt.Symbol)
	}

	eventemitter.Emit(eventemitter.TokenRegistered, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleTokenUnRegisteredEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	contractEvent := contractData.Event.(*ethaccessor.TokenUnRegisteredEvent)

	evt := contractEvent.ConvertDown()
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,token unregistered event,address:%s, symbol:%s", evt.Token.Hex(), evt.Symbol)
	}

	eventemitter.Emit(eventemitter.TokenUnRegistered, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleRinghashSubmitEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	if len(contractData.Topics) < 3 {
		return fmt.Errorf("extractor,ringhash registered event indexed fields number error")
	}

	contractEvent := contractData.Event.(*ethaccessor.RingHashSubmittedEvent)
	contractEvent.RingMiner = common.HexToAddress(contractData.Topics[1])
	contractEvent.RingHash = common.HexToHash(contractData.Topics[2])

	evt := contractEvent.ConvertDown()
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber
	evt.TxHash = common.HexToHash(contractData.TxHash)

	if l.commOpts.Develop {
		log.Debugf("extractor,ringhash submit event,ringhash:%s, ringMiner:%s", evt.RingHash.Hex(), evt.RingMiner.Hex())
	}

	eventemitter.Emit(eventemitter.RingHashSubmitted, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleAddressAuthorizedEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return fmt.Errorf("extractor,address authorized event indexed fields number error")
	}

	contractEvent := contractData.Event.(*ethaccessor.AddressAuthorizedEvent)
	contractEvent.ContractAddress = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,address authorized event address:%s, number:%d", evt.Protocol.Hex(), evt.Number)
	}

	eventemitter.Emit(eventemitter.AddressAuthorized, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleAddressDeAuthorizedEvent(input eventemitter.EventData) error {
	contractData := input.(ContractData)
	if len(contractData.Topics) < 2 {
		return fmt.Errorf("extractor,address deauthorized event indexed fields number error")
	}

	contractEvent := contractData.Event.(*ethaccessor.AddressDeAuthorizedEvent)
	contractEvent.ContractAddress = common.HexToAddress(contractData.Topics[1])

	evt := contractEvent.ConvertDown()
	evt.ContractAddress = common.HexToAddress(contractData.ContractAddress)
	evt.Time = contractData.Time
	evt.Blocknumber = contractData.BlockNumber

	if l.commOpts.Develop {
		log.Debugf("extractor,address deauthorized event,address:%s, number:%d", evt.Protocol.Hex(), evt.Number)
	}

	eventemitter.Emit(eventemitter.AddressAuthorized, evt)

	return nil
}

// todo: modify
func (l *ExtractorServiceImpl) getBlockNumberRange() (*big.Int, *big.Int) {
	var ret types.Block

	start := l.commOpts.DefaultBlockNumber
	end := l.commOpts.EndBlockNumber

	// 寻找分叉块，并归零分叉标记
	forkBlock, err := l.dao.FindForkBlock()
	if err == nil {
		forkBlock.ConvertUp(&ret)
		forkBlock.Fork = false
		l.dao.Update(forkBlock)
		return ret.BlockNumber, end
	}

	// 寻找最新块
	latestBlock, err := l.dao.FindLatestBlock()
	if err != nil {
		log.Debugf("extractor,get latest block number error:%s", err.Error())
		return start, end
	}
	if err := latestBlock.ConvertUp(&ret); err != nil {
		log.Fatalf("extractor,get blocknumber range convert up error:%s", err.Error())
	}

	return ret.BlockNumber, end
}
