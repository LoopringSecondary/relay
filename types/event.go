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
	Token           common.Address
	ContractAddress common.Address
	Symbol          string
	Blocknumber     *big.Int
	Time            *big.Int
}

type TokenUnRegisterEvent struct {
	Token           common.Address
	ContractAddress common.Address
	Symbol          string
	Blocknumber     *big.Int
	Time            *big.Int
}

type RinghashSubmittedEvent struct {
	RingHash        common.Hash
	RingMiner       common.Address
	ContractAddress common.Address
	TxHash          common.Hash
	Blocknumber     *big.Int
	Time            *big.Int
}

type AddressAuthorizedEvent struct {
	Protocol        common.Address
	ContractAddress common.Address
	Number          int
	Blocknumber     *big.Int
	Time            *big.Int
}

type AddressDeAuthorizedEvent struct {
	Protocol        common.Address
	ContractAddress common.Address
	Number          int
	Blocknumber     *big.Int
	Time            *big.Int
}

// todo: unpack transaction and create event
type EtherBalanceUpdateEvent struct {
	Owner common.Address
}

// todo: transfer change to
type TokenBalanceUpdateEvent struct {
	Owner       common.Address
	Value       *big.Int
	BlockNumber *big.Int
	BlockHash   common.Hash
}

// todo: erc20 event
type TokenAllowanceUpdateEvent struct {
	Owner       common.Address
	Spender     common.Address
	Value       *big.Int
	BlockNumber *big.Int
	BlockHash   common.Hash
}

type TransferEvent struct {
	From            common.Address
	To              common.Address
	ContractAddress common.Address
	Value           *big.Int
	Blocknumber     *big.Int
	Time            *big.Int
}

type ApprovalEvent struct {
	Owner           common.Address
	Spender         common.Address
	ContractAddress common.Address
	Value           *big.Int
	Blocknumber     *big.Int
	Time            *big.Int
}

type OrderFilledEvent struct {
	Ringhash        common.Hash
	PreOrderHash    common.Hash
	OrderHash       common.Hash
	NextOrderHash   common.Hash
	TxHash          common.Hash
	ContractAddress common.Address
	Owner           common.Address
	TokenS          common.Address
	TokenB          common.Address
	RingIndex       *big.Int
	Time            *big.Int
	Blocknumber     *big.Int
	AmountS         *big.Int
	AmountB         *big.Int
	LrcReward       *big.Int
	LrcFee          *big.Int
	SplitS          *big.Int
	SplitB          *big.Int
	Market          string
	FillIndex       *big.Int
}

type OrderCancelledEvent struct {
	OrderHash       common.Hash
	TxHash          common.Hash
	ContractAddress common.Address
	Time            *big.Int
	Blocknumber     *big.Int
	AmountCancelled *big.Int
}

type CutoffEvent struct {
	Owner           common.Address
	ContractAddress common.Address
	TxHash          common.Hash
	Time            *big.Int
	Blocknumber     *big.Int
	Cutoff          *big.Int
}

type RingMinedEvent struct {
	RingIndex          *big.Int
	Time               *big.Int
	Blocknumber        *big.Int
	TotalLrcFee        *big.Int
	TradeAmount        int
	Ringhash           common.Hash
	TxHash             common.Hash
	Miner              common.Address
	FeeRecipient       common.Address
	ContractAddress    common.Address
	IsRinghashReserved bool
}

type WethDepositMethodEvent struct {
	From            common.Address
	To              common.Address
	ContractAddress common.Address
	TxHash          common.Hash
	Value           *big.Int
	Time            *big.Int
	Blocknumber     *big.Int
}

type WethWithdrawalMethodEvent struct {
	From            common.Address
	To              common.Address
	ContractAddress common.Address
	TxHash          common.Hash
	Value           *big.Int
	Time            *big.Int
	Blocknumber     *big.Int
}

type SubmitRingMethodEvent struct {
	TxHash       common.Hash
	UsedGas      *big.Int
	UsedGasPrice *big.Int
	Err          error
}

type RingHashSubmitMethodEvent struct {
	RingMiner    common.Address
	RingHash     common.Hash
	TxHash       common.Hash
	UsedGas      *big.Int
	UsedGasPrice *big.Int
	Err          error
}

type BatchSubmitRingHashMethodEvent struct {
	RingHashMinerMap map[common.Hash]common.Address
	TxHash           common.Hash
	UsedGas          *big.Int
	UsedGasPrice     *big.Int
	Err              error
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
