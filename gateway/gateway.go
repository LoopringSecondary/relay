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

package gateway

import (
	"errors"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

var filters []Filter

type Filter interface {
	filter(o *types.Order) (bool, error)
}

func Initialize(options *config.GatewayFiltersOptions) {
	// add gateway watcher
	gatewayWatcher := &eventemitter.Watcher{Concurrent: false, Handle: HandleOrder}
	eventemitter.On(eventemitter.Gateway, gatewayWatcher)

	// add filters
	baseFilter := &BaseFilter{MinLrcFee: big.NewInt(options.BaseFilter.MinLrcFee)}

	tokenSFilter := &TokenSFilter{AllowTokens: make(map[common.Address]bool), DeniedTokens: make(map[common.Address]bool)}
	for _, v := range options.TokenSFilter.Allow {
		address := common.HexToAddress(v)
		tokenSFilter.AllowTokens[address] = true
	}
	for _, v := range options.TokenSFilter.Denied {
		address := common.HexToAddress(v)
		tokenSFilter.DeniedTokens[address] = true
	}

	tokenBFilter := &TokenBFilter{AllowTokens: make(map[common.Address]bool), DeniedTokens: make(map[common.Address]bool)}
	for _, v := range options.TokenBFilter.Allow {
		address := common.HexToAddress(v)
		tokenBFilter.AllowTokens[address] = true
	}
	for _, v := range options.TokenBFilter.Denied {
		address := common.HexToAddress(v)
		tokenBFilter.DeniedTokens[address] = true
	}

	signFilter := &SignFilter{}

	//cutoffFilter := &CutoffFilter{Cache: ob.cutoffcache}

	filters = append(filters, baseFilter)
	filters = append(filters, signFilter)
	filters = append(filters, tokenSFilter)
	filters = append(filters, tokenBFilter)
	//filters = append(filters, cutoffFilter)
}

func HandleOrder(input eventemitter.EventData) error {
	ord := input.(*types.Order)

	orderhash := ord.GenerateHash()
	ord.Hash = orderhash
	ord.GeneratePrice()

	for _, v := range filters {
		valid, err := v.filter(ord)
		if !valid {
			log.Errorf("gateway filter order error:%s", err.Error())
			return err
		}
	}

	state := &types.OrderState{}
	state.RawOrder = *ord

	log.Debugf("gateway accept new order hash:%s", orderhash.Hex())
	log.Debugf("gateway accept new order amountS:%s", ord.AmountS.String())
	log.Debugf("gateway accept new order amountB:%s", ord.AmountB.String())

	eventemitter.Emit(eventemitter.OrderManagerGatewayNewOrder, state)

	// todo: broadcast
	return nil
}

type BaseFilter struct {
	MinLrcFee *big.Int
}

func (f *BaseFilter) filter(o *types.Order) (bool, error) {
	if o.TokenB == o.TokenS {
		return false, errors.New("order " + o.Hash.Hex() + " tokenB == tokenS")
	}
	if f.MinLrcFee.Cmp(o.LrcFee) >= 0 {
		return false, errors.New("order " + o.Hash.Hex() + " lrcFee too tiny")
	}
	if len(o.Owner) != 20 {
		return false, errors.New("order " + o.Hash.Hex() + " owner address length error")
	}

	return true, nil
}

type SignFilter struct {
}

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
	AllowTokens  map[common.Address]bool
	DeniedTokens map[common.Address]bool
}

func (f *TokenSFilter) filter(o *types.Order) (bool, error) {
	if _, ok := f.AllowTokens[o.TokenS]; !ok {
		return false, errors.New("tokenS filter allowTokens do not contain " + o.TokenS.Hex())
	}
	if _, ok := f.DeniedTokens[o.TokenS]; ok {
		return false, errors.New("tokenS filter deniedTokens contain " + o.TokenS.Hex())
	}
	return true, nil
}

type TokenBFilter struct {
	AllowTokens  map[common.Address]bool
	DeniedTokens map[common.Address]bool
}

func (f *TokenBFilter) filter(o *types.Order) (bool, error) {
	if _, ok := f.AllowTokens[o.TokenB]; !ok {
		return false, errors.New("tokenB filter allowTokens do not contain " + o.TokenB.Hex())
	}
	if _, ok := f.DeniedTokens[o.TokenB]; ok {
		return false, errors.New("tokenB filter deniedTokens contain " + o.TokenB.Hex())
	}
	return true, nil
}

// todo: cutoff filter

//type CutoffFilter struct {
//	Cache *CutoffIndexCache
//}
//
//// 如果订单接收在cutoff(cancel)事件之后，则该订单直接过滤
//func (f *CutoffFilter) filter(o *types.Order) (bool, error) {
//	idx, ok := f.Cache.indexMap[o.Owner]
//	if !ok {
//		return true, nil
//	}
//
//	if o.Timestamp.Cmp(idx.Cutoff) < 0 {
//		return false, errors.New("")
//	}
//
//	return true, nil
//}
