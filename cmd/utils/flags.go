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

package utils

import "gopkg.in/urfave/cli.v1"

var (
	p2pFlag = cli.StringFlag{
		Name:  "network",
		Usage: "chose a p2p network, <ipfs>",
		Value: defaultP2PNetWork(),
	}

	subtopicFlag = cli.StringFlag{
		Name:  "topic",
		Usage: "chose a ifps pubsub sub topic,<topic>",
		Value: defaultIpfsSubTopic(),
	}
)

const (
	DEFAULT_NET_WORK       = "ipfs"
	DEFAULT_IPFS_SUB_TOPIC = "topic"
)

func defaultP2PNetWork() string {
	return DEFAULT_NET_WORK
}

func defaultIpfsSubTopic() string {
	return DEFAULT_IPFS_SUB_TOPIC
}

func GlobalFlags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:  "cnf,c",
			Usage: "config file",
		},
	}
}
