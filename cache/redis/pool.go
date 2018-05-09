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

package redis

import (
	"errors"
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"github.com/garyburd/redigo/redis"
	"time"
)

type RedisCacheImpl struct {
	options config.RedisOptions
	pool    *redis.Pool
}

func (impl *RedisCacheImpl) Initialize(cfg interface{}) {
	options := cfg.(config.RedisOptions)
	impl.options = options

	impl.pool = &redis.Pool{
		IdleTimeout: time.Duration(options.IdleTimeout) * time.Second,
		MaxIdle:     options.MaxIdle,
		MaxActive:   options.MaxActive,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			address := fmt.Sprintf("%s:%s", options.Host, options.Port)
			var (
				c   redis.Conn
				err error
			)
			if len(options.Password) > 0 {
				c, err = redis.Dial("tcp", address, redis.DialPassword(options.Password))
			} else {
				c, err = redis.Dial("tcp", address)
			}

			if err != nil {
				log.Fatal(err.Error())
				return nil, err
			}

			return c, nil
		},
	}
}

func (impl *RedisCacheImpl) Get(key string) ([]byte, error) {
	//log.Info("[REDIS-GET] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("get", key)

	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
		return []byte{}, err
	} else if nil == reply {
		if nil == err {
			err = fmt.Errorf("no this key:%s", key)
		}
		return []byte{}, err
	} else {
		return reply.([]byte), err
	}
}

func (impl *RedisCacheImpl) Exists(key string) (bool, error) {

	//log.Info("[REDIS-Exists] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("exists", key)

	if err != nil {
		log.Errorf(" key:%s, err:%s", key, err.Error())
		return false, err
	} else {
		exists := reply.(int64)
		if exists == 1 {
			return true, nil
		} else {
			return false, nil
		}
	}
}

func (impl *RedisCacheImpl) Set(key string, value []byte, ttl int64) error {

	//log.Info("[REDIS-SET] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("set", key, value); err != nil {
		log.Errorf(" key:%s, err:%s", key, err.Error())
		return err
	}

	if ttl > 0 {
		if _, err := conn.Do("expire", key, ttl); err != nil {
			log.Errorf(" key:%s, err:%s", key, err.Error())
			return err
		}
	}
	return nil
}

func (impl *RedisCacheImpl) Del(key string) error {

	//log.Info("[REDIS-Del] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	_, err := conn.Do("del", key)
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	}
	return err
}

func (impl *RedisCacheImpl) Dels(keys []string) error {
	conn := impl.pool.Get()
	defer conn.Close()

	var list []interface{}

	for _, v := range keys {
		list = append(list, v)
	}

	num, err := conn.Do("del", list...)
	if err != nil {
		log.Debugf("delete multi keys error:%s", err.Error())
	} else {
		log.Debugf("delete %d keys", num.(int64))
	}

	return nil
}

func (impl *RedisCacheImpl) Keys(keyFormat string) ([][]byte, error) {

	//log.Info("[REDIS-HMGET] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	vs := []interface{}{}
	vs = append(vs, keyFormat)
	reply, err := conn.Do("keys", vs...)
	res := [][]byte{}
	if nil != err {
		log.Errorf(" key:%s, err:%s", keyFormat, err.Error())
	} else if nil == err && nil != reply {
		rs := reply.([]interface{})
		for _, r := range rs {
			if nil == r {
				res = append(res, []byte{})
			} else {
				res = append(res, r.([]byte))
			}
		}
	}
	return res, err
}

func (impl *RedisCacheImpl) HMSet(key string, ttl int64, args ...[]byte) error {

	//log.Info("[REDIS-HMSET] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	if len(args)%2 != 0 {
		return errors.New("the length of `args` must be even")
	}
	vs := []interface{}{}
	vs = append(vs, key)
	for _, v := range args {
		vs = append(vs, v)
	}
	_, err := conn.Do("hmset", vs...)
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	}
	if ttl > 0 {
		if _, err := conn.Do("expire", key, ttl); err != nil {
			log.Errorf(" key:%s, err:%s", key, err.Error())
			return err
		}
	}
	return err
}

func (impl *RedisCacheImpl) ZAdd(key string, ttl int64, args ...[]byte) error {

	//log.Info("[REDIS-ZAdd] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	if len(args)%2 != 0 {
		return errors.New("the length of `args` must be even")
	}
	vs := []interface{}{}
	vs = append(vs, key)
	for _, v := range args {
		vs = append(vs, v)
	}
	_, err := conn.Do("zadd", vs...)
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	}
	if ttl > 0 {
		if _, err := conn.Do("expire", key, ttl); err != nil {
			log.Errorf(" key:%s, err:%s", key, err.Error())
			return err
		}
	}
	return err
}

func (impl *RedisCacheImpl) HMGet(key string, fields ...[]byte) ([][]byte, error) {

	//log.Info("[REDIS-HMGET] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	vs := []interface{}{}
	vs = append(vs, key)
	for _, v := range fields {
		vs = append(vs, v)
	}
	reply, err := conn.Do("hmget", vs...)

	res := [][]byte{}
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	} else if nil == err && nil != reply {
		rs := reply.([]interface{})
		for _, r := range rs {
			if nil == r {
				res = append(res, []byte{})
			} else {
				res = append(res, r.([]byte))
			}
		}
	}
	return res, err
}

