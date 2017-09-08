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
	"github.com/Loopring/ringminer/types"
	"github.com/Loopring/ringminer/log"
)

func GenOrder(data []byte) *types.Order {
	var ord types.Order
	err := ord.UnMarshalJson(data)
	if err != nil {
		log.Error(log.ERROR_P2P_LISTEN_ACCEPT,  err.Error())
	} else {
		log.Info(log.LOG_P2P_ACCEPT, string(data))
	}

	return &ord
}