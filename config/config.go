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

package config

import (
	"github.com/naoina/toml"
	"os"
)

func LoadConfig() *GlobalConfig {
	dir, _ := os.Getwd()
	file := dir + "/config/prod.toml"

	io, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer io.Close()

	var c GlobalConfig
	if err := toml.NewDecoder(io).Decode(&c); err != nil {
		panic(err)
	}

	return &c
}

type GlobalConfig struct {
	Title string
	Owner struct {
		Name string
	}
	Database DbOptions
	Ipfs IpfsOptions
	EthClient EthClientOptions
}

type IpfsOptions struct {
	Server string
	Port int
	Topic string
}

type DbOptions struct {
	Server string
	Port int
	Name string
	CacheCapacity int
	BufferCapacity int
}

type EthClientOptions struct {
	Server string
	Port int
}

func defaultConfig() {

}