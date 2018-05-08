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
	"time"
	"math/big"
)

type Pow struct {
	period time.Duration
	lastDifficult *big.Int
	currentDifficult *big.Int
	nextDifficult *big.Int
	started bool
}

type Difficult struct {
	diffi *big.Int
	expireTime int
}

var defaultPeriod = time.Minute * 5
var pow = Pow{started: false, period: defaultPeriod}
const powCurrentRedisKey = "POW_CHECK_CURRENT"
const powNextRedisKey = "POW_CHECK_NEXT"

func GetCurrentPow() Pow {
	if !pow.started {
		pow.start()
	}
	return pow
}

func GetDifficults() (current *big.Int, next *big.Int) {
	return pow.currentDifficult, pow.nextDifficult
}

func (p *Pow) start() {

}

func (p *Pow) calculateNewDifficult() {

}
