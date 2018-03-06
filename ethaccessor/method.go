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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"time"
)

func (accessor *ethNodeAccessor) Erc20Balance(tokenAddress, ownerAddress common.Address, blockParameter string) (*big.Int, error) {
	var balance types.Big
	callMethod := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddress)
	if err := callMethod(&balance, "balanceOf", blockParameter, ownerAddress); nil != err {
		return nil, err
	} else {
		return balance.BigInt(), err
	}
}

func (accessor *ethNodeAccessor) RetryCall(routeParam string, retry int, result interface{}, method string, args ...interface{}) error {
	var err error
	for i := 0; i < retry; i++ {
		if _, err = accessor.Call(routeParam, result, method, args...); nil != err {
			continue
		} else {
			return nil
		}
	}
	return err
}

func (accessor *ethNodeAccessor) Erc20Allowance(tokenAddress, ownerAddress, spenderAddress common.Address, blockParameter string) (*big.Int, error) {
	var allowance types.Big
	callMethod := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddress)
	if err := callMethod(&allowance, "allowance", blockParameter, ownerAddress, spenderAddress); nil != err {
		return nil, err
	} else {
		return allowance.BigInt(), err
	}
}

func (accessor *ethNodeAccessor) GetCancelledOrFilled(contractAddress common.Address, orderhash common.Hash, blockNumStr string) (*big.Int, error) {
	var amount types.Big
	if _, ok := accessor.ProtocolAddresses[contractAddress]; !ok {
		return nil, errors.New("accessor: contract address invalid -> " + contractAddress.Hex())
	}
	callMethod := accessor.ContractCallMethod(accessor.ProtocolImplAbi, contractAddress)
	if err := callMethod(&amount, "cancelledOrFilled", blockNumStr, orderhash); err != nil {
		return nil, err
	}

	return amount.BigInt(), nil
}

func (accessor *ethNodeAccessor) GetCutoff(result interface{}, contractAddress, owner common.Address, blockNumStr string) error {
	if _, ok := accessor.ProtocolAddresses[contractAddress]; !ok {
		return errors.New("accessor: contract address invalid -> " + contractAddress.Hex())
	}
	callMethod := accessor.ContractCallMethod(accessor.ProtocolImplAbi, contractAddress)
	if err := callMethod(result, "cutoffs", blockNumStr, owner); err != nil {
		return err
	}
	return nil
}

