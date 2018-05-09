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

package txmanager_test

import (
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/txmanager"
	"testing"
)

func TestRollbackCache(t *testing.T) {
	test.Cfg()

	from := 42710
	to := 42711
	if err := txmanager.RollbackCache(int64(from), int64(to)); err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("rollback tx cache success!")
}
