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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"reflect"
)

type AbiMethod struct {
	abi.Method
	ContractAddress types.Address
	rpcClient       *rpc.Client
	Pack            func(args ...interface{}) ([]byte, error)
}

func (method AbiMethod) Call(result interface{}, blockParameter string, args ...interface{}) error {
	dataBytes, err := method.Pack(args...)
	if nil != err {
		return err
	}
	data := types.ToHex(dataBytes)
	//when call a contract method，gas,gasPrice and value are not needed.
	arg := &CallArg{}
	arg.To = method.ContractAddress
	arg.Data = data
	return method.rpcClient.Call(result, "eth_call", arg, blockParameter)
}

func (m *AbiMethod) EstimateGas(args ...interface{}) (gas, gasPrice *big.Int, err error) {
	callData, err := m.Pack(args...)

	if nil != err {
		return
	}

	if err = m.rpcClient.Call(&gasPrice, "eth_GasPrice"); nil != err {
		return
	}

	callArg := &CallArg{}
	callArg.To = m.ContractAddress
	callArg.Data = types.ToHex(callData)
	callArg.GasPrice = *types.NewBigPtr(gasPrice)
	if err = m.rpcClient.Call(&gas, "eth_EstimateGas", callArg); nil != err {
		return
	}
	return
}

type AbiEvent struct {
	abi.Event
	ContractAddress types.Address
}

type Abi struct {
	abi.ABI
	abiStr string
}

func NewAbi(abiStr string) Abi {
	a := Abi{}
	a.abiStr = abiStr
	a.UnmarshalJSON([]byte(abiStr))
	return a
}

func (a Abi) NewMethod(methodName string, contractAddress types.Address, rpcClient *rpc.Client) AbiMethod {
	abiMethod := AbiMethod{}
	abiMethod.Name = methodName
	abiMethod.ContractAddress = contractAddress
	abiMethod.rpcClient = rpcClient
	abiMethod.Pack = a.newPack(methodName)
	return abiMethod
}

func (a Abi) newPack(name string) func(args ...interface{}) ([]byte, error) {
	return func(args ...interface{}) ([]byte, error) {
		return a.Pack(name, args...)
	}
}

func (a Abi) NewContract(result interface{}, address types.Address, rpcClient *rpc.Client) {
	res := reflect.ValueOf(result).Elem()
	t := reflect.TypeOf(AbiMethod{})
	res.FieldByName("Abi").Set(reflect.ValueOf(a))
	res.FieldByName("ContractAddress").Set(reflect.ValueOf(address))
	for i := 0; i < res.NumField(); i++ {
		f := res.Type().Field(i)
		if t == f.Type {
			methodName := f.Tag.Get("name")
			abiMethod := a.NewMethod(methodName, address, rpcClient)
			res.Field(i).Set(reflect.ValueOf(abiMethod))
		}
	}
}

type Erc20Token struct {
	Abi
	ContractAddress types.Address
	Name            string
	TotalSupply     AbiMethod `name:"totalSupply"`
	BalanceOf       AbiMethod `name:"balanceOf"`
	Transfer        AbiMethod
	TransferFrom    AbiMethod
	Approve         AbiMethod
	Allowance       AbiMethod
}
