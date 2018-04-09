package gateway_test

import (
	"testing"
	"fmt"
	"github.com/Loopring/relay/types"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/test"
)

func TestGetPow(t *testing.T) {

	globalConfig := test.LoadConfig()

	o := types.Order{}
	o.V = 56
	o.R = types.BytesToBytes32([]byte("0x506b2f36c551a910145b240a2ea235345a08c099a3d8e86a812bd189ff7ae036"))
	o.S = types.BytesToBytes32([]byte("0x087a52805f255b5109362b4655f256ddcb80c0ce40b0dace5dfb6299800b5891"))
	o.PowNonce = 1000

	difficulty := types.HexToBigint(globalConfig.GatewayFilters.PowFilter.Difficulty)
	powFilter := &gateway.PowFilter{Difficulty:difficulty}
	fmt.Println(types.BigintToHex(difficulty))

	for o.PowNonce < 1110 {
		fmt.Println(types.BigintToHex(gateway.GetPow(o.V, o.R, o.S, o.PowNonce)))
		fmt.Println(powFilter.Filter(&o))
		o.PowNonce = o.PowNonce + 1
	}
}