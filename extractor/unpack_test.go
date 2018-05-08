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

package extractor_test

import (
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
	"testing"
)

func init() {
	c := test.Cfg()
	println(c.Owner.Name)
}

func TestExtractorServiceImpl_UnpackSubmitRingMethod(t *testing.T) {
	input := "0xe78aadb20000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000024000000000000000000000000000000000000000000000000000000000000003e0000000000000000000000000000000000000000000000000000000000000044000000000000000000000000000000000000000000000000000000000000004a0000000000000000000000000000000000000000000000000000000000000054000000000000000000000000000000000000000000000000000000000000005e00000000000000000000000003acdf3e3d8ec52a768083f718e763727b021065000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000bb27332611e3f6372b37ef7d728a6f3a881f9391000000000000000000000000c02aaa39b223fe8d0a0e5c4f27ead9083c756cc2000000000000000000000000b94065482ad64d4c2b9252358d746b39e820a5820000000000000000000000008b0f7dad5a9a64c895fe54612b6949286d55f37c000000000000000000000000b94065482ad64d4c2b9252358d746b39e820a5820000000000000000000000001b793e49237758dbd8b752afc9eb4b329d5da016000000000000000000000000b94065482ad64d4c2b9252358d746b39e820a5820000000000000000000000002ffd520ac4d79caa0c1a7e5f1f7b37b4444da378000000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000025fa7a1faa41a00000000000000000000000000000000000000000000000026493009208ba100000000000000000000000000000000000000000000000000000000000005af07096000000000000000000000000000000000000000000000000000000005af1c21600000000000000000000000000000000000000000000000042c08d83f9a1000000000000000000000000000000000000000000000000000025fa7a1faa41a0000000000000000000000000000000000000000000000000056bc75e2d6310000000000000000000000000000000000000000000000000000000560a24872ba000000000000000000000000000000000000000000000000000000000005af1178e000000000000000000000000000000000000000000000000000000005af2690e00000000000000000000000000000000000000000000000005698eef066700000000000000000000000000000000000000000000000000056bc75e2d631000000000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000003200000000000000000000000000000000000000000000000000000000000000320000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000001b000000000000000000000000000000000000000000000000000000000000001b000000000000000000000000000000000000000000000000000000000000001c000000000000000000000000000000000000000000000000000000000000001b0000000000000000000000000000000000000000000000000000000000000004c14c87a401d3200487730431cc60cfbfd75086f24fc21819b91b2fe3d420b98a325b0fb4071bb3f7efef29bf567dd24ff05757c97af576be02ab8164b9a95c7a63936adf69c67c360d754b81d3980710ad0273cc9e2c5d8251363d8004e0b42e9dfb1ef1034d7939f70acf5cb717f73de4c13673ea9480bb8f2b39284dbcd49a00000000000000000000000000000000000000000000000000000000000000047faf5a6cc75d4aaf0ca207eafc1b3113c241dc3517cfa6fb7c06c79d901cff007b769c119d59a95252e008b3a35963dbd94f41f581c76b4d2af3b23d5daa33b97876371bd88382688660f2538a8681e14c68d62252c2a104738a08aa2f94d4496b5dbd58a834895e0cded1adc5068ee4771b0f1f23126491ad5f762df8683ea6"

	var ring ethaccessor.SubmitRingMethodInputs

	data := hexutil.MustDecode("0x" + input[10:])

	if err := ethaccessor.ProtocolImplAbi().UnpackMethodInput(&ring, "submitRing", data); err != nil {
		t.Fatalf(err.Error())
	}

	event, err := ring.ConvertDown()
	if err != nil {
		t.Fatalf(err.Error())
	}

	for k, v := range event.OrderList {
		t.Log(k, "protocol", v.Protocol.Hex())
		t.Log(k, "tokenS", v.TokenS.Hex())
		t.Log(k, "tokenB", v.TokenB.Hex())

		t.Log(k, "amountS", v.AmountS.String())
		t.Log(k, "amountB", v.AmountB.String())
		t.Log(k, "validSince", v.ValidSince.String())
		t.Log(k, "validUntil", v.ValidUntil.String())
		t.Log(k, "lrcFee", v.LrcFee.String())
		t.Log(k, "rateAmountS", ring.UintArgsList[k][5].String())

		t.Log(k, "marginSplitpercentage", v.MarginSplitPercentage)
		t.Log(k, "feeSelectionList", ring.Uint8ArgsList[k][0])

		t.Log(k, "buyNoMoreThanAmountB", v.BuyNoMoreThanAmountB)

		t.Log(k, "v", v.V)
		t.Log(k, "s", v.S.Hex())
		t.Log(k, "r", v.R.Hex())
	}

	t.Log("feeReceipt", event.FeeReceipt.Hex())
	t.Log("feeSelection", event.FeeSelection)
}

