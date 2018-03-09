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
	"github.com/ethereum/go-ethereum/rpc"
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
	tokenAddress := wethTokenAddress
	balance, err := ethaccessor.Erc20Balance(tokenAddress, owner, "latest")
	if err != nil {
		t.Fatalf("accessor get erc20 balance error:%s", err.Error())
	}

	t.Log(new(big.Rat).SetFrac(balance, big.NewInt(1e18)).FloatString(2))
}

func TestEthNodeAccessor_Approval(t *testing.T) {
	account := accounts.Account{Address: account2}
	//100000000000000000000
	tokenAddress := wethTokenAddress
	amount, _ := new(big.Int).SetString("1000000000000000000000000000000000000000000000000000000", 0) // 100weth
	//1000000000000000000000000000000000000
	//1000000000000000000000000000000000000000000000000000000
	spender := delegateAddress

	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.Erc20Abi(), tokenAddress)
	if result, err := callMethod(account.Address, "approve", big.NewInt(1000000), big.NewInt(21000000000), nil, spender, amount); nil != err {
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
	values := [7]*big.Int{state.RawOrder.AmountS, state.RawOrder.AmountB, state.RawOrder.ValidSince, state.RawOrder.ValidUntil, state.RawOrder.LrcFee, cancelAmount}
	buyNoMoreThanB := state.RawOrder.BuyNoMoreThanAmountB
	marginSplitPercentage := state.RawOrder.MarginSplitPercentage
	v := state.RawOrder.V
	s := state.RawOrder.S
	r := state.RawOrder.R

	// call cancel order
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[version])
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.ProtocolImplAbi(), protocol)
	if result, err = callMethod(account.Address, "cancelOrder", big.NewInt(200000), big.NewInt(21000000000), nil, addresses, values, buyNoMoreThanB, marginSplitPercentage, v, r, s); nil != err {
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
	if result, err := callMethod(account.Address, "setCutoff", nil, nil, nil, cutoff); nil != err {
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
	if result, err := callMethod(account.Address, "registerToken", nil, nil, nil, common.HexToAddress(registerTokenAddress), registerTokenSymbol); nil != err {
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
	if result, err := callMethod(account.Address, "unregisterToken", nil, nil, nil, common.HexToAddress(registerTokenAddress), registerTokenSymbol); nil != err {
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
	if result, err := callMethod(account.Address, "authorizeAddress", nil, nil, nil, protocol); nil != err {
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
	if result, err := callMethod(account.Address, "deauthorizeAddress", nil, nil, nil, protocol); nil != err {
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
	amount, _ := new(big.Int).SetString("100000000000000000000000000000000000000000000000000000000000000", 0)
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethAddr)
	if result, err := callMethod(account.Address, "deposit", big.NewInt(200000), big.NewInt(21000000000), amount); nil != err {
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
	if result, err := callMethod(account.Address, "withdraw", big.NewInt(200000), big.NewInt(21000000000), nil, amount); nil != err {
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
	if result, err := callMethod(account.Address, "transfer", big.NewInt(200000), big.NewInt(21000000000), nil, to, amount); nil != err {
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

func TestEthNodeAccessor_GetTransaction(t *testing.T) {
	tx := &ethaccessor.Transaction{}
	if err := ethaccessor.GetTransactionByHash(tx, "0x1cff70d0eecd86d008b633f62ef171bbe71516c132adea3ff73ed419d56232fd", "latest"); err == nil {
		t.Logf("tx gas:%s", tx.Gas.BigInt().String())
	} else {
		t.Fatalf(err.Error())
	}
}

//mainnet
//0x8924ce3be0895775b30f6ea7512c6d8318dc0c84da7e1eb4d1930e5658c92d04
//0x1cff70d0eecd86d008b633f62ef171bbe71516c132adea3ff73ed419d56232fd
//priv net
//0xebe2694d678ee784861268be37f73905415c118ab10523b075a8989636f6297d
func TestEthNodeAccessor_GetTransactionReceipt(t *testing.T) {
	var tx ethaccessor.TransactionReceipt
	if err := ethaccessor.GetTransactionReceipt(&tx, "0xebe2694d678ee784861268be37f73905415c118ab10523b075a8989636f6297d", "latest"); err == nil {
		t.Logf("tx gasUsed:%s status:%s", tx.GasUsed.BigInt().String(), tx.Status.BigInt().String())
	} else {
		t.Fatalf(err.Error())
	}
}

// 使用rpc.client调用eth call时应该使用 arg参数应该指针 保证unmarshal的正确性
func TestEthNodeAccessor_Call(t *testing.T) {
	var (
		arg1 ethaccessor.CallArg
		res1 string
	)

	arg1.To = common.HexToAddress("0x45245bc59219eeaAF6cD3f382e078A461FF9De7B")
	arg1.Data = "0x95d89b41"
	if err := ethaccessor.Call(&res1, &arg1, "latest"); err != nil {
		t.Fatal(err)
	}

	t.Log(res1)

	type CallArg struct {
		From     common.Address `json:"from"`
		To       common.Address `json:"to"`
		Gas      string         `json:"gas"`
		GasPrice string         `json:"gasPrice"`
		Value    string         `json:"value"`
		Data     string         `json:"data"`
		Nonce    string         `json:"nonce"`
	}

	var (
		client *rpc.Client
		err    error
		arg2   CallArg
		res2   string
	)

	url := "http://ec2-13-115-183-194.ap-northeast-1.compute.amazonaws.com:8545"
	if client, err = rpc.Dial(url); nil != err {
		t.Fatalf("rpc.Dail err : %s, url:%s", err.Error(), url)
	}

	arg2.To = common.HexToAddress("0x45245bc59219eeaAF6cD3f382e078A461FF9De7B")
	arg2.Data = "0x95d89b41"
	if err := client.Call(&res2, "eth_call", arg2, "latest"); err != nil {
		t.Fatal(err)
	}

	t.Log(res2)
}
