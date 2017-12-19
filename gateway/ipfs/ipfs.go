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
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/Loopring/relay/log"
	"github.com/ipfs/go-ipfs-api"
	pb "github.com/libp2p/go-floodsub/pb"
	peer "github.com/libp2p/go-libp2p-peer"
	"net/http"
)

var BootstrapPeers = []string{
	"",
}

type Record struct {
	*pb.Message
}

func (r Record) From() peer.ID {
	return peer.ID(r.Message.GetFrom())
}

func (r Record) Data() []byte {
	return r.Message.GetData()
}

func (r Record) SeqNo() int64 {
	return int64(binary.BigEndian.Uint64(r.Message.GetSeqno()))
}

func (r Record) TopicIDs() []string {
	return r.Message.GetTopicIDs()
}

type PubSubSubscription struct {
	reader *chunkedReader
}

func (s *PubSubSubscription) Next() (*Record, error) {
	msgData, err := s.reader.NextChunk()
	if nil != err {
		return nil, err
	}
	record := &Record{}
	if err := json.Unmarshal(msgData, record); nil != err {
		return nil, err
	}
	return record, nil
}

func PubSubSubscribe(url, topic string) (*PubSubSubscription, error) {
	req := shell.NewRequest(context.Background(), url, "pubsub/sub", topic)
	client := &http.Client{Transport: &http.Transport{
		DisableKeepAlives: true,
	},
	}
	if response, err := req.Send(client); nil != err {
		log.Errorf("err:%s", err.Error())
		return nil, err
	} else {
		if nil == response.Output {
			err := errors.New("can't connect to ipfs client")
			return nil, err
		}
		reader := NewChunkedReader(response.Output)
		return &PubSubSubscription{reader: reader}, nil
	}
}
