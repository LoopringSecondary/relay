package types

// todo: should add in contract
type TokenRegisterEvent struct {
}

// todo: should add in contract
type TokenUnRegisterEvent struct {
}

// todo: unpack transaction and create event
type EtherBalanceUpdateEvent struct {
	Owner Address
}

// todo: transfer change to
type TokenBalanceUpdateEvent struct {
	Owner       Address
	Value       *Big
	BlockNumber *Big
	BlockHash   Hash
}

// todo: erc20 event
type TokenAllowanceUpdateEvent struct {
	Owner       Address
	Spender     Address
	Value       *Big
	BlockNumber *Big
	BlockHash   *Hash
}

type OrderFilledEvent struct {
	Ringhash      Hash
	PreOrderHash  Hash
	OrderHash     Hash
	NextOrderHash Hash
	RingIndex     *Big
	Time          *Big
	Blocknumber   *Big
	AmountS       *Big
	AmountB       *Big
	LrcReward     *Big
	LrcFee        *Big
}

type OrderCancelledEvent struct {
	OrderHash       Hash
	Time            *Big
	Blocknumber     *Big
	AmountCancelled *Big
}

type CutoffEvent struct {
	Address     Address
	Time        *Big
	Blocknumber *Big
	Cutoff      *Big
}

type RingMinedEvent struct {
	RingIndex     *Big
	Time          *Big
	Blocknumber   *Big
	Ringhash      Hash
	Miner         Address
	FeeRecepient  Address
	RinghashFound bool
}
