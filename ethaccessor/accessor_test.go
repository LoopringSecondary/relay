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
	"fmt"
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
	"time"
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

func TestEthNodeAccessor_WethDeposit(t *testing.T) {
	account := account1
	wethAddr := wethTokenAddress
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(2))
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
	from := account1
	to := account2
	wethAddr := wethTokenAddress
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1))

	callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.WethAbi(), wethAddr)
	if result, err := callMethod(from, "transfer", gas, gasPrice, nil, to, amount); nil != err {
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

func TestEthNodeAccessor_EthBalance(t *testing.T) {
	account := account1

	var balance types.Big
	if err := ethaccessor.GetBalance(&balance, common.HexToAddress("0x8311804426A24495bD4306DAf5f595A443a52E32"), "0x53ca90"); err != nil {
		t.Fatalf(err.Error())
	} else {
		amount := new(big.Rat).SetFrac(balance.BigInt(), big.NewInt(1e18)).FloatString(2)
		t.Logf("eth account:%s amount:%s", account.Hex(), amount)
	}

	//time.Sleep(5 * time.Second)
}

func TestEthNodeAccessor_SetTokenBalance(t *testing.T) {
	reqs := ethaccessor.BatchBalanceReqs{}
	for _, v := range util.AllTokens {
		req := &ethaccessor.BatchBalanceReq{}
		req.BlockParameter = "latest"
		req.Token = v.Protocol
		req.Owner = common.HexToAddress("0xA44282dB26EC80C6cfdd748ec09f386a64D645bD")
		reqs = append(reqs, req)
	}
	//if err := ethaccessor.BatchErc20Balance("latest", reqs); nil != err {
	//	t.Errorf("err:%s", err.Error())
	//} else {
	//	for _,req := range reqs {
	//		if req.BalanceErr != nil {
	//			t.Errorf("eeeee:%s", req.BalanceErr.Error())
	//		} else {
	//			t.Logf("ddd:%s", req.Balance.BigInt().String())
	//		}
	//	}
	//}

	reqs1 := ethaccessor.BatchErc20AllowanceReqs{}
	for _, v := range util.AllTokens {
		for _, impl := range ethaccessor.ProtocolAddresses() {
			req := &ethaccessor.BatchErc20AllowanceReq{}
			req.BlockParameter = "latest"
			req.Spender = impl.DelegateAddress
			req.Token = v.Protocol
			req.Owner = common.HexToAddress("0xA44282dB26EC80C6cfdd748ec09f386a64D645bD")
			reqs1 = append(reqs1, req)
		}
	}

	reqs2 := []ethaccessor.BatchReq{reqs, reqs1}
	if err := ethaccessor.BatchCall("latest", reqs2); nil != err {
		t.Errorf("err:%s", err.Error())
	} else {
		for _, reqs3_ := range reqs2 {
			if reqs3, ok := reqs3_.(ethaccessor.BatchErc20AllowanceReqs); ok {

				for _, req := range reqs3 {

					if req.AllowanceErr != nil {
						t.Errorf("eeeee:%s", req.AllowanceErr.Error())
					} else {
						t.Logf("ddd:%s", req.Allowance.BigInt().String())
					}
				}
			} else {
				reqs3 := reqs3_.(ethaccessor.BatchBalanceReqs)
				for _, req := range reqs3 {
					if req.BalanceErr != nil {
						t.Errorf("eeeee:%s", req.BalanceErr.Error())
					} else {
						t.Logf("ddd:%s", req.Balance.BigInt().String())
					}
				}
			}
		}
	}
}

func TestEthNodeAccessor_ERC20Transfer(t *testing.T) {
	from := account1
	to := account2
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(180))

	erc20abi := ethaccessor.Erc20Abi()
	tokenAddress := lrcTokenAddress
	callMethod := ethaccessor.ContractSendTransactionMethod("latest", erc20abi, tokenAddress)
	if result, err := callMethod(from, "transfer", gas, gasPrice, nil, to, amount); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Logf("txhash:%s", common.HexToHash(result).Hex())
	}
}

func TestEthNodeAccessor_ERC20Balance(t *testing.T) {
	accounts := []common.Address{account1, account2, miner.Address}
	tokens := []common.Address{lrcTokenAddress, wethTokenAddress}

	for _, tokenAddress := range tokens {
		for _, account := range accounts {
			balance, err := ethaccessor.Erc20Balance(tokenAddress, account, "latest")
			if err != nil {
				t.Fatalf("accessor get erc20 balance error:%s", err.Error())
			}
			amount := new(big.Rat).SetFrac(balance, big.NewInt(1e18)).FloatString(2)
			symbol, _ := util.GetSymbolWithAddress(tokenAddress)
			t.Logf("token:%s account:%s amount:%s", symbol, account.Hex(), amount)
		}
	}
}

