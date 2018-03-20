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
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
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

	// new base filter
	baseFilter := &BaseFilter{MinLrcFee: big.NewInt(filterOptions.BaseFilter.MinLrcFee), MaxPrice: big.NewInt(filterOptions.BaseFilter.MaxPrice)}

	// new token filter
	tokenFilter := &TokenFilter{}

	// new sign filter
	signFilter := &SignFilter{}

	// new cutoff filter
	cutoffFilter := &CutoffFilter{om: om}

	gateway.filters = append(gateway.filters, baseFilter)
	gateway.filters = append(gateway.filters, signFilter)
	gateway.filters = append(gateway.filters, tokenFilter)
	gateway.filters = append(gateway.filters, cutoffFilter)
}

func HandleOrder(input eventemitter.EventData) error {
	var (
		state *types.OrderState
		err   error
	)

	order := input.(*types.Order)
	order.Hash = order.GenerateHash()

	var broadcastTime int

	//TODO(xiaolu) 这里需要测试一下，超时error和查询数据为空的error，处理方式不应该一样
	if state, err = gateway.om.GetOrderByHash(order.Hash); err != nil && err.Error() == "record not found" {
		if err = generatePrice(order); err != nil {
			return err
		}

		for _, v := range gateway.filters {
			valid, err := v.filter(order)
			if !valid {
				log.Errorf(err.Error())
				return err
			}
		}
		state = &types.OrderState{}
		state.RawOrder = *order
		broadcastTime = 0
		eventemitter.Emit(eventemitter.OrderManagerGatewayNewOrder, state)
	} else {
		broadcastTime = state.BroadcastTime
		log.Infof("gateway,order %s exist,will not insert again", order.Hash.Hex())
	}

	if gateway.isBroadcast && broadcastTime < gateway.maxBroadcastTime {
		//broadcast
		log.Infof(">>>>>>> broad to ipfs order : " + state.RawOrder.Hash.Hex())
		pubErr := gateway.ipfsPubService.PublishOrder(state.RawOrder)
		if pubErr != nil {
			log.Errorf("gateway,publish order %s failed", state.RawOrder.Hash.String())
		} else {
			if err = gateway.om.UpdateBroadcastTimeByHash(state.RawOrder.Hash, state.BroadcastTime+1); nil != err {
				return err
			}
		}
	}
	return nil
}

func generatePrice(order *types.Order) error {
	tokenS, err := util.AddressToToken(order.TokenS)
	if err != nil {
		return err
	}
	if tokenS.Decimals == nil || tokenS.Decimals.Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("order's tokenS decimals invalid")
	}

	tokenB, err := util.AddressToToken(order.TokenB)
	if err != nil {
		return err
	}
	if tokenB.Decimals == nil || tokenB.Decimals.Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("order's tokenS decimals invalid")
	}

	if order.AmountS == nil || order.AmountS.Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("order's amountS invalid")
	}

	if order.AmountB == nil || order.AmountB.Cmp(big.NewInt(0)) < 1 {
		return fmt.Errorf("order's amountB invalid")
	}

	order.Price = new(big.Rat).Mul(
		new(big.Rat).SetFrac(order.AmountS, order.AmountB),
		new(big.Rat).SetFrac(tokenB.Decimals, tokenS.Decimals),
	)

	return nil
}

type BaseFilter struct {
	MinLrcFee *big.Int
	MaxPrice  *big.Int
}

func (f *BaseFilter) filter(o *types.Order) (bool, error) {
	const (
		addrLength = 20
		hashLength = 32
	)

	if len(o.Hash) != hashLength {
		return false, fmt.Errorf("gateway,base filter,order %s length error", o.Hash.Hex())
	}
	if len(o.TokenB) != addrLength {
		return false, fmt.Errorf("gateway,base filter,order %s tokenB %s address length error", o.Hash.Hex(), o.TokenB.Hex())
	}
	if len(o.TokenS) != addrLength {
		return false, fmt.Errorf("gateway,base filter,order %s tokenS %s address length error", o.Hash.Hex(), o.TokenS.Hex())
	}
	if o.TokenB == o.TokenS {
		return false, fmt.Errorf("gateway,base filter,order %s tokenB == tokenS", o.Hash.Hex())
	}
	if len(o.Owner) != addrLength {
		return false, fmt.Errorf("gateway,base filter,order %s owner %s address length error", o.Hash.Hex(), o.Owner.Hex())
	}
	if len(o.Protocol) != addrLength {
		return false, fmt.Errorf("gateway,base filter,order %s protocol %s address length error", o.Hash.Hex(), o.Owner.Hex())
	}
	if o.Price.Cmp(new(big.Rat).SetFrac(f.MaxPrice, big.NewInt(1))) > 0 || o.Price.Cmp(new(big.Rat).SetFrac(big.NewInt(1), f.MaxPrice)) < 0 {
		return false, fmt.Errorf("dao order convert down,price out of range")
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
		return false, fmt.Errorf("gateway,sign filter,o.Owner %s and signeraddress %s are not match", o.Owner.Hex(), addr.Hex())
	}

	return true, nil
}

type TokenFilter struct {
	AllowTokens  map[common.Address]bool
	DeniedTokens map[common.Address]bool
}

func (f *TokenFilter) filter(o *types.Order) (bool, error) {
	supportTokenS := false
	supportTokenB := false
	for _, v := range util.AllTokens {
		if v.Protocol == o.TokenS && !v.Deny {
			supportTokenS = true
		}
		if v.Protocol == o.TokenB && !v.Deny {
			supportTokenB = true
		}
	}

	if !supportTokenS {
		return false, fmt.Errorf("gateway,token filter,tokenS:%s do not supported", o.TokenS.Hex())
	}
	if !supportTokenB {
		return false, fmt.Errorf("gateway,token filter,tokenB:%s do not supported", o.TokenB.Hex())
	}

	return true, nil
}

type CutoffFilter struct {
	om ordermanager.OrderManager
}

// 如果订单接收在cutoff(cancel)事件之后，则该订单直接过滤
func (f *CutoffFilter) filter(o *types.Order) (bool, error) {
	if f.om.IsOrderCutoff(o.Protocol, o.Owner, o.TokenS, o.TokenB, o.ValidSince) {
		return false, fmt.Errorf("gateway,cutoff filter order:%s should be cutoff", o.Owner.Hex())
	}

	return true, nil
}
