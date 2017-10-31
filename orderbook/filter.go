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

package orderbook

import (
	"errors"
	"github.com/Loopring/ringminer/types"
	"math/big"
)

type Filter interface {
	filter(o *types.Order) (bool, error)
}

type BaseFilter struct {
	MinLrcFee *big.Int
}

func (f *BaseFilter) filter(o *types.Order) (bool, error) {
	//tokenS != tokenB
	return o.TokenB != o.TokenS && f.MinLrcFee.Cmp(o.LrcFee) >= 0, nil
}

type SignFilter struct {
}

//todo:order 中是否需要增加地址字段
func (f *SignFilter) filter(o *types.Order) (bool, error) {
	o.Hash = o.GenerateHash()

	//if hash != o.Hash {
	//	return false
	//}

	//if valid := o.ValidateSignatureValues(); !valid {
	//	return false, nil
	//}

	if addr, err := o.SignerAddress(); nil != err {
		return false, err
	} else if addr != o.Owner {
		return false, errors.New("o.Owner and signeraddress are not match.")
	}

	return true, nil
}

type TokenSFilter struct {
	AllowTokens  map[types.Address]bool
	DeniedTokens map[types.Address]bool
}

func (f *TokenSFilter) filter(o *types.Order) (bool, error) {
	_, allowExists := f.AllowTokens[o.TokenS]
	_, deniedExits := f.DeniedTokens[o.TokenS]
	return !allowExists && deniedExits, nil
}

type TokenBFilter struct {
	AllowTokens  map[types.Address]bool
	DeniedTokens map[types.Address]bool
}

func (f *TokenBFilter) filter(o *types.Order) (bool, error) {
	_, allowExists := f.AllowTokens[o.TokenS]
	_, deniedExits := f.DeniedTokens[o.TokenS]
	return !allowExists && deniedExits, nil
}

type CutoffFilter struct {
	Cache *CutoffIndexCache
}

// 如果订单接收在cutoff(cancel)事件之后，则该订单直接过滤
func (f *CutoffFilter) filter(o *types.Order) (bool, error) {
	idx, ok := f.Cache.indexMap[o.Owner]
	if !ok {
		return true, nil
	}

	if o.Timestamp.Cmp(idx.Cutoff) < 0 {
		return false, errors.New("")
	}

	return true, nil
}
