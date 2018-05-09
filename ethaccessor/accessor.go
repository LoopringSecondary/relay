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

package ethaccessor

import (
	"errors"
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"sync"
)

var accessor *ethNodeAccessor

func BlockNumber(result interface{}) error {
	return accessor.RetryCall("latest", 5, result, "eth_blockNumber")
}

func GetBalance(result interface{}, address common.Address, blockNumber string) error {
	return accessor.RetryCall(blockNumber, 2, result, "eth_getBalance", address, blockNumber)
}

func SendRawTransaction(result interface{}, tx string) error {
	return accessor.RetryCall("latest", 2, result, "eth_sendRawTransaction", tx)
}

func GetTransactionCount(result interface{}, address common.Address, blockNumber string) error {
	return accessor.RetryCall(blockNumber, 2, result, "eth_getTransactionCount", address, blockNumber)
}

func Call(result interface{}, ethCall *CallArg, blockNumber string) error {
	return accessor.RetryCall(blockNumber, 2, result, "eth_call", ethCall, blockNumber)
}

func GetBlockByNumber(result interface{}, blockNumber *big.Int, withObject bool) error {
	return accessor.RetryCall(blockNumber.String(), 2, result, "eth_getBlockByNumber", fmt.Sprintf("%#x", blockNumber), withObject)
}

func GetBlockByHash(result types.CheckNull, blockHash string, withObject bool) error {
	for _, c := range accessor.clients {
		//todo:is it need retrycall
		if err := c.client.Call(result, "eth_getBlockByHash", blockHash, withObject); nil == err {
			if !result.IsNull() {
				return nil
			}
		}
	}
	return fmt.Errorf("no block with blockhash:%s", blockHash)

	//return accessor.RetryCall("latest", 2, result, "eth_getBlockByHash", blockHash, withObject)
}

func GetTransactionReceipt(result interface{}, txHash string, blockParameter string) error {
	return accessor.RetryCall(blockParameter, 2, result, "eth_getTransactionReceipt", txHash)
}

func GetTransactionByHash(result types.CheckNull, txHash string, blockParameter string) error {
	for _, c := range accessor.clients {
		if err := c.client.Call(result, "eth_getTransactionByHash", txHash); nil == err {
			if !result.IsNull() {
				return nil
			}
		}
	}
	return fmt.Errorf("no transaction with hash:%s", txHash)
}

func EstimateGasPrice(minGasPrice, maxGasPrice *big.Int) *big.Int {
	return accessor.gasPriceEvaluator.GasPrice(minGasPrice, maxGasPrice)
}

func GetBlockTransactionCountByHash(result interface{}, blockHash string, blockParameter string) error {
	return accessor.RetryCall("latest", 5, result, "eth_getBlockTransactionCountByHash", blockHash)

}

func GetBlockTransactionCountByNumber(result interface{}, blockNumber string) error {
	return accessor.RetryCall(blockNumber, 2, result, "eth_getBlockTransactionCountByNumber", blockNumber)

}

//func Synced() bool {
//	for _, c := range accessor.clients {
//		if c.syncingResult.isSynced() {
//			return true
//		}
//	}
//	return false
//}

func EstimateGas(callData []byte, to common.Address, blockNumber string) (gas, gasPrice *big.Int, err error) {
	return accessor.EstimateGas(blockNumber, callData, to)
}

func SignAndSendTransaction(sender common.Address, to common.Address, gas, gasPrice, value *big.Int, callData []byte, needPreExe bool) (string, error) {
	return accessor.ContractSendTransactionByData("latest", sender, to, gas, gasPrice, value, callData, needPreExe)
}

func ContractSendTransactionMethod(routeParam string, a *abi.ABI, contractAddress common.Address) func(sender common.Address, methodName string, gas, gasPrice, value *big.Int, args ...interface{}) (string, error) {
	return accessor.ContractSendTransactionMethod(routeParam, a, contractAddress)
}

func ContractCallMethod(a *abi.ABI, contractAddress common.Address) func(result interface{}, methodName, blockParameter string, args ...interface{}) error {
	return accessor.ContractCallMethod(a, contractAddress)
}

