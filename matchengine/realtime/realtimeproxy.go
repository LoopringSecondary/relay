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

package realtime

import (
	"sync"
	"github.com/Loopring/ringminer/types"
)

/**
todo：功能完整性上，必须要实现的部分
实时计算最小环，有效的算法
 */

type RealtimeProxy struct {
	mtx sync.RWMutex

	OrderChangeChan chan *types.Order   //订单改变的channel，在匹配过程中，订单改变可以及时终止或更改当前匹配

}