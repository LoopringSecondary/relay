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

package chainclient_test

import (
	"github.com/Loopring/ringminer/chainclient"
	"github.com/Loopring/ringminer/chainclient/eth"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/crypto"
	ethCryptoLib "github.com/Loopring/ringminer/crypto/eth"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
	"time"
)

var client *chainclient.Client
var imp *chainclient.LoopringProtocolImpl = &chainclient.LoopringProtocolImpl{}
var implAddress string = "0xfd9ecf92e3684451c2502cf9cdc45962a4febffa"
var registry *chainclient.LoopringRinghashRegistry = &chainclient.LoopringRinghashRegistry{}

func init() {
	globalConfig := config.LoadConfig("../config/ringminer.toml")
	log.Initialize(globalConfig.Log)

	crypto.CryptoInstance = &ethCryptoLib.EthCrypto{Homestead: false}

	ethClient := eth.NewChainClient(globalConfig.ChainClient)
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

func TestLoopringRingHash(t *testing.T) {

	order1 := createOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		big.NewInt(1000),
		big.NewInt(100000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
	)

	order2 := createOrder(
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		big.NewInt(100000),
		big.NewInt(1000),
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
	)

	var res1 string
	vList := []uint8{order1.V, order2.V, order1.V}
	rList := [][]byte{order1.R.Bytes(), order2.R.Bytes(), order1.R.Bytes()}
	sList := [][]byte{order1.S.Bytes(), order2.S.Bytes(), order1.S.Bytes()}
	err := registry.CalculateRinghash.Call(&res1, "pending", big.NewInt(2), vList, rList, sList)
	if nil != err {
		t.Error(err.Error())
	}
	t.Log(res1)
	ring := &types.Ring{}
	ring.Orders = make([]*types.FilledOrder, 0)
	fOrder1 := &types.FilledOrder{}
	fOrder1.OrderState = types.OrderState{}
	fOrder1.OrderState.RawOrder = *order1
	fOrder2 := &types.FilledOrder{}
	fOrder2.OrderState = types.OrderState{}
	fOrder2.OrderState.RawOrder = *order2
	ring.Orders = append(ring.Orders, fOrder1)
	ring.Orders = append(ring.Orders, fOrder2)
	t.Log(ring.GenerateHash().Hex())
}

func TestLoopringSigerToAddress(t *testing.T) {
	/**
		pragma solidity ^0.4.0;
		contract VTest {
			function calculateSignerAddress(
			bytes32 hash,
			uint8 v,
			bytes32 r,
			bytes32 s)
			public
			constant
			returns (address) {

			return ecrecover(
			    keccak256("\x19Ethereum Signed Message:\n32", hash),
			    v,
			    r,
			    s);
			}
	        }
	*/
	type VTest struct {
		chainclient.Contract
		CalculateSignerAddress chainclient.AbiMethod
		CalculateHashUint8     chainclient.AbiMethod
		CalculateHashUint      chainclient.AbiMethod
	}
	order := createOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		big.NewInt(1000),
		big.NewInt(100000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
	)

	vTest := &VTest{}
	client.NewContract(vTest, "0x75472d53ed6624cfa81a61b09175a32b2886ce58", `[{"constant":true,"inputs":[{"name":"i","type":"uint8"}],"name":"calculateHashUint8","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"hash","type":"bytes32"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"calculateSignerAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"addr","type":"address"}],"name":"calculateHashAddress","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"i","type":"uint256"}],"name":"calculateHashUint","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"b","type":"bool"}],"name":"calculateHashBool","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"}]`)
	var err error
	bs, _ := crypto.CryptoInstance.VRSToSig(byte(order.V), order.R.Bytes(), order.S.Bytes())
	addrBytes, err1 := crypto.CryptoInstance.SigToAddress(order.Hash.Bytes(), bs)
	if err1 != nil {
		t.Error(err1)
	} else {
		t.Logf("address calculated by local %x", addrBytes)
	}

	var res string
	err = vTest.CalculateSignerAddress.Call(&res, "pending", order.Hash.Bytes(), order.V, order.R.Bytes(), order.S.Bytes())
	if err != nil {
		t.Error(err)
	} else {
		t.Logf("address calculated by local %x", types.HexToAddress(res).Bytes())
	}
}

func TestNewContract(t *testing.T) {

	//var c *chainclient.Contract
	//c := &chainclient.Erc20Token{}
	//
	//erc20TokenAbiStr := `[{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"totalSupply","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_from","type":"address"},{"indexed":true,"name":"_to","type":"address"},{"indexed":false,"name":"_value","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_owner","type":"address"},{"indexed":true,"name":"_spender","type":"address"},{"indexed":false,"name":"_value","type":"uint256"}],"name":"Approval","type":"event"}]`
	//eth.NewContract(c, "0x211c9fb2c5ad60a31587a4a11b289e37ed3ea520", erc20TokenAbiStr)
	//var amount types.Big
	//err := c.BalanceOf.Call(&amount, "pending", common.HexToAddress("0xd86ee51b02c5ac295e59711f4335fed9805c0148"))
	//if err != nil {
	//	println(err.Error())
	//}
	//println(amount.Int64())

	type CTest struct {
		chainclient.Contract
		CalculateHashUint chainclient.AbiMethod
		CalculateHashBool chainclient.AbiMethod
	}

	abiStr := `[{"constant":true,"inputs":[{"name":"i","type":"uint8"}],"name":"calculateHashUint8","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"hash","type":"bytes32"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"calculateSignerAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"addr","type":"address"}],"name":"calculateHashAddress","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"i","type":"uint256"}],"name":"calculateHashUint","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"b","type":"bool"}],"name":"calculateHashBool","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"}]`
	config1 := &config.ChainClientOptions{}
	config1.RawUrl = "http://127.0.0.1:8545"
	ethClient := eth.NewChainClient(*config1)
	//var c *chainclient.Contract
	c := &CTest{}
	ethClient.NewContract(c, "0x75472d53ed6624cfa81a61b09175a32b2886ce58", abiStr)
	var amount types.Big
	//c1 := (*CTest)(c)
	err := c.CalculateHashBool.Call(&amount, "pending", true)
	if nil != err {
		t.Log(err.Error())
	}
	t.Log(types.ToHex(amount.BigInt().Bytes()))
}

func TestRingHashRegistry(t *testing.T) {
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
	fOrder1.RateAmountS = order1.AmountS
	fOrder1.FeeSelection = uint8(0)
	fOrder2 := &types.FilledOrder{}
	fOrder2.OrderState = types.OrderState{}
	fOrder2.OrderState.RawOrder = *order2
	fOrder2.RateAmountS = order2.AmountS
	fOrder2.FeeSelection = uint8(0)

	ring.Orders = append(ring.Orders, fOrder1)
	ring.Orders = append(ring.Orders, fOrder2)
	ring.Hash = ring.GenerateHash()
	ring.ThrowIfTokenAllowanceOrBalanceIsInsuffcient = false

	t.Logf("ring.Hash:%x", ring.Hash)
	_, _, _, _, vList, rList, sList, _, _, _ := generateAddressList(ring)

	vList = append(vList, ring.V)
	rList = append(rList, ring.R.Bytes())
	sList = append(sList, ring.S.Bytes())
	var res string

	//res, err := registry.SubmitRinghash.SendTransaction(types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	//	big.NewInt(2),
	//	types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	//	vList,
	//	rList,
	//	sList,
	//)
	err := registry.RinghashFound.Call(&res, "pending", common.HexToHash("0x02b5c83d5df78a1c4e52c542ae263e865c8d06b6d390bdab330c5e109aea92f4"))

	if nil != err {
		t.Error(err)
	} else {
		t.Log(res)
	}
}

func TestSubmitRing(t *testing.T) {
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
	fOrder1.RateAmountS = order1.AmountS
	fOrder1.FeeSelection = uint8(0)
	fOrder2 := &types.FilledOrder{}
	fOrder2.OrderState = types.OrderState{}
	fOrder2.OrderState.RawOrder = *order2
	fOrder2.RateAmountS = order2.AmountS
	fOrder2.FeeSelection = uint8(0)

	ring.Orders = append(ring.Orders, fOrder1)
	ring.Orders = append(ring.Orders, fOrder2)
	ring.Hash = ring.GenerateHash()
	ring.ThrowIfTokenAllowanceOrBalanceIsInsuffcient = false

	t.Logf("ring.Hash:%x", ring.Hash)
	addressList, uintArgsList, uint8ArgsList, buyNoMoreThanAmountBList, vList, rList, sList, ringminer, feeRecepient, throwIfLRCIsInsuffcient := generateAddressList(ring)

	vList = append(vList, ring.V)
	rList = append(rList, ring.R.Bytes())
	sList = append(sList, ring.S.Bytes())
	var res string

	res, err := imp.SubmitRing.SendTransaction(types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
		addressList,
		uintArgsList,
		uint8ArgsList,
		buyNoMoreThanAmountBList,
		vList,
		rList,
		sList,
		ringminer,
		feeRecepient,
		throwIfLRCIsInsuffcient,
	)
	if nil != err {
		t.Error(err)
	} else {
		t.Log(res)
	}
}

func generateAddressList(ring *types.Ring) (addressList [][2]common.Address,
	uintArgsList [][7]*big.Int,
	uint8ArgsList [][2]uint8,
	buyNoMoreThanAmountBList []bool,
	vList []uint8,
	rList, sList [][]byte,
	ringminer common.Address,
	feeRecepient common.Address,
	throwIfLRCIsInsuffcient bool,
) {
	addressList = [][2]common.Address{}
	uintArgsList = [][7]*big.Int{}
	uint8ArgsList = [][2]uint8{}
	buyNoMoreThanAmountBList = make([]bool, 0)
	vList = make([]uint8, 0)
	rList = make([][]byte, 0)
	sList = make([][]byte, 0)

	for _, filledOrder := range ring.Orders {
		order := filledOrder.OrderState.RawOrder
		addressList = append(addressList, [2]common.Address{common.BytesToAddress(order.Owner.Bytes()), common.BytesToAddress(order.TokenS.Bytes())})

		uintArgsList = append(uintArgsList, [7]*big.Int{order.AmountS, order.AmountB, order.Timestamp, order.Ttl, order.Salt, order.LrcFee, filledOrder.RateAmountS})

		uint8ArgsList = append(uint8ArgsList, [2]uint8{order.MarginSplitPercentage, filledOrder.FeeSelection})

		buyNoMoreThanAmountBList = append(buyNoMoreThanAmountBList, order.BuyNoMoreThanAmountB)

		vList = append(vList, order.V)
		rList = append(rList, order.R.Bytes())
		sList = append(sList, order.S.Bytes())

		ringminer = common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")
		feeRecepient = common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")
		throwIfLRCIsInsuffcient = ring.ThrowIfTokenAllowanceOrBalanceIsInsuffcient

	}

	return addressList, uintArgsList, uint8ArgsList, buyNoMoreThanAmountBList, vList, rList, sList, ringminer, feeRecepient, throwIfLRCIsInsuffcient
}

func TestA1(t *testing.T) {
	//fullTypeRegex := regexp.MustCompile(`(\[([0-9]*)\])?((?:(?:\[[0-9]*\])*)[a-zA-Z0-9]+)`)
	////fullTypeRegex := regexp.MustCompile(`([a-zA-Z0-9]+(?:(?:\[[0-9]*\])*))(\[([0-9]*)\])`)
	//s := "address[34][56]"
	//res := fullTypeRegex.FindAllStringSubmatch(s, -1)[0]
	//println("s0", res[0],"s1", res[1],"s2", res[2],"s3", res[3])
	//
	//res = fullTypeRegex.FindAllStringSubmatch(res[3], -1)[0]
	//println("s0", res[0],"s1", res[1],"s2", res[2],"s3", res[3])
	//
	//res = fullTypeRegex.FindAllStringSubmatch(res[3], -1)[0]
	//println("s0", res[0],"s1", res[1],"s2", res[2],"s3", res[3])

	//uintArgsList := [2][]*big.Int{}
	//
	//for i:=0;i<=2;i++ {
	//	uintArgsList[0] = append(uintArgsList[0], big.NewInt(int64(i)))
	//	uintArgsList[1] = append(uintArgsList[1], big.NewInt(int64(i*100)))
	//}
	//for _,a := range uintArgsList {
	//	for _,b := range a {
	//		println(b.Int64())
	//	}
	//}

	var res string
	if err := client.GetTransactionReceipt(&res, "0x71aa2e2d4137b1c4b8e847f04bba8877005f212dec451f89ada3eb220477da56"); nil != err {
		println(err.Error())
	} else {
		println(res)
	}
}

func TestB(t *testing.T) {
	type CTest struct {
		chainclient.Contract
		MutilAddress chainclient.AbiMethod
	}

	cTest := &CTest{}
	client.NewContract(cTest, "0x23bbc1ca3bb70c609122cb8f86fc7c4f106fb962", `[{"constant":true,"inputs":[{"name":"addressList","type":"address[2][]"}],"name":"mutilAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"}]`)
	addressList := [][2]common.Address{}
	addressList = append(addressList, [2]common.Address{common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"), common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")})
	var err error
	var res string
	err = cTest.MutilAddress.Call(&res, "pending", addressList)
	if nil != err {
		t.Error(err)
	} else {
		t.Log(res)
	}

}

