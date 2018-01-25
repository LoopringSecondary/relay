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

package redis_test

import (
	"encoding/json"
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/config"
	"testing"
	"time"
)

func cfg() *config.RedisOptions {
	return &config.RedisOptions{}
}

func TestRedisCacheImpl_SetExpire(t *testing.T) {
	// test expire time
	if err := cache.Set("test_expire", []byte("hahhah"), 20); err != nil {
		t.Fatalf(err.Error())
	}

	if data, err := cache.Get("test_expire"); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Log(string(data))
	}

	time.Sleep(22 * time.Second)

	if data, err := cache.Get("test_expire"); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Log(data)
	}
}

func TestRedisCacheImpl_SetStruct(t *testing.T) {
	type user struct {
		Name   string `json:name`
		Height int    `json:height`
	}

	u := user{Name: "tom", Height: 181}

	bs, err := json.Marshal(u)
	if err != nil {
		t.Fatalf(err.Error())
	}
	if err := cache.Set("test_struct", bs, 20); err != nil {
		t.Fatalf(err.Error())
	}

	if data, err := cache.Get("test_struct"); err != nil {
		t.Fatalf(err.Error())
	} else {
		var u1 user
		if err := json.Unmarshal(data, &u1); err != nil {
			t.Fatalf(err.Error())
		}
		t.Logf("name:%s, height:%d", u1.Name, u1.Height)
	}
}