func TestEthNodeAccessor_Approval(t *testing.T) {
	accounts := []common.Address{account1, account2}
	spender := delegateAddress
	tokenAddress := lrcTokenAddress
	amount := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(1000000))
	//amount,_ := big.NewInt(0).SetString("9223372036854775806000000000000000000", 0)

	for _, account := range accounts {
		callMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.Erc20Abi(), tokenAddress)
		if result, err := callMethod(account, "approve", gas, gasPrice, nil, spender, amount); nil != err {
			t.Fatalf("call method approve error:%s", err.Error())
		} else {
			t.Logf("approve result:%s", result)
		}
	}
}

func TestEthNodeAccessor_Allowance(t *testing.T) {
	tokens := []common.Address{lrcTokenAddress, wethTokenAddress}
	accounts := []common.Address{account1, account2}
	//tokens := []common.Address{common.HexToAddress("0xb5f64747127be058Ee7239b363269FC8cF3F4A87")}
	//accounts := []common.Address{common.HexToAddress("0x8311804426A24495bD4306DAf5f595A443a52E32")}
	spender := delegateAddress

	for _, tokenAddress := range tokens {
		for _, account := range accounts {
			if allowance, err := ethaccessor.Erc20Allowance(tokenAddress, account, spender, "latest"); err != nil {
				t.Fatalf("accessor get erc20 approval error:%s", err.Error())
			} else {
				amount := new(big.Rat).SetFrac(allowance, big.NewInt(1e18)).FloatString(2)
				symbol, _ := util.GetSymbolWithAddress(tokenAddress)
				t.Logf("token:%s, account:%s, amount:%s", symbol, account.Hex(), amount)
			}
		}
	}
}