func Erc20Balance(tokenAddress, ownerAddress common.Address, blockParameter string) (*big.Int, error) {
	return accessor.Erc20Balance(tokenAddress, ownerAddress, blockParameter)
}

func Erc20Allowance(tokenAddress, ownerAddress, spender common.Address, blockParameter string) (*big.Int, error) {
	return accessor.Erc20Allowance(tokenAddress, ownerAddress, spender, blockParameter)
}

// todo(fuk): 需要测试，如果没有，合约是否返回为0
func GetCutoff(contractAddress, owner common.Address, blockNumber string) (*big.Int, error) {
	var cutoff types.Big
	err := accessor.GetCutoff(&cutoff, contractAddress, owner, blockNumber)
	return cutoff.BigInt(), err
}

// todo(fuk): 需要测试，如果没有，合约是否返回为0
func GetCutoffPair(contractAddress, owner, token1, token2 common.Address, blockNumber string) (*big.Int, error) {
	var cutoff types.Big
	err := accessor.GetCutoffPair(&cutoff, contractAddress, owner, token1, token2, blockNumber)
	return cutoff.BigInt(), err
}

func GetCancelledOrFilled(contractAddress common.Address, orderhash common.Hash, blockNumber string) (*big.Int, error) {
	return accessor.GetCancelledOrFilled(contractAddress, orderhash, blockNumber)
}

func GetCancelled(contractAddress common.Address, orderhash common.Hash, blockNumber string) (*big.Int, error) {
	return accessor.GetCancelled(contractAddress, orderhash, blockNumber)
}

func BatchErc20BalanceAndAllowance(routeParam string, reqs []*BatchErc20Req) error {
	return accessor.BatchErc20BalanceAndAllowance(routeParam, reqs)
}

func BatchCall(routeParam string, reqs []BatchReq) error {
	var err error
	elems := []rpc.BatchElem{}
	elemsLength := []int{}
	for _, req := range reqs {
		elems1 := req.ToBatchElem()
		elemsLength = append(elemsLength, len(elems1))
		elems = append(elems, elems1...)
	}
	if elems, err = accessor.BatchCall(routeParam, elems); nil != err {
		return err
	} else {
		startId := 0
		for idx, req := range reqs {
			endId := startId + elemsLength[idx]
			req.FromBatchElem(elems[startId:endId])
			startId = endId
		}
		return nil
	}
}

func BatchTransactions(reqs []*BatchTransactionReq, blockNumber string) error {
	return accessor.BatchTransactions(blockNumber, 5, reqs)
}

func BatchTransactionRecipients(reqs []*BatchTransactionRecipientReq, blockNumber string) error {
	return accessor.BatchTransactionRecipients(blockNumber, 5, reqs)
}

func NewBlockIterator(startNumber, endNumber *big.Int, withTxData bool, confirms uint64) *BlockIterator {
	return accessor.BlockIterator(startNumber, endNumber, withTxData, confirms)
}

func GetSpenderAddress(protocolAddress common.Address) (spender common.Address, err error) {
	impl, ok := accessor.ProtocolAddresses[protocolAddress]
	if !ok {
		return common.Address{}, errors.New("accessor method:invalid protocol address")
	}

	return impl.DelegateAddress, nil
}

func GetFullBlock(blockNumber *big.Int, withObject bool) (interface{}, error) {
	return accessor.GetFullBlock(blockNumber, withObject)
}

func IsSpenderAddress(spender common.Address) bool {
	_, exists := accessor.DelegateAddresses[spender]
	return exists
}

func ProtocolAddresses() map[common.Address]*ProtocolAddress {
	return accessor.ProtocolAddresses
}

func DelegateAddresses() map[common.Address]bool {
	return accessor.DelegateAddresses
}

func SupportedDelegateAddress(delegate common.Address) bool {
	return accessor.DelegateAddresses[delegate]
}

func IsRelateProtocol(protocol, delegate common.Address) bool {
	protocolAddress, ok := accessor.ProtocolAddresses[protocol]
	if ok {
		return protocolAddress.DelegateAddress == delegate
	} else {
		return false
	}
}

func ProtocolImplAbi() *abi.ABI {
	return accessor.ProtocolImplAbi
}

