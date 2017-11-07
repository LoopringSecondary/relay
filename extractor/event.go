package extractor

import (
	"github.com/Loopring/ringminer/types"
)

// todo: should add in contract
type TokenRegisterEvent struct {
}

// todo: should add in contract
type TokenUnRegisterEvent struct {
}

// todo: unpack transaction and create event
type EtherBalanceUpdateEvent struct {
	Owner types.Address
}

// todo: transfer change to
type TokenBalanceUpdateEvent struct {
	Owner       types.Address
	Value       *types.Big
	BlockNumber *types.Big
	BlockHash   types.Hash
}

// todo: erc20 event
type TokenAllowanceUpdateEvent struct {
	Owner       types.Address
	Spender     types.Address
	Value       *types.Big
	BlockNumber *types.Big
	BlockHash   *types.Hash
}

type OrderFilledEvent struct {
	Ringhash      types.Hash
	PreOrderHash  types.Hash
	OrderHash     types.Hash
	NextOrderHash types.Hash
	BlockHash     types.Hash
	RingIndex     *types.Big
	Time          *types.Big
	Blocknumber   *types.Big
	AmountS       *types.Big
	AmountB       *types.Big
	LrcReward     *types.Big
	LrcFee        *types.Big
}

type OrderCancelledEvent struct {
	OrderHash       types.Hash
	BlockHash       types.Hash
	Time            *types.Big
	Blocknumber     *types.Big
	AmountCancelled *types.Big
}

type CutoffEvent struct {
	Address     types.Address
	BlockHash   types.Hash
	Time        *types.Big
	Blocknumber *types.Big
	Cutoff      *types.Big
}

type RingMinedEvent struct {
	RingIndex     *types.Big
	Time          *types.Big
	Blocknumber   *types.Big
	BlockHash     types.Hash
	Ringhash      types.Hash
	Miner         types.Address
	FeeRecepient  types.Address
	RinghashFound bool
}
