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

	Dels(keys []string) error

	Exists(key string) (bool, error)

	Keys(keyFormat string) ([][]byte, error)

	HMSet(key string, ttl int64, args ...[]byte) error

	HMGet(key string, fields ...[]byte) ([][]byte, error)

	HDel(key string, fields ...[]byte) (int64, error)

	HGetAll(key string) ([][]byte, error)

	HVals(key string) ([][]byte, error)

	HExists(key string, field []byte) (bool, error)

	SAdd(key string, ttl int64, members ...[]byte) error
	SCard(key string) (int64, error)
	SRem(key string, members ...[]byte) (int64, error)

	SMembers(key string) ([][]byte, error)

	SIsMember(key string, member []byte) (bool, error)

	ZAdd(key string, ttl int64, args ...[]byte) error

	ZRange(key string, start, stop int64, withScores bool) ([][]byte, error)
	ZRemRangeByScore(key string, start, stop int64) (int64, error)
}

func NewCache(cfg interface{}) {
	redisCache := &myredis.RedisCacheImpl{}
	redisCache.Initialize(cfg)
	cache = redisCache
}

func Set(key string, value []byte, ttl int64) error { return cache.Set(key, value, ttl) }
func Get(key string) ([]byte, error)                { return cache.Get(key) }
func Del(key string) error                          { return cache.Del(key) }
func Dels(keys []string) error                      { return cache.Dels(keys) }
func Exists(key string) (bool, error)               { return cache.Exists(key) }
func Keys(keyFormat string) ([][]byte, error)       { return cache.Keys(keyFormat) }

func HMSet(key string, ttl int64, args ...[]byte) error {
	return cache.HMSet(key, ttl, args...)
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
func SAdd(key string, ttl int64, members ...[]byte) error {
	return cache.SAdd(key, ttl, members...)
}
func SCard(key string) (int64, error) {
	return cache.SCard(key)
}

func SMembers(key string) ([][]byte, error) {
	return cache.SMembers(key)
}

func SIsMember(key string, member []byte) (bool, error) {
	return cache.SIsMember(key, member)
}

func SRem(key string, members ...[]byte) (int64, error) {
	return cache.SRem(key, members...)
}

func HDel(key string, fields ...[]byte) (int64, error) {
	return cache.HDel(key, fields...)
}

func ZAdd(key string, ttl int64, args ...[]byte) error {
	return cache.ZAdd(key, ttl, args...)
}

func ZRange(key string, start, stop int64, withScores bool) ([][]byte, error) {
	return cache.ZRange(key, start, stop, withScores)
}
func ZRemRangeByScore(key string, start, stop int64) (int64, error) {
	return cache.ZRemRangeByScore(key, start, stop)
}
