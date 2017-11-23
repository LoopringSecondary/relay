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
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	Erc20TokenAbiStr       string = `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferFrom","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"who","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"},{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"owner","type":"address"},{"indexed":true,"name":"spender","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`
	ImplAbiStr             string = `[{"constant":true,"inputs":[{"name":"signer","type":"address"},{"name":"hash","type":"bytes32"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"verifySignature","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"orderHash","type":"bytes32"}],"name":"getOrderCancelled","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_MAX_VALUE","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"filled","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"bytes32"}],"name":"cancelled","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"MARGIN_SPLIT_PERCENTAGE_BASE","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"ringIndex","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"addresses","type":"address[3]"},{"name":"orderValues","type":"uint256[7]"},{"name":"buyNoMoreThanAmountB","type":"bool"},{"name":"marginSplitPercentage","type":"uint8"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"cancelOrder","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"RATE_RATIO_SCALE","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"lrcTokenAddress","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"tokenRegistryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"addressList","type":"address[2][]"},{"name":"uintArgsList","type":"uint256[7][]"},{"name":"uint8ArgsList","type":"uint8[2][]"},{"name":"buyNoMoreThanAmountBList","type":"bool[]"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"},{"name":"relay","type":"address"},{"name":"feeRecepient","type":"address"},{"name":"throwIfLRCIsInsuffcient","type":"bool"}],"name":"submitRing","outputs":[{"name":"","type":"bytes32"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"delegateAddress","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"orderHash","type":"bytes32"}],"name":"getOrderFilled","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"tokenAddress","type":"address"},{"name":"tokenOwner","type":"address"}],"name":"getSpendable","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"maxRingSize","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"ringhashRegistryAddress","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"cutoff","type":"uint256"}],"name":"setCutoff","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_LRC","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"cutoffs","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"rateRatioCVSThreshold","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"FEE_SELECT_MARGIN_SPLIT","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"inputs":[{"name":"_lrcTokenAddress","type":"address"},{"name":"_tokenRegistryAddress","type":"address"},{"name":"_ringhashRegistryAddress","type":"address"},{"name":"_delegateAddress","type":"address"},{"name":"_maxRingSize","type":"uint256"},{"name":"_rateRatioCVSThreshold","type":"uint256"}],"payable":false,"type":"constructor"},{"payable":true,"type":"fallback"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_ringIndex","type":"uint256"},{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_ringhash","type":"bytes32"},{"indexed":true,"name":"_miner","type":"address"},{"indexed":true,"name":"_feeRecepient","type":"address"},{"indexed":false,"name":"_ringhashFound","type":"bool"}],"name":"RingMinedEvent","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_ringIndex","type":"uint256"},{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_ringhash","type":"bytes32"},{"indexed":false,"name":"_prevOrderHash","type":"bytes32"},{"indexed":true,"name":"_orderHash","type":"bytes32"},{"indexed":false,"name":"_nextOrderHash","type":"bytes32"},{"indexed":false,"name":"_amountS","type":"uint256"},{"indexed":false,"name":"_amountB","type":"uint256"},{"indexed":false,"name":"_lrcReward","type":"uint256"},{"indexed":false,"name":"_lrcFee","type":"uint256"}],"name":"OrderFilled","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_orderHash","type":"bytes32"},{"indexed":false,"name":"_amountCancelled","type":"uint256"}],"name":"OrderCancelled","type":"event"},{"anonymous":false,"inputs":[{"indexed":false,"name":"_time","type":"uint256"},{"indexed":false,"name":"_blocknumber","type":"uint256"},{"indexed":true,"name":"_address","type":"address"},{"indexed":false,"name":"_cutoff","type":"uint256"}],"name":"CutoffTimestampChanged","type":"event"}]`
	RinghashRegistryAbiStr string = `[{"constant":true,"inputs":[{"name":"relay","type":"address"},{"name":"ringSize","type":"uint256"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"}],"name":"calculateRinghash","outputs":[{"name":"","type":"bytes32"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"ringhash","type":"bytes32"}],"name":"ringhashFound","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"blocksToLive","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"ringhash","type":"bytes32"},{"name":"relay","type":"address"}],"name":"canSubmit","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"ringSize","type":"uint256"},{"name":"relay","type":"address"},{"name":"vList","type":"uint8[]"},{"name":"rList","type":"bytes32[]"},{"name":"sList","type":"bytes32[]"}],"name":"submitRinghash","outputs":[],"payable":false,"type":"function"},{"inputs":[{"name":"_blocksToLive","type":"uint256"}],"payable":false,"type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"name":"_ringminer","type":"address"},{"indexed":true,"name":"_ringhash","type":"bytes32"}],"name":"RinghashSubmitted","type":"event"}]`
	TransferDelegateAbiStr string = `[{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"addVersion","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"token","type":"address"},{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transferToken","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"address"}],"name":"versioned","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"getVersions","outputs":[{"name":"","type":"address[]"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"versions","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"addr","type":"address"}],"name":"removeVersion","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"tokenAddress","type":"address"},{"name":"_owner","type":"address"}],"name":"getSpendable","outputs":[{"name":"","type":"uint256"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"addr","type":"address"},{"indexed":false,"name":"version","type":"uint256"}],"name":"VersionAdded","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"addr","type":"address"},{"indexed":false,"name":"version","type":"uint256"}],"name":"VersionRemoved","type":"event"}]`
	TokenRegistryAbiStr    string = `[{"constant":false,"inputs":[{"name":"_token","type":"address"},{"name":"_symbol","type":"string"}],"name":"unregisterToken","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"symbol","type":"string"}],"name":"getAddressBySymbol","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"_token","type":"address"}],"name":"isTokenRegistered","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"","type":"uint256"}],"name":"tokens","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":true,"inputs":[],"name":"owner","outputs":[{"name":"","type":"address"}],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"_token","type":"address"},{"name":"_symbol","type":"string"}],"name":"registerToken","outputs":[],"payable":false,"type":"function"},{"constant":false,"inputs":[{"name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"payable":false,"type":"function"},{"constant":true,"inputs":[{"name":"symbol","type":"string"}],"name":"isTokenRegisteredBySymbol","outputs":[{"name":"","type":"bool"}],"payable":false,"type":"function"}]`
)

