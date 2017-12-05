package gateway

import (
	"github.com/Loopring/relay/ethaccessor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type EthForwarder struct {
	Accessor ethaccessor.EthNodeAccessor
}

func (e *EthForwarder) GetBalance(address, blockNumber string) (result string, err error) {
	err = e.Accessor.Call(&result, "eth_getBalance", common.StringToAddress(address), blockNumber)
	return
}

func (e *EthForwarder) SendRawTransaction(tx string) (result string, err error) {
	hex := hexutil.Bytes{}
	hex.UnmarshalText([]byte(tx))
	//err = e.Accessor.Call(&result, "eth_sendRawTransaction", hex)
	return
}

func (e *EthForwarder) GetTransactionCount(address , blockNumber string) (result string, err error) {
	//err = e.Accessor.Call(&result, "eth_sendRawTransaction", hex)
	return "0x1", nil
}
