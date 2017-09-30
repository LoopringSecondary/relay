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
	"errors"
	"github.com/Loopring/ringminer/log"
	types "github.com/Loopring/ringminer/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"reflect"
	"strings"
)

type AbiMethod struct {
	abi.Method
	Abi     *abi.ABI
	Address string
	Client  *EthClient
}

func (m *AbiMethod) Call(result interface{}, blockParameter string, args ...interface{}) error {
	dataBytes, err := m.Abi.Pack(m.Name, args...)
	if nil != err {
		return err
	}
	data := common.ToHex(dataBytes)
	//when call a contract method，gas,gasPrice and value are not needed.
	arg := &CallArg{}
	arg.From = m.Address
	arg.To = m.Address //when call a contract method this arg is unnecessary.
	arg.Data = data
	//todo:m.Abi.Unpack
	return m.Client.Call(result, arg, blockParameter)
}

//contract transaction
func (m *AbiMethod) SendTransaction(from string, args ...interface{}) (string, error) {
	var gas, gasPrice *types.Big
	dataBytes, err := m.Abi.Pack(m.Name, args...)

	if nil != err {
		return "", err
	}

	if err = m.Client.GasPrice(&gasPrice); nil != err {
		return "", err
	}

	dataHex := common.ToHex(dataBytes)
	callArg := &CallArg{}
	callArg.From = from
	callArg.To = m.Address
	callArg.Data = dataHex
	callArg.GasPrice = *gasPrice
	if err = m.Client.EstimateGas(&gas, callArg); nil != err {
		return "", err
	}

	//todo: m.Abi.Pack is double used
	return m.SendTransactionWithSpecificGas(from, gas.BigInt(), gasPrice.BigInt(), args...)
}

func (m *AbiMethod) SendTransactionWithSpecificGas(from string, gas, gasPrice *big.Int, args ...interface{}) (string, error) {
	dataBytes, err := m.Abi.Pack(m.Name, args...)

	if nil != err {
		return "", err
	}

	if nil == gasPrice || gasPrice.Cmp(big.NewInt(0)) <= 0 {
		return "", errors.New("gasPrice must be setted.")
	}

	if nil == gas || gas.Cmp(big.NewInt(0)) <= 0 {
		return "", errors.New("gas must be setted.")
	}

	var nonce types.Big
	if err = m.Client.GetTransactionCount(&nonce, from, "pending"); nil != err {
		return "", err
	}

	transaction := ethTypes.NewTransaction(nonce.Uint64(),
		common.HexToAddress(m.Address),
		big.NewInt(0),
		gas,
		gasPrice,
		dataBytes)
	var txHash string

	err = m.Client.SignAndSendTransaction(&txHash, from, transaction)
	return txHash, err
}

func applyAbiMethod(e reflect.Value, cabi *abi.ABI, address string, ethClient *EthClient) {
	for _, method := range cabi.Methods {
		methodName := strings.ToUpper(method.Name[0:1]) + method.Name[1:]
		abiMethod := &AbiMethod{}
		abiMethod.Name = method.Name
		abiMethod.Abi = cabi
		abiMethod.Address = address
		abiMethod.Client = ethClient
		field := e.FieldByName(methodName)
		if field.IsValid() {
			field.Set(reflect.ValueOf(abiMethod))
		}
	}
}

func (ethClient *EthClient) newContract(contract interface{}, address, abiStr string) error {
	cabi := &abi.ABI{}
	if err := cabi.UnmarshalJSON([]byte(abiStr)); err != nil {
		log.Fatalf("error:%s", err.Error())
	}

	e := reflect.ValueOf(contract).Elem()

	e.FieldByName("Abi").Set(reflect.ValueOf(cabi))
	e.FieldByName("Address").Set(reflect.ValueOf(address))

	applyAbiMethod(e, cabi, address, ethClient)
	return nil
}