func TestExtractorServiceImpl_UnpackWethWithdrawalMethod(t *testing.T) {
	input := "0x2e1a7d4d0000000000000000000000000000000000000000000000000000000000000064"

	var withdrawal ethaccessor.WethWithdrawalMethod

	data := hexutil.MustDecode("0x" + input[10:])

	if err := ethaccessor.WethAbi().UnpackMethodInput(&withdrawal.Value, "withdraw", data); err != nil {
		t.Fatalf(err.Error())
	}

	evt := withdrawal.ConvertDown()
	t.Logf("withdrawal event value:%s", evt.Amount)
}

func TestExtractorServiceImpl_UnpackCancelOrderMethod(t *testing.T) {
	input := "0x8c59f7ca000000000000000000000000b1018949b241d76a1ab2094f473e9befeabb5ead000000000000000000000000480037780d0b0e766941b8c5e99e685bf8812c39000000000000000000000000f079e0612e869197c5f4c7d0a95df570b163232b000000000000000000000000b1018949b241d76a1ab2094f473e9befeabb5ead00000000000000000000000047fe1648b80fa04584241781488ce4c0aaca23e400000000000000000000000000000000000000000000003635c9adc5dea00000000000000000000000000000000000000000000000000000016345785d8a0000000000000000000000000000000000000000000000000000000000005ad8a62f000000000000000000000000000000000000000000000000000000005b5c7c2f00000000000000000000000000000000000000000000000029a2241af62c00000000000000000000000000000000000000000000000000001bc16d674ec8000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001b39026cca9b4e4e42ac957182e6bbeebd88d327c9368f905620b8edbf2be687af12e190eb0ec2fc5b337487834aeb9ce9df2f0275f281b3e7ca5bdec13246444f"

	var method ethaccessor.CancelOrderMethod

	data := hexutil.MustDecode("0x" + input[10:])

	//for i := 0; i < len(data)/32; i++ {
	//	t.Logf("index:%d -> %s", i, common.ToHex(data[i*32:(i+1)*32]))
	//}

	if err := ethaccessor.ProtocolImplAbi().UnpackMethodInput(&method, "cancelOrder", data); err != nil {
		t.Fatalf(err.Error())
	}

	order, cancelAmount, err := method.ConvertDown()
	if err != nil {
		t.Fatalf(err.Error())
	}

	order.DelegateAddress = common.HexToAddress("0xf49733091a3e1ddec740bca4c325f8aaee6ee307")
	order.Hash = order.GenerateHash()
	t.Log("de", order.DelegateAddress.Hex())
	t.Log("orderHash", order.Hash.Hex())
	t.Log("owner", order.Owner.Hex())
	t.Log("wallet", order.WalletAddress.Hex())
	t.Log("auth", order.AuthAddr.Hex())
	t.Log("tokenS", order.TokenS.Hex())
	t.Log("tokenB", order.TokenB.Hex())
	t.Log("amountS", order.AmountS.String())
	t.Log("amountB", order.AmountB.String())
	t.Log("validSince", order.ValidSince.String())
	t.Log("validUntil", order.ValidUntil.String())
	t.Log("lrcFee", order.LrcFee.String())
	t.Log("cancelAmount", method.OrderValues[5].String())
	t.Log("buyNoMoreThanAmountB", order.BuyNoMoreThanAmountB)
	t.Log("marginSplitpercentage", order.MarginSplitPercentage)
	t.Log("v", order.V)
	t.Log("s", order.S.Hex())
	t.Log("r", order.R.Hex())
	t.Log("cancelAmount", cancelAmount)
}

func TestExtractorServiceImpl_UnpackApproveMethod(t *testing.T) {
	input := "0x095ea7b300000000000000000000000045aa504eb94077eec4bf95a10095a8e3196fc5910000000000000000000000000000000000000000000000008ac7230489e80000"

	var method ethaccessor.ApproveMethod

	data := hexutil.MustDecode("0x" + input[10:])
	for i := 0; i < len(data)/32; i++ {
		t.Logf("index:%d -> %s", i, common.ToHex(data[i*32:(i+1)*32]))
	}

	if err := ethaccessor.Erc20Abi().UnpackMethodInput(&method, "approve", data); err != nil {
		t.Fatalf(err.Error())
	}

	approve := method.ConvertDown()
	t.Logf("approve spender:%s, value:%s", approve.Spender.Hex(), approve.Amount.String())
}

