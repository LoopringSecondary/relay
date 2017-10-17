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

package test

import (
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	ethCryptoLib "github.com/Loopring/ringminer/crypto/eth"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"time"
)

func CreateOrder(tokenS, tokenB types.Address, amountS, amountB *big.Int, pkBytes []byte) *types.Order {
	order := &types.Order{}
	order.Protocol = types.HexToAddress("0xfd9ecf92e3684451c2502cf9cdc45962a4febffa")
	order.TokenS = tokenS
	order.TokenB = tokenB
	order.AmountS = amountS
	order.AmountB = amountB
	order.Timestamp = big.NewInt(time.Now().Unix())
	order.Ttl = big.NewInt(10000)
	order.Salt = big.NewInt(1000)
	order.LrcFee = big.NewInt(100)
	order.BuyNoMoreThanAmountB = false
	order.MarginSplitPercentage = 0
	order.GenerateHash()
	//order.GenerateAndSetSignature(types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"))
	order.GenerateAndSetSignature(pkBytes)
	return order
}

func init() {
	globalConfig := config.LoadConfig("/Users/yuhongyu/Desktop/service/go/src/github.com/Loopring/ringminer/config/ringminer.toml")
	log.Initialize(globalConfig.Log)

	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	ethClient := eth.NewChainClient(globalConfig.ChainClient)

	database := db.NewDB(globalConfig.Database)
	ringClient := miner.NewRingClient(database, ethClient.Client)
	//
	miner.Initialize(globalConfig.Miner, ringClient.Chainclient)

}