func Erc20Abi() *abi.ABI {
	return accessor.Erc20Abi
}

func WethAbi() *abi.ABI {
	return accessor.WethAbi
}

func TokenRegistryAbi() *abi.ABI {
	return accessor.TokenRegistryAbi
}

func DelegateAbi() *abi.ABI {
	return accessor.DelegateAbi
}

//
//func NameRegistryAbi() *abi.ABI {
//	return accessor.NameRegistryAbi
//}

func Initialize(accessorOptions config.AccessorOptions, commonOptions config.CommonOptions, wethAddress common.Address) error {
	var err error
	accessor = &ethNodeAccessor{}
	accessor.mtx = sync.RWMutex{}
	if accessorOptions.FetchTxRetryCount > 0 {
		accessor.fetchTxRetryCount = accessorOptions.FetchTxRetryCount
	} else {
		accessor.fetchTxRetryCount = 60
	}
	accessor.AddressNonce = make(map[common.Address]*big.Int)
	accessor.MutilClient = NewMutilClient(accessorOptions.RawUrls)
	if nil != err {
		return err
	}

	if accessor.Erc20Abi, err = NewAbi(commonOptions.Erc20Abi); nil != err {
		return err
	}

	if accessor.WethAbi, err = NewAbi(commonOptions.WethAbi); nil != err {
		return err
	}
	accessor.WethAddress = wethAddress

	accessor.ProtocolAddresses = make(map[common.Address]*ProtocolAddress)
	accessor.DelegateAddresses = make(map[common.Address]bool)

	if protocolImplAbi, err := NewAbi(commonOptions.ProtocolImpl.ImplAbi); nil != err {
		return err
	} else {
		accessor.ProtocolImplAbi = protocolImplAbi
	}

	if transferDelegateAbi, err := NewAbi(commonOptions.ProtocolImpl.DelegateAbi); nil != err {
		return err
	} else {
		accessor.DelegateAbi = transferDelegateAbi
	}

	if tokenRegistryAbi, err := NewAbi(commonOptions.ProtocolImpl.TokenRegistryAbi); nil != err {
		return err
	} else {
		accessor.TokenRegistryAbi = tokenRegistryAbi
	}

	//if nameRegistryAbi, err := NewAbi(commonOptions.ProtocolImpl.NameRegistryAbi); nil != err {
	//	return err
	//} else {
	//	accessor.NameRegistryAbi = nameRegistryAbi
	//}

	for version, address := range commonOptions.ProtocolImpl.Address {
		impl := &ProtocolAddress{Version: version, ContractAddress: common.HexToAddress(address)}
		callMethod := accessor.ContractCallMethod(accessor.ProtocolImplAbi, impl.ContractAddress)
		var addr string
		if err := callMethod(&addr, "lrcTokenAddress", "latest"); nil != err {
			return err
		} else {
			log.Debugf("version:%s, contract:%s, lrcTokenAddress:%s", version, address, addr)
			impl.LrcTokenAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "tokenRegistryAddress", "latest"); nil != err {
			return err
		} else {
			log.Debugf("version:%s, contract:%s, tokenRegistryAddress:%s", version, address, addr)
			impl.TokenRegistryAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "delegateAddress", "latest"); nil != err {
			return err
		} else {
			log.Debugf("version:%s, contract:%s, delegateAddress:%s", version, address, addr)
			impl.DelegateAddress = common.HexToAddress(addr)
		}
		//if err := callMethod(&addr, "nameRegistryAddress", "latest"); nil != err {
		//	return err
		//} else {
		//	log.Debugf("version:%s, contract:%s, nameRegistryAddress:%s", version, address, addr)
		//	impl.NameRegistryAddress = common.HexToAddress(addr)
		//}
		accessor.ProtocolAddresses[impl.ContractAddress] = impl
		accessor.DelegateAddresses[impl.DelegateAddress] = true
	}

	accessor.MutilClient.startSyncBlockNumber()
	return nil
}

func IncludeGasPriceEvaluator() {
	accessor.gasPriceEvaluator = &GasPriceEvaluator{}
	accessor.gasPriceEvaluator.start()
}
