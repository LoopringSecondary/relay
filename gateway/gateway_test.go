package gateway_test

import (
	"fmt"
	"github.com/Loopring/relay/types"
	"testing"
	//"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/gateway"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/usermanager"
	common2 "github.com/ethereum/go-ethereum/common"
	"math/big"
	"time"
)

func Ttt() {
	fmt.Println("ttt ")
}

func TestGetPow(t *testing.T) {

	globalConfig := test.LoadConfig()

	fmt.Println(util.CalculatePrice("23399999899980370", "2027934200000000", "0xfe5aFA7BfF3394359af2D277aCc9f00065CdBe2f", "0x639687b7f8501f174356D3aCb1972f749021CCD0"))
	fmt.Println(util.GetSide("0xfe5aFA7BfF3394359af2D277aCc9f00065CdBe2f", "0x639687b7f8501f174356D3aCb1972f749021CCD0"))

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
	//fmt.Println(ws.GetOrders(&oq))
	//trendm.LoadCache()
	ore := &types.OrderFilledEvent{}
	ore.Market = "LRC-WETH"
	ore.TokenS = common2.HexToAddress("0xfe5aFA7BfF3394359af2D277aCc9f00065CdBe2f")
	ore.TokenB = common2.HexToAddress("0x639687b7f8501f174356D3aCb1972f749021CCD0")
	ore.Protocol = common2.HexToAddress("0xD8516b98cFe5e33907488B144112e800632055a5")
	ore.Owner = common2.HexToAddress("0x750aD4351bB728ceC7d639A9511F9D6488f1E259")
	ore.RingIndex = big.NewInt(492)
	ore.BlockNumber = big.NewInt(483818)
	ore.BlockTime = 1523859094
	ore.Status = types.TX_STATUS_SUCCESS
	ore.AmountS = big.NewInt(23399999899980370)
	ore.AmountB = big.NewInt(2027934200000000)
	ore.FillIndex = big.NewInt(1)
	//trendm.ProofRead()
	time.Sleep(time.Minute * 10)
	trendm.HandleOrderFilled(ore)
	fmt.Println(ws.GetTrend(gateway.TrendQuery{Market: "LRC-WETH", Interval: "1Hr"}))
	//fmt.Println(ws.GetTrend(gateway.TrendQuery{Market:"LRC-WETH", Interval:"2Hr"}))
	fmt.Println(ws.GetTicker())

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