func (accessor *ethNodeAccessor) BatchErc20BalanceAndAllowance(routeParam string, reqs []*BatchErc20Req) error {
	reqElems := make([]rpc.BatchElem, 2*len(reqs))
	erc20Abi := accessor.Erc20Abi

	for idx, req := range reqs {
		balanceOfData, _ := erc20Abi.Pack("balanceOf", req.Owner)
		balanceOfArg := &CallArg{}
		balanceOfArg.To = req.Token
		balanceOfArg.Data = common.ToHex(balanceOfData)

		allowanceData, _ := erc20Abi.Pack("allowance", req.Owner, req.Spender)
		allowanceArg := &CallArg{}
		allowanceArg.To = req.Token
		allowanceArg.Data = common.ToHex(allowanceData)
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

	if _, err := accessor.MutilClient.BatchCall(routeParam, reqElems); err != nil {
		return err
	}

	for idx, req := range reqs {
		req.BalanceErr = reqElems[2*idx].Error
		req.AllowanceErr = reqElems[2*idx+1].Error
	}
	return nil
}

func (accessor *ethNodeAccessor) BatchTransactions(routeParam string, retry int, reqs []*BatchTransactionReq) error {
	if len(reqs) < 1 || retry < 1 {
		return fmt.Errorf("ethaccessor:batchTransactions retry or reqs invalid")
	}

	reqElems := make([]rpc.BatchElem, len(reqs))
	for idx, req := range reqs {
		reqElems[idx] = rpc.BatchElem{
			Method: "eth_getTransactionByHash",
			Args:   []interface{}{req.TxHash},
			Result: &req.TxContent,
		}
	}

	var err error
	for i := 0; i < retry; i++ {
		if _, err = accessor.MutilClient.BatchCall(routeParam, reqElems); err == nil {
			break
		}
	}
	if err != nil {
		return err
	}

	for idx, v := range reqElems {
		var (
			tx     Transaction
			txhash string = reqs[idx].TxHash
		)

		if v.Error == nil {
			continue
		}

		for i := 0; i < retry; i++ {
			if _, v.Error = accessor.Call(routeParam, &tx, "eth_getTransactionByHash", txhash); v.Error == nil {
				break
			}
		}
		if v.Error != nil {
			return v.Error
		}
	}

	return nil
}

func (accessor *ethNodeAccessor) BatchTransactionRecipients(routeParam string, retry int, reqs []*BatchTransactionRecipientReq) error {
	if len(reqs) < 1 || retry < 1 {
		return fmt.Errorf("ethaccessor:batchTransactionRecipients retry or reqs invalid")
	}

	reqElems := make([]rpc.BatchElem, len(reqs))
	for idx, req := range reqs {
		reqElems[idx] = rpc.BatchElem{
			Method: "eth_getTransactionReceipt",
			Args:   []interface{}{req.TxHash},
			Result: &req.TxContent,
		}
	}

	var err error
	for i := 0; i < retry; i++ {
		if _, err = accessor.BatchCall(routeParam, reqElems); err == nil {
			break
		}
	}
	if err != nil {
		return err
	}

	for idx, v := range reqElems {
		var (
			tx     TransactionReceipt
			txhash string = reqs[idx].TxHash
		)

		if v.Error == nil {
			continue
		}

		for i := 0; i < retry; i++ {
			if _, v.Error = accessor.Call(routeParam, &tx, "eth_getTransactionReceipt", txhash); v.Error == nil {
				break
			}
		}
		if v.Error != nil {
			return v.Error
		}
	}

	return nil
}

func (accessor *ethNodeAccessor) EstimateGas(routeParam string, callData []byte, to common.Address) (gas, gasPrice *big.Int, err error) {
	var gasBig, gasPriceBig types.Big
	if nil == accessor.gasPriceEvaluator.gasPrice {
		if err = accessor.RetryCall(routeParam, 2, &gasPriceBig, "eth_gasPrice"); nil != err {
			return
		}
	} else {
		gasPriceBig = new(types.Big).SetInt(accessor.gasPriceEvaluator.gasPrice)
	}

	callArg := &CallArg{}
	callArg.To = to
	callArg.Data = common.ToHex(callData)
	callArg.GasPrice = gasPriceBig
	if err = accessor.RetryCall(routeParam, 2, &gasBig, "eth_estimateGas", callArg); nil != err {
		return
	}
	gasPrice = gasPriceBig.BigInt()
	gas = gasBig.BigInt()
	return
}

func (accessor *ethNodeAccessor) ContractCallMethod(a *abi.ABI, contractAddress common.Address) func(result interface{}, methodName, blockParameter string, args ...interface{}) error {
	return func(result interface{}, methodName string, blockParameter string, args ...interface{}) error {
		if callData, err := a.Pack(methodName, args...); nil != err {
			return err
		} else {
			arg := &CallArg{}
			arg.From = contractAddress
			arg.To = contractAddress
			arg.Data = common.ToHex(callData)
			return accessor.RetryCall(blockParameter, 2, result, "eth_call", arg, blockParameter)
		}
	}
}

func (ethAccessor *ethNodeAccessor) SignAndSendTransaction(result interface{}, sender common.Address, tx *ethTypes.Transaction) error {
	var err error
	if tx, err = crypto.SignTx(sender, tx, nil); nil != err {
		return err
	}
	if txData, err := rlp.EncodeToBytes(tx); nil != err {
		return err
	} else {
		log.Debugf("txhash:%s, nonce:%d, value:%s, gas:%s, gasPrice:%s", tx.Hash().Hex(), tx.Nonce(), tx.Value().String(), tx.Gas().String(), tx.GasPrice().String())
		err = ethAccessor.RetryCall("latest", 2, result, "eth_sendRawTransaction", common.ToHex(txData))
		if err != nil {
			log.Errorf("accessor, Sign and send transaction error:%s", err.Error())
		}
		return err
	}
}

func (accessor *ethNodeAccessor) ContractSendTransactionByData(routeParam string, sender common.Address, to common.Address, gas, gasPrice, value *big.Int, callData []byte) (string, error) {
	if nil == gasPrice || gasPrice.Cmp(big.NewInt(0)) <= 0 {
		return "", errors.New("gasPrice must be setted.")
	}
	if nil == gas || gas.Cmp(big.NewInt(0)) <= 0 {
		return "", errors.New("gas must be setted.")
	}
	var txHash string
	var nonce types.Big
	if err := accessor.RetryCall(routeParam, 2, &nonce, "eth_getTransactionCount", sender.Hex(), "pending"); nil != err {
		return "", err
	}
	if value == nil {
		value = big.NewInt(0)
	}
	// todo: modify gas
	gas.SetString("1000000", 0)
	transaction := ethTypes.NewTransaction(nonce.Uint64(),
		common.HexToAddress(to.Hex()),
		value,
		gas,
		gasPrice,
		callData)
	if err := accessor.SignAndSendTransaction(&txHash, sender, transaction); nil != err {
		return "", err
	} else {
		return txHash, err
	}
}

//gas, gasPrice can be set to nil
func (accessor *ethNodeAccessor) ContractSendTransactionMethod(routeParam string, a *abi.ABI, contractAddress common.Address) func(sender common.Address, methodName string, gas, gasPrice, value *big.Int, args ...interface{}) (string, error) {
	return func(sender common.Address, methodName string, gas, gasPrice, value *big.Int, args ...interface{}) (string, error) {
		if callData, err := a.Pack(methodName, args...); nil != err {
			return "", err
		} else {
			if nil == gas || nil == gasPrice {
				if gas, gasPrice, err = accessor.EstimateGas(routeParam, callData, contractAddress); nil != err {
					return "", err
				}
			}
			gas.Add(gas, big.NewInt(int64(1000)))
			return accessor.ContractSendTransactionByData(routeParam, sender, contractAddress, gas, gasPrice, value, callData)
		}
	}
}

func (iterator *BlockIterator) Next() (interface{}, error) {
	if nil != iterator.endNumber && iterator.endNumber.Cmp(big.NewInt(0)) > 0 && iterator.endNumber.Cmp(iterator.currentNumber) < 0 {
		return nil, errors.New("finished")
	}

	var blockNumber types.Big
	if err := iterator.ethClient.RetryCall("latest", 2, &blockNumber, "eth_blockNumber"); nil != err {
		return nil, err
	} else {
		confirmNumber := iterator.currentNumber.Uint64() + iterator.confirms
		if blockNumber.Uint64() < confirmNumber {
		hasNext:
			for {
				select {
				// todo(fk):modify this duration
				case <-time.After(time.Duration(5 * time.Second)):
					if err1 := iterator.ethClient.RetryCall("latest", 2, &blockNumber, "eth_blockNumber"); nil == err1 && blockNumber.Uint64() >= confirmNumber {
						break hasNext
					}
				}
			}
		}
	}

	block, err := iterator.ethClient.getFullBlock(iterator.currentNumber, iterator.withTxData)
	if nil == err {
		iterator.currentNumber.Add(iterator.currentNumber, big.NewInt(1))
	}
	return block, err
}

func (accessor *ethNodeAccessor) getFullBlockFromCacheByHash(hash string) (*BlockWithTxAndReceipt, error) {
	blockWithTxAndReceipt := &BlockWithTxAndReceipt{}

	if blockData, err := cache.Get(hash); nil == err {
		if err = json.Unmarshal(blockData, blockWithTxAndReceipt); nil == err {
			return blockWithTxAndReceipt, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (accessor *ethNodeAccessor) getFullBlock(blockNumber *big.Int, withTxObject bool) (interface{}, error) {
	blockWithTxHash := &BlockWithTxHash{}

	if err := accessor.RetryCall(blockNumber.String(), 2, &blockWithTxHash, "eth_getBlockByNumber", fmt.Sprintf("%#x", blockNumber), false); nil != err {
		return nil, err
	} else {
		if !withTxObject {
			return blockWithTxHash, nil
		} else {

			if blockWithTxAndReceipt, err := accessor.getFullBlockFromCacheByHash(blockWithTxHash.Hash.Hex()); nil == err && nil != blockWithTxAndReceipt {
				return blockWithTxAndReceipt, nil
			} else {
				blockWithTxAndReceipt := &BlockWithTxAndReceipt{}
				blockWithTxAndReceipt.Block = blockWithTxHash.Block
				blockWithTxAndReceipt.Transactions = []Transaction{}
				blockWithTxAndReceipt.Receipts = []TransactionReceipt{}

				txno := len(blockWithTxHash.Transactions)
				if txno == 0 {
					return blockWithTxAndReceipt, nil
				}

				var (
					txReqs = make([]*BatchTransactionReq, txno)
					rcReqs = make([]*BatchTransactionRecipientReq, txno)
				)
				for idx, txstr := range blockWithTxHash.Transactions {
					var (
						txreq        BatchTransactionReq
						rcreq        BatchTransactionRecipientReq
						tx           Transaction
						rc           TransactionReceipt
						txerr, rcerr error
					)
					txreq.TxHash = txstr
					txreq.TxContent = tx
					txreq.Err = txerr

					rcreq.TxHash = txstr
					rcreq.TxContent = rc
					rcreq.Err = rcerr

					txReqs[idx] = &txreq
					rcReqs[idx] = &rcreq
				}

				if err := BatchTransactions(txReqs, blockWithTxAndReceipt.Number.BigInt().String()); err != nil {
					return nil, err
				}
				if err := BatchTransactionRecipients(rcReqs, blockWithTxAndReceipt.Number.BigInt().String()); err != nil {
					return nil, err
				}

				for idx, _ := range txReqs {
					blockWithTxAndReceipt.Transactions = append(blockWithTxAndReceipt.Transactions, txReqs[idx].TxContent)
					blockWithTxAndReceipt.Receipts = append(blockWithTxAndReceipt.Receipts, rcReqs[idx].TxContent)
				}

				if blockData, err := json.Marshal(blockWithTxAndReceipt); nil == err {
					cache.Set(blockWithTxHash.Hash.Hex(), blockData, int64(86400))
				}
				return blockWithTxAndReceipt, nil
			}

		}
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
	if err := iterator.ethClient.RetryCall(prevNumber.String(), 2, &block, "eth_getBlockByNumber", fmt.Sprintf("%#x", prevNumber), iterator.withTxData); nil != err {
		return nil, err
	} else {
		if nil == block {
			return nil, errors.New("there isn't a block with number:" + prevNumber.String())
		}
		iterator.currentNumber.Sub(iterator.currentNumber, big.NewInt(1))
		return block, nil
	}
}

func (ethAccessor *ethNodeAccessor) BlockIterator(startNumber, endNumber *big.Int, withTxData bool, confirms uint64) *BlockIterator {
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

func (ethAccessor *ethNodeAccessor) GetSenderAddress(protocol common.Address) (common.Address, error) {
	impl, ok := ethAccessor.ProtocolAddresses[protocol]
	if !ok {
		return common.Address{}, errors.New("accessor method:invalid protocol address")
	}

	return impl.DelegateAddress, nil
}
