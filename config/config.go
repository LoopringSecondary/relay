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
	"os"
	"reflect"
)

func LoadConfig(file string) *GlobalConfig {
	if "" == file {
		dir, _ := os.Getwd()
		file = dir + "/config/ringminer.toml"
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

	return c
}

type GlobalConfig struct {
	Title string `required:"true"`
	Owner struct {
		Name string
	}
	Database    DbOptions
	Ipfs        IpfsOptions
	ChainClient ChainClientOptions
	Miner       MinerOptions
	Orderbook   OrderBookOptions
	Log         zap.Config
}

func (c *GlobalConfig) defaultConfig() {

}

type IpfsOptions struct {
	Server string
	Port   int
	Topic  string
}

type DbOptions struct {
	Server         string `required:"true"`
	Port           int    `required:"true"`
	Name           string `required:"true"`
	DataDir        string
	CacheCapacity  int
	BufferCapacity int
}

type ChainClientOptions struct {
	RawUrl     string `required:"true"`
	Senders              map[string]string `required:"true"` //address->encrypted private key, used to send transaction
	Passphrase string //密码，用于加密私钥，最长为32个字符，安全起见，建议不出现在配置文件中
	Eth        struct {
		GasPrice    int
		GasLimit    int
		   }
}

type MinerOptions struct {
	Passphrase string //密码，用于加密私钥，最长为32个字符，安全起见，建议不出现在配置文件中
	LoopringImps         []ContractOpts `required:"true"`
	LoopringFingerprints []ContractOpts `required:"true"`
	RingMaxLength        int  `required:"true"` //recommended value:4
	Miner                string `required:"true"` //private key, used to sign the ring
	FeeRecepient         string          //address the recepient of fee
}

type OrderBookOptions struct {
	Filters struct {
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

	OrderMinAmounts map[string]string //最小的订单金额，低于该数，则终止匹配订单，每个token的值不同
}

type ContractOpts struct {
	Abi     string `required:"true"`
	Address string `required:"true"`
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
