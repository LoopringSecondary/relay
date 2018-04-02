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
	"math/big"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/naoina/toml"
	"go.uber.org/zap"
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

	//if c.Common.Develop {
	//	basedir := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/"
	//	c.Keystore.Keydir = basedir + c.Keystore.Keydir
	//
	//	for idx, path := range c.Log.ZapOpts.OutputPaths {
	//		if !strings.HasPrefix(path, "std") {
	//			c.Log.ZapOpts.OutputPaths[idx] = basedir + path
	//		}
	//	}
	//}

	// extractor.IsDevNet default false

	return c
}

type GlobalConfig struct {
	Title string `required:"true"`
	Mode  string `required:"true"`
	Owner struct {
		Name string
	}
	Mysql          MysqlOptions
	Redis          RedisOptions
	Ipfs           IpfsOptions
	Jsonrpc        JsonrpcOptions
	Websocket      WebsocketOptions
	GatewayFilters GatewayFiltersOptions
	OrderManager   OrderManagerOptions
	Gateway        GateWayOptions
	Accessor       AccessorOptions
	Extractor      ExtractorOptions
	Common         CommonOptions
	Miner          MinerOptions
	Log            LogOptions
	Keystore       KeyStoreOptions
	Market         MarketOptions
	MarketCap      MarketCapOptions
	UserManager    UserManagerOptions
}

type JsonrpcOptions struct {
	Port string
}

type WebsocketOptions struct {
	Port string
}

func (c *GlobalConfig) defaultConfig() {

}

type OrderManagerOptions struct {
	CutoffCacheExpireTime int64
	CutoffCacheCleanTime  int64
	DustOrderValue        int64
}

type IpfsOptions struct {
	Server          string
	Port            int
	ListenTopics    []string
	BroadcastTopics []string
}

func (opts IpfsOptions) Url() string {
	url := opts.Server
	if !strings.HasSuffix(url, ":") {
		url = url + ":"
	}
	return url + strconv.Itoa(opts.Port)
}

type AccessorOptions struct {
	RawUrls []string `required:"true"`
}

type ExtractorOptions struct {
	StartBlockNumber   *big.Int
	EndBlockNumber     *big.Int
	ConfirmBlockNumber uint64
	Debug              bool
}

type KeyStoreOptions struct {
	Keydir  string
	ScryptN int
	ScryptP int
}

type ProtocolOptions struct {
	Address          map[string]string
	ImplAbi          string
	DelegateAbi      string
	TokenRegistryAbi string
	NameRegistryAbi  string
}

type CommonOptions struct {
	Erc20Abi        string
	WethAbi         string
	ProtocolImpl    ProtocolOptions  `required:"true"`
	OrderMinAmounts map[string]int64 //最小的订单金额，低于该数，则终止匹配订单，每个token的值不同
}

type LogOptions struct {
	ZapOpts zap.Config
}

type TimingMatcher struct {
	RoundOrdersCount     int
	Duration             int64
	DelayedNumber        int64
	MaxCacheRoundsLength int
}

type PercentMinerAddress struct {
	Address    string
	FeePercent float64 //the gasprice will be calculated by (FeePercent/100)*(legalFee/eth-price)/gaslimit
	StartFee   float64 //If received reaches StartReceived, it will use feepercent to ensure eth confirm this tx quickly.
}

type NormalMinerAddress struct {
	Address         string
	MaxPendingTtl   int   //if a tx is still pending after MaxPendingTtl blocks, the nonce used by it will be used again.
	MaxPendingCount int64 //this addr will be used to send tx again until the count of pending txs belows MaxPendingCount.
	GasPriceLimit   int64 //the max gas price
}

type MinerOptions struct {
	RingMaxLength         int `` //recommended value:4
	Name                  string
	NormalMiners          []NormalMinerAddress  //
	PercentMiners         []PercentMinerAddress //
	TimingMatcher         *TimingMatcher
	RateRatioCVSThreshold int64
	MinGasLimit           int64
	MaxGasLimit           int64
}

type MarketOptions struct {
	TokenFile             string
	OldVersionWethAddress string
}

type MarketCapOptions struct {
	BaseUrl  string
	Currency string
	Duration int
}

type GatewayFiltersOptions struct {
	BaseFilter struct {
		MinLrcFee             int64
		MaxPrice              int64
		MinSplitPercentage    float64
		MaxSplitPercentage    float64
		MinTokeSAmount        map[string]string
		MinTokenSUsdAmount    float64
		MaxValidSinceInterval int64
	}
}

type GateWayOptions struct {
	IsBroadcast      bool
	MaxBroadcastTime int
}

type MysqlOptions struct {
	Hostname    string
	Port        string
	User        string
	Password    string
	DbName      string
	TablePrefix string
	Debug       bool
}

type RedisOptions struct {
	Host        string
	Port        string
	Password    string
	IdleTimeout int
	MaxIdle     int
	MaxActive   int
}

type UserManagerOptions struct {
	WhiteListOpen            bool
	WhiteListCacheExpireTime int64
	WhiteListCacheCleanTime  int64
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
