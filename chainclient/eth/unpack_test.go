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

package eth_test

import (
	"github.com/Loopring/relay/chainclient/eth"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"strings"
	"testing"
)

// 字符串转地址必须使用hexToAddress
func TestAddress(t *testing.T) {
	account := "0x56d9620237fff8a6c0f98ec6829c137477887ec4"
	t.Log(account)
	t.Log(common.HexToAddress(account).String())
}

const (
	testAbiStr = `[
	{"constant":false,"inputs":[
		{"name":"hash","type":"bytes32"},
		{"name":"accountS","type":"address"},
		{"name":"accountB","type":"address"},
		{"name":"amountS","type":"uint256"},
		{"name":"amountB","type":"uint256"}],"name":"submitTransfer","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[
		{"name":"condition","type":"bool"},
		{"name":"message","type":"string"}],"name":"check","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":true,"inputs":[
		{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},
	{"constant":false,"inputs":[
		{"name":"_id","type":"bytes32"},
		{"name":"_owner","type":"address"},
		{"name":"_amount","type":"uint256"}],
		"name":"submitDeposit","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"anonymous":false,"inputs":[
		{"indexed":false,"name":"hash","type":"bytes32"},
		{"indexed":false,"name":"account","type":"address"},
		{"indexed":false,"name":"amount","type":"uint256"},
		{"indexed":false,"name":"ok","type":"bool"}],"name":"DepositFilled","type":"event"},
	{"anonymous":false,"inputs":[
		{"indexed":false,"name":"hash","type":"bytes32"},
		{"indexed":false,"name":"accountS","type":"address"},
		{"indexed":false,"name":"accountB","type":"address"},
		{"indexed":false,"name":"amountS","type":"uint256"},
		{"indexed":false,"name":"amountB","type":"uint256"},
		{"indexed":false,"name":"ok","type":"bool"}],"name":"OrderFilled","type":"event"},
	{"anonymous":false,"inputs":[
		{"indexed":false,"name":"message","type":"string"}],"name":"Exception","type":"event"}]
`

	lrcAbiStr string = `[{"constant":true,"inputs":[{"name":"signer","type":"address"},{"name":"hash","type":"bytes32"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"verifySignature","outputs":[],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"orderHash","type":"bytes32"}],"name":"getOrderCancelled","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_MAX_VALUE","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"filled","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"cancelled","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"MARGIN_SPLIT_PERCENTAGE_BASE","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"ringIndex","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"addresses","type":"address[3]"},{"name":"orderValues","type":"uint256[7]"},{"name":"buyNoMoreThanAmountB","type":"bool"},{"name":"marginSplitPercentage","type":"uint8"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"cancelOrder","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"RATE_RATIO_SCALE","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"lrcTokenAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"tokenRegistryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"addressList","type":"address[2][]"},{"name":"uintArgsList","type":"uint256[7][]"},{"name":"uint8ArgsList","type":"uint8[2][]"},{"name":"buyNoMoreThanAmountBList","type":"bool[]"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"},{"name":"ringminer","type":"address"},{"name":"feeRecepient","type":"address"},{"name":"throwIfLRCIsInsuffcient","type":"bool"}],"name":"submitRing","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"delegateAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"orderHash","type":"bytes32"}],"name":"getOrderFilled","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"maxRingSize","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"ringhashRegistryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"cutoff","type":"uint256"}],"name":"setCutoff","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_LRC","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"cutoffs","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"rateRatioCVSThreshold","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_MARGIN_SPLIT","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"inputs":[{"name":"_lrcTokenAddress","type":"address"},{"name":"_tokenRegistryAddress","type":"address"},{"name":"_ringhashRegistryAddress","type":"address"},{"name":"_delegateAddress","type":"address"},{"name":"_maxRingSize","type":"uint256"},{"name":"_rateRatioCVSThreshold","type":"uint256"}],"payable":false,"stateMutability":"nonpayable","type":"constructor"},{"payable":true,"stateMutability":"payable","type":"fallback"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_ringIndex","type":"uint256"},{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_ringhash","type":"bytes32"},{"indexed":true,"name":"_miner","type":"address"},{"indexed":true,"name":"_feeRecepient","type":"address"},{"indexed":false,"name":"_ringhashFound","type":"bool"}],"name":"RingMined","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_ringIndex","type":"uint256"},{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_ringhash","type":"bytes32"},{"indexed":false,"name":"_prevOrderHash","type":"bytes32"},{"indexed":true,"name":"_orderHash","type":"bytes32"},{"indexed":false,"name":"_nextOrderHash","type":"bytes32"},{"indexed":false,"name":"_amountS","type":"uint256"},{"indexed":false,"name":"_amountB","type":"uint256"},{"indexed":false,"name":"_lrcReward","type":"uint256"},{"indexed":false,"name":"_lrcFee","type":"uint256"}],"name":"OrderFilled","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_orderHash","type":"bytes32"},{"indexed":false,"name":"_amountCancelled","type":"uint256"}],"name":"OrderCancelled","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_address","type":"address"},{"indexed":false,"name":"_cutoff","type":"uint256"}],"name":"CutoffTimestampChanged","type":"event"}]`
)

