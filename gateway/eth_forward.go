package gateway

import (
	"fmt"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/ethereum/go-ethereum/common"
)

type EthForwarder struct {
	Accessor ethaccessor.EthNodeAccessor
}

func (e *EthForwarder) GetBalance(address, blockNumber string) (result string, err error) {
	fmt.Println("get balance log")
	fmt.Println("intput is " + address + " " + blockNumber)
	err = e.Accessor.Call(&result, "eth_getBalance", common.HexToAddress(address), blockNumber)
	fmt.Println(result)
	return
}

func (e *EthForwarder) SendRawTransaction(tx string) (result string, err error) {
	err = e.Accessor.Call(&result, "eth_sendRawTransaction", tx)
	return
}

func (e *EthForwarder) GetTransactionCount(address, blockNumber string) (result string, err error) {
	fmt.Println("get balance log")
	fmt.Println("intput is " + address + " " + blockNumber)
	err = e.Accessor.Call(&result, "eth_getTransactionCount", common.HexToAddress(address), blockNumber)
	fmt.Println(result)
	return result, nil
}

func (e *EthForwarder) Call(ethCall ethaccessor.CallArg, blockNumber string) (result string, err error) {
	fmt.Println("get balance log")
	fmt.Println("intput is " + blockNumber)
	fmt.Println(ethCall)
	err = e.Accessor.Call(&result, "eth_call", ethCall, blockNumber)
	fmt.Println(result)
	return result, nil
}
