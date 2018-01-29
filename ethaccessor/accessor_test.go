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
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

var (
	version              = test.Version
	registerTokenAddress = "0x8b62ff4ddc9baeb73d0a3ea49d43e4fe8492935a"
	registerTokenSymbol  = "wrdn"
	account1             = test.Entity().Accounts[0].Address
	account2             = test.Entity().Accounts[1].Address
	lrcTokenAddress      = util.AllTokens["LRC"].Protocol
	wethTokenAddress     = util.AllTokens["WETH"].Protocol
	delegateAddress      = test.Delegate()
)

func TestEthNodeAccessor_SetTokenBalance(t *testing.T) {
	owner := account2
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000000))
	test.SetTokenBalance("LRC", owner, amount)
}

func TestEthNodeAccessor_Erc20Balance(t *testing.T) {
	owner := account2
	tokenAddress := lrcTokenAddress
	balance, err := ethaccessor.Erc20Balance(tokenAddress, owner, "latest")
	if err != nil {
		t.Fatalf("accessor get erc20 balance error:%s", err.Error())
	}

	t.Log(new(big.Rat).SetFrac(balance, big.NewInt(1e18)).FloatString(2))
}

func TestEthNodeAccessor_Approval(t *testing.T) {
	account := accounts.Account{Address: account2}

	tokenAddress := wethTokenAddress
	amount, _ := new(big.Int).SetString("100000000000000000000", 0) // 100weth
	spender := delegateAddress

	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.Erc20Abi(), tokenAddress)
	if result, err := callMethod(account, "approve", nil, nil, nil, spender, amount); nil != err {
		t.Fatalf("call method approve error:%s", err.Error())
	} else {
		t.Logf("approve result:%s", result)
	}
}

func TestEthNodeAccessor_Allowance(t *testing.T) {
	owner := account2
	tokenAddress := wethTokenAddress
	spender := delegateAddress

	if allowance, err := ethaccessor.Erc20Allowance(tokenAddress, owner, spender, "latest"); err != nil {
		t.Fatalf("accessor get erc20 approval error:%s", err.Error())
	} else {
		t.Log(allowance.String())
	}
}

func TestEthNodeAccessor_CancelOrder(t *testing.T) {
	var (
		model           *dao.Order
		state           types.OrderState
		err             error
		result          string
		orderhash       = common.HexToHash("0x38a4ff508746feddb75e1652e2c430e23a179f949f331ebe81e624801b4b6f4d")
		cancelAmount, _ = new(big.Int).SetString("1000000000000000000000", 0)
	)

	// load config
	c := test.Cfg()

	// get order
	rds := test.Rds()
	if model, err = rds.GetOrderByHash(orderhash); err != nil {
		t.Fatalf(err.Error())
	}
	if err := model.ConvertUp(&state); err != nil {
		t.Fatalf(err.Error())
	}

	account := accounts.Account{Address: state.RawOrder.Owner}

	// create cancel order contract function parameters
	addresses := [3]common.Address{state.RawOrder.Owner, state.RawOrder.TokenS, state.RawOrder.TokenB}
	values := [7]*big.Int{state.RawOrder.AmountS, state.RawOrder.AmountB, state.RawOrder.Timestamp, state.RawOrder.Ttl, state.RawOrder.Salt, state.RawOrder.LrcFee, cancelAmount}
	buyNoMoreThanB := state.RawOrder.BuyNoMoreThanAmountB
	marginSplitPercentage := state.RawOrder.MarginSplitPercentage
	v := state.RawOrder.V
	s := state.RawOrder.S
	r := state.RawOrder.R

	// call cancel order
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.ProtocolImplAbi(), protocol)
	if result, err = callMethod(account, "cancelOrder", big.NewInt(200000), big.NewInt(21000000000), nil, addresses, values, buyNoMoreThanB, marginSplitPercentage, v, r, s); nil != err {
		t.Fatalf("call method cancelOrder error:%s", err.Error())
	} else {
		t.Logf("cancelOrder result:%s", result)
	}
}

func TestEthNodeAccessor_GetCancelledOrFilled(t *testing.T) {
	c := test.Cfg()
	orderhash := common.HexToHash("0x77aecf96a71d260074ab6ad9352365f0e83cd87d4d3a424071e76cafc393f549")

	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	if amount, err := ethaccessor.GetCancelledOrFilled(protocol, orderhash, "latest"); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("cancelOrFilled amount:%s", amount.String())
	}
}

// cutoff的值必须在两个块的timestamp之间
func TestEthNodeAccessor_Cutoff(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: account2}
	cutoff := big.NewInt(1518700280)

	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.ProtocolImplAbi(), protocol)
	if result, err := callMethod(account, "setCutoff", nil, nil, nil, cutoff); nil != err {
		t.Fatalf("call method setCutoff error:%s", err.Error())
	} else {
		t.Logf("cutoff result:%s", result)
	}
}

func TestEthNodeAccessor_GetCutoff(t *testing.T) {
	c := test.Cfg()
	owner := account1
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	if timestamp, err := ethaccessor.GetCutoff(protocol, owner, "latest"); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("cutoff timestamp:%s", timestamp.String())
	}
}

