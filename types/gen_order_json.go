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
	"encoding/json"
	"errors"
	"reflect"
)

func (ord Order) MarshalJson() ([]byte,error) {
	type order struct {
		Protocol              string	`json:"protocol"`
		TokenS                string	`json:"tokenS"`
		TokenB                string	`json:"tokenB"`
		AmountS               uint64	`json:"amountS"`
		AmountB               uint64	`json:"amountB"`
		Rand                  uint64	`json:"rand"`
		Expiration            uint64	`json:"expiration"`
		LrcFee                uint64	`json:"lrcFee"`
		SavingSharePercentage int		`json:"savingShareRate"`
		buyNoMoreThanAmountB  bool		`json:"buyNoMoreThanAmountB"`
		V                     uint8		`json:"v"`
		R                     string	`json:"r"`
		S                     string	`json:"s"`
	}

	var enc order

	enc.Protocol = ord.Protocol.Str()
	enc.TokenS = ord.TokenS.Str()
	enc.TokenB = ord.TokenB.Str()

	enc.AmountS = ord.AmountS.Uint64()
	enc.AmountB = ord.AmountB.Uint64()

	enc.Rand = ord.Rand.Uint64()
	enc.Expiration = ord.Expiration
	enc.LrcFee = ord.LrcFee.Uint64()
	enc.SavingSharePercentage = ord.SavingSharePercentage
	enc.buyNoMoreThanAmountB = ord.BuyNoMoreThanAmountB

	enc.V = ord.V
	enc.R = ord.R.Str()
	enc.S = ord.S.Str()

	return json.Marshal(enc)
}

func (ord *Order) UnMarshalJson(input []byte) error {
	type order struct {
		Protocol              string	`json:"protocol"`
		TokenS                string	`json:"tokenS"`
		TokenB                string	`json:"tokenB"`
		AmountS               uint64	`json:"amountS"`
		AmountB               uint64	`json:"amountB"`
		Rand                  uint64	`json:"rand"`
		Expiration            uint64	`json:"expiration"`
		LrcFee                uint64	`json:"lrcFee"`
		SavingSharePercentage int		`json:"savingShareRate"`
		buyNoMoreThanAmountB  bool		`json:"buyNoMoreThanAmountB"`
		V                     uint8		`json:"v"`
		R                     string	`json:"r"`
		S                     string	`json:"s"`
	}

	var dec order
	err := json.Unmarshal(input, &dec)
	if err != nil {
		return err
	}

	if !reflect.ValueOf(dec.Protocol).IsValid() {
		return errors.New("missing required field 'Protocol' for order")
	}
	ord.Protocol = StringToAddress(dec.Protocol)

	if !reflect.ValueOf(dec.TokenS).IsValid() {
		return errors.New("missing required field 'tokenS' for order")
	}
	ord.TokenS = StringToAddress(dec.TokenS)

	if !reflect.ValueOf(dec.TokenB).IsValid() {
		return errors.New("missing required field 'tokenB' for order")
	}
	ord.TokenB = StringToAddress(dec.TokenB)

	if !reflect.ValueOf(dec.AmountS).IsValid() {
		return errors.New("missing required field 'amountS' for order")
	}
	ord.AmountS = IntToBig(int64(dec.AmountS))

	if !reflect.ValueOf(dec.AmountB).IsValid() {
		return errors.New("missing required field 'amountB' for order")
	}
	ord.AmountB = IntToBig(int64(dec.AmountB))

	if !reflect.ValueOf(dec.Rand).IsValid() {
		return errors.New("missing required field 'rand' for order")
	}
	ord.Rand = IntToBig(int64(dec.Rand))

	if !reflect.ValueOf(dec.Expiration).IsValid() {
		return errors.New("missing required field 'expiration' for order")
	}
	ord.Expiration = dec.Expiration

	if !reflect.ValueOf(dec.LrcFee).IsValid() {
		return errors.New("missing required field 'lrcFee' for order")
	}
	ord.LrcFee = IntToBig(int64(dec.LrcFee))

	if !reflect.ValueOf(dec.SavingSharePercentage).IsValid() {
		return errors.New("missing required field 'savingSharePercentage' for order")
	}
	ord.SavingSharePercentage = dec.SavingSharePercentage

	if !reflect.ValueOf(dec.buyNoMoreThanAmountB).IsValid() {
		return errors.New("missing required field 'fullyFilled' for order")
	}
	ord.BuyNoMoreThanAmountB = dec.buyNoMoreThanAmountB

	if !reflect.ValueOf(dec.V).IsValid() {
		return errors.New("missing required field 'ECDSA.V' for order")
	}
	ord.V = dec.V

	if !reflect.ValueOf(dec.S).IsValid() {
		return errors.New("missing required field 'ECDSA.S' for order")
	}
	ord.S = StringToSign(dec.S)

	if  !reflect.ValueOf(dec.R).IsValid() {
		return errors.New("missing required field 'ECSA.R' for order")
	}
	ord.R = StringToSign(dec.R)

	return nil
}

