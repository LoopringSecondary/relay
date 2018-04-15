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
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

type BatchErc20Req struct {
	Owner          common.Address
	Token          common.Address
	Symbol         string
	Spender        common.Address
	BlockParameter string
	Balance        types.Big
	Allowance      types.Big
	BalanceErr     error
	AllowanceErr   error
}

type BatchReq interface {
	ToBatchElem() []rpc.BatchElem
	FromBatchElem(batchElems []rpc.BatchElem)
}

type BatchBalanceReq struct {
	Owner          common.Address
	Token          common.Address
	BlockParameter string
	Balance        types.Big
	BalanceErr     error
}

type BatchBalanceReqs []*BatchBalanceReq

func (reqs BatchBalanceReqs) ToBatchElem() []rpc.BatchElem {
	reqElems := make([]rpc.BatchElem, len(reqs))
	erc20Abi := accessor.Erc20Abi

	for idx, req := range reqs {
		if types.IsZeroAddress(req.Token) {
			reqElems[idx] = rpc.BatchElem{
				Method: "eth_getBalance",
				Args:   []interface{}{req.Owner.Hex(), req.BlockParameter},
				Result: &req.Balance,
			}
		} else {
			balanceOfData, _ := erc20Abi.Pack("balanceOf", req.Owner)
			balanceOfArg := &CallArg{}
			balanceOfArg.To = req.Token
			balanceOfArg.Data = common.ToHex(balanceOfData)

			reqElems[idx] = rpc.BatchElem{
				Method: "eth_call",
				Args:   []interface{}{balanceOfArg, req.BlockParameter},
				Result: &req.Balance,
			}
		}
	}
	return reqElems
}

func (reqs BatchBalanceReqs) FromBatchElem(elems []rpc.BatchElem) {
	for idx, req := range elems {
		reqs[idx].BalanceErr = req.Error
	}
}

type BatchErc20AllowanceReq struct {
	Owner          common.Address
	Token          common.Address
	BlockParameter string
	Spender        common.Address
	Allowance      types.Big
	AllowanceErr   error
}

type BatchErc20AllowanceReqs []*BatchErc20AllowanceReq

func (reqs BatchErc20AllowanceReqs) ToBatchElem() []rpc.BatchElem {
	reqElems := make([]rpc.BatchElem, len(reqs))
	erc20Abi := accessor.Erc20Abi

	for idx, req := range reqs {
		balanceOfData, _ := erc20Abi.Pack("allowance", req.Owner, req.Spender)
		balanceOfArg := &CallArg{}
		balanceOfArg.To = req.Token
		balanceOfArg.Data = common.ToHex(balanceOfData)

		reqElems[idx] = rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{balanceOfArg, req.BlockParameter},
			Result: &req.Allowance,
		}
	}
	return reqElems
}

func (reqs BatchErc20AllowanceReqs) FromBatchElem(elems []rpc.BatchElem) {
	for idx, req := range elems {
		reqs[idx].AllowanceErr = req.Error
	}
}

type BatchTransactionReq struct {
	TxHash    string
	TxContent Transaction
	Err       error
}

type BatchTransactionRecipientReq struct {
	TxHash    string
	TxContent TransactionReceipt
	Err       error
}
