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
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type Gateway struct {
	filters          []Filter
	om               ordermanager.OrderManager
	isBroadcast      bool
	maxBroadcastTime int
	ipfsPubService   IPFSPubService
}

var gateway Gateway

type Filter interface {
	filter(o *types.Order) (bool, error)
}

func Initialize(filterOptions *config.GatewayFiltersOptions, options *config.GateWayOptions, ipfsOptions *config.IpfsOptions, om ordermanager.OrderManager) {
	// add gateway watcher
	gatewayWatcher := &eventemitter.Watcher{Concurrent: false, Handle: HandleOrder}
	eventemitter.On(eventemitter.Gateway, gatewayWatcher)

	gateway = Gateway{filters: make([]Filter, 0), om: om, isBroadcast: options.IsBroadcast, maxBroadcastTime: options.MaxBroadcastTime}
	gateway.ipfsPubService = NewIPFSPubService(ipfsOptions)

	// add filters
	baseFilter := &BaseFilter{MinLrcFee: big.NewInt(filterOptions.BaseFilter.MinLrcFee)}

	tokenSFilter := &TokenSFilter{AllowTokens: make(map[common.Address]bool), DeniedTokens: make(map[common.Address]bool)}
	for _, v := range filterOptions.TokenSFilter.Allow {
		address := common.HexToAddress(v)
		tokenSFilter.AllowTokens[address] = true
	}
	for _, v := range filterOptions.TokenSFilter.Denied {
		address := common.HexToAddress(v)
		tokenSFilter.DeniedTokens[address] = true
	}

	tokenBFilter := &TokenBFilter{AllowTokens: make(map[common.Address]bool), DeniedTokens: make(map[common.Address]bool)}
	for _, v := range filterOptions.TokenBFilter.Allow {
		address := common.HexToAddress(v)
		tokenBFilter.AllowTokens[address] = true
	}
	for _, v := range filterOptions.TokenBFilter.Denied {
		address := common.HexToAddress(v)
		tokenBFilter.DeniedTokens[address] = true
	}

	signFilter := &SignFilter{}

	//cutoffFilter := &CutoffFilter{Cache: ob.cutoffcache}

	gateway.filters = append(gateway.filters, baseFilter)
	gateway.filters = append(gateway.filters, signFilter)
	gateway.filters = append(gateway.filters, tokenSFilter)
	gateway.filters = append(gateway.filters, tokenBFilter)
	//filters = append(filters, cutoffFilter)
}

func HandleOrder(input eventemitter.EventData) error {
	ord := input.(*types.Order)

	orderHash := ord.GenerateHash()
	ord.Hash = orderHash

	orderState, err := gateway.om.GetOrderByHash(ord.Hash)

	//TODO(xiaolu) 这里需要测试一下，超时error和查询数据为空的error，处理方式不应该一样
	if err != nil {
		ord.GeneratePrice()

		for _, v := range gateway.filters {
			valid, err := v.filter(ord)
			if !valid {
				log.Errorf("gateway filter order error:%s", err.Error())
				return err
			}
		}

		state := &types.OrderState{}
		state.RawOrder = *ord

		if gateway.isBroadcast && gateway.maxBroadcastTime > 0 {
			state.BroadcastTime = 1
			go func() {
				pubErr := gateway.ipfsPubService.PublishOrder(*ord)
				if pubErr != nil {
					log.Error("publish order failed, orderHash : " + ord.Hash.Str())
				}
			}()
		}

		log.Debugf("gateway accept new order hash:%s", orderHash.Hex())
		log.Debugf("gateway accept new order amountS:%s", ord.AmountS.String())
		log.Debugf("gateway accept new order amountB:%s", ord.AmountB.String())

		eventemitter.Emit(eventemitter.OrderManagerGatewayNewOrder, state)
	} else if gateway.isBroadcast && orderState.BroadcastTime < gateway.maxBroadcastTime {
		//broadcast
		go func() {
			pubErr := gateway.ipfsPubService.PublishOrder(*ord)
			if pubErr != nil {
				log.Error("publish order failed, orderHash : " + ord.Hash.Str())
			} else {
				gateway.om.UpdateBroadcastTimeByHash(orderState.RawOrder.Hash, orderState.BroadcastTime+1)
			}
		}()
	}

	return nil
}

type BaseFilter struct {
	MinLrcFee *big.Int
}

func (f *BaseFilter) filter(o *types.Order) (bool, error) {
	const (
		addrLength = 20
		hashLength = 32
	)

	if len(o.Hash) != hashLength {
		return false, fmt.Errorf("gateway base filter,order %s length error", o.Hash.Hex())
	}
	if len(o.TokenB) != addrLength {
		return false, fmt.Errorf("gateway base filter,order %s tokenB %s address length error", o.Hash.Hex(), o.TokenB.Hex())
	}
	if len(o.TokenS) != addrLength {
		return false, fmt.Errorf("gateway base filter,order %s tokenS %s address length error", o.Hash.Hex(), o.TokenS.Hex())
	}
	if o.TokenB == o.TokenS {
		return false, fmt.Errorf("gateway base filter,order %s tokenB == tokenS", o.Hash.Hex())
	}
	if f.MinLrcFee.Cmp(o.LrcFee) >= 0 {
		return false, fmt.Errorf("gateway base filter,order %s lrc fee %s invalid", o.Hash.Hex(), o.LrcFee.String())
	}
	if len(o.Owner) != addrLength {
		return false, fmt.Errorf("gateway base filter,order %s owner %s address length error", o.Hash.Hex(), o.Owner.Hex())
	}
	if len(o.Protocol) != addrLength {
		return false, fmt.Errorf("gateway base filter,order %s protocol %s address length error", o.Hash.Hex(), o.Owner.Hex())
	}
	return true, nil
}

type SignFilter struct {
}

func (f *SignFilter) filter(o *types.Order) (bool, error) {
	o.Hash = o.GenerateHash()

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
