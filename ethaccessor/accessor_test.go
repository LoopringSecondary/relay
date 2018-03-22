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
	registerTokenAddress = "0xf079E0612E869197c5F4c7D0a95DF570B163232b"
	registerTokenSymbol  = "WETH"
	miner                = test.Entity().Creator
	account1             = test.Entity().Accounts[0].Address
	account2             = test.Entity().Accounts[1].Address
	lrcTokenAddress      = util.AllTokens["LRC"].Protocol
	wethTokenAddress     = util.AllTokens["WETH"].Protocol
	delegateAddress      = test.Delegate()
	gas                  = big.NewInt(200000)
	gasPrice             = big.NewInt(21000000000)
)

func TestEthNodeAccessor_SetTokenBalance(t *testing.T) {
	owner := account2
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2000000))
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
	tokenAddress := wethTokenAddress
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100))
	spender := delegateAddress

	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.Erc20Abi(), tokenAddress)
	if result, err := callMethod(account.Address, "approve", gas, gasPrice, nil, spender, amount); nil != err {
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
		t.Log(new(big.Rat).SetFrac(allowance, big.NewInt(1e18)).FloatString(2))
	}
}

func TestEthNodeAccessor_CancelOrder(t *testing.T) {
	var (
		model        *dao.Order
		state        types.OrderState
		err          error
		result       string
		orderhash    = common.HexToHash("0xf9e4657a74b947edbc3028013640fa6cc052b3ba7432175b93c0906959042146")
		cancelAmount = new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2))
	)

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
	addresses := [4]common.Address{state.RawOrder.Owner, state.RawOrder.TokenS, state.RawOrder.TokenB, state.RawOrder.AuthAddr}
	values := [7]*big.Int{state.RawOrder.AmountS, state.RawOrder.AmountB, state.RawOrder.ValidSince, state.RawOrder.ValidUntil, state.RawOrder.LrcFee, state.RawOrder.WalletId, cancelAmount}
	buyNoMoreThanB := state.RawOrder.BuyNoMoreThanAmountB
	marginSplitPercentage := state.RawOrder.MarginSplitPercentage
	v := state.RawOrder.V
	s := state.RawOrder.S
	r := state.RawOrder.R

	// call cancel order
	protocol := test.Protocol()
	implAddress := ethaccessor.ProtocolAddresses()[protocol].ContractAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.ProtocolImplAbi(), implAddress)
	if result, err = callMethod(account.Address, "cancelOrder", gas, gasPrice, nil, addresses, values, buyNoMoreThanB, marginSplitPercentage, v, r, s); nil != err {
		t.Fatalf("call method cancelOrder error:%s", err.Error())
	} else {
		t.Logf("cancelOrder result:%s", result)
	}
}

func TestEthNodeAccessor_GetCancelledOrFilled(t *testing.T) {
	orderhash := common.HexToHash("0x77aecf96a71d260074ab6ad9352365f0e83cd87d4d3a424071e76cafc393f549")

	protocol := test.Protocol()
	implAddress := ethaccessor.ProtocolAddresses()[protocol].ContractAddress
	if amount, err := ethaccessor.GetCancelledOrFilled(implAddress, orderhash, "latest"); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("cancelOrFilled amount:%s", amount.String())
	}
}

// cutoff的值必须在两个块的timestamp之间
func TestEthNodeAccessor_CutoffAll(t *testing.T) {
	account := common.HexToAddress("0xb1018949b241D76A1AB2094f473E9bEfeAbB5Ead")
	cutoff := big.NewInt(1581107175)

	protocol := test.Protocol()
	implAddress := ethaccessor.ProtocolAddresses()[protocol].ContractAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.ProtocolImplAbi(), implAddress)
	if result, err := callMethod(account, "cancelAllOrders", gas, gasPrice, nil, cutoff); nil != err {
		t.Fatalf("call method cancelAllOrders error:%s", err.Error())
	} else {
		t.Logf("cutoff result:%s", result)
	}
}

func TestEthNodeAccessor_GetCutoffAll(t *testing.T) {
	owner := account1
	protocol := test.Protocol()
	implAddress := ethaccessor.ProtocolAddresses()[protocol].ContractAddress
	if timestamp, err := ethaccessor.GetCutoff(implAddress, owner, "latest"); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("cutoff timestamp:%s", timestamp.String())
	}
}

func TestEthNodeAccessor_CutoffPair(t *testing.T) {
	account := common.HexToAddress("0xb1018949b241D76A1AB2094f473E9bEfeAbB5Ead")
	cutoff := big.NewInt(1531107175)
	token1 := lrcTokenAddress
	token2 := wethTokenAddress

	protocol := test.Protocol()
	implAddress := ethaccessor.ProtocolAddresses()[protocol].ContractAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.ProtocolImplAbi(), implAddress)
	if result, err := callMethod(account, "cancelAllOrdersByTradingPair", gas, gasPrice, nil, token1, token2, cutoff); nil != err {
		t.Fatalf("call method cancelAllOrdersByTradingPair error:%s", err.Error())
	} else {
		t.Logf("cutoff result:%s", result)
	}
}

func TestEthNodeAccessor_GetCutoffPair(t *testing.T) {
	owner := accounts.Account{Address: account2}
	token1 := lrcTokenAddress
	token2 := wethTokenAddress
	protocol := test.Protocol()
	implAddress := ethaccessor.ProtocolAddresses()[protocol].ContractAddress
	if timestamp, err := ethaccessor.GetCutoffPair(implAddress, owner.Address, token1, token2, "latest"); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("cutoffpair timestamp:%s", timestamp.String())
	}
}

