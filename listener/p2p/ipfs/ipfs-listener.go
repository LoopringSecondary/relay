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

package ipfs

import (
	"encoding/json"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"github.com/ipfs/go-ipfs-api"
	"sync"
)

type Whisper struct {
	PeerOrderChan chan *types.Order
}

type IPFSListener struct {
	options config.IpfsOptions
	sh      *shell.Shell
	sub     *shell.PubSubSubscription
	whisper *Whisper
	stop    chan struct{}
	lock    sync.RWMutex
}

func NewListener(options config.IpfsOptions, whisper *Whisper) *IPFSListener {
	l := &IPFSListener{}

	l.options = options

	l.sh = shell.NewLocalShell()
	sub, err := l.sh.PubSubSubscribe(options.Topic)
	if err != nil {
		panic(err.Error())
	}
	l.sub = sub
	l.whisper = whisper

	return l
}

func (l *IPFSListener) Start() {
	l.stop = make(chan struct{})
	go func() {
		for {
			if record, err := l.sub.Next(); nil != err {
				log.Errorf("err:%s", err.Error())
			} else {
				data := record.Data()
				ord := &types.Order{}
				err := json.Unmarshal(data, ord)
				if err != nil {
					log.Errorf(log.ERROR_P2P_LISTEN_ACCEPT, err.Error())
				} else {
					log.Debugf(log.LOG_P2P_ACCEPT, string(data))
					l.whisper.PeerOrderChan <- ord
				}
			}
		}
	}()
}

func (listener *IPFSListener) Stop() {
	listener.lock.Lock()
	close(listener.stop)
	listener.lock.Unlock()
}

func (listener *IPFSListener) Restart() {

}

func (listener *IPFSListener) Name() string {
	return "ipfs-listener"
}
