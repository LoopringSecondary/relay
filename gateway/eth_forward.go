package gateway

import (
	"github.com/Loopring/relay/ethaccessor"
	"github.com/ethereum/go-ethereum/common"
)

type EthForwarder struct {
}

func (e *EthForwarder) GetBalance(address, blockNumber string) (result string, err error) {
	err = ethaccessor.GetBalance(&result, common.HexToAddress(address), blockNumber)
	//err = e.Accessor.RetryCall("latest", 2, &result, "eth_getBalance", common.HexToAddress(address), blockNumber)
	return
}

func (e *EthForwarder) SendRawTransaction(tx string) (result string, err error) {
	err = ethaccessor.SendRawTransaction(&result, tx)
	//err = e.Accessor.RetryCall("latest", 2, &result, "eth_sendRawTransaction", tx)
	return
}

func (e *EthForwarder) GetTransactionCount(address, blockNumber string) (result string, err error) {
	err = ethaccessor.GetTransactionCount(&result, common.HexToAddress(address), blockNumber)
	return
	//err = e.Accessor.RetryCall("latest", 2, &result, "eth_getTransactionCount", common.HexToAddress(address), blockNumber)
	//return result, nil
}

func (e *EthForwarder) Call(ethCall *ethaccessor.CallArg, blockNumber string) (result string, err error) {
	err = ethaccessor.Call(&result, ethCall, blockNumber)
	return
	//err = e.Accessor.RetryCall("latest", 2, &result, "eth_call", ethCall, blockNumber)
	//return result, nil
}