func (impl *RedisCacheImpl) ZRange(key string, start, stop int64, withScores bool) ([][]byte, error) {

	//log.Info("[REDIS-ZRANGE] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	vs := []interface{}{}
	vs = append(vs, key, start, stop)
	if withScores {
		vs = append(vs, []byte("WITHSCORES"))
	}
	reply, err := conn.Do("ZRANGE", vs...)

	res := [][]byte{}
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	} else if nil == err && nil != reply {
		rs := reply.([]interface{})
		for _, r := range rs {
			if nil == r {
				res = append(res, []byte{})
			} else {
				res = append(res, r.([]byte))
			}
		}
	}
	return res, err
}

func (impl *RedisCacheImpl) HDel(key string, fields ...[]byte) (int64, error) {

	//log.Info("[REDIS-HDEL] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	vs := []interface{}{}
	vs = append(vs, key)
	for _, v := range fields {
		vs = append(vs, v)
	}
	reply, err := conn.Do("hdel", vs...)

	if err != nil {
		log.Errorf(" key:%s, err:%s", key, err.Error())
		return 0, err
	} else {
		res := reply.(int64)
		return res, err
	}
}
func (impl *RedisCacheImpl) SCard(key string) (int64, error) {

	//log.Info("[REDIS-SCARD] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	vs := []interface{}{}
	vs = append(vs, key)
	reply, err := conn.Do("scard", vs...)

	if err != nil {
		log.Errorf(" key:%s, err:%s", key, err.Error())
		return 0, err
	} else {
		res := reply.(int64)
		return res, err
	}
}

func (impl *RedisCacheImpl) ZRemRangeByScore(key string, start, stop int64) (int64, error) {

	//log.Info("[REDIS-ZRemRangeByScore] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	vs := []interface{}{}
	vs = append(vs, key, start, stop)

	reply, err := conn.Do("ZREMRANGEBYSCORE", vs...)

	if err != nil {
		log.Errorf(" key:%s, err:%s", key, err.Error())
		return 0, err
	} else {
		res := reply.(int64)
		return res, err
	}
}

func (impl *RedisCacheImpl) SRem(key string, members ...[]byte) (int64, error) {

	//log.Info("[REDIS-SRem] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	vs := []interface{}{}
	vs = append(vs, key)
	for _, v := range members {
		vs = append(vs, v)
	}
	reply, err := conn.Do("srem", vs...)

	if err != nil {
		log.Errorf(" key:%s, err:%s", key, err.Error())
		return 0, err
	} else {
		res := reply.(int64)
		return res, err
	}
}

func (impl *RedisCacheImpl) SIsMember(key string, member []byte) (bool, error) {
	conn := impl.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("sismember", key, member)
	if err != nil {
		log.Errorf("key:%s, err:%s", key, err.Error())
		return false, err
	} else {
		return reply.(int64) > 0, nil
	}
}

func (impl *RedisCacheImpl) HGetAll(key string) ([][]byte, error) {

	//log.Info("[REDIS-HGetAll] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("hgetall", key)

	res := [][]byte{}
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	} else if nil == err && nil != reply {
		rs := reply.([]interface{})
		for _, r := range rs {
			res = append(res, r.([]byte))
		}
	}
	return res, err
}
func (impl *RedisCacheImpl) HVals(key string) ([][]byte, error) {

	//log.Info("[REDIS-HVals] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	//todo:test nil result
	reply, err := conn.Do("hvals", key)

	res := [][]byte{}
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	} else if nil == err && nil != reply {
		rs := reply.([]interface{})
		for _, r := range rs {
			res = append(res, r.([]byte))
		}
	}
	return res, err
}

func (impl *RedisCacheImpl) HExists(key string, field []byte) (bool, error) {

	//log.Info("[REDIS-HExists] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("hexists", key, field)
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	} else if nil == err && nil != reply {
		exists := reply.(int64)
		return exists > 0, nil
	}

	return false, err
}

func (impl *RedisCacheImpl) SAdd(key string, ttl int64, members ...[]byte) error {

	//log.Info("[REDIS-SAdd] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	vs := []interface{}{}
	vs = append(vs, key)
	for _, v := range members {
		vs = append(vs, v)
	}
	_, err := conn.Do("sadd", vs...)
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	}
	if ttl > 0 {
		if _, err := conn.Do("expire", key, ttl); err != nil {
			log.Errorf(" key:%s, err:%s", key, err.Error())
			return err
		}
	}
	return err
}

func (impl *RedisCacheImpl) SMembers(key string) ([][]byte, error) {

	//log.Info("[REDIS-SMembers] key : " + key)

	conn := impl.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("smembers", key)

	res := [][]byte{}
	if nil != err {
		log.Errorf(" key:%s, err:%s", key, err.Error())
	} else if nil == err && nil != reply {
		rs := reply.([]interface{})
		for _, r := range rs {
			res = append(res, r.([]byte))
		}
	}
	return res, err
}
