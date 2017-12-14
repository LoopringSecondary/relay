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
	options      config.AccessorOptions
	commOpts     config.CommonOptions
	accessor     *ethaccessor.EthNodeAccessor
	dao          dao.RdsService
	stop         chan struct{}
	lock         sync.RWMutex
	events       map[common.Hash]EventData
	methods      map[string]MethodData
	protocols    map[common.Address]string
	syncComplete bool
}

func NewExtractorService(options config.AccessorOptions,
	commonOpts config.CommonOptions,
	accessor *ethaccessor.EthNodeAccessor,
	rds dao.RdsService) *ExtractorServiceImpl {
	var l ExtractorServiceImpl

	l.commOpts = commonOpts
	l.accessor = accessor
	l.dao = rds
	l.syncComplete = false

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

			// get current block
			block := inter.(*ethaccessor.BlockWithTxObject)
			log.Debugf("extractor,get block:%s->%s", block.Number.BigInt().String(), block.Hash.Hex())

			currentBlock := &types.Block{}
			currentBlock.BlockNumber = block.Number.BigInt()
			currentBlock.ParentHash = block.ParentHash
			currentBlock.BlockHash = block.Hash
			currentBlock.CreateTime = block.Timestamp.Int64()

			// sync chain block number
			if l.syncComplete == false {
				var syncBlock types.Big
				if err := l.accessor.Call(&syncBlock, "eth_blockNumber"); err != nil {
					log.Fatalf("extractor,sync chain block,get ethereum node current block number error:%s", err.Error())
				}
				if syncBlock.BigInt().Cmp(currentBlock.BlockNumber) <= 0 && l.syncComplete == false {
					eventemitter.Emit(eventemitter.SyncChainComplete, syncBlock)
					l.syncComplete = true
					log.Debugf("extractor,sync chain block complete!")
				} else {
					log.Debugf("extractor,chain block syncing... ")
				}
			}

			// emit new block
			blockEvent := &types.BlockEvent{}
			blockEvent.BlockNumber = block.Number.BigInt()
			blockEvent.BlockHash = block.Hash
			eventemitter.Emit(eventemitter.Block_New, blockEvent)

			// convert block to dao entity
			var entity dao.Block
			if err := entity.ConvertDown(currentBlock); err != nil {
				log.Debugf("extractor,convert block to dao/entity error:%s", err.Error())
			} else {
				l.dao.Add(&entity)
			}

			// base filter
			txcnt := len(block.Transactions)
			if txcnt < 1 {
				log.Debugf("extractor,get none block transaction")
				continue
			} else {
				log.Debugf("extractor,get block transaction list length %d", txcnt)
			}

			// detect chain fork
			if err := l.detectFork(currentBlock); err != nil {
				log.Debugf("extractor,detect fork error:%s", err.Error())
				continue
			}

			// process block
			for _, tx := range block.Transactions {
				log.Debugf("extractor,get transaction hash:%s", tx.Hash)

				_, err := l.processEvent(&tx, block.Timestamp.BigInt())
				if err != nil {
					log.Errorf(err.Error())
				}

				// 解析method，获得ring内等orders并发送到orderbook保存
				//if err := l.processMethod(tx.Hash, block.Timestamp.BigInt(), block.Number.BigInt(), logAmount); err != nil {
				//	log.Errorf(err.Error())
				//}
			}
		}
	}()
}

func (l *ExtractorServiceImpl) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.syncComplete = false
	close(l.stop)
}

// 重启(分叉)时先关停subscribeEvents，然后关
func (l *ExtractorServiceImpl) Restart() {
	l.Stop()
	time.Sleep(1 * time.Second)
	l.Start()
}

func (l *ExtractorServiceImpl) processMethod(txhash string, time, blockNumber *big.Int, logAmount int) error {
	var tx ethaccessor.Transaction
	if err := l.accessor.Call(&tx, "eth_getTransactionByHash", txhash); err != nil {
		return fmt.Errorf("extractor,get transaction error:%s", err.Error())
	}

	input := common.FromHex(tx.Input)
	var (
		contract MethodData
		ok       bool
	)

	// 过滤方法
	if len(input) < 4 || len(tx.Input) < 10 {
		return fmt.Errorf("extractor,contract method id %s length invalid", common.ToHex(input))
	}
	id := common.ToHex(input[0:4])
	if contract, ok = l.methods[id]; !ok {
		return fmt.Errorf("extractor,contract method id error:%s", id)
	}

	contract.BlockNumber = tx.BlockNumber.BigInt()
	contract.Time = time
	contract.ContractAddress = tx.To
	contract.From = tx.From
	contract.To = tx.To
	contract.TxHash = tx.Hash
	contract.Value = tx.Value.BigInt()
	contract.BlockNumber = blockNumber
	contract.Input = tx.Input
	contract.Gas = tx.Gas.BigInt()
	contract.GasPrice = tx.Gas.BigInt()
	contract.LogAmount = logAmount

	eventemitter.Emit(contract.Id, contract)
	return nil
}

