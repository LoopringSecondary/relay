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
	"github.com/ethereum/go-ethereum/rpc"
	"net"
)

func (*JsonrpcServiceImpl) Ping(val string, val2 int) (res string, err error) {
	res = "pong for first connect, meaning server is OK"
	return
}

type JsonrpcService interface {
	Start(port string)
	Stop()
}

type JsonrpcServiceImpl struct {
	port          string
	walletService WalletServiceImpl
}

func NewJsonrpcService(port string, walletService WalletServiceImpl) *JsonrpcServiceImpl {
	l := &JsonrpcServiceImpl{}
	l.port = port
	l.walletService = walletService
	return l
}

func (j *JsonrpcServiceImpl) Start() {
	log.Info("start jsonrpc service now.......1")
	handler := rpc.NewServer()
	if err := handler.RegisterName("loopring", j.walletService); err != nil {
		fmt.Println(err)
		return
	}

	log.Info("start jsonrpc service now.......2")

	var (
		listener net.Listener
		err      error
	)

	log.Info("start jsonrpc service now.......3")
	if listener, err = net.Listen("tcp", ":8083"); err != nil {
		return
	}
	go rpc.NewHTTPServer([]string{"*"}, handler).Serve(listener)
	log.Info(fmt.Sprintf("HTTP endpoint opened on 8083"))

	return
}
