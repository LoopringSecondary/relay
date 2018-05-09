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

type TxStatus uint8

const (
	TX_STATUS_UNKNOWN TxStatus = 0
	TX_STATUS_PENDING TxStatus = 1
	TX_STATUS_SUCCESS TxStatus = 2
	TX_STATUS_FAILED  TxStatus = 3
)

func StatusStr(status TxStatus) string {
	var ret string
	switch status {
	case TX_STATUS_PENDING:
		ret = "pending"
	case TX_STATUS_SUCCESS:
		ret = "success"
	case TX_STATUS_FAILED:
		ret = "failed"
	default:
		ret = "unknown"
	}

	return ret
}

func StrToTxStatus(txType string) TxStatus {
	var ret TxStatus
	switch txType {
	case "pending":
		ret = TX_STATUS_PENDING
	case "success":
		ret = TX_STATUS_SUCCESS
	case "failed":
		ret = TX_STATUS_FAILED
	default:
		ret = TX_STATUS_UNKNOWN
	}

	return ret
}

type TxInfo struct {
	Protocol        common.Address `json:"from"`
	DelegateAddress common.Address `json:"to"`
	From            common.Address `json:"from"`
	To              common.Address `json:"to"`
	BlockHash       common.Hash    `json:"block_hash"`
	BlockNumber     *big.Int       `json:"block_number"`
	BlockTime       int64          `json:"block_time"`
	TxHash          common.Hash    `json:"tx_hash"`
	TxIndex         int64          `json:"tx_index"`
	TxLogIndex      int64          `json:"tx_log_index"`
	Value           *big.Int       `json:"value"`
	Status          TxStatus       `json:"status"`
	GasLimit        *big.Int       `json:"gas_limit"`
	GasUsed         *big.Int       `json:"gas_used"`
	GasPrice        *big.Int       `json:"gas_price"`
	Nonce           *big.Int       `json:"nonce"`
	Identify        string         `json:"identify"`
}

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
	Amount   *big.Int
}

type ApprovalEvent struct {
	TxInfo
	Owner   common.Address
	Spender common.Address
	Amount  *big.Int
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
	Err          error
}

type WethDepositEvent struct {
	TxInfo
	Dst    common.Address
	Amount *big.Int
}

type WethWithdrawalEvent struct {
	TxInfo
	Src    common.Address
	Amount *big.Int
}

type SubmitRingMethodEvent struct {
	TxInfo
	OrderList    []Order
	FeeReceipt   common.Address
	FeeSelection uint16
	Err          error
}

type RingSubmitResultEvent struct {
	RingHash     common.Hash
	RingUniqueId common.Hash
	TxHash       common.Hash
	Status       TxStatus
	RingIndex    *big.Int
	BlockNumber  *big.Int
	UsedGas      *big.Int
	Err          error
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
	BlockTime   int64
}

type ExtractorWarningEvent struct{}

type TransactionEvent struct {
	Tx TxInfo
}

type DepthUpdateEvent struct {
	DelegateAddress string
	Market          string
}

type BalanceUpdateEvent struct {
	DelegateAddress string
	Owner           string
}