func (ord OrderState) MarshalJson() ([]byte,error) {
	type state struct {
		Protocol              string	`json:"protocol"`
		TokenS                string	`json:"tokenS"`
		TokenB                string	`json:"tokenB"`
		AmountS               uint64	`json:"amountS"`
		AmountB               uint64	`json:"amountB"`
		Rand                  uint64	`json:"rand"`
		Expiration            uint64	`json:"expiration"`
		LrcFee                uint64	`json:"lrcFee"`
		SavingSharePercentage int		`json:"savingShareRate"`
		buyNoMoreThanAmountB  bool		`json:"buyNoMoreThanAmountB"`
		V                     uint8		`json:"v"`
		R                     string	`json:"r"`
		S                     string	`json:"s"`
		Owner 				  string	`json:"owner"`
		OrderHash 			  string    `json:"hash"`
		RemainedAmountS 	  uint64  	`json:"remainedAmountS"`
		RemainedAmountB 	  uint64	`json:"remainedAmountB"`
		Status 				  uint8		`json:"status"`
	}

	var enc state

	enc.Protocol = ord.RawOrder.Protocol.Str()
	enc.TokenS = ord.RawOrder.TokenS.Str()
	enc.TokenB = ord.RawOrder.TokenB.Str()

	enc.AmountS = ord.RawOrder.AmountS.Uint64()
	enc.AmountB = ord.RawOrder.AmountB.Uint64()

	enc.Rand = ord.RawOrder.Rand.Uint64()
	enc.Expiration = ord.RawOrder.Expiration
	enc.LrcFee = ord.RawOrder.LrcFee.Uint64()
	enc.SavingSharePercentage = ord.RawOrder.SavingSharePercentage
	enc.buyNoMoreThanAmountB = ord.RawOrder.BuyNoMoreThanAmountB

	enc.V = ord.RawOrder.V
	enc.R = ord.RawOrder.R.Str()
	enc.S = ord.RawOrder.S.Str()

	enc.Owner = ord.Owner.Str()
	enc.OrderHash = ord.OrderHash.Str()
	enc.RemainedAmountS = ord.RemainedAmountS.Uint64()
	enc.RemainedAmountB = ord.RemainedAmountB.Uint64()
	enc.Status = uint8(ord.Status)

	return json.Marshal(enc)
}

