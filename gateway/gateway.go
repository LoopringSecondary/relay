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
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"qiniupkg.com/x/errors.v7"
	"time"
)

type Gateway struct {
	filters          []Filter
	om               ordermanager.OrderManager
	am               market.AccountManager
	isBroadcast      bool
	maxBroadcastTime int
	ipfsPubService   IPFSPubService
	marketCap        marketcap.MarketCapProvider
}

var gateway Gateway

type Filter interface {
	filter(o *types.Order) (bool, error)
}

func Initialize(filterOptions *config.GatewayFiltersOptions, options *config.GateWayOptions, ipfsOptions *config.IpfsOptions, om ordermanager.OrderManager, marketCap marketcap.MarketCapProvider, am market.AccountManager) {
	// add gateway watcher
	gatewayWatcher := &eventemitter.Watcher{Concurrent: false, Handle: HandleOrder}
	eventemitter.On(eventemitter.GatewayNewOrder, gatewayWatcher)

	gateway = Gateway{filters: make([]Filter, 0), om: om, isBroadcast: options.IsBroadcast, maxBroadcastTime: options.MaxBroadcastTime, am: am}
	//gateway.ipfsPubService = NewIPFSPubService(ipfsOptions)

	gateway.marketCap = marketCap

	// new pow filter
	powFilter := &PowFilter{Difficulty: types.HexToBigint(filterOptions.PowFilter.Difficulty)}

	// new base filter
	baseFilter := &BaseFilter{
		MinLrcFee:             big.NewInt(filterOptions.BaseFilter.MinLrcFee),
		MinLrcHold:            filterOptions.BaseFilter.MinLrcHold,
		MaxPrice:              big.NewInt(filterOptions.BaseFilter.MaxPrice),
		MinSplitPercentage:    filterOptions.BaseFilter.MinSplitPercentage,
		MaxSplitPercentage:    filterOptions.BaseFilter.MaxSplitPercentage,
		MinTokeSAmount:        make(map[string]*big.Int),
		MinTokenSUsdAmount:    filterOptions.BaseFilter.MinTokenSUsdAmount,
		MaxValidSinceInterval: filterOptions.BaseFilter.MaxValidSinceInterval,
	}
	for k, v := range filterOptions.BaseFilter.MinTokeSAmount {
		minAmount := big.NewInt(0)
		amount, succ := minAmount.SetString(v, 10)
		if succ {
			baseFilter.MinTokeSAmount[k] = amount
		}
	}

	// new token filter
	tokenFilter := &TokenFilter{}

	// new sign filter
	signFilter := &SignFilter{}

	// new cutoff filter
	cutoffFilter := &CutoffFilter{om: om}

	gateway.filters = append(gateway.filters, powFilter)
	gateway.filters = append(gateway.filters, baseFilter)
	gateway.filters = append(gateway.filters, signFilter)
	gateway.filters = append(gateway.filters, tokenFilter)
	gateway.filters = append(gateway.filters, cutoffFilter)
}

func HandleInputOrder(input eventemitter.EventData) (orderHash string, err error) {
	var (
		state *types.OrderState
	)

	order := input.(*types.Order)
	order.Hash = order.GenerateHash()
	orderHash = order.Hash.Hex()

	//var broadcastTime int

	//TODO(xiaolu) 这里需要测试一下，超时error和查询数据为空的error，处理方式不应该一样
	if state, err = gateway.om.GetOrderByHash(order.Hash); err != nil && err.Error() == "record not found" {
		if err = generatePrice(order); err != nil {
			return orderHash, err
		}

		for _, v := range gateway.filters {
			valid, err := v.filter(order)
			if !valid {
				log.Errorf(err.Error())
				return orderHash, err
			}
		}
		state = &types.OrderState{}
		state.RawOrder = *order
		//broadcastTime = 0
		eventemitter.Emit(eventemitter.NewOrder, state)
	} else {
		//broadcastTime = state.BroadcastTime
		log.Infof("gateway,order %s exist,will not insert again", order.Hash.Hex())
		return orderHash, errors.New("order existed, please not submit again")
	}

	//if gateway.isBroadcast && broadcastTime < gateway.maxBroadcastTime {
	//	//broadcast
	//	log.Infof(">>>>>>> broad to ipfs order : " + state.RawOrder.Hash.Hex())
	//	pubErr := gateway.ipfsPubService.PublishOrder(state.RawOrder)
	//	if pubErr != nil {
	//		log.Errorf("gateway,publish order %s failed", state.RawOrder.Hash.String())
	//	} else {
	//		if err = gateway.om.UpdateBroadcastTimeByHash(state.RawOrder.Hash, state.BroadcastTime+1); nil != err {
	//			return err
	//		}
	//	}
	//}
	return orderHash, err
}

