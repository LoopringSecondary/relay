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

package eth

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"reflect"
	"strings"
	"github.com/ethereum/go-ethereum/common"
	"github.com/Loopring/ringminer/chainclient"
)

//合约的数据结构相关的，其余的不在此处
const Erc20TokenAbiStr = `[{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"totalSupply","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_from","type":"address"},{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"balance","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_to","type":"address"},{"name":"_value","type":"uint256"}],"name":"transfer","outputs":[{"name":"success","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_owner","type":"address"},{"name":"_spender","type":"address"}],"name":"allowance","outputs":[{"name":"remaining","type":"uint256"}],"payable":false,"type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_from","type":"address"},{"indexed":true,"name":"_to","type":"address"},{"indexed":false,"name":"_value","type":"uint256"}],"name":"Transfer","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_owner","type":"address"},{"indexed":true,"name":"_spender","type":"address"},{"indexed":false,"name":"_value","type":"uint256"}],"name":"Approval","type":"event"}]`

type Contract struct {
	Abi *abi.ABI
	Address     string
}

func (c *Contract) GetAbi() interface{} {
	return c.Abi
}

func (c *Contract) GetAddress() string {
	return c.Address
}

type AbiMethod struct {
	Abi *abi.ABI
	Address	string
	abi.Method
}

//todo：如何进行签名等处理～～～
//调用eth_call，不会产生交易
func (m *AbiMethod) Call(result interface{}, blockParameter string, args ...interface{}) error {
	dataBytes, err := m.Abi.Pack(m.Name, args...)
	if (nil != err) {
		return err
	}
	data := common.ToHex(dataBytes)
	//合约的call，gas、gasPrice、value等暂为0
	arg := &CallArgs{}
	arg.From = m.Address	//设置地址，因为rpc.Client.Call不仅给eth_call使用，所以要求地址
	arg.To = m.Address
	arg.Data = data
	//发送到区块链
	return EthClient.Call(result, arg, blockParameter)
}

func (m *AbiMethod) SendTransaction(contractAddress string, args ...interface{}) error {
	//如何签名
	dataBytes, err := m.Abi.Pack(m.Name, args)
	if (nil != err) {
		return err
	}
	data := common.ToHex(dataBytes)

	println(data)
	//发送到区块链
	//ChainClient.SendRawTransaction()
	return nil
}


func  applyAbiMethod(token *chainclient.Erc20Token) {
	e := reflect.ValueOf(token).Elem()
	abi := token.GetAbi().(abi.ABI)

	for _, method := range abi.Methods {
		methodName := strings.ToUpper(method.Name[0:1]) + method.Name[1:]
		abiMethod := &AbiMethod{}
		abiMethod.Name = method.Name
		abiMethod.Abi = &abi
		abiMethod.Address = token.GetAddress()
		e.FieldByName(methodName).Set(reflect.ValueOf(abiMethod))
	}
}

func NewErc20Token(address string) *chainclient.Erc20Token {
	erc20Token := &chainclient.Erc20Token{}

	contract := &Contract{}
	cabi := &abi.ABI{}
	cabi.UnmarshalJSON([]byte(Erc20TokenAbiStr))
	contract.Abi = cabi
	contract.Address = address

	erc20Token.Contract = contract
	applyAbiMethod(erc20Token)
	return erc20Token
}