func TestExtractorServiceImpl_UnpackTransferMethod(t *testing.T) {
	input := "0xa9059cbb0000000000000000000000008311804426a24495bd4306daf5f595a443a52e32000000000000000000000000000000000000000000000000000000174876e800"
	data := hexutil.MustDecode("0x" + input[10:])
	var method ethaccessor.TransferMethod
	if err := ethaccessor.Erc20Abi().UnpackMethodInput(&method, "transfer", data); err != nil {
		t.Fatalf(err.Error())
	}
	transfer := method.ConvertDown()
	t.Logf("transfer receiver:%s, value:%s", transfer.Receiver.Hex(), transfer.Amount.String())
}

func TestExtractorServiceImpl_UnpackTransferEvent(t *testing.T) {
	inputs := []string{
		"0x00000000000000000000000000000000000000000000001d2666491321fc5651",
		"0x0000000000000000000000000000000000000000000000008ac7230489e80000",
		"0x0000000000000000000000000000000000000000000000004c0303a413a39039",
		"0x000000000000000000000000000000000000000000000000016345785d8a0000",
	}
	transfer := &ethaccessor.TransferEvent{}

	for _, input := range inputs {
		data := hexutil.MustDecode(input)

		if err := ethaccessor.Erc20Abi().Unpack(transfer, "Transfer", data, abi.SEL_UNPACK_EVENT); err != nil {
			t.Fatalf(err.Error())
		}

		t.Logf("transfer value:%s", transfer.Value.String())
	}
}

func TestExtractorServiceImpl_UnpackRingMined(t *testing.T) {
	input := "0x00000000000000000000000000000000000000000000000000000000000000070000000000000000000000004bad3053d574cd54513babe21db3f09bea1d387d0000000000000000000000004bad3053d574cd54513babe21db3f09bea1d387d0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000eece69a21bb35f7566d4d7e447cb2765cf464c308ba0352d6ad90af4a744794eb0000000000000000000000001b978a1d302335a6f2ebe4b8823b5e17c3c84135000000000000000000000000f079e0612e869197c5f4c7d0a95df570b163232b000000000000000000000000000000000000000000000000016345785d8a000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffffffffffffffffffffffffffffffffffffffffffffffba9c6e7dbb0c00006987b1498573ad4fed2d2a1becb054c57d351f775c1dd3d80a42a25dd31c18e3000000000000000000000000b1018949b241d76a1ab2094f473e9befeabb5ead000000000000000000000000ae79693db742d72576db8349142f9cd8b9d8535500000000000000000000000000000000000000000000001db12d6c17abe45651000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000016cdb44ad2b111aa0000000000000000000000000000000000000000000000000000000000000000"
	//input := "0x00000000000000000000000000000000000000000000000000000000000000080000000000000000000000004bad3053d574cd54513babe21db3f09bea1d387d0000000000000000000000004bad3053d574cd54513babe21db3f09bea1d387d0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000000e779a662897f805cee228e4c0349ec8a5c05c190652287b47daddc3008d78a28b000000000000000000000000b1018949b241d76a1ab2094f473e9befeabb5ead000000000000000000000000ae79693db742d72576db8349142f9cd8b9d8535500000000000000000000000000000000000000000000001043561a8829300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000016cdb44ad2b111b40000000000000000000000000000000000000000000000000000000000000000af78d9d04c29924ff9dcdda4f034f77e230d186415fe433bc653e980d4d6771f0000000000000000000000001b978a1d302335a6f2ebe4b8823b5e17c3c84135000000000000000000000000f079e0612e869197c5f4c7d0a95df570b163232b00000000000000000000000000000000000000000000000000c297138f8e6f8100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ffffffffffffffffffffffffffffffffffffffffffffffffba9c6e7dbb0c0000"
	ringmined := &ethaccessor.RingMinedEvent{}

	data := hexutil.MustDecode(input)

	for i := 0; i < len(data)/32; i++ {
		t.Logf("index:%d -> %s", i, common.ToHex(data[i*32:(i+1)*32]))
	}

	if err := ethaccessor.ProtocolImplAbi().Unpack(ringmined, "RingMined", data, abi.SEL_UNPACK_EVENT); err != nil {
		t.Fatalf(err.Error())
	}

	evt, fills, err := ringmined.ConvertDown()
	if err != nil {
		t.Fatalf(err.Error())
	}

	for k, fill := range fills {
		t.Logf("k:%d --> ringindex:%s", k, fill.RingIndex.String())
		t.Logf("k:%d --> fillIndex:%s", k, fill.FillIndex.String())
		t.Logf("k:%d --> orderhash:%s", k, fill.OrderHash.Hex())
		t.Logf("k:%d --> preorder:%s", k, fill.PreOrderHash.Hex())
		t.Logf("k:%d --> nextorder:%s", k, fill.NextOrderHash.Hex())
		t.Logf("k:%d --> owner:%s", k, fill.Owner.Hex())
		t.Logf("k:%d --> tokenS:%s", k, fill.TokenS.Hex())
		t.Logf("k:%d --> tokenB:%s", k, fill.TokenB.Hex())
		t.Logf("k:%d --> amountS:%s", k, fill.AmountS.String())
		t.Logf("k:%d --> amountB:%s", k, fill.AmountB.String())
		t.Logf("k:%d --> lrcReward:%s", k, fill.LrcReward.String())
		t.Logf("k:%d --> lrcFee:%s", k, fill.LrcFee.String())
		t.Logf("k:%d --> splitS:%s", k, fill.SplitS.String())
		t.Logf("k:%d --> splitB:%s", k, fill.SplitB.String())
	}

	t.Logf("totalLrcFee:%s", evt.TotalLrcFee.String())
	t.Logf("tradeAmount:%d", evt.TradeAmount)
}