func TestEthNodeAccessor_TokenRegister(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: common.HexToAddress(c.Miner.Miner)}

	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	address := ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.TokenRegistryAbi(), address)
	if result, err := callMethod(account, "registerToken", nil, nil, nil, common.HexToAddress(registerTokenAddress), registerTokenSymbol); nil != err {
		t.Fatalf("call method registerToken error:%s", err.Error())
	} else {
		t.Logf("registerToken result:%s", result)
	}
}

func TestEthNodeAccessor_TokenUnRegister(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: common.HexToAddress(c.Miner.Miner)}

	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	address := ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.TokenRegistryAbi(), address)
	if result, err := callMethod(account, "unregisterToken", nil, nil, nil, common.HexToAddress(registerTokenAddress), registerTokenSymbol); nil != err {
		t.Fatalf("call method unregisterToken error:%s", err.Error())
	} else {
		t.Logf("unregisterToken result:%s", result)
	}
}

func TestEthNodeAccessor_GetAddressBySymbol(t *testing.T) {
	c := test.Cfg()
	var result string
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.TokenRegistryAbi(), ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress)
	if err := callMethod(&result, "getAddressBySymbol", "latest", registerTokenSymbol); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("symbol map:%s->%s", registerTokenSymbol, common.HexToAddress(result).Hex())
	}
}

// 注册合约
func TestEthNodeAccessor_AuthorizedAddress(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: common.HexToAddress(c.Miner.Miner)}

	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.DelegateAbi(), ethaccessor.ProtocolAddresses()[protocol].DelegateAddress)
	if result, err := callMethod(account, "authorizeAddress", nil, nil, nil, protocol); nil != err {
		t.Fatalf("call method authorizeAddress error:%s", err.Error())
	} else {
		t.Logf("authorizeAddress result:%s", result)
	}
}

func TestEthNodeAccessor_DeAuthorizedAddress(t *testing.T) {
	c := test.Cfg()
	account := accounts.Account{Address: common.HexToAddress(c.Miner.Miner)}

	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.DelegateAbi(), ethaccessor.ProtocolAddresses()[protocol].DelegateAddress)
	if result, err := callMethod(account, "deauthorizeAddress", nil, nil, nil, protocol); nil != err {
		t.Fatalf("call method deauthorizeAddress error:%s", err.Error())
	} else {
		t.Logf("deauthorizeAddress result:%s", result)
	}
}

func TestEthNodeAccessor_IsAddressAuthorized(t *testing.T) {
	c := test.Cfg()

	var result string
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.DelegateAbi(), ethaccessor.ProtocolAddresses()[protocol].DelegateAddress)
	if err := callMethod(&result, "isAddressAuthorized", "latest", protocol); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("symbol map:%s->%s", registerTokenSymbol, result)
	}
}

func TestEthNodeAccessor_WethDeposit(t *testing.T) {
	account := accounts.Account{Address: account1}

	wethAddr := wethTokenAddress
	amount := new(big.Int).SetInt64(1000000)
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethAddr)
	if result, err := callMethod(account, "deposit", big.NewInt(200000), big.NewInt(21000000000), amount); nil != err {
		t.Fatalf("call method weth-deposit error:%s", err.Error())
	} else {
		t.Logf("weth-deposit result:%s", result)
	}
}

func TestEthNodeAccessor_WethWithdrawal(t *testing.T) {
	account := accounts.Account{Address: account1}

	wethAddr := wethTokenAddress
	amount, _ := new(big.Int).SetString("100", 0)
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethAddr)
	if result, err := callMethod(account, "withdraw", big.NewInt(200000), big.NewInt(21000000000), nil, amount); nil != err {
		t.Fatalf("call method weth-withdraw error:%s", err.Error())
	} else {
		t.Logf("weth-withdraw result:%s", result)
	}
}

func TestEthNodeAccessor_WethTransfer(t *testing.T) {
	account := accounts.Account{Address: account1}

	wethAddr := wethTokenAddress
	amount := new(big.Int).SetInt64(100)
	to := account2
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethAddr)
	if result, err := callMethod(account, "transfer", big.NewInt(200000), big.NewInt(21000000000), nil, to, amount); nil != err {
		t.Fatalf("call method weth-transfer error:%s", err.Error())
	} else {
		t.Logf("weth-transfer result:%s", result)
	}
}

func TestEthNodeAccessor_TokenAddress(t *testing.T) {
	c := test.Cfg()

	symbol := "WETH"
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.TokenRegistryAbi(), protocol)
	var result string
	if err := callMethod(&result, "getAddressBySymbol", "latest", symbol); nil != err {
		t.Fatalf("call method tokenAddress error:%s", err.Error())
	} else {
		t.Logf("symbol:%s-> address:%s", symbol, result)
	}
}

func TestEthNodeAccessor_BlockTransactions(t *testing.T) {
	blockNumber := big.NewInt(4976099)
	block := &ethaccessor.BlockWithTxHash{}
	if err := ethaccessor.GetBlockByNumber(block, blockNumber, false); err != nil {
		t.Fatal(err.Error())
	}

	t.Logf("length of block transactions :%d", len(block.Transactions))
	for _, v := range block.Transactions {
		if v == "0x68ce1331a561e5b693a4780458910fab302da2f52bea4cef05f9ec9d5860e632" {
			t.Logf("transaction:%s", v)
		}
	}
}
