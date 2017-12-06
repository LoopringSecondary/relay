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
	"github.com/Loopring/relay/config"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type EthNodeAccessor struct {
	Erc20Abi            *abi.ABI
	ProtocolImplAbi     *abi.ABI
	DelegateAbi         *abi.ABI
	RinghashRegistryAbi *abi.ABI
	TokenRegistryAbi    *abi.ABI

	ProtocolAddresses map[common.Address]*ProtocolAddress
	*rpc.Client
}

func NewAccessor(accessorOptions config.AccessorOptions, commonOptions config.CommonOptions) (*EthNodeAccessor, error) {
	var err error
	accessor := &EthNodeAccessor{}
	accessor.Client, err = rpc.Dial(accessorOptions.RawUrl)
	if nil != err {
		return nil, err
	}

	if accessor.Erc20Abi, err = NewAbi(commonOptions.Erc20Abi); nil != err {
		return nil, err
	}
	accessor.ProtocolAddresses = make(map[common.Address]*ProtocolAddress)

	if protocolImplAbi, err := NewAbi(commonOptions.ProtocolImpl.ImplAbi); nil != err {
		return nil, err
	} else {
		accessor.ProtocolImplAbi = protocolImplAbi
	}
	if registryAbi, err := NewAbi(commonOptions.ProtocolImpl.RegistryAbi); nil != err {
		return nil, err
	} else {
		accessor.RinghashRegistryAbi = registryAbi
	}
	if transferDelegateAbi, err := NewAbi(commonOptions.ProtocolImpl.DelegateAbi); nil != err {
		return nil, err
	} else {
		accessor.DelegateAbi = transferDelegateAbi
	}
	if tokenRegistryAbi, err := NewAbi(commonOptions.ProtocolImpl.TokenRegistryAbi); nil != err {
		return nil, err
	} else {
		accessor.TokenRegistryAbi = tokenRegistryAbi
	}

	for version, address := range commonOptions.ProtocolImpl.Address {
		impl := &ProtocolAddress{Version: version, ContractAddress: common.HexToAddress(address)}
		callMethod := accessor.ContractCallMethod(accessor.ProtocolImplAbi, impl.ContractAddress)
		var addr string
		if err := callMethod(&addr, "lrcTokenAddress", "latest"); nil != err {
			return nil, err
		} else {
			impl.LrcTokenAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "ringhashRegistryAddress", "latest"); nil != err {
			return nil, err
		} else {
			impl.RinghashRegistryAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "tokenRegistryAddress", "latest"); nil != err {
			return nil, err
		} else {
			impl.TokenRegistryAddress = common.HexToAddress(addr)
		}
		if err := callMethod(&addr, "delegateAddress", "latest"); nil != err {
			return nil, err
		} else {
			impl.DelegateAddress = common.HexToAddress(addr)
		}
		accessor.ProtocolAddresses[impl.ContractAddress] = impl
	}

	return accessor, nil
}
