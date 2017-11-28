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
	"errors"
	"github.com/naoina/toml"
	"go.uber.org/zap"
	"math/big"
	"os"
	"reflect"
	"strings"
)

func LoadConfig(file string) *GlobalConfig {
	if "" == file {
		dir, _ := os.Getwd()
		file = dir + "/config/relay.toml"
	}

	io, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer io.Close()

	c := &GlobalConfig{}
	c.defaultConfig()
	if err := toml.NewDecoder(io).Decode(c); err != nil {
		panic(err)
	}

	if c.Common.Develop {
		basedir := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/"
		c.Keystore.Keydir = basedir + c.Keystore.Keydir

		for idx, path := range c.Log.ZapOpts.OutputPaths {
			if !strings.HasPrefix(path, "std") {
				c.Log.ZapOpts.OutputPaths[idx] = basedir + path
			}
		}
	}

	return c
}

type GlobalConfig struct {
	Title string `required:"true"`
	Owner struct {
		Name string
	}
	Mysql          MysqlOptions
	Ipfs           IpfsOptions
	Jsonrpc        JsonrpcOptions
	GatewayFilters GatewayFiltersOptions
	Gateway        GateWayOptions
	Accessor       AccessorOptions
	Common         CommonOptions
	Miner          MinerOptions
	OrderManager   OrderManagerOptions
	Log            LogOptions
	Keystore       KeyStoreOptions
}

type JsonrpcOptions struct {
	Port int
}

func (c *GlobalConfig) defaultConfig() {

}

type IpfsOptions struct {
	Server          string
	Port            int
	ListenTopics    []string
	BroadcastTopics []string
}

type AccessorOptions struct {
	RawUrl string `required:"true"`
	Eth    struct {
		GasPrice int
		GasLimit int
	}
}

type KeyStoreOptions struct {
	Keydir  string
	ScryptN int
	ScryptP int
}

type ProtocolOptions struct {
	Address          string
	ImplAbi          string
	RegistryAbi      string
	DelegateAbi      string
	TokenRegistryAbi string
}

type CommonOptions struct {
	Erc20Abi           string
	ProtocolImpls      map[string]ProtocolOptions `required:"true"`
	FilterTopics       []string                   `required:"true"`
	DefaultBlockNumber *big.Int                   `required:"true"`
	EndBlockNumber     *big.Int                   `required:"true"`
	Develop            bool                       `required:"true"`
	OrderMinAmounts    map[string]int64           //最小的订单金额，低于该数，则终止匹配订单，每个token的值不同
}

type LogOptions struct {
	ZapOpts zap.Config
}

type MinerOptions struct {
	RingMaxLength           int    `required:"true"` //recommended value:4
	Miner                   string `required:"true"` //private key, used to sign the ring
	FeeRecepient            string //address the recepient of fee
	IfRegistryRingHash      bool
	ThrowIfLrcIsInsuffcient bool
	RateProvider            struct {
		BaseUrl       string
		Currency      string
		CurrenciesMap map[string]string //address -> name
	}
	RateRatioCVSThreshold int64
}

type OrderManagerOptions struct {
	TickerDuration int
	BlockPeriod    int
}

type GatewayFiltersOptions struct {
	BaseFilter struct {
		MinLrcFee int64
	}
	TokenSFilter struct {
		Allow  []string
		Denied []string
	}
	TokenBFilter struct {
		Allow  []string
		Denied []string
	}
}

type GateWayOptions struct {
	IsBroadcast bool
	MaxBroadcastTime int
}


type MysqlOptions struct {
	User        string
	Password    string
	DbName      string
	Loc         string
	TablePrefix string
}

func Validator(cv reflect.Value) (bool, error) {
	for i := 0; i < cv.NumField(); i++ {
		cvt := cv.Type().Field(i)

		if cv.Field(i).Type().Kind() == reflect.Struct {
			if res, err := Validator(cv.Field(i)); nil != err {
				return res, err
			}
		} else {
			if "true" == cvt.Tag.Get("required") {
				if !isSet(cv.Field(i)) {
					return false, errors.New("The field " + cvt.Name + " in config must be setted")
				}
			}
		}
	}

	return true, nil
}

func isSet(v reflect.Value) bool {
	switch v.Type().Kind() {
	case reflect.Invalid:
		return false
	case reflect.String:
		return v.String() != ""
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() != 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Map:
		return len(v.MapKeys()) != 0
	}
	return true
}
