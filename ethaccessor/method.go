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
	"errors"
	"fmt"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"time"
)

func (accessor *EthNodeAccessor) Erc20Balance(tokenAddress, address types.Address, blockParameter string) (*big.Int, error) {
	var balance types.Big
	var balanceOf AbiMethod
	if token, exists := accessor.Erc20Tokens[tokenAddress]; exists {
		balanceOf = token.BalanceOf
	} else {
		balanceOf = accessor.Erc20Abi.NewMethod("balanceOf", tokenAddress, accessor.Client)
	}
	err := balanceOf.Call(&balance, blockParameter, address)
	return balance.BigInt(), err
}

func (accessor *EthNodeAccessor) Erc20Allowance(tokenAddress, address, senderAddress types.Address, blockParameter string) (*big.Int, error) {
	var balance types.Big
	var allowance AbiMethod
	if token, exists := accessor.Erc20Tokens[tokenAddress]; exists {
		allowance = token.Allowance
	} else {
		allowance = accessor.Erc20Abi.NewMethod("allowance", tokenAddress, accessor.Client)
	}
	err := allowance.Call(&balance, blockParameter, address, senderAddress)
	return balance.BigInt(), err
}

func (accessor *EthNodeAccessor) BatchErc20BalanceAndAllowance(reqs []*BatchErc20BalanceAndAllowanceReq) error {
	reqElems := make([]rpc.BatchElem, 2*len(reqs))
	erc20Abi := accessor.Erc20Abi

	for idx, req := range reqs {
		balanceOfData, _ := erc20Abi.Pack("balanceOf", req.Address)
		balanceOfArg := &CallArg{}
		balanceOfArg.To = req.Token
		balanceOfArg.Data = types.ToHex(balanceOfData)

		allowanceData, _ := erc20Abi.Pack("allowance", req.Address, req.Spender)
		allowanceArg := &CallArg{}
		allowanceArg.To = req.Token
		allowanceArg.Data = types.ToHex(allowanceData)
		reqElems[2*idx] = rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{balanceOfArg, req.BlockParameter},
			Result: &req.Balance,
		}
		reqElems[2*idx+1] = rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{allowanceArg, req.BlockParameter},
			Result: &req.Allowance,
		}
	}

	if err := accessor.Client.BatchCall(reqElems); err != nil {
		return err
	}

	for idx, req := range reqs {
		req.BalanceErr = reqElems[2*idx].Error
		req.AllowanceErr = reqElems[2*idx+1].Error
	}
	return nil
}

func (iterator *BlockIterator) Next() (interface{}, error) {
	var block interface{}
	if iterator.withTxData {
		block = &BlockWithTxObject{}
	} else {
		block = &BlockWithTxHash{}
	}
	if nil != iterator.endNumber && iterator.endNumber.Cmp(big.NewInt(0)) > 0 && iterator.endNumber.Cmp(iterator.currentNumber) < 0 {
		return nil, errors.New("finished")
	}

	var blockNumber types.Big
	if err := iterator.ethClient.Call(&blockNumber, "eth_blockNumber"); nil != err {
		return nil, err
	} else {
		confirmNumber := iterator.currentNumber.Uint64() + iterator.confirms
		if blockNumber.Uint64() < confirmNumber {
		hasNext:
			for {
				select {
				// todo(fk):modify this duration
				case <-time.After(time.Duration(5 * time.Second)):
					if err1 := iterator.ethClient.Call(&blockNumber, "eth_blockNumber"); nil == err1 && blockNumber.Uint64() >= confirmNumber {
						break hasNext
					}
				}
			}
		}
	}

	if err := iterator.ethClient.Call(&block, "eth_getBlockByNumber", fmt.Sprintf("%#x", iterator.currentNumber), iterator.withTxData); nil != err {
		return nil, err
	} else {
		iterator.currentNumber.Add(iterator.currentNumber, big.NewInt(1))
		return block, nil
	}
}

func (iterator *BlockIterator) Prev() (interface{}, error) {
	var block interface{}
	if iterator.withTxData {
		block = &BlockWithTxObject{}
	} else {
		block = &BlockWithTxHash{}
	}
	if nil != iterator.startNumber && iterator.startNumber.Cmp(big.NewInt(0)) > 0 && iterator.startNumber.Cmp(iterator.currentNumber) > 0 {
		return nil, errors.New("finished")
	}
	prevNumber := new(big.Int).Sub(iterator.currentNumber, big.NewInt(1))
	if err := iterator.ethClient.Call(&block, "eth_getBlockByNumber", fmt.Sprintf("%#x", prevNumber), iterator.withTxData); nil != err {
		return nil, err
	} else {
		if nil == block {
			return nil, errors.New("there isn't a block with number:" + prevNumber.String())
		}
		iterator.currentNumber.Sub(iterator.currentNumber, big.NewInt(1))
		return block, nil
	}
}

func (ethAccessor *EthNodeAccessor) BlockIterator(startNumber, endNumber *big.Int, withTxData bool, confirms uint64) *BlockIterator {
	iterator := &BlockIterator{
		startNumber:   new(big.Int).Set(startNumber),
		endNumber:     endNumber,
		currentNumber: new(big.Int).Set(startNumber),
		ethClient:     ethAccessor,
		withTxData:    withTxData,
		confirms:      confirms,
	}
	return iterator
}
