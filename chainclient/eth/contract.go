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
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"reflect"
	"strings"
	"fmt"
)

type AbiMethod struct {
	abi.Method
	Abi     *abi.ABI
	Address string
	Client  *EthClient
}

func (m *AbiMethod) Id() string {
	return m.Id()
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
func (m *AbiMethod) SendTransaction(from types.Address, args ...interface{}) (string, error) {
	if from.IsZero() {
		for address, _ := range m.Client.senders {
			from = address
			break
		}
	}
	var gas, gasPrice *hexutil.Big
	dataBytes, err := m.Abi.Pack(m.Name, args...)

	if nil != err {
		return "", err
	}

	if err = m.Client.GasPrice(&gasPrice); nil != err {
		return "", err
	}

	callArg := &CallArg{}
	callArg.From = from.Hex()
	callArg.To = m.Address
	callArg.Data = common.ToHex(dataBytes)
	callArg.GasPrice = *gasPrice
	if err = m.Client.EstimateGas(&gas, callArg); nil != err {
		return "", err
	}

	return m.doSendTransaction(from, gas.ToInt(), gasPrice.ToInt(), dataBytes)
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

type sendArgs struct {
	From     common.Address  `json:"from"`
	To       common.Address `json:"to"`
	Gas      string   `json:"gas"`
	GasPrice string `json:"gasPrice"`
	Value    string    `json:"value"`
	Data     string   `json:"data"`
	Nonce    string `json:"nonce"`
}

func (m *AbiMethod) doSendTransaction(from types.Address, gas, gasPrice *big.Int, data []byte ) (string,error) {
	var txHash string
	var err error
	var nonce types.Big
	if err = m.Client.GetTransactionCount(&nonce, from.Hex(), "pending"); nil != err {
		return "", err
	}

	transaction := ethTypes.NewTransaction(nonce.Uint64(),
		common.HexToAddress(m.Address),
		big.NewInt(0),
		gas,
		gasPrice,
		data)

	if _, exists := m.Client.senders[from]; exists {
		err = m.Client.SignAndSendTransaction(&txHash, from, transaction)
	} else {
		args := &sendArgs{
			From:common.HexToAddress(from.Hex()),
			To:common.HexToAddress(m.Address),
			Gas:fmt.Sprintf("%#x", gas),
			GasPrice:fmt.Sprintf("%#x", gasPrice),
			Value:fmt.Sprintf("%#x", big.NewInt(int64(0))),
			Data:common.ToHex(data),
			Nonce:fmt.Sprintf("%#x", nonce.BigInt()),
		}
		err = m.Client.SendTransaction(&txHash, args)
	}
	return txHash, err
}

type AbiEvent struct {
	abi.Event
	Address string
	Client  *EthClient
}

func (e *AbiEvent) Id() string {
	return e.Event.Id().String()
}

func (e *AbiEvent) Name() string {
	return e.Event.Name
}

//todo:impl it
func (e *AbiEvent) Subscribe() {
	e.Event.Id().String()
}

func (e *AbiEvent) Unpack(v interface{}, output []byte, topics []string) error {
	return UnpackEvent(e.Inputs, v, output, topics)
}

func (m *AbiMethod) Unpack(v interface{}, hex string) error {
	return UnpackTransaction(m.Method, v, hex)
}

func applyAbiMethod(e reflect.Value, cabi *abi.ABI, address string, ethClient *EthClient) {
	for _, method := range cabi.Methods {
		methodName := strings.ToUpper(method.Name[0:1]) + method.Name[1:]
		abiMethod := &AbiMethod{}
		abiMethod.Name = method.Name
		abiMethod.Abi = cabi
		abiMethod.Address = address
		abiMethod.Client = ethClient
		abiMethod.Method = cabi.Methods[method.Name]
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