func newAbi(definition string) abi.ABI {
	tabi, err := abi.JSON(strings.NewReader(definition))
	if err != nil {
		panic(err)
	}

	return tabi
}

func TestUnpackEvent(t *testing.T) {
	tabi := newAbi(testAbiStr)

	type DepositEvent struct {
		Hash    []byte         `alias:"hash"`
		Account common.Address `alias:"account"`
		Amount  *big.Int       `alias:"amount"`
		Ok      bool           `alias:"ok"`
	}

	event := DepositEvent{}
	name := "DepositFilled"
	str := "0x5ad6fe3e08ffa01bb1db674ac8e66c47511e364a4500115dd2feb33dad972d7e00000000000000000000000056d9620237fff8a6c0f98ec6829c137477887ec4000000000000000000000000000000000000000000000000000000000bebc2010000000000000000000000000000000000000000000000000000000000000001"

	data := hexutil.MustDecode(str)
	abievent, ok := tabi.Events[name]
	if !ok {
		t.Error("event do not exist")
	}

	if err := eth.UnpackEvent(abievent.Inputs, &event, data, []string{}); err != nil {
		panic(err)
	}

	t.Log(common.BytesToHash(event.Hash).Hex())
	t.Log(event.Account.Hex())
	t.Log(event.Amount)
	t.Log(event.Ok)
}

func TestUnpackTransaction(t *testing.T) {
	tabi := newAbi(testAbiStr)

	type Deposit struct {
		Id     []byte         `alias:"_id"`
		Owner  common.Address `alias:"_owner"`
		Amount *big.Int       `alias:"_amount"`
	}

	tx := "0x8a024a21000000000000000000000000000000000000000000000000000000000000000100000000000000000000000046c5683c754b2eba04b2701805617c0319a9b4dd000000000000000000000000000000000000000000000000000000001dcd6500"
	method, _ := tabi.Methods["submitDeposit"]
	out := &Deposit{}

	if err := eth.UnpackTransaction(method, out, tx); err != nil {
		panic(err)
	}

	t.Log(common.BytesToHash(out.Id).Hex())
	t.Log(out.Owner.Hex())
	t.Log(out.Amount.String())
}