func TestExtractorServiceImpl_UnpackWethDeposit(t *testing.T) {
	input := "0x0000000000000000000000000000000000000000000000000de0b6b3a7640000"
	deposit := &ethaccessor.WethDepositEvent{}

	data := hexutil.MustDecode(input)

	if err := ethaccessor.WethAbi().Unpack(deposit, "Deposit", data, abi.SEL_UNPACK_EVENT); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Logf("deposit value:%s", deposit.Value.String())
	}
}

func TestExtractorServiceImpl_UnpackTokenRegistry(t *testing.T) {
	input := "0x000000000000000000000000f079e0612e869197c5f4c7d0a95df570b163232b0000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000457455448"

	tokenRegistry := &ethaccessor.TokenRegisteredEvent{}

	data := hexutil.MustDecode(input)

	println("====token registry", len(data))

	if err := ethaccessor.WethAbi().Unpack(tokenRegistry, "TokenRegistered", data, abi.SEL_UNPACK_EVENT); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Logf("TokenRegistered symbol:%s, address:%s", tokenRegistry.Symbol, tokenRegistry.Token.Hex())
	}
}

func TestExtractorServiceImpl_UnpackTokenUnRegistry(t *testing.T) {
	input := "0x000000000000000000000000529540ee6862158f47d647ae023098f6705210a90000000000000000000000000000000000000000000000000000000000000040000000000000000000000000000000000000000000000000000000000000000457455448"

	tokenUnRegistry := &ethaccessor.TokenUnRegisteredEvent{}

	data := hexutil.MustDecode(input)

	println("====token unregistry", len(data))

	if err := ethaccessor.WethAbi().Unpack(tokenUnRegistry, "TokenUnregistered", data, abi.SEL_UNPACK_EVENT); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Logf("TokenUnregistered symbol:%s, address:%s", tokenUnRegistry.Symbol, tokenUnRegistry.Token.Hex())
	}
}

func TestExtractorServiceImpl_Compare(t *testing.T) {
	str1 := "547722557505166136913"
	str2 := "1000000000000000000000"
	num1, _ := big.NewInt(0).SetString(str1, 0)
	num2, _ := big.NewInt(0).SetString(str2, 0)
	if num1.Cmp(num2) > 0 {
		t.Logf("%s > %s", str1, str2)
	} else {
		t.Logf("%s <= %s", str1, str2)
	}
}

func TestExtractorServiceImpl_UnpackNumbers(t *testing.T) {
	str1 := "0xffffffffffffffffffffffffffffffffffffffffffffffffffa1d2c1fb1c2d9f"
	str2 := "0xffffffffffffffffffffffffffffffffffffffffffffffffff90c5f64e557fa4"
	str3 := "0x0000000000000000000000000000000000000000000000026508392204063330"
	str4 := "0x0000000000000000000000000000000000000000000000031307535724740700"
	list := []string{str1, str2, str3, str4}

	for _, v := range list {
		n1 := safeBig(v)
		t.Logf("init data:%s -> number:%s", v, n1.String())
	}
}

func safeBig(input string) *big.Int {
	bytes := hexutil.MustDecode(input)
	num := new(big.Int).SetBytes(bytes[:])
	if bytes[0] > uint8(128) {
		num.Xor(types.MaxUint256, num)
		num.Not(num)
	}
	return num
}
