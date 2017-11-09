package gateway

import (
	"errors"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/eventemiter"
	"github.com/Loopring/ringminer/types"
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

	tokenSFilter := &TokenSFilter{}
	for _, v := range options.TokenSFilter.Allow {
		address := types.HexToAddress(v)
		tokenSFilter.AllowTokens[address] = true
	}
	for _, v := range options.TokenSFilter.Denied {
		address := types.HexToAddress(v)
		tokenSFilter.DeniedTokens[address] = true
	}

	tokenBFilter := &TokenBFilter{}
	for _, v := range options.TokenBFilter.Allow {
		address := types.HexToAddress(v)
		tokenBFilter.AllowTokens[address] = true
	}
	for _, v := range options.TokenBFilter.Denied {
		address := types.HexToAddress(v)
		tokenBFilter.DeniedTokens[address] = true
	}

	//cutoffFilter := &CutoffFilter{Cache: ob.cutoffcache}

	filters = append(filters, baseFilter)
	filters = append(filters, tokenSFilter)
	filters = append(filters, tokenBFilter)
	//filters = append(filters, cutoffFilter)
}

func HandleOrder(input eventemitter.EventData) error {
	for _, v := range filters {
		valid, err := v.filter(input.(*types.Order))
		if !valid {
			return err
		}
	}

	eventemitter.Emit(eventemitter.OrderBookPeer, input)

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
		return false, errors.New("order " + o.Hash.Hex() + "lrcFee too tiny")
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
