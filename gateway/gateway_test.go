package gateway_test

import (
	"fmt"
	"github.com/Loopring/relay/types"
	"testing"
	//"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/usermanager"
)

func TestGetPow(t *testing.T) {

	globalConfig := test.LoadConfig()

	rds := dao.NewRdsService(globalConfig.Mysql)

	um := usermanager.NewUserManager(&globalConfig.UserManager, rds)
	mc := marketcap.NewMarketCapProvider(globalConfig.MarketCap)

	om := ordermanager.NewOrderManager(&globalConfig.OrderManager, rds, um, mc)
	trendm := market.NewTrendManager(rds)
	am := market.NewAccountManager()

	ethf := gateway.EthForwarder{}
	collector := market.NewCollector()
	protocols := make(map[string]string)
	ws := gateway.NewWalletService(trendm, om, am, mc, &ethf, *collector, rds, "", protocols)
	oq := gateway.OrderQuery{}
	oq.ContractVersion = "v1.4"
	oq.Side = "buy"
	fmt.Println(ws.GetOrders(&oq))

	//q := make(map[string]interface{})
	//q["side"] = "buy"
	//q["protocol"] = "0x13d8d3c7318B118b28ed810E15F1A7B9Bea46fAA"
	//os := make([]types.OrderStatus, 0)
	//pr, _ := om.GetOrders(q, os, 1, 30)
	//fmt.Println(">>>>>>")
	//fmt.Println(gateway.BuildOrderResult(pr))
	//fmt.Println("2>>>>>>")
	//ooo := pr.Data[0].(types.OrderState)
	//fmt.Println(ooo)
	//fmt.Println(gateway.OrderStateToJson(ooo))

	o := types.Order{}
	o.V = 27
	o.R = types.BytesToBytes32([]byte("0x12345"))
	o.S = types.BytesToBytes32([]byte("0x12345"))
	o.PowNonce = 30

	//difficulty := types.HexToBigint(globalConfig.GatewayFilters.PowFilter.Difficulty)
	//powFilter := &gateway.PowFilter{Difficulty:difficulty}
	//fmt.Println(types.BigintToHex(gateway.GetPow(o.V, o.R, o.S, o.PowNonce)))

	//for o.PowNonce < 1110 {
	//	fmt.Println(types.BigintToHex(gateway.GetPow(o.V, o.R, o.S, o.PowNonce)))
	//	fmt.Println(powFilter.Filter(&o))
	//	o.PowNonce = o.PowNonce + 1
	//}

}
