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

package ethaccessor_test

import (
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rlp"
)

func TestGenerateSubmitRingMethodInputsData(t *testing.T) {
	protocol := common.HexToAddress("0x456044789a41b277f033e4d79fab2139d69cd154")
	delegate := common.HexToAddress("0xa0af16edd397d9e826295df9e564b10d57e3c457")
	token1 := common.HexToAddress("0xe1C541BA900cbf212Bc830a5aaF88aB499931751")
	token2 := common.HexToAddress("0x639687b7f8501f174356d3acb1972f749021ccd0")
	feeReceipt := common.HexToAddress("0xAc399518Cd6415fF746dab204f3d3176F62035cD")
	protocolAbi := &abi.ABI{}
	if err := protocolAbi.UnmarshalJSON([]byte(`[{"constant":false,"inputs":[{"name":"addressList","type":"address[4][]"},{"name":"uintArgsList","type":"uint256[6][]"},{"name":"uint8ArgsList","type":"uint8[1][]"},{"name":"buyNoMoreThanAmountBList","type":"bool[]"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"},{"name":"feeRecipient","type":"address"},{"name":"feeSelections","type":"uint16"}],"name":"submitRing","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"}]`)); nil != err {
		t.Fatal(err.Error())
	}

	order1, err1 := generateOrder(protocol, delegate, token1, token2, big.NewInt(int64(10000)), big.NewInt(int64(10000)), "0xc20e06ac4531f99ca0ede0bc936cdebf3e19d91eba78b1c8e55ca09e731490fb")
	if nil != err1 {
		t.Fatal(err1.Error())
	}
	filledOrder1 := &types.FilledOrder{}
	filledOrder1.OrderState = *order1
	filledOrder1.FeeSelection = uint8(0)
	filledOrder1.RateAmountS = new(big.Rat).SetInt(order1.RawOrder.AmountS)

	order2, err2 := generateOrder(protocol, delegate, token2, token1, big.NewInt(int64(10000)), big.NewInt(int64(10000)), "0x117130880d2f345a1bb1ebe25979e369d2705cd2a68b77d5f3eafc77d9738fba")
	if nil != err2 {
		t.Fatal(err2.Error())
	}
	filledOrder2 := &types.FilledOrder{}
	filledOrder2.OrderState = *order2
	filledOrder2.FeeSelection = uint8(0)
	filledOrder2.RateAmountS = new(big.Rat).SetInt(order2.RawOrder.AmountS)

	ring := &types.Ring{}
	ring.Orders = []*types.FilledOrder{filledOrder1, filledOrder2}

	var callData []byte
	if callData, err1 = ethaccessor.GenerateSubmitRingMethodInputsData(ring, feeReceipt, protocolAbi); nil != err1 {
		t.Fatal(err1.Error())
	} else {
		t.Logf("inputsData:%s", common.ToHex(callData))
	}

	//send to ethaccessor
	//if err := sendToETH(callData, common.HexToAddress("0x750aD4351bB728ceC7d639A9511F9D6488f1E259"), protocol); nil != err {
	//	t.Fatal(err.Error())
	//}

}

func sendToETH(callData []byte, sender, protocol common.Address) error {
	test.LoadConfig()
	var nonce types.Big

	ethaccessor.GetTransactionCount(&nonce, sender, "latest")
	transaction := ethTypes.NewTransaction(nonce.Uint64(),
		protocol,
		big.NewInt(int64(0)),
		big.NewInt(int64(500000)),
		big.NewInt(int64(1000000000)),
		callData)

	if tx, err := crypto.SignTx(sender, transaction, nil); nil != err {
		return err
	} else {
		if txData, err := rlp.EncodeToBytes(tx); nil != err {
			return err
		} else {
			var txhash string
			return ethaccessor.SendRawTransaction(txhash, common.ToHex(txData))
		}
	}
}

func generateOrder(protocol, delegate, tokenS, tokenB common.Address, amountS, amountB *big.Int, ownerPrivateKeyStr string) (*types.OrderState, error) {
	lrcFee := big.NewInt(int64(100000))

	ownerPrivateKey, _ := crypto.NewPrivateKeyCrypto(false, ownerPrivateKeyStr)
	ownerAddr := ownerPrivateKey.Address()
	authPrivateKey, _ := crypto.NewPrivateKeyCrypto(false, "0x11a22b9b094422fef93eb6d37d3e6f7809d32e6965865bb403eaa6489a532d9d")
	order := &types.Order{}
	order.Protocol = protocol
	order.DelegateAddress = delegate
	order.TokenS = tokenS
	order.TokenB = tokenB
	order.AmountS = amountS
	order.AmountB = amountB
	order.ValidSince = big.NewInt(time.Now().Unix())
	order.ValidUntil = big.NewInt(time.Now().Unix() + 20000)
	order.LrcFee = lrcFee
	order.BuyNoMoreThanAmountB = false
	order.MarginSplitPercentage = 0
	order.Owner = ownerAddr
	order.PowNonce = 1
	order.AuthPrivateKey = authPrivateKey
	order.AuthAddr = order.AuthPrivateKey.Address()
	order.WalletAddress = ownerAddr
	order.Hash = order.GenerateHash()

	if sig, err := ownerPrivateKey.Sign(order.Hash.Bytes(), ownerAddr); nil != err {
		return nil, err
	} else {
		v, r, s := ownerPrivateKey.SigToVRS(sig)
		order.V = uint8(v)
		order.R = types.BytesToBytes32(r)
		order.S = types.BytesToBytes32(s)
	}

	state := &types.OrderState{}
	state.RawOrder = *order

	return state, nil
}
