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

package miner_test

import (
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	ethCryptoLib "github.com/Loopring/ringminer/crypto/eth"
	"github.com/Loopring/ringminer/db"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/miner"
	"github.com/Loopring/ringminer/types"
	"math/big"
	"regexp"
	"testing"
	"time"
)

var client *chainclient.Client
var imp *chainclient.LoopringProtocolImpl = &chainclient.LoopringProtocolImpl{}
var implAddress string = "0xfd9ecf92e3684451c2502cf9cdc45962a4febffa"
var registry *chainclient.LoopringRinghashRegistry = &chainclient.LoopringRinghashRegistry{}
var ringClient *miner.RingClient

func init() {
	globalConfig := config.LoadConfig("../config/ringminer.toml")
	log.Initialize(globalConfig.Log)

	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	ethClient := eth.NewChainClient(globalConfig.ChainClient)

	database := db.NewDB(globalConfig.Database)
	ringClient = miner.NewRingClient(database, ethClient.Client)
	//
	miner.Initialize(globalConfig.Miner, ringClient.Chainclient)

	client = ethClient.Client
	client.NewContract(imp, implAddress, chainclient.CurrentImplAbiStr)

	var lrcTokenAddressHex string
	imp.LrcTokenAddress.Call(&lrcTokenAddressHex, "pending")
	lrcTokenAddress := types.HexToAddress(lrcTokenAddressHex)
	lrcToken := &chainclient.Erc20Token{}
	client.NewContract(lrcToken, lrcTokenAddress.Hex(), chainclient.Erc20TokenAbiStr)

	var registryAddressHex string
	imp.RinghashRegistryAddress.Call(&registryAddressHex, "pending")
	registryAddress := types.HexToAddress(registryAddressHex)
	client.NewContract(registry, registryAddress.Hex(), chainclient.CurrentRinghashRegistryAbiStr)
}

