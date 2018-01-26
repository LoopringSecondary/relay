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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

var accessor *ethNodeAccessor

func BlockNumber(result interface{}) error {
	return accessor.RetryCall("latest", 2, result, "eth_blockNumber")
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

func Call(result interface{}, ethCall CallArg, blockNumber string) error {
	return accessor.RetryCall(blockNumber, 2, result, "eth_call", ethCall, blockNumber)
}

func GetBlockByNumber(result interface{}, blockNumber string, withObject bool) error {
	return accessor.RetryCall(blockNumber, 2, result, "eth_getBlockByNumber", fmt.Sprintf("%#x", blockNumber), withObject)
}

func GetBlockByHash(result types.CheckNull, blockHash string, withObject bool) error {
	for _,c := range accessor.clients {
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
	for _,c := range accessor.clients {
		if err := c.client.Call(result, "eth_getTransactionByHash", txHash); nil == err {
			if !result.IsNull() {
				return nil
			}
		}
	}
	return fmt.Errorf("no transaction with hash:%s", txHash)
}

//todo:
func GasPrice() {}

func GetBlockTransactionCountByHash(result interface{}, blockHash string, blockParameter string) error {
	return accessor.RetryCall("latest", 2, result, "eth_getBlockTransactionCountByHash", blockHash)

}

func GetBlockTransactionCountByNumber(result interface{}, blockNumber string) error {
	return accessor.RetryCall(blockNumber, 2, result, "eth_getBlockTransactionCountByNumber", blockNumber)

}

func Synced() bool {
	for _, c := range accessor.clients {
		if c.syncingResult.isSynced() {
			return true
		}
	}
	return false
}

func EstimateGas(callData []byte, to common.Address, blockNumber string) (gas, gasPrice *big.Int, err error) {
	return accessor.EstimateGas(blockNumber, callData, to)
}

func SignAndSendTransaction(sender accounts.Account, to common.Address, gas, gasPrice, value *big.Int, callData []byte) (string, error) {
	return accessor.ContractSendTransactionByData("latest", sender, to, gas, gasPrice, value, callData)
}

func ContractSendTransactionMethod(routeParam string, a *abi.ABI, contractAddress common.Address) func(sender accounts.Account, methodName string, gas, gasPrice, value *big.Int, args ...interface{}) (string, error) {
	return accessor.ContractSendTransactionMethod(routeParam, a, contractAddress)
}

func ContractCallMethod(a *abi.ABI, contractAddress common.Address) func(result interface{}, methodName, blockParameter string, args ...interface{}) error {
	return accessor.ContractCallMethod(a, contractAddress)
}

func ProtocolCanSubmit(implAddress *ProtocolAddress, ringhash common.Hash, miner common.Address) (bool, error) {
	callMethod := accessor.ContractCallMethod(accessor.RinghashRegistryAbi, implAddress.RinghashRegistryAddress)
	var canSubmit types.Big
	if err := callMethod(&canSubmit, "canSubmit", "latest", ringhash, miner); nil != err {
		return false, err
	} else {
		if canSubmit.Int() <= 0 {
			return true, nil
		}
	}
	return false, nil
}

func Erc20Balance(tokenAddress, ownerAddress common.Address, blockParameter string) (*big.Int, error) {
	return accessor.Erc20Balance(tokenAddress, ownerAddress, blockParameter)
}

func Erc20Allowance(tokenAddress, ownerAddress, spender common.Address, blockParameter string) (*big.Int, error) {
	return accessor.Erc20Allowance(tokenAddress, ownerAddress, spender, blockParameter)
}

func GetCutoff(contractAddress, owner common.Address, blockNumber string) (*big.Int, error) {
	var cutoff types.Big
	err := accessor.GetCutoff(&cutoff, contractAddress, owner, blockNumber)
	return cutoff.BigInt(), err
}

func GetCancelledOrFilled(contractAddress common.Address, orderhash common.Hash, blockNumber string) (*big.Int, error) {
	return accessor.GetCancelledOrFilled(contractAddress, orderhash, blockNumber)
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

func ProtocolAddresses() map[common.Address]*ProtocolAddress {
	return accessor.ProtocolAddresses
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

func RinghashRegistryAbi() *abi.ABI {
	return accessor.RinghashRegistryAbi
}

func DelegateAbi() *abi.ABI {
	return accessor.DelegateAbi
}

func Initialize(accessorOptions config.AccessorOptions, commonOptions config.CommonOptions, wethAddress common.Address) error {
	var err error
	accessor = &ethNodeAccessor{}
	accessor.MutilClient = &MutilClient{}
	accessor.MutilClient.Dail(accessorOptions.RawUrls)
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

	if protocolImplAbi, err := NewAbi(commonOptions.ProtocolImpl.ImplAbi); nil != err {
		return err
	} else {
		accessor.ProtocolImplAbi = protocolImplAbi
	}
	if registryAbi, err := NewAbi(commonOptions.ProtocolImpl.RegistryAbi); nil != err {
		return err
	} else {
		accessor.RinghashRegistryAbi = registryAbi
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

	for version, address := range commonOptions.ProtocolImpl.Address {
		impl := &ProtocolAddress{Version: version, ContractAddress: common.HexToAddress(address)}
		callMethod := accessor.ContractCallMethod(accessor.ProtocolImplAbi, impl.ContractAddress)
		var addr string
		if err := callMethod(&addr, "lrcTokenAddress", "latest"); nil != err {
			return err
		} else {
			impl.LrcTokenAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "ringhashRegistryAddress", "latest"); nil != err {
			return err
		} else {
			impl.RinghashRegistryAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "tokenRegistryAddress", "latest"); nil != err {
			return err
		} else {
			impl.TokenRegistryAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "delegateAddress", "latest"); nil != err {
			return err
		} else {
			impl.DelegateAddress = common.HexToAddress(addr)
		}
		accessor.ProtocolAddresses[impl.ContractAddress] = impl
	}
	accessor.MutilClient.startSyncStatus()
	return nil
}
