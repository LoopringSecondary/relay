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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/gateway/ipfs"
	"github.com/Loopring/relay/types"

	"github.com/Loopring/relay/log"
	"sync"
)

type IPFSSubService interface {

	// Register register topic in options and start ipfs sub client
	Register(topic string) error

	// Unregister unregister topic from options and stop ipfs sub client
	Unregister(topic string) error

	// Start default start ipfs sub client
	Start()

	// Stop
	Stop()

	// Restart
	Restart()
}

type IPFSSubServiceImpl struct {
	options config.IpfsOptions
	subs    map[string]*subProxy
	stop    chan struct{}
	mtx     sync.Mutex
	url     string
}

func NewIPFSSubService(options config.IpfsOptions) *IPFSSubServiceImpl {
	l := &IPFSSubServiceImpl{}
	l.url = options.Url()
	l.options = options
	l.subs = make(map[string]*subProxy)

	// TODO: get topics from mysql and combine with toml config

	for _, topic := range l.options.ListenTopics {
		proxy, err := l.newSubProxy(topic)
		if err != nil {
			log.Fatalf("ipfs impl create sub scribe error:%s", err.Error())
		}
		l.subs[topic] = proxy
	}

	return l
}

func (l *IPFSSubServiceImpl) Register(topic string) error {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	var (
		proxy *subProxy
		err   error
		ok    bool
	)

	if proxy, ok = l.subs[topic]; ok {
		return fmt.Errorf("ipfs sub,topic %s already exist", topic)
	}

	if proxy, err = l.newSubProxy(topic); err != nil {
		return fmt.Errorf("ipfs sub,register new topic %s error %s", topic, err.Error())
	}

	proxy.listen()
	l.subs[topic] = proxy

	// todo: add in mysql
	return nil
}

func (l *IPFSSubServiceImpl) Unregister(topic string) error {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	var (
		proxy *subProxy
		ok    bool
	)

	if proxy, ok = l.subs[topic]; !ok {
		return fmt.Errorf("ipfs sub, topic %s do not exist", topic)
	}

	proxy.quit()
	delete(l.subs, topic)

	// todo: delete in mysql
	return nil
}

func (l *IPFSSubServiceImpl) Start() {
	for _, v := range l.subs {
		v.listen()
	}
}

func (l *IPFSSubServiceImpl) Stop() {
	for _, v := range l.subs {
		v.quit()
	}
}

func (l *IPFSSubServiceImpl) Restart() {
	for _, v := range l.subs {
		v.quit()
		v.listen()
	}
}

type subProxy struct {
	topic    string
	iterator *ipfs.PubSubSubscription
	stop     chan struct{}
}

func (l *IPFSSubServiceImpl) newSubProxy(topic string) (*subProxy, error) {
	s := &subProxy{}
	s.topic = topic
	scribe, err := ipfs.PubSubSubscribe(l.url, topic)
	if err != nil {
		return nil, err
	}
	s.iterator = scribe

	return s, nil
}

func (p *subProxy) listen() {
	p.stop = make(chan struct{})

	go func() {
		for {
			record, err := p.iterator.Next()
			if err != nil {
				log.Fatalf("ipfs sub,ipfs occurs err:%s shut down!", err.Error())
			}
			//record.data() have to contain two char: '{' and '}'
			if len(record.Data()) > 2 {
				ord := &types.Order{}
				if err := ord.UnmarshalJSON(record.Data()); err != nil {
					log.Errorf("ipfs sub,failed to accept data %s", err.Error())
					continue
				}
				log.Debugf("ipfs sub,accept data from topic %s and data is %s", p.topic, string(record.Data()))
				eventemitter.Emit(eventemitter.GatewayNewOrder, ord)
			}
		}
	}()
}

func (p *subProxy) quit() {
	close(p.stop)
}