func HandleOrder(input eventemitter.EventData) error {
	_, err := HandleInputOrder(input)
	return err
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
	MinLrcFee             *big.Int
	MinLrcHold            int64
	MinSplitPercentage    float64
	MaxSplitPercentage    float64
	MaxPrice              *big.Int
	MinTokeSAmount        map[string]*big.Int
	MinTokenSUsdAmount    float64
	MaxValidSinceInterval int64
}

func (f *BaseFilter) filter(o *types.Order) (bool, error) {
	const (
		addrLength = 20
		hashLength = 32
	)

	if o.TokenB != util.AliasToAddress("LRC") {
		balances, err := gateway.am.GetBalanceWithSymbolResult(o.Owner)

		if err != nil {
			return false, fmt.Errorf("gateway,base filter,owner holds lrc less than %d ", f.MinLrcHold)
		}

		if b, ok := balances["LRC"]; ok {
			lrcHold := big.NewInt(f.MinLrcHold)
			lrcHold = lrcHold.Mul(lrcHold, util.AllTokens["LRC"].Decimals)
			if b.Cmp(lrcHold) < 1 {
				return false, fmt.Errorf("gateway,base filter,owner holds lrc less than %d ", f.MinLrcHold)
			}

		} else {
			return false, fmt.Errorf("gateway,base filter,owner holds lrc less than %d ", f.MinLrcHold)
		}

	}

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

	now := time.Now().Unix()

	// validSince check
	if o.ValidSince.Int64()-f.MaxValidSinceInterval > now {
		return false, fmt.Errorf("valid since is too small, order must be valid before %d second timestamp", now-f.MaxValidSinceInterval)
	}

	// validUntil check
	if o.ValidUntil.Int64() < now {
		return false, fmt.Errorf("order expired, please check validUntil")
	}

	// MarginSplitPercentage range check
	if float64(o.MarginSplitPercentage)/100.0 < f.MinSplitPercentage || float64(o.MarginSplitPercentage)/100.0 > f.MaxSplitPercentage {
		return false, fmt.Errorf("margin split percentage out of range")
	}

	// tokenS min amount check
	tokenS, err := util.AddressToToken(o.TokenS)
	if err != nil {
		return false, fmt.Errorf("tokenS is not support now")
	}

	if minAmount, ok := f.MinTokeSAmount[tokenS.Symbol]; ok && o.AmountS.Cmp(minAmount) < 0 {
		return false, fmt.Errorf("tokenS amount is too small")
	}

	// USD min amount check
	tokenSPrice, err := gateway.marketCap.GetMarketCapByCurrency(o.TokenS, "USD")
	if err != nil || tokenSPrice == nil {
		return false, fmt.Errorf("get price error. please retry later")
	}
	tokenSFloatPrice, _ := tokenSPrice.Float64()
	if tokenSFloatPrice <= 0 {
		return false, fmt.Errorf("get zero token s price. symbol : " + tokenS.Symbol)
	}

	amountDivDecimal, _ := new(big.Rat).SetFrac(o.AmountS, tokenS.Decimals).Float64()
	usdAmount := amountDivDecimal * tokenSFloatPrice
	if usdAmount < f.MinTokenSUsdAmount {
		return false, fmt.Errorf("tokenS usd amount is too small, price:%f, amount:%f, value:%f, usdMinValue:%f", tokenSFloatPrice, amountDivDecimal, usdAmount, f.MinTokenSUsdAmount)
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

type PowFilter struct {
	Difficulty *big.Int
}

func (f *PowFilter) filter(o *types.Order) (bool, error) {

	if o.PowNonce <= 0 {
		return false, fmt.Errorf("invalid pow nonce")
	}

	pow := GetPow(o.V, o.R, o.S, o.PowNonce)

	if pow.Cmp(f.Difficulty) < 0 {
		return false, fmt.Errorf("invalid pow")
	}
	return true, nil
}

func GetPow(v uint8, r types.Bytes32, s types.Bytes32, powNonce uint64) *big.Int {

	input := make([]byte, 0)
	input = append(input, v)
	input = append(input, r.Bytes()...)
	input = append(input, s.Bytes()...)
	nonce := Uint64ToByteArray(powNonce)
	input = append(input, nonce...)

	hash := sha256.New()
	fmt.Println(input)
	fmt.Println(common.Bytes2Hex(input))
	hash.Write(input)

	rst := hash.Sum(nil)
	bigRst := big.NewInt(0)
	bigRst.SetBytes(rst)
	return bigRst
}

func Uint64ToByteArray(src uint64) []byte {
	rst := make([]byte, 8)
	binary.LittleEndian.PutUint64(rst, src)
	return rst
}