func (l *ExtractorServiceImpl) processEvent(tx *ethaccessor.Transaction, time *big.Int) (int, error) {
	var receipt ethaccessor.TransactionReceipt

	if err := l.accessor.Call(&receipt, "eth_getTransactionReceipt", tx.Hash); err != nil {
		return 0, fmt.Errorf("extractor,get transaction receipt error:%s", err.Error())
	}

	if len(receipt.Logs) == 0 {
		log.Debugf("extractor,transaction %s recipient do not have any logs", tx.Hash)
		return 0, nil
	}

	for _, evtLog := range receipt.Logs {
		var (
			contract EventData
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
			log.Debugf("extractor,contract event id error:%s", id.Hex())
			continue
		}

		// 记录event log
		if l.commOpts.SaveEventLog {
			if bs, err := json.Marshal(evtLog); err != nil {
				log.Debugf("extractor,json unmarshal evtlog error:%s", err.Error())
			} else {
				el := &dao.EventLog{}
				el.Protocol = evtLog.Address
				el.TxHash = tx.Hash
				el.BlockNumber = evtLog.BlockNumber.Int64()
				el.CreateTime = time.Int64()
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
		contract.Time = time
		contract.ContractAddress = evtLog.Address
		contract.TxHash = tx.Hash

		eventemitter.Emit(contract.Id.Hex(), contract)
	}

	return len(receipt.Logs), nil
}

// 只需要解析submitRing,cancel，cutoff这些方法在event里，如果方法不成功也不用执行后续逻辑
func (l *ExtractorServiceImpl) handleSubmitRingMethod(input eventemitter.EventData) error {
	contract := input.(MethodData)

	// emit to miner
	var evt types.SubmitRingMethodEvent
	evt.TxHash = common.HexToHash(contract.TxHash)
	evt.UsedGas = contract.Gas
	evt.UsedGasPrice = contract.GasPrice
	evt.Err = contract.IsValid()
	eventemitter.Emit(eventemitter.Miner_SubmitRing_Method, &evt)

	ring := contract.Method.(*ethaccessor.SubmitRingMethod)
	ring.Protocol = common.HexToAddress(contract.To)

	data := hexutil.MustDecode("0x" + contract.Input[10:])
	if err := contract.CAbi.UnpackMethodInput(ring, contract.Name, data); err != nil {
		return fmt.Errorf("extractor,submitRing method,unpack error:%s", err.Error())
	}
	orderList, err := ring.ConvertDown()
	if err != nil {
		return fmt.Errorf("extractor,submitRing method,convert order data error:%s", err.Error())
	}

	for _, v := range orderList {
		if l.commOpts.Develop {
			log.Debugf("extractor,submitRing method,order,protocol:%s,owner:%s,tokenS:%s,tokenB:%s,"+
				"amountS:%s,amountB:%s,timestamp:%s,lrcFee:%s,marginSplit:%d,buyNoMoreThanBAmount:%t",
				v.Protocol.Hex(), v.Owner.Hex(), v.TokenS.Hex(), v.TokenB.Hex(),
				v.AmountS.String(), v.AmountB.String(), v.Timestamp.String(), v.LrcFee.String(), v.MarginSplitPercentage, v.BuyNoMoreThanAmountB)
		}
		v.Protocol = common.HexToAddress(contract.ContractAddress)
		eventemitter.Emit(eventemitter.Gateway, v)
	}

	return nil
}

func (l *ExtractorServiceImpl) handleSubmitRingHashMethod(input eventemitter.EventData) error {
	contract := input.(MethodData)
	method := contract.Method.(*ethaccessor.SubmitRingHashMethod)

	data := hexutil.MustDecode("0x" + contract.Input[10:])
	if err := contract.CAbi.UnpackMethodInput(method, contract.Name, data); err != nil {
		return fmt.Errorf("extractor,submitRingHash method,unpack error:%s", err.Error())
	}
	evt, err := method.ConvertDown()
	if err != nil {
		return fmt.Errorf("extractor,submitRingHash method,convert order data error:%s", err.Error())
	}

	evt.TxHash = common.HexToHash(contract.TxHash)
	evt.UsedGas = contract.Gas
	evt.UsedGasPrice = contract.GasPrice
	evt.Err = contract.IsValid()

	eventemitter.Emit(eventemitter.Miner_SubmitRingHash_Method, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleBatchSubmitRingHashMethod(input eventemitter.EventData) error {
	contract := input.(MethodData)
	method := contract.Method.(*ethaccessor.BatchSubmitRingHashMethod)

	data := hexutil.MustDecode("0x" + contract.Input[10:])
	if err := contract.CAbi.UnpackMethodInput(method, contract.Name, data); err != nil {
		return fmt.Errorf("extractor,batchSubmitRingHash method,unpack error:%s", err.Error())
	}
	evt, err := method.ConvertDown()
	if err != nil {
		return fmt.Errorf("extractor,batchSubmitRingHash method,convert order data error:%s", err.Error())
	}

	evt.TxHash = common.HexToHash(contract.TxHash)
	evt.UsedGas = contract.Gas
	evt.UsedGasPrice = contract.GasPrice
	evt.Err = contract.IsValid()

	eventemitter.Emit(eventemitter.Miner_SubmitRingHash_Method, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleCancelOrderMethod(input eventemitter.EventData) error {
	contract := input.(MethodData)
	cancel := contract.Method.(*ethaccessor.CancelOrderMethod)

	data := hexutil.MustDecode("0x" + contract.Input[10:])
	if err := contract.CAbi.UnpackMethodInput(cancel, contract.Name, data); err != nil {
		return fmt.Errorf("extractor,cancelOrder method,unpack error:%s", err.Error())
	}

	order, err := cancel.ConvertDown()
	if err != nil {
		return fmt.Errorf("extractor,cancelOrder method,convert order data error:%s", err.Error())
	}

	if l.commOpts.Develop {
		log.Debugf("extractor,cancelOrder method,order tokenS:%s,tokenB:%s,amountS:%s,amountB:%s", order.TokenS.Hex(), order.TokenB.Hex(), order.AmountS.String(), order.AmountB.String())
	}

	order.Protocol = common.HexToAddress(contract.ContractAddress)
	eventemitter.Emit(eventemitter.Gateway, order)

	return nil
}

func (l *ExtractorServiceImpl) handleWethDepositMethod(input eventemitter.EventData) error {
	contractData := input.(MethodData)

	var deposit types.WethDepositMethodEvent
	deposit.From = common.HexToAddress(contractData.From)
	deposit.To = common.HexToAddress(contractData.To)
	deposit.Value = contractData.Value
	deposit.Time = contractData.Time
	deposit.Blocknumber = contractData.BlockNumber
	deposit.TxHash = common.HexToHash(contractData.TxHash)
	deposit.ContractAddress = common.HexToAddress(contractData.ContractAddress)

	if l.commOpts.Develop {
		log.Debugf("extractor,wethDeposit method,from:%s, to:%s, value:%s", deposit.From.Hex(), deposit.To.Hex(), deposit.Value.String())
	}

	eventemitter.Emit(eventemitter.WethDepositMethod, deposit)
	return nil
}

func (l *ExtractorServiceImpl) handleWethWithdrawalMethod(input eventemitter.EventData) error {
	contractData := input.(MethodData)
	contractMethod := contractData.Method.(*ethaccessor.WethWithdrawalMethod)

	data := hexutil.MustDecode("0x" + contractData.Input[10:])
	if err := contractData.CAbi.UnpackMethodInput(contractMethod, contractData.Name, data); err != nil {
		return fmt.Errorf("extractor,wethWithdrawal method,unpack error:%s", err.Error())
	}

	withdrawal := contractMethod.ConvertDown()
	withdrawal.From = common.HexToAddress(contractData.From)
	withdrawal.To = common.HexToAddress(contractData.To)
	withdrawal.Time = contractData.Time
	withdrawal.Blocknumber = contractData.BlockNumber
	withdrawal.TxHash = common.HexToHash(contractData.TxHash)
	withdrawal.ContractAddress = common.HexToAddress(contractData.ContractAddress)

	if l.commOpts.Develop {
		log.Debugf("extractor,wethWithdrawal method,from:%s, to:%s, value:%s", withdrawal.From.Hex(), withdrawal.To.Hex(), withdrawal.Value.String())
	}

	eventemitter.Emit(eventemitter.WethWithdrawalMethod, withdrawal)
	return nil
}

func (l *ExtractorServiceImpl) handleRingMinedEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)
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
		fill.Market, _ = util.WrapMarketByAddress(fill.TokenS.Hex(), fill.TokenB.Hex())

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
	contractData := input.(EventData)
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
	contractData := input.(EventData)
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
		log.Debugf("extractor,cutoffTimestampChanged event,ownerAddress:%s, cutOffTime:%s", evt.Owner.Hex(), evt.Cutoff.String())
	}

	eventemitter.Emit(eventemitter.OrderManagerExtractorCutoff, evt)

	return nil
}

func (l *ExtractorServiceImpl) handleTransferEvent(input eventemitter.EventData) error {
	contractData := input.(EventData)

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
	contractData := input.(EventData)
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
	contractData := input.(EventData)
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
	contractData := input.(EventData)
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
	contractData := input.(EventData)
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
	contractData := input.(EventData)
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
	contractData := input.(EventData)
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
		blockHash := common.HexToHash(forkBlock.BlockHash)
		l.dao.SetForkBlock(blockHash)
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
