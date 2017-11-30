package gateway

import (
	"github.com/Loopring/relay/ethaccessor"
	"github.com/ethereum/go-ethereum/common"
)

type EthForwarder struct {
	Accessor ethaccessor.EthNodeAccessor
}

func (e *EthForwarder) GetBalance(address, blockNumber string) (result string, err error) {
	err = e.Accessor.Call(&result, "eth_getBalance", common.StringToAddress(address), blockNumber)
	return
}