type EthNodeAccessor struct {
	Erc20Abi    Abi
	Erc20Tokens map[types.Address]*Erc20Token

	signer  *ethTypes.HomesteadSigner
	Senders map[types.Address]*Account

	*rpc.Client
}

func NewAccessor(clientConfig config.ChainClientOptions) *EthNodeAccessor {
	accessor := &EthNodeAccessor{}
	var err error
	accessor.Client, err = rpc.Dial(clientConfig.RawUrl)
	if nil != err {
		panic(err)
	}

	//todo:change to keystore
	//passphrase := &types.Passphrase{}
	//passphrase.SetBytes(passphraseBytes)
	//if accounts, err := DecryptAccounts(passphrase, clientConfig.Senders); nil != err {
	//	panic(err)
	//} else {
	//	ethClient.Senders = accounts
	//}

	accessor.Erc20Abi = NewAbi(Erc20TokenAbiStr)

	accessor.Erc20Tokens = make(map[types.Address]*Erc20Token)

	//todo:cache tokens
	//erc20Token := &Erc20Token{}
	//erc20Abi.NewContract(erc20Token, types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e"), Accessor.Client)
	//Tokens.Erc20Tokens[types.HexToAddress("0x937ff659c8a9d85aac39dfa84c4b49bb7c9b226e")] = erc20Token

	return accessor
}

func (ethAccessor *EthNodeAccessor) SignAndSendTransaction(result interface{}, from types.Address, tx *ethTypes.Transaction) error {
	if account, ok := ethAccessor.Senders[from]; !ok {
		return errors.New("there isn't a private key for this address:" + from.Hex())
	} else {
		signer := &ethTypes.HomesteadSigner{}

		if signature, err := crypto.Sign(signer.Hash(tx).Bytes(), account.PrivKey); nil != err {
			return err
		} else {
			if tx, err = tx.WithSignature(signer, signature); nil != err {
				return err
			} else {
				if txData, err := rlp.EncodeToBytes(tx); nil != err {
					return err
				} else {
					log.Debugf("txhash:%s, sig:%s, value:%s, gas:%s, gasPrice:%s", tx.Hash().Hex(), common.ToHex(signature), tx.Value().String(), tx.Gas().String(), tx.GasPrice().String())
					err = ethAccessor.Call(result, "eth_sendRawTransaction", types.ToHex(txData))
					return err
				}
			}
		}
	}
}
