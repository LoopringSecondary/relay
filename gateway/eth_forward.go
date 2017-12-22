package gateway

import (
	"github.com/Loopring/relay/ethaccessor"
	"github.com/ethereum/go-ethereum/common"
)

type EthForwarder struct {
	Accessor ethaccessor.EthNodeAccessor
}

func (e *EthForwarder) GetBalance(address, blockNumber string) (result string, err error) {
	err = e.Accessor.RetryCall(2, &result, "eth_getBalance", common.HexToAddress(address), blockNumber)
	return
}

func (e *EthForwarder) SendRawTransaction(tx string) (result string, err error) {
	err = e.Accessor.RetryCall(2, &result, "eth_sendRawTransaction", tx)
	return
}

func (e *EthForwarder) GetTransactionCount(address, blockNumber string) (result string, err error) {
	err = e.Accessor.RetryCall(2, &result, "eth_getTransactionCount", common.HexToAddress(address), blockNumber)
	return result, nil
}

func (e *EthForwarder) Call(ethCall ethaccessor.CallArg, blockNumber string) (result string, err error) {
	err = e.Accessor.RetryCall(2, &result, "eth_call", ethCall, blockNumber)
	return result, nil
}