func TestFilter(t *testing.T) {
	var filterId string
	filterReq := &eth.FilterQuery{}
	filterReq.Address = []common.Address{
		common.HexToAddress("0xfd9ecf92e3684451c2502cf9cdc45962a4febffa"),
		common.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		common.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
	}
	filterReq.FromBlock = "latest"
	filterReq.ToBlock = "latest"
	//todo:topics, eventId
	//filterReq.Topics =
	if err := client.NewFilter(&filterId, filterReq); nil != err {
		t.Error(err)
	} else {

		logChan := make(chan []eth.Log)
		if err := client.Subscribe(&logChan, filterId); nil != err {
			log.Errorf("error:%s", err.Error())
		} else {
			for {
				select {
				case logs := <-logChan:
					for _, log1 := range logs {
						println(log1.BlockHash)
					}
				}
			}
		}
	}

}

func TestApprove(t *testing.T) {
	/**
	  HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
	  		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
	*/
	cTest := &chainclient.Erc20Token{}
	client.NewContract(cTest, "0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511", chainclient.Erc20TokenAbiStr)
	var res string
	var err error
	//if res,err = cTest.Approve.SendTransaction(types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	//	common.HexToAddress("0xfd9ecf92e3684451c2502cf9cdc45962a4febffa"),
	//	big.NewInt(2000),
	//); nil != err {
	//	t.Error(err)
	//} else {
	//	t.Log(res)
	//}
	//if res,err = cTest.Approve.SendTransaction(types.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A"),
	//	common.HexToAddress("0xfd9ecf92e3684451c2502cf9cdc45962a4febffa"),
	//	big.NewInt(2000),
	//); nil != err {
	//	t.Error(err)
	//} else {
	//	t.Log(res)
	//}
	//
	//client.NewContract(cTest, "0x96124db0972e3522a9b3910578b3f2e1a50159c7",chainclient.Erc20TokenAbiStr)
	//if res,err = cTest.Approve.SendTransaction(types.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A"),
	//	common.HexToAddress("0xfd9ecf92e3684451c2502cf9cdc45962a4febffa"),
	//	big.NewInt(2000),
	//); nil != err {
	//	t.Error(err)
	//} else {
	//	t.Log(res)
	//}
	//if res,err = cTest.Approve.SendTransaction(types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	//	common.HexToAddress("0xfd9ecf92e3684451c2502cf9cdc45962a4febffa"),
	//	big.NewInt(2000),
	//); nil != err {
	//	t.Error(err)
	//} else {
	//	t.Log(res)
	//}
	res, err = cTest.Transfer.SendTransaction(types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
		common.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A"),
		big.NewInt(20000),
	)
	if nil != err {
		t.Error(err)
	} else {
		t.Log(res)
	}

}
