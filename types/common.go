package types

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"math/big"
)

func Int2BlockNumHex(height int) string {
	data := big.NewInt(int64(height))
	return hexutil.EncodeBig(data) //common.Bytes2Hex(height.Bytes())
}
