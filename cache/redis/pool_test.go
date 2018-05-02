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
	"github.com/Loopring/relay/test"
	"github.com/lydy/go-ethereum/common"
	"os/exec"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestRedisCacheImpl_SetExpire(t *testing.T) {
	cache.NewCache(test.Cfg().Redis)

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
	cache.NewCache(test.Cfg().Redis)

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

	for i := 0; i < 1000; i++ {
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
}

func TestRedisCacheImpl_SAdd(t *testing.T) {
	cache.NewCache(test.Cfg().Redis)

	//err := cache.HMSet("test1", []byte("k1"), []byte("v1"), []byte("k2"), []byte("v2"))

	repl, err := cache.HGetAll("test1")

	if nil != err {
		t.Error(err.Error())
	} else {
		//println(string(repl))
		for _, r := range repl {
			t.Log(string(r))
		}
	}
}

func TestRedisCacheImpl_HMSet(t *testing.T) {
	cache.NewCache(test.Cfg().Redis)

	start := time.Now().UnixNano()
	for i := 0; i < 100000; i++ {
		if err := cache.HMSet("test_hmset", 0, []byte("balance_"+strconv.Itoa(i)), []byte(strconv.Itoa(i))); nil != err {
			t.Errorf(err.Error())
		}
	}
	end := time.Now().UnixNano()
	t.Logf("time1: %d", (end - start))

	cache.HGetAll("test1")

	end1 := time.Now().UnixNano()
	t.Logf("time2: %d", (end1 - end))
}

func TestRedisCacheImpl_BenchSyncPool(t *testing.T) {
	cache.NewCache(test.Cfg().Redis)

	for i := 0; i < 1000000; i++ {
		if err := cache.Set("test_expire", []byte(strconv.Itoa(i)), 1000); err != nil {
			t.Fatalf(err.Error())
		}
	}

	execCmd("netstat -an|grep 6379|grep ESTABLISHED -c", t)
	execCmd("netstat -an|grep 6379|grep TIME_WAIT|wc -l", t)
}

func TestRedisCacheImpl_BenchAsyncPool(t *testing.T) {
	cache.NewCache(test.Cfg().Redis)

	var wg sync.WaitGroup
	for i := 0; i < 100000; i++ {
		wg.Add(1)
		go func(num int) {
			wg.Done()
			if err := cache.Set("test_expire", []byte(strconv.Itoa(num)), 1000); err != nil {
				t.Fatalf(err.Error())
			}
		}(i)
	}
	wg.Wait()
	execCmd("netstat -an|grep 6379|grep ESTABLISHED -c", t)
	execCmd("netstat -an|grep 6379|grep TIME_WAIT -c", t)
}

func execCmd(cmdStr string, t *testing.T) {
	cmd := exec.Command("sh", "-c", cmdStr)
	if bs, err := cmd.Output(); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Log(cmdStr)
		t.Log(string(bs))
	}
}
