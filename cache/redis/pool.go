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
	conn := impl.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("get", key)

	if nil == reply {
		if nil == err {
			err = fmt.Errorf("no this key:%s", key)
		}
		return []byte{}, err
	} else {
		return reply.([]byte), err
	}
}

func (impl *RedisCacheImpl) Exists(key string) (bool, error) {
	conn := impl.pool.Get()
	defer conn.Close()

	reply, err := conn.Do("exists", key)

	if err != nil {
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
	conn := impl.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("set", key, value); err != nil {
		return err
	}
	if _, err := conn.Do("expire", key, ttl); err != nil {
		return err
	}
	return nil
}

func (impl *RedisCacheImpl) Del(key string) error {
	conn := impl.pool.Get()
	defer conn.Close()

	_, err := conn.Do("del", key)

	return err
}