func createOrder(tokenS, tokenB types.Address, amountS, amountB *big.Int, pkBytes []byte) *types.Order {
	order := &types.Order{}
	order.Protocol = types.HexToAddress(implAddress)
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

func TestRingClient_NewRing(t *testing.T) {
	order1 := createOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		big.NewInt(100),
		big.NewInt(1000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
	)
	order1.Owner = types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")

	order2 := createOrder(
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		big.NewInt(1000),
		big.NewInt(100),
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
	)
	order2.Owner = types.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A")

	ring := &types.Ring{}
	ring.Orders = make([]*types.FilledOrder, 0)
	fOrder1 := &types.FilledOrder{}
	fOrder1.OrderState = types.OrderState{}
	fOrder1.OrderState.RawOrder = *order1
	//fOrder1.RateAmountS = order1.AmountS
	//fOrder1.RateAmountS = big.NewInt(99)

	fOrder1.FeeSelection = uint8(0)

	fOrder2 := &types.FilledOrder{}
	fOrder2.OrderState = types.OrderState{}
	fOrder2.OrderState.RawOrder = *order2
	//fOrder2.RateAmountS = order2.AmountS
	//fOrder2.RateAmountS = big.NewInt(999)
	fOrder2.FeeSelection = uint8(0)

	cTest := &chainclient.Erc20Token{}
	client.NewContract(cTest, "0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511", chainclient.Erc20TokenAbiStr)

	cTest1 := &chainclient.Erc20Token{}
	client.NewContract(cTest1, "0x96124db0972e3522a9b3910578b3f2e1a50159c7", chainclient.Erc20TokenAbiStr)

	miner.LoopringInstance.Tokens[types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511")] = cTest
	miner.LoopringInstance.Tokens[types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7")] = cTest1

	miner.AvailableAmountS(fOrder1)
	t.Log(fOrder1.AvailableAmountS.Int64())
	ring.Orders = append(ring.Orders, fOrder1)
	ring.Orders = append(ring.Orders, fOrder2)
	ring.Hash = ring.GenerateHash()
	ring.ThrowIfTokenAllowanceOrBalanceIsInsuffcient = false

	t.Logf("ring.Hash:%x", ring.Hash)

	ringState := &types.RingState{}
	ringState.RawRing = ring
	miner.PriceRateCVSquare(ringState)
	//ringClient.NewRing(ringState)
}

func TestB(t *testing.T) {
	//rateRatios := []*big.Int{big.NewInt(1), big.NewInt(5), big.NewInt(6), big.NewInt(8), big.NewInt(10), big.NewInt(40), big.NewInt(65), big.NewInt(88)}
	////rateRatios := []*big.Int{big.NewInt(0), big.NewInt(10), big.NewInt(20), big.NewInt(30), big.NewInt(40)}
	//cvs := miner.CVSquare(rateRatios, big.NewInt(1000))
	//println(cvs.Int64())
	//RATE_RATIO_SCALE := big.NewInt(10000)
	//ratio := new(big.Int).Set(RATE_RATIO_SCALE)
	//ratio.Mul(ratio, big.NewInt(100)).Div(ratio, big.NewInt(1))
	//rateRatios = append(rateRatios, ratio)
	//
	//t.Log(rateRatios[0].Int64())

	var s string = "address[1][2][3]"
	fullTypeRegex1 := regexp.MustCompile(`([a-zA-Z0-9]+(?:(?:\[[0-9]*\])*))(\[([0-9]*)\])`)
	var res []string
	if res = fullTypeRegex1.FindAllStringSubmatch(s, -1)[0]; len(res) == 0 {
		println(s)
	} else {

	}

	typeRegex := regexp.MustCompile("([a-zA-Z]+)(([0-9]+)(x([0-9]+))?)?")

	//println(len(res))
	println("aaa", len(res))
	println(res[0], res[1], res[2], res[3], "aa")
	res = typeRegex.FindAllStringSubmatch(s, -1)[0]
	println(res[0], res[1])
}

func TestCVSquare(t *testing.T) {

	rateRatios := []*big.Int{big.NewInt(1), big.NewInt(5), big.NewInt(6), big.NewInt(8), big.NewInt(10), big.NewInt(40), big.NewInt(65), big.NewInt(88)}
	//rateRatios := []*big.Int{big.NewInt(0), big.NewInt(10)}
	scale := big.NewInt(10000)
	cvs := miner.CVSquare(rateRatios, scale)
	println("cvs:", cvs.Int64())

	type Contract struct {
		chainclient.Contract
		Cvsquare chainclient.AbiMethod
	}

	c := &Contract{}
	client.NewContract(c, "0xb002cf4f4595a0e19c21913ce7c6b2678dc17167", `[{"constant":true,"inputs":[{"name":"arr","type":"uint256[]"},{"name":"scale","type":"uint256"}],"name":"cvsquare","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`)
	var res types.Big
	c.Cvsquare.Call(&res, "pending", rateRatios, scale)
	println("res:", res.Int64())
}

func TestCSV(t *testing.T) {
	scale := big.NewInt(10000)
	//e1 := big.NewInt(181)
	//e1.Mul(e1, scale)
	//e1.Div(e1, big.NewInt(200))
	//e2 := big.NewInt(89)
	//e2.Mul(e2, scale)
	//e2.Div(e2, big.NewInt(100))
	//e3 := big.NewInt(8999)
	//e3.Mul(e3, scale)
	//e3.Div(e3, big.NewInt(10000))
	//rateRatios := []*big.Int{e1,e2, e3}
	rateRatios := []*big.Int{big.NewInt(100), big.NewInt(200)}
	cvs := miner.CVSquare(rateRatios, scale)
	println("cvs:", cvs.Int64())

	nums := []float64{1.0, 10000000000.0}
	//cvsF(nums, 100.0)
	//cvsF1(nums, 100.0)
	//cvsF2(nums, 0.55)
	for i := 100000000; i <= 100000000000000000; i = i * 10 {
		nums = []float64{10000000000000.0, float64(i)}
		//cvsF2(nums, 0)
		cvsF2(nums, 0.9)
	}
	//length:2 [1.0,2.0] a=1 max:2 [1.0, 100000.0] a=1 max:2
	//length:3
}

func cvsF(nums []float64, a float64) {
	length := float64(len(nums))
	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	avg := sum / length

	s1 := 0.0
	for _, n := range nums {
		s1 += (n - avg) * (n - avg)
	}

	println(s1)
	cv := s1 / (avg * avg)
	println(cv)
}

func cvsF1(nums []float64, a float64) {
	length := float64(len(nums))

	sum := 0.0
	for _, n := range nums {
		sum += n
	}
	sum1 := 0.0
	for _, n := range nums {
		sum1 = sum1 + (n*length-sum)*(n*length-sum)
	}

	println(sum1 / (sum * sum))
}

func cvsF2(nums []float64, a float64) {
	neg := -1.0
	length := float64(len(nums))

	sum := 0.0
	for _, n := range nums {
		n1 := 1 / n
		sum = sum + neg*n1
	}
	sum1 := 0.0
	for _, n := range nums {
		n1 := neg * 1 / n
		sum1 = sum1 + (n1*length-sum)*(n1*length-sum)
	}

	sum2 := 0.0
	for _, n := range nums {
		n1 := neg * 1 / n
		sum2 += a + n1
	}
	println("last:", nums[1], "p:", a, "cvs", sum1/(sum2*sum2))

	//s := 0.0
	//x1 := 2*(-0.9/100000000) + (-0.9/100000000 - 0.9/100000000000000000)
	//x2 := 2*(-0.9/100000000000000000) + (-0.9/100000000 - 0.9/100000000000000000)
	//s = (x1*x1 + x2*x2)/((0.9 - 0.9/100000000000000000 - 0.9/100000000) * (0.9 - 0.9/100000000000000000 - 0.9/100000000))
	//println(s)
}
