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

package orderbook_test

import (
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/orderbook"
	"github.com/Loopring/ringminer/types"
	"os"
	"path/filepath"
	"testing"
)

const (
	dbname = "leveldb"
	topic1 = "OrderFilled"
	topic2 = "RingMined"
)

var sep = func() string { return string(filepath.Separator) }

func file() string {
	gopath := os.Getenv("GOPATH")
	proj := "github.com/Loopring/ringminer"
	return gopath + sep() + "src" + sep() + proj + sep() + dbname
}

func getOrderBook() *orderbook.OrderBook {
	database := db.NewDB(config.DbOptions{DataDir: file(), BufferCapacity: 8, CacheCapacity: 4})
	opts := config.OrderBookOptions{FilterTopics: []string{topic1, topic2}, DefaultBlockNumber: 100}

	peerOrderChan := make(chan *types.Order)
	chainOrderChan := make(chan *types.OrderMined)
	engineOrderChan := make(chan *types.OrderState)
	whisper := &orderbook.Whisper{PeerOrderChan: peerOrderChan, EngineOrderChan: engineOrderChan, ChainOrderChan: chainOrderChan}
	ob := orderbook.NewOrderBook(opts, database, whisper)

	ob.RuntimeDataReady()
	return ob
}

func TestOrderBook_GetBlockNumber(t *testing.T) {
	ob := getOrderBook()
	t.Log(ob.GetBlockNumber())
}

func TestOrderBook_SetBlockNumber(t *testing.T) {
	ob := getOrderBook()
	ob.SetBlockNumber(topic1, 200)
	num := ob.GetBlockNumber()
	t.Log("height:", num)
}

func TestOrderBook_SetTransaction(t *testing.T) {
	ob := getOrderBook()
	height := ob.GetBlockNumber()
	tx := types.HexToHash("0x4f07194d6601954c8de8cec792f1702ed29b61d6f4d8bf3587f113abbce544f2")
	data := []byte("222")
	ob.SetTransaction(topic1, height, tx, data)

	data, height, err := ob.GetTransaction(topic1, tx)
	if err != nil {
		t.Error(err)
	} else {
		t.Log("data:", string(data))
		t.Log("height:", height)
	}
}

//func getOrderWrap() *types.OrderWrap {
//	var (
//		ord types.Order
//		odw types.OrderWrap
//	)
//
//	ord.Id = types.StringToHash("test1")
//	ord.Protocol = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.Owner = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.OutToken = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.InToken = types.StringToAddress("0xb794f5ea0ba39494ce839613fffba74279579268")
//	ord.OutAmount = types.IntToBig(20000)
//	ord.InAmount = types.IntToBig(800)
//	ord.Expiration = uint64(time.Now().Unix())
//	ord.Fee = types.IntToBig(30)
//	ord.SavingShare = types.IntToBig(51)
//	ord.V = 8
//	ord.R = types.StringToSign("hhhhhhhh")
//	ord.S = types.StringToSign("fjalskdf")
//
//	odw.RawOrder = &ord
//	odw.InAmount = types.IntToBig(400)
//	odw.OutAmount = types.IntToBig(10000)
//	odw.Fee = types.IntToBig(15)
//	odw.PeerId = "Qme85LtECPhvx4Px5i7s2Ht2dXdHrgXYpqkDsKvxdpFQP4"
//
//	return &odw
//}
//
//func TestNewOrder(t *testing.T) {
//	conf := &orderbook.OrderBookConfig{DBName: file(), DBCacheCapcity: 12, DBBufferCapcity: 12}
//	orderbook.InitializeOrderBook(conf)
//	odw := getOrderWrap()
//	orderbook.NewOrder(odw)
//}
//
//func TestGetOrder(t *testing.T) {
//	conf := &orderbook.OrderBookConfig{DBName: file(), DBCacheCapcity: 12, DBBufferCapcity: 12}
//	orderbook.InitializeOrderBook(conf)
//
//	if w, err := orderbook.GetOrder(types.StringToHash("test1")); err != nil {
//		t.Log(err.Error())
//	} else {
//		t.Log(w.RawOrder.Id.Str())
//		t.Log(w.RawOrder.OutAmount.Uint64())
//		t.Log(w.RawOrder.InToken.Str())
//		t.Log(w.RawOrder.OutToken.Str())
//		t.Log(w.PeerId)
//		t.Log(w.OutAmount.Uint64())
//	}
//}
