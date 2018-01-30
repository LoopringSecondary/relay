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
	"fmt"
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

type WsClient struct {
	websocket   *websocket.Conn
	clientIP    string
	connectType string
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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	if err := handler.RegisterName("eth", j.ethForwarder); err != nil {
		fmt.Println(err)
		return
	}
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", ":8083"); err != nil {
		return
	}
	go rpc.NewHTTPServer([]string{"*"}, handler).Serve(listener)
	log.Info(fmt.Sprintf("HTTP endpoint opened on 8083"))

	return
}

func (ws *WebsocketServiceImpl) serve(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err.Error())
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
