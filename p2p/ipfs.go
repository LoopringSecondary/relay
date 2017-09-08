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

package p2p

import (
	"github.com/ipfs/go-ipfs-api"
	"sync"
	"github.com/Loopring/ringminer/types"
	"github.com/Loopring/ringminer/config"
)

type IpfsConfig struct {
	topic string
}

type IPFSListener struct {
	conf IpfsConfig
	toml config.IpfsOptions
	sh *shell.Shell
	sub *shell.PubSubSubscription
	stop chan struct{}
	whisper *types.Whispers
	lock sync.RWMutex
}

func (l *IPFSListener) loadConfig() {
	l.conf.topic = l.toml.Topic
}

func NewListener(whisper *types.Whispers, options config.IpfsOptions) *IPFSListener {
	l := &IPFSListener{}

	l.toml = options
	l.loadConfig()

	l.sh = shell.NewLocalShell()
	sub, err := l.sh.PubSubSubscribe(l.conf.topic)
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
			record, _ := l.sub.Next()
			data := record.Data()
			ord := GenOrder(data)
			l.whisper.PeerOrderChan <- ord
		}
	}()
}

func (listener *IPFSListener) Stop() {
	listener.lock.Lock()
	close(listener.stop)
	listener.lock.Unlock()
}

func (listener *IPFSListener) Name() string {
	return "ipfs-listener"
}
