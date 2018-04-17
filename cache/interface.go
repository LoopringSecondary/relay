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

	Exists(key string) (bool, error)

	HMSet(key string, args ...[]byte) error

	HMGet(key string, fields ...[]byte) ([][]byte, error)

	HDel(key string, fields ...[]byte) (int64, error)

	HGetAll(key string) ([][]byte, error)

	HVals(key string) ([][]byte, error)

	HExists(key string, field []byte) (bool, error)

	SAdd(key string, members ...[]byte) error

	SRem(key string, members ...[]byte) (int64,error)

	SMembers(key string) ([][]byte, error)
}

func NewCache(cfg interface{}) {
	redisCache := &myredis.RedisCacheImpl{}
	redisCache.Initialize(cfg)
	cache = redisCache
}

func Set(key string, value []byte, ttl int64) error { return cache.Set(key, value, ttl) }
func Get(key string) ([]byte, error)                { return cache.Get(key) }
func Del(key string) error                          { return cache.Del(key) }
func Exists(key string) (bool, error)               { return cache.Exists(key) }

func HMSet(key string, args ...[]byte) error {
	return cache.HMSet(key, args...)
}

func HMGet(key string, fields ...[]byte) ([][]byte, error) {
	return cache.HMGet(key, fields...)
}

func HGetAll(key string) ([][]byte, error) {
	return cache.HGetAll(key)
}

func HVals(key string) ([][]byte, error) {
	return cache.HVals(key)
}

func HExists(key string, field []byte) (bool, error) {
	return cache.HExists(key, field)
}
func SAdd(key string, members ...[]byte) error {
	return cache.SAdd(key, members...)
}

func SMembers(key string) ([][]byte, error) {
	return cache.SMembers(key)
}

func SRem(key string, members ...[]byte) (int64,error) {
	return cache.SRem(key, members...)
}

func HDel(key string, fields ...[]byte) (int64, error) {
	return cache.HDel(key, fields...)
}