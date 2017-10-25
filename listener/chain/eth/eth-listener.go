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
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"sync"
)

/**
区块链的listener, 得到order以及ring的事件，
*/

const (
	BLOCK_HASH_TABLE_NAME       = "block_hash_table"
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
	txhashTable    db.Database
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

func (l *EthClientListener) Start() {
	l.stop = make(chan struct{})

	log.Info("eth listener start...")

	start, end := l.getBlockNumberRange()

	go func() {
		iterator := l.ethClient.BlockIterator(start, end)
		for {
			inter, err := iterator.Next()
			if err != nil {
				log.Fatalf("eth listener iterator next error:%s", err.Error())
			}

			block := inter.(eth.BlockWithTxObject)
			log.Debugf("eth listener get block:%s->%s", block.Number.BigInt().String(), block.Hash.Hex())

			txcnt := len(block.Transactions)
			if txcnt < 1 {
				log.Debugf("eth listener get none block transaction")
				continue
			} else {
				log.Infof("eth listener get block transaction list length %d", txcnt)
			}

			if err := l.saveBlock(block); err != nil {
				log.Errorf("eth listener save block hash error:%s", err.Error())
				continue
			}

			if l.commOpts.Develop {
				if idx, err := l.getTransactions(block.Hash); err != nil {
					for _, v := range idx.Txs {
						log.Debugf("eth listener block transaction %s", v.Hex())
					}
				}
			}

			l.doBlock(block)
		}
	}()
}

func (l *EthClientListener) doBlock(block eth.BlockWithTxObject) {
	txs := []types.Hash{}

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

		// 解析event,并发送到orderbook
		var receipt eth.TransactionReceipt
		err := l.ethClient.GetTransactionReceipt(&receipt, tx.Hash)
		if err != nil {
			log.Errorf("eth listener get transaction receipt error:%s", err.Error())
			continue
		}

		log.Debugf("transaction receipt  event logs number:%d", len(receipt.Logs))

		contractAddr := types.HexToAddress(receipt.To)
		for _, v := range receipt.Logs {
			if err := l.doEvent(v, contractAddr); err != nil {
				log.Errorf("eth listener do event error:%s", err.Error())
			} else {
				txhash := types.HexToHash(tx.Hash)
				txs = append(txs, txhash)
			}
		}
	}

	if err := l.saveTransactions(block.Hash, txs); err != nil {
		log.Errorf("eth listener save transactions error:%s", err.Error())
	}
}

// 只需要解析submitRing,cancel，cutoff这些方法在event里，如果方法不成功也不用执行后续逻辑
func (l *EthClientListener) doMethod(input string) {
	// todo: unpack method
	// input := tx.Input
	// l.ethClient
}

// 解析相关事件
// todo(fuk): how to process approval,transfer,version add/remove...
func (l *EthClientListener) doEvent(v eth.Log, to types.Address) error {
	impl, ok := miner.LoopringInstance.LoopringImpls[to]
	if !ok {
		return errors.New("eth listener do event contract address do not exsit")
	}

	topic := v.Topics[0]
	data := hexutil.MustDecode(v.Data)

	// todo:delete after test
	log.Debugf("eth listener log data:%s", v.Data)
	log.Debugf("eth listener log topic:%s", topic)
	log.Debugf("impl order filled id:%s", impl.OrderFilledEvent.Id())

	switch topic {
	case impl.OrderFilledEvent.Id():
		evt := chainclient.OrderFilledEvent{}
		log.Debugf("eth listener event order filled")
		if err := impl.OrderFilledEvent.Unpack(&evt, data, v.Topics); err != nil {
			return err
		}

		if l.commOpts.Develop {
			log.Debugf("eth listener order filled event ringhash -> %s", types.Bytes2Hex(evt.Ringhash))
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

		evt.ConvertDown(ord)
		l.whisper.ChainOrderChan <- ord

	case impl.OrderCancelledEvent.Id():
		log.Debugf("eth listener event order cancelled")
		evt := chainclient.OrderCancelledEvent{}
		if err := impl.OrderCancelledEvent.Unpack(&evt, data, v.Topics); err != nil {
			return err
		}

		hash := types.BytesToHash(evt.OrderHash)
		ord, err := l.ob.GetOrder(hash)
		if err != nil {
			return err
		}

		evt.ConvertDown(ord)
		l.whisper.ChainOrderChan <- ord

	case impl.CutoffTimestampChangedEvent.Id():

	default:
		//log.Errorf("event id %s not found", topic)
	}

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