func TestLrcUnpackRing(t *testing.T) {
	tabi := newAbi(lrcAbiStr)

	/*type RingSubmitArgs struct {
		AddressList              [][2]common.Address `alias:"addressList"`
		UintArgsList             [][7]*big.Int       `alias:"uintArgsList"`
		Uint8ArgsList            [][2]uint8          `alias:"uint8ArgsList"`
		BuyNoMoreThanAmountBList []bool              `alias:"buyNoMoreThanAmountBList"`
		VList                    []uint8             `alias:"vList"`
		RList                    [][]byte            `alias:"rList"`
		SList                    [][]byte            `alias:"sList"`
		Ringminer                common.Address      `alias:"ringminer"`
		FeeRecepient             common.Address      `alias:"feeRecepient"`
		ThrowIfLRCIsInsuffcient  bool                `alias:"throwIfLRCIsInsuffcient"`
	}*/

	type RingSubmitArgs struct {
		AddressList              []common.Address `alias:"addressList"`
		UintArgsList             []*big.Int       `alias:"uintArgsList"`
		Uint8ArgsList            []*big.Int       `alias:"uint8ArgsList"`
		BuyNoMoreThanAmountBList []bool           `alias:"buyNoMoreThanAmountBList"`
		VList                    []uint8          `alias:"vList"`
		RList                    [][]byte         `alias:"rList"`
		SList                    [][]byte         `alias:"sList"`
		Ringminer                common.Address   `alias:"ringminer"`
		FeeRecepient             common.Address   `alias:"feeRecepient"`
		ThrowIfLRCIsInsuffcient  bool             `alias:"throwIfLRCIsInsuffcient"`
	}

	tx := "0x64c86dda000000000000000000000000000000000000000000000000000000000000014000000000000000000000000000000000000000000000000000000000000001e000000000000000000000000000000000000000000000000000000000000003c0000000000000000000000000000000000000000000000000000000000000046000000000000000000000000000000000000000000000000000000000000004c0000000000000000000000000000000000000000000000000000000000000054000000000000000000000000000000000000000000000000000000000000005c0000000000000000000000000b5fab0b11776aad5ce60588c16bd59dcfd61a1c2000000000000000000000000b5fab0b11776aad5ce60588c16bd59dcfd61a1c200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000b5fab0b11776aad5ce60588c16bd59dcfd61a1c20000000000000000000000000c0b638ffccb4bdc4c0d0d5fef062fc512c9251100000000000000000000000048ff2269e58a373120ffdbbdee3fbcea854ac30a00000000000000000000000096124db0972e3522a9b3910578b3f2e1a50159c7000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000003e800000000000000000000000000000000000000000000000000000000000186a00000000000000000000000000000000000000000000000000000000059dee59a000000000000000000000000000000000000000000000000000000000000271000000000000000000000000000000000000000000000000000000000000003e8000000000000000000000000000000000000000000000000000000000000006400000000000000000000000000000000000000000000000000000000000003e800000000000000000000000000000000000000000000000000000000000186a000000000000000000000000000000000000000000000000000000000000003e80000000000000000000000000000000000000000000000000000000059dee59a000000000000000000000000000000000000000000000000000000000000271000000000000000000000000000000000000000000000000000000000000003e8000000000000000000000000000000000000000000000000000000000000006400000000000000000000000000000000000000000000000000000000000186a0000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000001b000000000000000000000000000000000000000000000000000000000000001c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000375c0ba3b509046f3d2121c39d75fc483be0e189c9477bc2898472ba0a56940062716ed79270d409911afa22599644bb925cbd954606bd868f1cb686cf5066f08000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000035be279edef04a941c48c60157be68650169503ff3a784de6707e9ff056a537137ae8546aa85f36707edbc7a53ce717c875ddd4f8acaf859d435ed43f8ed042b70000000000000000000000000000000000000000000000000000000000000000"
	method, _ := tabi.Methods["submitRing"]
	out := &RingSubmitArgs{}

	if err := eth.UnpackTransaction(method, out, tx); err != nil {
		panic(err)
	}

	for _, v := range out.AddressList {
		t.Log(v.Hex())
	}
	for _, v := range out.UintArgsList {
		t.Log(v.String())
	}
	for _, v := range out.Uint8ArgsList {
		t.Log(v.String())
	}
	for _, v := range out.RList {
		t.Log(common.Bytes2Hex(v))
	}
	t.Log(common.Bytes2Hex(out.VList))
}