func TestEthNodeAccessor_CancelOrder(t *testing.T) {
	var (
		model        *dao.Order
		state        types.OrderState
		err          error
		result       string
		orderhash    = common.HexToHash("0x30c053e1f7da3161c36591eacd27077b07b91525ec849fc2b7911d13729e9a2d")
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
	addresses := [5]common.Address{state.RawOrder.Owner, state.RawOrder.TokenS, state.RawOrder.TokenB, state.RawOrder.WalletAddress, state.RawOrder.AuthAddr}
	values := [6]*big.Int{state.RawOrder.AmountS, state.RawOrder.AmountB, state.RawOrder.ValidSince, state.RawOrder.ValidUntil, state.RawOrder.LrcFee, cancelAmount}
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
	cutoff := big.NewInt(1531808145)

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

func TestEthNodeAccessor_IsTokenRegistried(t *testing.T) {
	var result string

	protocol := test.Protocol()
	tokenRegistryAddress := ethaccessor.ProtocolAddresses()[protocol].TokenRegistryAddress
	symbol := "LRC"
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.TokenRegistryAbi(), tokenRegistryAddress)

	if err := callMethod(&result, "isTokenRegistered", "latest", symbol); nil != err {
		t.Fatalf("call method isTokenRegistered error:%s", err.Error())
	} else {
		t.Logf("isTokenRegistered result:%s", result)
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

func TestEthNodeAccessor_TokenAddress(t *testing.T) {
	symbol := "LRC"
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

func TestEthNodeAccessor_BlockTransactionStatus(t *testing.T) {
	const (
		startBlock = 5444365
		endBlock   = startBlock + 2
	)

	for i := startBlock; i < endBlock; i++ {
		blockNumber := big.NewInt(int64(i))

	blockMark:
		var blockWithTxHash ethaccessor.BlockWithTxHash
		if err := ethaccessor.GetBlockByNumber(&blockWithTxHash, blockNumber, false); err != nil {
			time.Sleep(1 * time.Second)
			fmt.Printf("........err:%s nil\n", err.Error())
			goto blockMark
		} else if len(blockWithTxHash.Transactions) == 0 {
			time.Sleep(1 * time.Second)
			fmt.Printf("........tx 0\n")
			goto blockMark
		}

		blockWithTxAndReceipt := &ethaccessor.BlockWithTxAndReceipt{}
		blockWithTxAndReceipt.Block = blockWithTxHash.Block
		blockWithTxAndReceipt.Transactions = []ethaccessor.Transaction{}
		blockWithTxAndReceipt.Receipts = []ethaccessor.TransactionReceipt{}

		txno := len(blockWithTxHash.Transactions)
		var rcReqs = make([]*ethaccessor.BatchTransactionRecipientReq, txno)
		for idx, txstr := range blockWithTxHash.Transactions {
			var (
				rcreq ethaccessor.BatchTransactionRecipientReq
				rc    ethaccessor.TransactionReceipt
				rcerr error
			)

			rcreq.TxHash = txstr
			rcreq.TxContent = rc
			rcreq.Err = rcerr

			rcReqs[idx] = &rcreq
		}

		if err := ethaccessor.BatchTransactionRecipients(rcReqs, blockWithTxAndReceipt.Number.BigInt().String()); err != nil {
			t.Fatalf(err.Error())
		}

		for idx, _ := range rcReqs {
			blockWithTxAndReceipt.Receipts = append(blockWithTxAndReceipt.Receipts, rcReqs[idx].TxContent)
		}

		var (
			success = 0
			failed  = 0
			invalid = 0
		)
		for _, v := range blockWithTxAndReceipt.Receipts {
			if v.StatusInvalid() {
				invalid++
				fmt.Printf("tx:%s status is nil\n", v.TransactionHash)
			} else if v.Status.BigInt().Cmp(big.NewInt(1)) < 0 {
				failed++
				fmt.Printf("tx:%s status:%s\n", v.TransactionHash, v.Status.BigInt().String())
			} else {
				success++
			}
		}
		fmt.Printf("blockNumber:%s, blockHash:%s, txNumber:%d, successTx:%d failed:%d nil:%d \n",
			blockNumber.String(), blockWithTxHash.Hash.Hex(), txno, success, failed, invalid)
	}
}

func TestEthNodeAccessor_GetTransaction(t *testing.T) {
	tx := &ethaccessor.Transaction{}
	if err := ethaccessor.GetTransactionByHash(tx, "0x26383249d29e13c4c5f73505775813829875d0b0bf496f2af2867548e2bf8108", "pending"); err == nil {
		t.Logf("tx blockNumber:%s, from:%s, to:%s, gas:%s value:%s", tx.BlockNumber.BigInt().String(), tx.From, tx.To, tx.Gas.BigInt().String(), tx.Value.BigInt().String())
		t.Logf("tx input:%s", tx.Input)
	} else {
		t.Fatalf(err.Error())
	}
}

func TestEthNodeAccessor_GetTransactionReceipt(t *testing.T) {
	var tx ethaccessor.TransactionReceipt
	if err := ethaccessor.GetTransactionReceipt(&tx, "0x26383249d29e13c4c5f73505775813829875d0b0bf496f2af2867548e2bf8108", "latest"); err == nil {
		t.Logf("tx blockNumber:%s gasUsed:%s status:%s logs:%d", tx.BlockNumber.BigInt().String(), tx.GasUsed.BigInt().String(), tx.Status.BigInt().String(), len(tx.Logs))
		idx := len(tx.Logs) - 1
		t.Logf("tx event:%d data:%s", idx, tx.Logs[idx].Data)
		for _, v := range tx.Logs[idx].Topics {
			t.Logf("topic:%s", v)
		}
	} else {
		t.Fatalf(err.Error())
	}
}

func TestEthNodeAccessor_GetBlock(t *testing.T) {
	hash := "0x25d526f4d913a563783fd09a1e5472c505d644fc2f3ac17eae8f2704943dd033"
	var block ethaccessor.Block
	if err := ethaccessor.GetBlockByHash(&block, hash, false); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Logf("number:%s, hash:%s, time:%s", block.Number.BigInt().String(), block.Hash.Hex(), block.Timestamp.BigInt().String())
	}
}

func TestEthNodeAccessor_GetTransactionCount(t *testing.T) {
	var count types.Big
	user := common.HexToAddress("0x71c079107b5af8619d54537a93dbf16e5aab4900")
	if err := ethaccessor.GetTransactionCount(&count, user, "latest"); err != nil {
		t.Fatalf(err.Error())
	} else {
		t.Logf("transaction count:%d", count.Int64())
	}
}

func TestEthNodeAccessor_GetFullBlock(t *testing.T) {
	blockNumber := big.NewInt(5514801)
	withObject := true
	ret, err := ethaccessor.GetFullBlock(blockNumber, withObject)
	if err != nil {
		t.Fatalf(err.Error())
	}
	block := ret.(*ethaccessor.BlockWithTxAndReceipt)
	for _, v := range block.Transactions {
		t.Logf("hash:%s", v.Hash)
	}
	t.Logf("length of block:%s is %d", blockNumber.String(), len(block.Transactions))
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

func TestAccessor_MutilClient(t *testing.T) {
	test.Delegate()

}