func TestEthNodeAccessor_TokenRegister(t *testing.T) {
	account := accounts.Account{Address: test.Entity().Creator.Address}

	protocol := test.Protocol()
	tokenRegistryAddress := ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.TokenRegistryAbi(), tokenRegistryAddress)
	if result, err := callMethod(account.Address, "registerToken", gas, gasPrice, nil, common.HexToAddress(registerTokenAddress), registerTokenSymbol); nil != err {
		t.Fatalf("call method registerToken error:%s", err.Error())
	} else {
		t.Logf("registerToken result:%s", result)
	}
}

func TestEthNodeAccessor_TokenUnRegister(t *testing.T) {
	account := accounts.Account{Address: test.Entity().Creator.Address}

	protocol := test.Protocol()
	tokenRegistryAddress := ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.TokenRegistryAbi(), tokenRegistryAddress)
	if result, err := callMethod(account.Address, "unregisterToken", gas, gasPrice, nil, common.HexToAddress(registerTokenAddress), registerTokenSymbol); nil != err {
		t.Fatalf("call method unregisterToken error:%s", err.Error())
	} else {
		t.Logf("unregisterToken result:%s", result)
	}
}

func TestEthNodeAccessor_GetAddressBySymbol(t *testing.T) {
	var result string
	protocol := test.Protocol()
	symbol := "LRC"
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.TokenRegistryAbi(), ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress)
	if err := callMethod(&result, "getAddressBySymbol", "latest", symbol); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("symbol map:%s->%s", symbol, common.HexToAddress(result).Hex())
	}
}

// 注册合约
func TestEthNodeAccessor_AuthorizedAddress(t *testing.T) {
	account := accounts.Account{Address: test.Entity().Creator.Address}

	protocol := test.Protocol()
	delegateAddress := ethaccessor.ProtocolAddresses()[protocol].DelegateAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.DelegateAbi(), delegateAddress)
	if result, err := callMethod(account.Address, "authorizeAddress", gas, gasPrice, nil, protocol); nil != err {
		t.Fatalf("call method authorizeAddress error:%s", err.Error())
	} else {
		t.Logf("authorizeAddress result:%s", result)
	}
}

func TestEthNodeAccessor_DeAuthorizedAddress(t *testing.T) {
	account := accounts.Account{Address: test.Entity().Creator.Address}

	protocol := test.Protocol()
	delegateAddress := ethaccessor.ProtocolAddresses()[protocol].DelegateAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.DelegateAbi(), delegateAddress)
	if result, err := callMethod(account.Address, "deauthorizeAddress", gas, gasPrice, nil, protocol); nil != err {
		t.Fatalf("call method deauthorizeAddress error:%s", err.Error())
	} else {
		t.Logf("deauthorizeAddress result:%s", result)
	}
}

func TestEthNodeAccessor_IsAddressAuthorized(t *testing.T) {
	var result string
	protocol := test.Protocol()
	delegateAddress := ethaccessor.ProtocolAddresses()[protocol].DelegateAddress
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.DelegateAbi(), delegateAddress)
	if err := callMethod(&result, "isAddressAuthorized", "latest", delegateAddress); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("symbol map:%s->%s", registerTokenSymbol, result)
	}
}

func TestEthNodeAccessor_WethDeposit(t *testing.T) {
	account := account1
	wethAddr := wethTokenAddress
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(100))
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethAddr)
	if result, err := callMethod(account, "deposit", gas, gasPrice, amount); nil != err {
		t.Fatalf("call method weth-deposit error:%s", err.Error())
	} else {
		t.Logf("weth-deposit result:%s", result)
	}
}

func TestEthNodeAccessor_WethWithdrawal(t *testing.T) {
	account := account1
	wethAddr := wethTokenAddress
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1))
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethAddr)
	if result, err := callMethod(account, "withdraw", gas, gasPrice, nil, amount); nil != err {
		t.Fatalf("call method weth-withdraw error:%s", err.Error())
	} else {
		t.Logf("weth-withdraw result:%s", result)
	}
}

func TestEthNodeAccessor_WethTransfer(t *testing.T) {
	account := accounts.Account{Address: account1}

	wethAddr := wethTokenAddress
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1))
	to := account2
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethAddr)
	if result, err := callMethod(account.Address, "transfer", gas, gasPrice, nil, to, amount); nil != err {
		t.Fatalf("call method weth-transfer error:%s", err.Error())
	} else {
		t.Logf("weth-transfer result:%s", result)
	}
}

func TestEthNodeAccessor_EthTransfer(t *testing.T) {
	sender := account1
	receiver := account2
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1))
	if hash, err := ethaccessor.SignAndSendTransaction(sender, receiver, gas, gasPrice, amount, []byte("test")); err != nil {
		t.Errorf(err.Error())
	} else {
		t.Logf("txhash:%s", hash)
	}
}

func TestEthNodeAccessor_TokenAddress(t *testing.T) {
	symbol := "WETH"
	protocol := test.Protocol()
	tokenRegistryAddress := ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.TokenRegistryAbi(), tokenRegistryAddress)
	var result string
	if err := callMethod(&result, "getAddressBySymbol", "latest", symbol); nil != err {
		t.Fatalf("call method tokenAddress error:%s", err.Error())
	} else {
		t.Logf("symbol:%s-> address:%s", symbol, result)
	}
}

func TestEthNodeAccessor_GetRegistryName(t *testing.T) {
	protocol := test.Protocol()
	miner := test.Entity().Creator.Address
	nameRegistryAddress := ethaccessor.ProtocolAddresses()[protocol].NameRegistryAddress
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.NameRegistryAbi(), nameRegistryAddress)
	var result string
	if err := callMethod(&result, "nameMap", "latest", miner); nil != err {
		t.Fatalf("call method nameMap error:%s", err.Error())
	} else {
		t.Logf("name:%s-> address:%s", result, miner.Hex())
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
