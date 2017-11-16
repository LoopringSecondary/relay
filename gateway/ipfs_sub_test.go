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

package gateway_test

import (
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/test"
	"github.com/ipfs/go-ipfs-api"
	"testing"
	"time"
)

var (
	options config.IpfsOptions
	impl    *gateway.IPFSSubServiceImpl
	sh      *shell.Shell
)

func prepare() {
	globalConfig := test.LoadConfig()
	impl = gateway.NewIPFSSubService(globalConfig.Ipfs)
	options = globalConfig.Ipfs
	sh = shell.NewLocalShell()
}

func putMessage(topic string) {
	order := `{
	"protocol":"0x29d4178372d890e3127d35c3f49ee5ee215d6fe8",
	"tokenS":"0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f",
	"tokenB":"0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e",
	"amountS":"0xc8",
	"amountB":"0xa",
	"timestamp":"0x59ef0cc8",
	"ttl":"0x2710",
	"salt":"0x3e8",
	"lrcFee":"0x64",
	"buyNoMoreThanAmountB":false,
	"marginSplitPercentage":0,
	"v":27,
	"r":"0xecdfe5d96346e1a4fffce7a63fe0c8ff6111b13c3c387a296cdc6d9a10599fb0",
	"s":"0x18640bbb9ccc6b667a05abcd349531b58211084b33fbb73270f1eb1861d6559a",
	"owner":"0x48ff2269e58a373120ffdbbdee3fbcea854ac30a",
	"hash":"0x9b7857b006236a148e70e8b07adf6347610a7d1beb88328810528d98f20496e8"
	}`

	err := sh.PubSubPublish(topic, order)
	if err != nil {
		panic(err.Error())
	}
}

func TestIPFSSubServiceImpl_Start(t *testing.T) {
	prepare()
	impl.Start()
	time.Sleep(1 * time.Second)

	putMessage(options.ListenTopics[0])
	time.Sleep(1 * time.Second)
}

func TestIPFSSubServiceImpl_Stop(t *testing.T) {
	prepare()
	topic := options.ListenTopics[0]

	impl.Start()
	putMessage(topic)
	time.Sleep(3 * time.Second)
	t.Log("start......")

	impl.Stop()
	putMessage(topic)
	time.Sleep(3 * time.Second)
	t.Log("stop......")
}

func TestIPFSSubServiceImpl_Register(t *testing.T) {
	prepare()
	topic1 := options.ListenTopics[0]
	topic2 := "topic_nn"

	impl.Start()
	time.Sleep(1 * time.Second)
	t.Log("start......")

	if err := impl.Register(topic2); err != nil {
		t.Fatalf(err.Error())
	}
	putMessage(topic1)
	putMessage(topic2)
	time.Sleep(1 * time.Second)
	t.Log("register and put message in topic1 and topic2")
}

func TestIPFSSubServiceImpl_Unregister(t *testing.T) {
	prepare()
	topic1 := options.ListenTopics[0]
	topic2 := "topic_nn"

	impl.Start()
	time.Sleep(1 * time.Second)
	t.Log("start......")

	if err := impl.Register(topic2); err != nil {
		t.Fatalf(err.Error())
	}
	putMessage(topic2)
	time.Sleep(1 * time.Second)
	t.Log("register and put message......")

	if err := impl.Unregister(topic2); err != nil {
		t.Fatalf(err.Error())
	}
	putMessage(topic1)
	putMessage(topic2)
	time.Sleep(1 * time.Second)
	t.Log("unregister and put topic1 and topic2 message")
}
