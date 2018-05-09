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

type TxType uint8

const (
	SYMBOL_ETH  = "ETH"
	SYMBOL_WETH = "WETH"
	SYMBOL_LRC  = "LRC"
)

// send/receive/sell/buy/wrap/unwrap/cancelOrder/approve
const (
	// common type
	TX_TYPE_UNKNOWN              TxType = 0
	TX_TYPE_APPROVE              TxType = 1
	TX_TYPE_SEND                 TxType = 2
	TX_TYPE_RECEIVE              TxType = 3
	TX_TYPE_SELL                 TxType = 4
	TX_TYPE_BUY                  TxType = 5
	TX_TYPE_CONVERT_INCOME       TxType = 7
	TX_TYPE_CONVERT_OUTCOME      TxType = 8
	TX_TYPE_CANCEL_ORDER         TxType = 9
	TX_TYPE_CUTOFF               TxType = 10
	TX_TYPE_CUTOFF_PAIR          TxType = 11
	TX_TYPE_UNSUPPORTED_CONTRACT TxType = 12
	TX_TYPE_LRC_FEE              TxType = 13
	TX_TYPE_LRC_REWARD           TxType = 14
)

func TypeStr(typ TxType) string {
	var ret string

	switch typ {
	case TX_TYPE_APPROVE:
		ret = "approve"
	case TX_TYPE_SEND:
		ret = "send"
	case TX_TYPE_RECEIVE:
		ret = "receive"
	case TX_TYPE_SELL:
		ret = "sell"
	case TX_TYPE_BUY:
		ret = "buy"
	case TX_TYPE_CONVERT_INCOME:
		ret = "convert_income"
	case TX_TYPE_CONVERT_OUTCOME:
		ret = "convert_outcome"
	case TX_TYPE_CANCEL_ORDER:
		ret = "cancel_order"
	case TX_TYPE_CUTOFF:
		ret = "cutoff"
	case TX_TYPE_CUTOFF_PAIR:
		ret = "cutoff_trading_pair"
	case TX_TYPE_UNSUPPORTED_CONTRACT:
		ret = "unsupported_contract"
	case TX_TYPE_LRC_FEE:
		ret = "lrc_fee"
	case TX_TYPE_LRC_REWARD:
		ret = "lrc_reward"
	default:
		ret = "unknown"
	}

	return ret
}

func StrToTxType(typ string) TxType {
	var ret TxType
	switch typ {
	case "approve":
		ret = TX_TYPE_APPROVE
	case "send":
		ret = TX_TYPE_SEND
	case "receive":
		ret = TX_TYPE_RECEIVE
	case "sell":
		ret = TX_TYPE_SELL
	case "buy":
		ret = TX_TYPE_BUY
	case "convert_income":
		ret = TX_TYPE_CONVERT_INCOME
	case "convert_outcome":
		ret = TX_TYPE_CONVERT_OUTCOME
	case "cancel_order":
		ret = TX_TYPE_CANCEL_ORDER
	case "cutoff":
		ret = TX_TYPE_CUTOFF
	case "cutoff_trading_pair":
		ret = TX_TYPE_CUTOFF_PAIR
	case "unsupported_contract":
		ret = TX_TYPE_UNSUPPORTED_CONTRACT
	case "lrc_fee":
		ret = TX_TYPE_LRC_FEE
	case "lrc_reward":
		ret = TX_TYPE_LRC_REWARD
	default:
		ret = TX_TYPE_UNKNOWN
	}

	return ret
}
