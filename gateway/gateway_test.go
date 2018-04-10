package gateway_test

import (
	"fmt"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/types"
	"testing"
	//"github.com/Loopring/relay/test"
)

func TestGetPow(t *testing.T) {

	//globalConfig := test.LoadConfig()

	o := types.Order{}
	o.V = 27
	o.R = types.BytesToBytes32([]byte("0x12345"))
	o.S = types.BytesToBytes32([]byte("0x12345"))
	o.PowNonce = 30

	//difficulty := types.HexToBigint(globalConfig.GatewayFilters.PowFilter.Difficulty)
	//powFilter := &gateway.PowFilter{Difficulty:difficulty}
	fmt.Println(types.BigintToHex(gateway.GetPow(o.V, o.R, o.S, o.PowNonce)))

	//for o.PowNonce < 1110 {
	//	fmt.Println(types.BigintToHex(gateway.GetPow(o.V, o.R, o.S, o.PowNonce)))
	//	fmt.Println(powFilter.Filter(&o))
	//	o.PowNonce = o.PowNonce + 1
	//}
}
