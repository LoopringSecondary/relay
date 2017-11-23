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
	"github.com/Loopring/relay/chainclient"
	"github.com/Loopring/relay/chainclient/eth"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

var testParams *test.TestParams

func init() {
	testParams = test.LoadConfigAndGenerateTestParams()
}

func TestLoopringRingHash(t *testing.T) {

	order1 := test.CreateOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		testParams.ImplAddress,
		big.NewInt(1000),
		big.NewInt(100000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)

	order2 := test.CreateOrder(
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		testParams.ImplAddress,
		big.NewInt(100000),
		big.NewInt(1000),
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)

	var res1 string
	t.Log(order1.V)
	t.Log(order2.V)
	vList := []uint8{order1.V - 27, order2.V - 27, order1.V - 27}
	rList := [][]byte{order1.R.Bytes(), order2.R.Bytes(), order1.R.Bytes()}
	sList := [][]byte{order1.S.Bytes(), order2.S.Bytes(), order1.S.Bytes()}
	err := testParams.Registry.CalculateRinghash.Call(&res1, "pending", big.NewInt(2), vList, rList, sList)
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
	ring.Miner = types.HexToAddress("0xB5FAB0B11776AAD5cE60588C16bd59DCfd61a1c2")
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
	order := test.CreateOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		testParams.ImplAddress,
		big.NewInt(1000),
		big.NewInt(100000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)

	vTest := &VTest{}
	testParams.Client.NewContract(vTest, "0x75472d53ed6624cfa81a61b09175a32b2886ce58", `[{"constant":true,"inputs":[{"name":"i","type":"uint8"}],"name":"calculateHashUint8","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"hash","type":"bytes32"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"calculateSignerAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"addr","type":"address"}],"name":"calculateHashAddress","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"i","type":"uint256"}],"name":"calculateHashUint","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"b","type":"bool"}],"name":"calculateHashBool","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"}]`)
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
	config1 := &config.AccessorOptions{}
	config1.RawUrl = "http://127.0.0.1:8545"
	ethClient := eth.NewChainClient(*config1, "sa")
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
	order1 := test.CreateOrder(
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		testParams.ImplAddress,
		big.NewInt(100),
		big.NewInt(1000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)
	order1.Owner = types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")

	order2 := test.CreateOrder(
		types.HexToAddress("0x96124db0972e3522a9b3910578b3f2e1a50159c7"),
		types.HexToAddress("0x0c0b638ffccb4bdc4c0d0d5fef062fc512c92511"),
		testParams.ImplAddress,
		big.NewInt(1000),
		big.NewInt(100),
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)
	order2.Owner = types.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A")

	ring := &types.Ring{}
	ring.Orders = make([]*types.FilledOrder, 0)
	fOrder1 := &types.FilledOrder{}
	fOrder1.OrderState = types.OrderState{}
	fOrder1.OrderState.RawOrder = *order1
	fOrder1.RateAmountS = new(big.Rat).SetInt(order1.AmountS)
	fOrder1.FeeSelection = uint8(0)
	fOrder2 := &types.FilledOrder{}
	fOrder2.OrderState = types.OrderState{}
	fOrder2.OrderState.RawOrder = *order2
	fOrder2.RateAmountS = new(big.Rat).SetInt(order2.AmountS)
	fOrder1.FeeSelection = uint8(0)
	fOrder2.FeeSelection = uint8(0)

	ring.Orders = append(ring.Orders, fOrder1)
	ring.Orders = append(ring.Orders, fOrder2)
	ring.Miner = types.HexToAddress("0xB5FAB0B11776AAD5cE60588C16bd59DCfd61a1c2")

	ring.Hash = ring.GenerateHash()

	t.Logf("ring.Hash:%x", ring.Hash)
	ringSubmitArgs := ring.GenerateSubmitArgs(testParams.MinerPrivateKey)

	var res string

	t.Logf("order1.V:%d,order1.R:%s, order1.V1:%d, order1.R1:%s", order1.V, order2.R.Hex(), ringSubmitArgs.VList[2],
		types.BytesToSign(ringSubmitArgs.RList[2]).Hex(),
	)
	//vList := []uint8{ringSubmitArgs.VList[0]-27, ringSubmitArgs.VList[1]-27, ringSubmitArgs.VList[2]-27}

	t.Log(ringSubmitArgs.Ringminer.Hex())
	err := testParams.Registry.CalculateRinghash.Call(&res, "pending",
		ringSubmitArgs.Ringminer,
		big.NewInt(2),
		ringSubmitArgs.VList,
		ringSubmitArgs.RList,
		ringSubmitArgs.SList,
	)
	//err := registry.RinghashFound.Call(&res, "pending", common.HexToHash("0x02b5c83d5df78a1c4e52c542ae263e865c8d06b6d390bdab330c5e109aea92f4"))
	//
	if nil != err {
		t.Error(err)
	} else {
		t.Log(res)
	}
}

func TestSubmitRing(t *testing.T) {
	order1 := test.CreateOrder(
		types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"),
		types.HexToAddress("0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f"),
		testParams.ImplAddress,
		big.NewInt(100),
		big.NewInt(1000),
		types.Hex2Bytes("11293da8fdfe3898eae7637e429e7e93d17d0d8293a4d1b58819ac0ca102b446"),
		types.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2"),
	)

	order2 := test.CreateOrder(
		types.HexToAddress("0x8711ac984e6ce2169a2a6bd83ec15332c366ee4f"),
		types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"),
		testParams.ImplAddress,
		big.NewInt(1000),
		big.NewInt(100),
		types.Hex2Bytes("07ae9ee56203d29171ce3de536d7742e0af4df5b7f62d298a0445d11e466bf9e"),
		types.HexToAddress("0x48ff2269e58a373120FFdBBdEE3FBceA854AC30A"),
	)

	ring := &types.Ring{}
	ring.Orders = make([]*types.FilledOrder, 0)
	fOrder1 := &types.FilledOrder{}
	fOrder1.OrderState = types.OrderState{}
	fOrder1.OrderState.RawOrder = *order1
	fOrder1.RateAmountS = new(big.Rat).SetInt(order1.AmountS)
	fOrder1.FeeSelection = uint8(0)
	t.Logf("order1.h : %s, order2.h : %s", order1.Hash.Hex(), order2.R.Hex())
	fOrder2 := &types.FilledOrder{}
	fOrder2.OrderState = types.OrderState{}
	fOrder2.OrderState.RawOrder = *order2
	fOrder2.RateAmountS = new(big.Rat).SetInt(order2.AmountS)
	fOrder2.FeeSelection = uint8(0)

	ring.Orders = append(ring.Orders, fOrder1)
	ring.Orders = append(ring.Orders, fOrder2)
	ring.Miner = types.HexToAddress("0xB5FAB0B11776AAD5cE60588C16bd59DCfd61a1c2")
	ring.Hash = ring.GenerateHash()

	t.Logf("ring.Hash:%x", ring.Hash)

	var res string
	var err error
	ringSubmitArgs := ring.GenerateSubmitArgs(testParams.MinerPrivateKey)
	//
	//err = imp.GetSpendable.Call(&res, "pending", common.HexToAddress(order1.TokenS.Hex()), common.HexToAddress(order1.Owner.Hex()))
	//if nil != err {
	//	t.Error(err)
	//} else {
	//	t.Log(res)
	//}
	res, err = testParams.Imp.SubmitRing.SendTransaction(types.HexToAddress("0x"),
		ringSubmitArgs.AddressList,
		ringSubmitArgs.UintArgsList,
		ringSubmitArgs.Uint8ArgsList,
		ringSubmitArgs.BuyNoMoreThanAmountBList,
		ringSubmitArgs.VList,
		ringSubmitArgs.RList,
		ringSubmitArgs.SList,
		ringSubmitArgs.Ringminer,
		ringSubmitArgs.FeeRecepient,
	)
	if nil != err {
		t.Error(err)
	} else {
		t.Log(res)
	}
	//err = registry.CalculateRinghash.Call(&res, "pending",
	//	ringSubmitArgs.Ringminer,
	//	big.NewInt(2),
	//	ringSubmitArgs.VList,
	//	ringSubmitArgs.RList,
	//	ringSubmitArgs.SList,
	//)
	//if nil != err {
	//	t.Error(err)
	//} else {
	//	t.Log(res)
	//}
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
	if err := testParams.Client.NewFilter(&filterId, filterReq); nil != err {
		t.Error(err)
	} else {

		logChan := make(chan []eth.Log)
		if err := testParams.Client.Subscribe(&logChan, filterId); nil != err {
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

func TestClient_BlockIterator(t *testing.T) {
	iterator := testParams.Client.BlockIterator(big.NewInt(4069), big.NewInt(4080), false)
	for {
		_, err := iterator.Next()
		if nil != err {
			println(err.Error())
			break
		} else {
			//block := b.(eth.Block)
			//println(block.Number.BigInt().String(), block.Hash.Hex())
		}
	}
}
