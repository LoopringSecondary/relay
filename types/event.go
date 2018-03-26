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

package types

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type TokenRegisterEvent struct {
	TxInfo
	Token  common.Address
	Symbol string
}

type TokenUnRegisterEvent struct {
	TxInfo
	Token  common.Address
	Symbol string
}

type AddressAuthorizedEvent struct {
	TxInfo
	Protocol common.Address
	Number   int
}

type AddressDeAuthorizedEvent struct {
	TxInfo
	Protocol common.Address
	Number   int
}

type TransferEvent struct {
	TxInfo
	Sender   common.Address
	Receiver common.Address
	Value    *big.Int
}

type ApprovalEvent struct {
	TxInfo
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
}

type OrderFilledEvent struct {
	TxInfo
	Ringhash      common.Hash
	PreOrderHash  common.Hash
	OrderHash     common.Hash
	NextOrderHash common.Hash
	Owner         common.Address
	TokenS        common.Address
	TokenB        common.Address
	SellTo        common.Address
	BuyFrom       common.Address
	RingIndex     *big.Int
	AmountS       *big.Int
	AmountB       *big.Int
	LrcReward     *big.Int
	LrcFee        *big.Int
	SplitS        *big.Int
	SplitB        *big.Int
	Market        string
	FillIndex     *big.Int
}

type OrderCancelledEvent struct {
	TxInfo
	OrderHash       common.Hash
	AmountCancelled *big.Int
}

type CutoffEvent struct {
	TxInfo
	Owner         common.Address
	Cutoff        *big.Int
	OrderHashList []common.Hash
}

type CutoffPairEvent struct {
	TxInfo
	Owner         common.Address
	Token1        common.Address
	Token2        common.Address
	Cutoff        *big.Int
	OrderHashList []common.Hash
}

type RingMinedEvent struct {
	TxInfo
	RingIndex    *big.Int
	TotalLrcFee  *big.Int
	TradeAmount  int
	Ringhash     common.Hash
	Miner        common.Address
	FeeRecipient common.Address
}

type WethDepositEvent struct {
	TxInfo
	Dst   common.Address
	Value *big.Int
}

type WethWithdrawalEvent struct {
	TxInfo
	Src   common.Address
	Value *big.Int
}

type WethDepositMethodEvent struct {
	TxInfo
	Dst   common.Address
	Value *big.Int
}

type WethWithdrawalMethodEvent struct {
	TxInfo
	Src   common.Address
	Value *big.Int
}

type ApproveMethodEvent struct {
	TxInfo
	Spender common.Address
	Value   *big.Int
	Owner   common.Address
}

type TransferMethodEvent struct {
	TxInfo
	Sender   common.Address
	Receiver common.Address
	Value    *big.Int
}

type SubmitRingMethodEvent struct {
	TxInfo
	Err error
}

type CutoffMethodEvent struct {
	TxInfo
	Value *big.Int
	Owner common.Address
}

type CutoffPairMethodEvent struct {
	TxInfo
	Value  *big.Int
	Token1 common.Address
	Token2 common.Address
	Owner  common.Address
}

type RingSubmitFailedEvent struct {
	RingHash common.Hash
	Err      error
}

type ForkedEvent struct {
	DetectedBlock *big.Int
	DetectedHash  common.Hash
	ForkBlock     *big.Int
	ForkHash      common.Hash
}

type BlockEvent struct {
	BlockNumber *big.Int
	BlockHash   common.Hash
}

type TransactionEvent struct {
	Tx Transaction
	test string
}
