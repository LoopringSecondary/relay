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

package cache

import (
	myredis "github.com/Loopring/relay/cache/redis"
)

var cache Cache

type Cache interface {
	Set(key string, value []byte, ttl int64) error

	Get(key string) ([]byte, error)

	Del(key string) error
}

func NewCache(cfg interface{}) {
	redis := &myredis.RedisCacheImpl{}
	redis.Initialize(cfg)
	cache = redis
}

func Set(key string, value []byte, ttl int64) error { return cache.Set(key, value, ttl) }
func Get(key string) ([]byte, error)                { return cache.Get(key) }
func Del(key string) error                          { return cache.Del(key) }
