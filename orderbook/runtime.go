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

package orderbook

import (
	"errors"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/types"
	"strconv"
)

const (
	DEFAULT_BLOCKER_NUMBER_FIELD = "defaultBlockerNumber"
	TX_INDEX_TABLE_PREFIX        = "index"
	TX_CONTENT_TABLE_PREFIX      = "log"
)

// db根据events/topic保存两张表，一张存储块高度，一张存储内容
// (为了保险起见，我们还可以将原始的tx数据保存一份)
// 这些表用于存储kv blockNumber&transaction链上过程数据
// 每次根据listener根据blockNumber生成filterId,进而监听events
// 每张表表名为event_blockNumber
// 每张表包含几个固定字段:blockNumber,createTime,updateTime

// RuntimeDataReady create tables for topics
func (ob *OrderBook) RuntimeDataReady() {
	ob.runtimeTables = make(map[string]db.Database)
	for _, topic := range ob.commOpts.FilterTopics {
		idx := getTableName(topic, TX_INDEX_TABLE_PREFIX)
		con := getTableName(topic, TX_CONTENT_TABLE_PREFIX)
		ob.runtimeTables[idx] = db.NewTable(ob.db, idx)
		ob.runtimeTables[con] = db.NewTable(ob.db, con)
	}
}

// SetBlockNumber record recent block number
func (ob *OrderBook) SetBlockNumber(topic string, height int) {
	ob.db.Put(defaultBlockNumKey(), int2bytes(height))
}

// GetBlockNumber get recent block number from db or toml config
func (ob *OrderBook) GetBlockNumber() int {
	println("-----tst1")
	println(string(defaultBlockNumKey()))
	bs, err := ob.db.Get(defaultBlockNumKey())
	println("-----tst2")
	num := bytes2int(bs)
	if err != nil {
		num = ob.commOpts.DefaultBlockNumber
		ob.db.Put(defaultBlockNumKey(), int2bytes(num))
	}

	return num
}

// todo(fk):filter transaction,return bool/false ,if do not exist setTransaction
func (ob *OrderBook) FilterTransaction(topic string, height int, tx types.Hash, data []byte) bool {
	return true
}

// SetTransaction set transaction
func (ob *OrderBook) SetTransaction(topic string, height int, tx types.Hash, data []byte) error {
	oldHeight := ob.GetBlockNumber()
	currentHeight := height

	idxtn := getTableName(topic, TX_INDEX_TABLE_PREFIX)
	contn := getTableName(topic, TX_CONTENT_TABLE_PREFIX)

	ob.lock.Lock()
	defer ob.lock.Unlock()

	idxtb, idxok := ob.runtimeTables[idxtn]
	contb, conok := ob.runtimeTables[contn]

	if !idxok {
		return errors.New("table for " + topic + "'s index do not exist")
	}
	if !conok {
		return errors.New("table for " + topic + "'s content do not exist")
	}

	idxtb.Put(tx.Bytes(), int2bytes(currentHeight))
	contb.Put(tx.Bytes(), data)

	if oldHeight != currentHeight {
		ob.SetBlockNumber(topic, currentHeight)
	}

	return nil
}

func (ob *OrderBook) GetTransaction(topic string, tx types.Hash) ([]byte, int, error) {
	ob.lock.RLock()
	defer ob.lock.RUnlock()

	idxtn := getTableName(topic, TX_INDEX_TABLE_PREFIX)
	contn := getTableName(topic, TX_CONTENT_TABLE_PREFIX)

	idxtb, idxok := ob.runtimeTables[idxtn]
	contb, conok := ob.runtimeTables[contn]

	if !idxok {
		return nil, 0, errors.New("table for " + topic + "'s index do not exist")
	}
	if !conok {
		return nil, 0, errors.New("table for " + topic + "'s content do not exist")
	}

	height, idxerr := idxtb.Get(tx.Bytes())
	if idxerr != nil {
		return nil, 0, errors.New(tx.Str() + " index do not exist")
	}
	content, conerr := contb.Get(tx.Bytes())
	if conerr != nil {
		return nil, bytes2int(height), errors.New(tx.Str() + " content do not exist")
	}

	return content, bytes2int(height), nil
}

func getTableName(topic, prefix string) string { return topic + "_" + prefix }
func defaultBlockNumKey() []byte               { return []byte(DEFAULT_BLOCKER_NUMBER_FIELD) }

func int2bytes(num int) []byte {
	str := strconv.Itoa(num)
	return []byte(str)
}

func bytes2int(bs []byte) int {
	num, _ := strconv.Atoi(string(bs))
	return num
}