func (ord *OrderState) UnMarshalJson(input []byte) error {
	type state struct {
		Protocol              string	`json:"protocol"`
		TokenS                string	`json:"tokenS"`
		TokenB                string	`json:"tokenB"`
		AmountS               uint64	`json:"amountS"`
		AmountB               uint64	`json:"amountB"`
		Rand                  uint64	`json:"rand"`
		Expiration            uint64	`json:"expiration"`
		LrcFee                uint64	`json:"lrcFee"`
		SavingSharePercentage int		`json:"savingShareRate"`
		buyNoMoreThanAmountB  bool		`json:"buyNoMoreThanAmountB"`
		V                     uint8		`json:"v"`
		R                     string	`json:"r"`
		S                     string	`json:"s"`
		Owner 				  string	`json:"owner"`
		OrderHash 			  string    `json:"hash"`
		RemainedAmountS 	  uint64  	`json:"remainedAmountS"`
		RemainedAmountB 	  uint64	`json:"remainedAmountB"`
		Status 				  uint8		`json:"status"`
	}

	var dec state
	err := json.Unmarshal(input, &dec)
	if err != nil {
		return err
	}

	if !reflect.ValueOf(dec.Protocol).IsValid() {
		return errors.New("missing required field 'Protocol' for orderState")
	}
	ord.RawOrder.Protocol = StringToAddress(dec.Protocol)

	if !reflect.ValueOf(dec.TokenS).IsValid() {
		return errors.New("missing required field 'tokenS' for orderState")
	}
	ord.RawOrder.TokenS = StringToAddress(dec.TokenS)

	if !reflect.ValueOf(dec.TokenB).IsValid() {
		return errors.New("missing required field 'tokenB' for orderState")
	}
	ord.RawOrder.TokenB = StringToAddress(dec.TokenB)

	if !reflect.ValueOf(dec.AmountS).IsValid() {
		return errors.New("missing required field 'amountS' for orderState")
	}
	ord.RawOrder.AmountS = IntToBig(int64(dec.AmountS))

	if !reflect.ValueOf(dec.AmountB).IsValid() {
		return errors.New("missing required field 'amountB' for orderState")
	}
	ord.RawOrder.AmountB = IntToBig(int64(dec.AmountB))

	if !reflect.ValueOf(dec.Rand).IsValid() {
		return errors.New("missing required field 'rand' for orderState")
	}
	ord.RawOrder.Rand = IntToBig(int64(dec.Rand))

	if !reflect.ValueOf(dec.Expiration).IsValid() {
		return errors.New("missing required field 'expiration' for orderState")
	}
	ord.RawOrder.Expiration = dec.Expiration

	if !reflect.ValueOf(dec.LrcFee).IsValid() {
		return errors.New("missing required field 'lrcFee' for orderState")
	}
	ord.RawOrder.LrcFee = IntToBig(int64(dec.LrcFee))

	if !reflect.ValueOf(dec.SavingSharePercentage).IsValid() {
		return errors.New("missing required field 'savingSharePercentage' for orderState")
	}
	ord.RawOrder.SavingSharePercentage = dec.SavingSharePercentage

	if !reflect.ValueOf(dec.buyNoMoreThanAmountB).IsValid() {
		return errors.New("missing required field 'fullyFilled' for orderState")
	}
	ord.RawOrder.BuyNoMoreThanAmountB = dec.buyNoMoreThanAmountB

	if !reflect.ValueOf(dec.V).IsValid() {
		return errors.New("missing required field 'ECDSA.V' for orderState")
	}
	ord.RawOrder.V = dec.V

	if !reflect.ValueOf(dec.S).IsValid() {
		return errors.New("missing required field 'ECDSA.S' for orderState")
	}
	ord.RawOrder.S = StringToSign(dec.S)

	if  !reflect.ValueOf(dec.R).IsValid() {
		return errors.New("missing required field 'ECSA.R' for orderState")
	}
	ord.RawOrder.R = StringToSign(dec.R)

	if !reflect.ValueOf(dec.Owner).IsValid() {
		return errors.New("missing required field 'owner' for orderState")
	}
	ord.Owner = StringToAddress(dec.Owner)

	if !reflect.ValueOf(dec.OrderHash).IsValid() {
		return errors.New("missing required field 'orderHash' for orderState")
	}
	ord.OrderHash = StringToHash(dec.OrderHash)

	if !reflect.ValueOf(dec.RemainedAmountS).IsValid() {
		return errors.New("missing required field 'remainedAmountS' for orderState")
	}
	ord.RemainedAmountS = IntToBig(int64(dec.RemainedAmountS))

	if !reflect.ValueOf(dec.RemainedAmountB).IsValid() {
		return errors.New("missing required field 'remainedAmountB' for orderState")
	}
	ord.RemainedAmountB = IntToBig(int64(dec.RemainedAmountB))

	if !reflect.ValueOf(dec.Status).IsValid() {
		return errors.New("missing required field 'status' for orderState")
	}
	ord.Status = OrderStatus(dec.Status)

	return nil
}