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

package gateway

import (
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/marketcap"
	"github.com/gorilla/websocket"
	"net/http"
)

type WebsocketService interface {
	Start(port string)
	Stop()
}

type WebsocketServiceImpl struct {
	port           string
	trendManager   market.TrendManager
	accountManager market.AccountManager
	marketCap      marketcap.MarketCapProvider
	upgrader       websocket.Upgrader
}

type NodeType int

const (
	TICKER NodeType = iota
	PORTFOLIO
	MARKETCAP
	BALANCE
	TRANSACTION
)

var MsgTypeRoute = map[NodeType]string{
	TICKER:      "ticker",
	PORTFOLIO:   "portfolio",
	MARKETCAP:   "marketcap",
	BALANCE:     "balance",
	TRANSACTION: "transaction",
}

type WebsocketRequest struct {
	Params interface{}
}

func NewWebsocketService(port string, trendManager market.TrendManager, accountManager market.AccountManager, capProvider marketcap.MarketCapProvider) *WebsocketServiceImpl {
	l := &WebsocketServiceImpl{}
	l.port = port
	l.trendManager = trendManager
	l.accountManager = accountManager
	l.marketCap = capProvider
	l.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	return l
}

func (ws *WebsocketServiceImpl) Start() {

	for k, v := range MsgTypeRoute {
		node := newSocketNode(k)
		go node.run()
		http.HandleFunc("/socket/"+v, func(w http.ResponseWriter, r *http.Request) {
			ws.serve(node, w, r)
		})
	}

	err := http.ListenAndServe(":"+ws.port, nil)
	if err != nil {
		log.Fatal("ListenAndServe Websocket Error : " + err.Error())
	}

	return
}

func (ws *WebsocketServiceImpl) serve(node *SocketNode, w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("get ws connection error , " + err.Error())
		return
	}
	client := &SocketClient{node: node, conn: conn, send: make(chan []byte, 256)}
	client.node.register <- client
	go client.write()
	go client.read()
}
