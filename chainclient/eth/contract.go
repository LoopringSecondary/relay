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
	Abi             *abi.ABI
	ContractAddress types.Address
	Client          *EthClient
}

func (m *AbiMethod) MethodId() string {
	return types.BytesToHash(m.Id()).Hex()
}

func (m *AbiMethod) Address() types.Address {
	return m.ContractAddress
}

func (m *AbiMethod) Call(result interface{}, blockParameter string, args ...interface{}) error {
	dataBytes, err := m.Abi.Pack(m.Name, args...)
	if nil != err {
		return err
	}
	data := common.ToHex(dataBytes)
	//when call a contract method，gas,gasPrice and value are not needed.
	arg := &CallArg{}
	arg.From = m.ContractAddress
	arg.To = m.ContractAddress //when call a contract method this arg is unnecessary.
	arg.Data = data
	//todo:m.Abi.Unpack
	return m.Client.Call(result, arg, blockParameter)
}

//contract transaction
func (m *AbiMethod) SendTransaction(from types.Address, args ...interface{}) (string, error) {
	if from.IsZero() {
		for address, _ := range m.Client.senders {
			from = address
			break
		}
	}
	var gas, gasPrice types.Big
	dataBytes, err := m.Abi.Pack(m.Name, args...)

	if nil != err {
		return "", err
	}

	if err = m.Client.GasPrice(&gasPrice); nil != err {
		return "", err
	}

	callArg := &CallArg{}
	callArg.From = from
	callArg.To = m.ContractAddress
	callArg.Data = common.ToHex(dataBytes)
	callArg.GasPrice = gasPrice
	if err = m.Client.EstimateGas(&gas, callArg); nil != err {
		return "", err
	}

	return m.doSendTransaction(from, gas.BigInt(), gasPrice.BigInt(), dataBytes)
}

func (m *AbiMethod) SendTransactionWithSpecificGas(from types.Address, gas, gasPrice *big.Int, args ...interface{}) (string, error) {
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

	return m.doSendTransaction(from, gas, gasPrice, dataBytes)
}

func (m *AbiMethod) doSendTransaction(from types.Address, gas, gasPrice *big.Int, data []byte) (string, error) {
	var txHash string
	var err error
	var nonce types.Big
	if err = m.Client.GetTransactionCount(&nonce, from.Hex(), "pending"); nil != err {
		return "", err
	}

	if _, exists := m.Client.senders[from]; exists {
		transaction := ethTypes.NewTransaction(nonce.Uint64(),
			common.HexToAddress(m.ContractAddress.Hex()),
			big.NewInt(0),
			gas,
			gasPrice,
			data)
		err = m.Client.SignAndSendTransaction(&txHash, from, transaction)
	} else {
		args := &CallArg{
			From:     from,
			To:       m.ContractAddress,
			Gas:      new(types.Big).SetInt(gas),
			GasPrice: new(types.Big).SetInt(gasPrice),
			Value:    new(types.Big).SetInt(big.NewInt(0)),
			Data:     common.ToHex(data),
			Nonce:    new(types.Big).SetInt(nonce.BigInt()),
		}
		err = m.Client.SendTransaction(&txHash, args)
	}
	return txHash, err
}

type AbiEvent struct {
	abi.Event
	ContractAddress types.Address
	Client  *EthClient
}

func (e AbiEvent) Id() string {
	return e.Event.Id().String()
}

func (e AbiEvent) Name() string {
	return e.Event.Name
}

//todo:impl it
func (e AbiEvent) Subscribe() {
	e.Event.Id().String()
}

func (e AbiEvent) Unpack(v interface{}, output []byte, topics []string) error {
	return UnpackEvent(e.Inputs, v, output, topics)
}

func (m *AbiMethod) Unpack(v interface{}, hex string) error {
	return UnpackTransaction(m.Method, v, hex)
}

func applyAbiMethod(e reflect.Value, cabi *abi.ABI, address types.Address, ethClient *EthClient) {
	for _, method := range cabi.Methods {
		methodName := strings.ToUpper(method.Name[0:1]) + method.Name[1:]
		abiMethod := &AbiMethod{}
		abiMethod.Name = method.Name
		abiMethod.Abi = cabi
		abiMethod.ContractAddress = address
		abiMethod.Client = ethClient
		abiMethod.Method = cabi.Methods[method.Name]
		field := e.FieldByName(methodName)
		if field.IsValid() {
			field.Set(reflect.ValueOf(abiMethod))
		}
	}
}

func applyAbiEvent(e reflect.Value, cabi *abi.ABI, address types.Address, ethClient *EthClient) {
	for _, event := range cabi.Events {
		// todo(fuk): process this "Event"
		eventName := strings.ToUpper(event.Name[0:1]) + event.Name[1:] + "Event"
		abiEvent := &AbiEvent{}
		abiEvent.Event = event
		abiEvent.ContractAddress = address
		abiEvent.Client = ethClient
		abiEvent.Event.Name = event.Name
		field := e.FieldByName(eventName)

		if field.IsValid() {
			field.Set(reflect.ValueOf(abiEvent))
		}
	}
}

func (ethClient *EthClient) newContract(contract interface{}, addressStr, abiStr string) error {
	cabi := &abi.ABI{}
	if err := cabi.UnmarshalJSON([]byte(abiStr)); err != nil {
		log.Fatalf("error:%s", err.Error())
	}

	e := reflect.ValueOf(contract).Elem()

	address := types.HexToAddress(addressStr)
	e.FieldByName("Abi").Set(reflect.ValueOf(cabi))
	e.FieldByName("Address").Set(reflect.ValueOf(address))

	applyAbiMethod(e, cabi, address, ethClient)
	applyAbiEvent(e, cabi, address, ethClient)

	return nil
}
