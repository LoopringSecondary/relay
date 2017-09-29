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

package log

//type Template string

const (
	ERROR_P2P_LISTEN_START  = "failed to start listener"
	ERROR_P2P_LISTEN_ACCEPT = "failed to accept ipfs data:%s"
	ERROR_P2P_LISTEN_STOP   = "p2p network stopped"
	ERROR_LDB_CREATE_FAILED = "leveldb create failed:%s"
	LOG_P2P_ACCEPT          = "accept p2p network order:%s"
	LOG_P2P_STOP            = "stop p2p network success"
)
