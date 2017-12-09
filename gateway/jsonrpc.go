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
	"errors"
	"fmt"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"math/big"
	"net"
	"sort"
	"strconv"
	"strings"
)

func (*JsonrpcServiceImpl) Ping(val string, val2 int) (res string, err error) {
	res = "pong for first connect, meaning server is OK"
	return
}

type PageResult struct {
	Data      []interface{} `json:"data"`
	PageIndex int           `json:"pageIndex"`
	PageSize  int           `json:"pageSize"`
	Total     int           `json:"total"`
}

type Depth struct {
	ContractVersion string `json:"contractVersion"`
	Market          string `json:"market"`
	Depth           AskBid `json:"depth"`
}

type AskBid struct {
	Buy  [][]string `json:"buy"`
	Sell [][]string `json:"sell"`
}

type CommonTokenRequest struct {
	ContractVersion string `json:"contractVersion"`
	Owner           string `json:"owner"`
}

type OrderQuery struct {
	Status          string `json:"status"`
	PageIndex       int    `json:"pageIndex"`
	PageSize        int    `json:"pageSize"`
	ContractVersion string `json:"contractVersion"`
	Owner           string `json:"owner"`
	Market          string `json:"market"`
	OrderHash       string `json:"orderHash"`
}

type DepthQuery struct {
	Length          int    `json:"length"`
	ContractVersion string `json:"contractVersion"`
	Market          string `json:"market"`
}

type FillQuery struct {
	ContractVersion string
	Market          string
	Owner           string
	OrderHash       string
	RingHash        string
	PageIndex       int
	PageSize        int
}

type RingMinedQuery struct {
	ContractVersion string
	RingHash        string
	PageIndex       int
	PageSize        int
}

type RawOrderJsonResult struct {
	Protocol              string `json:"protocol"` // 智能合约地址
	Owner                 string `json:"address"`
	Hash                  string `json:"hash"`
	TokenS                string `json:"tokenS"`  // 卖出erc20代币智能合约地址
	TokenB                string `json:"tokenB"`  // 买入erc20代币智能合约地址
	AmountS               string `json:"amountS"` // 卖出erc20代币数量上限
	AmountB               string `json:"amountB"` // 买入erc20代币数量上限
	Timestamp             int64  `json:"timestamp"`
	Ttl                   string `json:"ttl"` // 订单过期时间
	Salt                  string `json:"salt"`
	LrcFee                string `json:"lrcFee"` // 交易总费用,部分成交的费用按该次撮合实际卖出代币额与比例计算
	BuyNoMoreThanAmountB  bool   `json:"buyNoMoreThanAmountB"`
	MarginSplitPercentage string `json:"marginSplitPercentage"` // 不为0时支付给交易所的分润比例，否则视为100%
	V                     string `json:"v"`
	R                     string `json:"r"`
	S                     string `json:"s"`
}

type OrderJsonResult struct {
	RawOrder         RawOrderJsonResult `json:"originalOrder"`
	DealtAmountS     string             `json:"dealtAmountS"`
	DealtAmountB     string             `json:"dealtAmountB"`
	CancelledAmountS string             `json:"cancelledAmountS"`
	CancelledAmountB string             `json:"cancelledAmountB"`
	Status           string             `json:"status"`
}

type PriceQuote struct {
	Currency string `json:"currency"`
	Tokens [] TokenPrice `json:"tokens"`
}

type TokenPrice struct {
	Token string  `json:"token"`
	Price float64 `json:"price"`
}

var RemoteAddrContextKey = "RemoteAddr"

type JsonrpcService interface {
	Start(port string)
	Stop()
}

type JsonrpcServiceImpl struct {
	port           string
	trendManager   market.TrendManager
	orderManager   ordermanager.OrderManager
	accountManager market.AccountManager
	ethForwarder   *EthForwarder
	marketCap      *marketcap.MarketCapProvider
}

func NewJsonrpcService(port string, trendManager market.TrendManager, orderManager ordermanager.OrderManager, accountManager market.AccountManager, ethForwarder *EthForwarder, capProvider *marketcap.MarketCapProvider) *JsonrpcServiceImpl {
	l := &JsonrpcServiceImpl{}
	l.port = port
	l.trendManager = trendManager
	l.orderManager = orderManager
	l.accountManager = accountManager
	l.ethForwarder = ethForwarder
	l.marketCap = capProvider
	return l
}

func (j *JsonrpcServiceImpl) Start() {
	handler := rpc.NewServer()
	if err := handler.RegisterName("loopring", j); err != nil {
		fmt.Println(err)
		return
	}
	if err := handler.RegisterName("eth", j.ethForwarder); err != nil {
		fmt.Println(err)
		return
	}
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", ":8083"); err != nil {
		return
	}
	go rpc.NewHTTPServer([]string{"*"}, handler).Serve(listener)
	log.Info(fmt.Sprintf("HTTP endpoint opened: http://%s", ":8083"))

	return
}

func (j *JsonrpcServiceImpl) SubmitOrder(order *types.OrderJsonRequest) (res string, err error) {
	err = HandleOrder(types.ToOrder(order))
	if err != nil {
		fmt.Println(err)
	}
	res = "SUBMIT_SUCCESS"
	return res, err
}

func (j *JsonrpcServiceImpl) GetOrders(query *OrderQuery) (res PageResult, err error) {
	orderQuery, pi, ps := convertFromQuery(query)
	queryRst, err := j.orderManager.GetOrders(orderQuery, pi, ps)
	if err != nil {
		fmt.Println(err)
	}
	return buildOrderResult(queryRst), err
}

func (j *JsonrpcServiceImpl) GetDepth(query DepthQuery) (res Depth, err error) {

	mkt := strings.ToLower(query.Market)
	protocol := query.ContractVersion
	length := query.Length

	if mkt == "" || protocol == "" || util.ContractVersionConfig[protocol] == "" {
		err = errors.New("market and correct contract version must be applied")
		return
	}

	if length <= 0 || length > 20 {
		length = 20
	}

	a, b := util.UnWrap(mkt)
	if !util.IsSupportedToken(a) || !util.IsSupportedToken(b) {
		err = errors.New("unsupported market type")
		return
	}

	empty := make([][]string, 0)

	for i := range empty {
		empty[i] = make([]string, 0)
	}
	askBid := AskBid{Buy: empty, Sell: empty}
	depth := Depth{ContractVersion: util.ContractVersionConfig[protocol], Market: mkt, Depth: askBid}

	//(TODO) 考虑到需要聚合的情况，所以每次取2倍的数据，先聚合完了再cut, 不是完美方案，后续再优化
	asks, askErr := j.orderManager.GetOrderBook(
		common.HexToAddress(util.ContractVersionConfig[protocol]),
		util.AllTokens[a].Protocol,
		util.AllTokens[b].Protocol, length*2)

	if askErr != nil {
		err = errors.New("get depth error , please refresh again")
		return
	}

	depth.Depth.Sell = calculateDepth(asks, length, true)

	bids, bidErr := j.orderManager.GetOrderBook(
		common.HexToAddress(util.ContractVersionConfig[protocol]),
		util.AllTokens[b].Protocol,
		util.AllTokens[a].Protocol, length*2)

	if bidErr != nil {
		err = errors.New("get depth error , please refresh again")
		return
	}

	depth.Depth.Buy = calculateDepth(bids, length, false)

	return depth, err
}

func (j *JsonrpcServiceImpl) GetFills(query FillQuery) (dao.PageResult, error) {
	res, err := j.orderManager.FillsPageQuery(fillQueryToMap(query))

	if err != nil {
		return dao.PageResult{}, nil
	}

	result := dao.PageResult{PageIndex:res.PageIndex, PageSize:res.PageSize, Total:res.Total, Data:make([]interface{}, 0)}

	for _, f := range res.Data {
		fill := f.(dao.FillEvent)
		fill.TokenS = util.AddressToAlias(fill.TokenS)
		fill.TokenB = util.AddressToAlias(fill.TokenB)
		result.Data = append(result.Data, fill)
	}
	fmt.Println(result)
	return result, nil
}

func (j *JsonrpcServiceImpl) GetTicker(contractVersion string) (res []market.Ticker, err error) {
	res, err = j.trendManager.GetTicker()

	for _, t := range res {
		j.fillBuyAndSell(&t, contractVersion)
	}
	return
}

func (j *JsonrpcServiceImpl) GetTrend(market string) (res []market.Trend, err error) {
	res, err = j.trendManager.GetTrends(market)
	return
}

func (j *JsonrpcServiceImpl) GetRingMined(query RingMinedQuery) (res dao.PageResult, err error) {
	return j.orderManager.RingMinedPageQuery(ringMinedQueryToMap(query))
}

func (j *JsonrpcServiceImpl) GetBalance(balanceQuery CommonTokenRequest) (res market.AccountJson, err error) {
	account := j.accountManager.GetBalance(balanceQuery.ContractVersion, balanceQuery.Owner)
	ethBalance := market.Balance{Token:"ETH", Balance:big.NewInt(0)}
	b, bErr := j.ethForwarder.GetBalance(balanceQuery.Owner, "latest")
	if bErr == nil {
		ethBalance.Balance = types.HexToBigint(b)
		account.Balances["ETH"] = ethBalance
	}
	res = account.ToJsonObject(balanceQuery.ContractVersion)
	return
}

func (j *JsonrpcServiceImpl) GetCutoff(address, contractVersion, blockNumber string) (result string, err error) {
	cutoff, err := j.ethForwarder.Accessor.GetCutoff(common.HexToAddress(address), common.HexToAddress(util.ContractVersionConfig[contractVersion]), blockNumber)
	if err != nil {
		return "", err
	}
	return cutoff.String(), nil
}

func (j *JsonrpcServiceImpl) GetPriceQuote(currency string) (result PriceQuote, err error) {

	rst := PriceQuote{currency, make([]TokenPrice, 0)}
	for k, v := range util.AllTokens {
		price := j.marketCap.GetMarketCapByCurrency(v.Protocol, marketcap.StringToLegalCurrency(currency))
		floatPrice, _ := price.Float64()
		rst.Tokens = append(rst.Tokens, TokenPrice{k, floatPrice})
	}

	return rst, nil
}

func convertFromQuery(orderQuery *OrderQuery) (query map[string]interface{}, pageIndex int, pageSize int) {

	query = make(map[string]interface{})
	status := convertStatus(orderQuery.Status)
	if uint8(status) != 0 {
		query["status"] = uint8(status)
	}
	if orderQuery.Owner != "" {
		query["owner"] = orderQuery.Owner
	}
	if util.ContractVersionConfig[orderQuery.ContractVersion] != "" {
		query["protocol"] = util.ContractVersionConfig[orderQuery.ContractVersion]
	}
	if orderQuery.Market != "" {
		query["market"] = orderQuery.Market
	}
	if orderQuery.OrderHash != "" {
		query["order_hash"] = orderQuery.OrderHash
	}
	pageIndex = orderQuery.PageIndex
	pageSize = orderQuery.PageSize
	return

}

func convertStatus(s string) types.OrderStatus {
	switch s {
	case "ORDER_NEW":
		return types.ORDER_NEW
	case "ORDER_PARTIAL":
		return types.ORDER_PARTIAL
	case "ORDER_FINISHED":
		return types.ORDER_FINISHED
	case "ORDER_CANCELED":
		return types.ORDER_CANCEL
	case "ORDER_CUTOFF":
		return types.ORDER_CUTOFF
	}
	return types.ORDER_UNKNOWN
}

func getStringStatus(s types.OrderStatus) string {
	switch s {
	case types.ORDER_NEW:
		return "ORDER_NEW"
	case types.ORDER_PARTIAL:
		return "ORDER_PARTIAL"
	case types.ORDER_FINISHED:
		return "ORDER_FINISHED"
	case types.ORDER_CANCEL:
		return "ORDER_CANCELED"
	case types.ORDER_CUTOFF:
		return "ORDER_CUTOFF"
	}
	return "ORDER_UNKNOWN"
}

func calculateDepth(states []types.OrderState, length int, isAsk bool) [][]string {

	if len(states) == 0 {
		return [][]string{}
	}

	depth := make([][]string, 0)
	for i := range depth {
		depth[i] = make([]string, 0)
	}

	depthMap := make(map[string]big.Rat)

	for _, s := range states {

		price := *s.RawOrder.Price
		amountS, amountB := s.RemainedAmount()

		if isAsk {
			price = *price.Inv(&price)
			priceFloatStr := price.FloatString(10)
			if v, ok := depthMap[priceFloatStr]; ok {
				depthMap[priceFloatStr] = *v.Add(&v, amountS)
			} else {
				depthMap[priceFloatStr] = *amountS
			}
		} else {
			priceFloatStr := price.FloatString(10)
			if v, ok := depthMap[priceFloatStr]; ok {
				depthMap[priceFloatStr] = *v.Add(&v, amountB)
			} else {
				depthMap[priceFloatStr] = *amountB
			}
		}
	}

	for k, v := range depthMap {
		amount, _ := v.Float64()
		depth = append(depth, []string{k, strconv.FormatFloat(amount/util.WeiToEther, 'f', 10, 64)})
	}

	sort.Slice(depth, func(i, j int) bool {
		cmpA, _ := strconv.ParseFloat(depth[i][0], 64)
		cmpB, _ := strconv.ParseFloat(depth[j][0], 64)
		if isAsk {
			return cmpA < cmpB
		} else {
			return cmpA > cmpB
		}

	})

	if length < len(depth) {
		return depth[:length]
	}
	return depth
}

func fillQueryToMap(q FillQuery) (map[string]interface{}, int, int) {
	rst := make(map[string]interface{})
	var pi, ps int
	if q.Market != "" {
		rst["market"] = q.Market
	}
	if q.PageIndex <= 0 {
		pi = 1
	} else {
		pi = q.PageIndex
	}
	if q.PageSize <= 0 || q.PageSize > 20 {
		ps = 20
	} else {
		ps = q.PageIndex
	}
	if q.ContractVersion != "" {
		rst["contract_address"] = util.ContractVersionConfig[q.ContractVersion]
	}
	if q.Owner != "" {
		rst["owner"] = q.Owner
	}
	if q.OrderHash != "" {
		rst["order_hash"] = q.OrderHash
	}
	if q.RingHash != "" {
		rst["ring_hash"] = q.RingHash
	}

	return rst, pi, ps
}

func orderQueryToMap(q FillQuery) (map[string]string, int, int) {
	rst := make(map[string]string)
	var pi, ps int
	if q.Market != "" {
		rst["market"] = q.Market
	}
	if q.PageIndex <= 0 {
		pi = 1
	} else {
		pi = q.PageIndex
	}
	if q.PageSize <= 0 || q.PageSize > 20 {
		ps = 20
	} else {
		ps = q.PageIndex
	}
	if q.ContractVersion != "" {
		rst["contract_version"] = util.ContractVersionConfig[q.ContractVersion]
	}
	if q.Owner != "" {
		rst["owner"] = q.Owner
	}
	if q.OrderHash != "" {
		rst["order_hash"] = q.OrderHash
	}
	if q.RingHash != "" {
		rst["ring_hash"] = q.RingHash
	}

	return rst, pi, ps
}

func ringMinedQueryToMap(q RingMinedQuery) (map[string]interface{}, int, int) {
	rst := make(map[string]interface{})
	var pi, ps int
	if q.PageIndex <= 0 {
		pi = 1
	} else {
		pi = q.PageIndex
	}
	if q.PageSize <= 0 || q.PageSize > 20 {
		ps = 20
	} else {
		ps = q.PageSize
	}
	if q.ContractVersion != "" {
		rst["contract_address"] = util.ContractVersionConfig[q.ContractVersion]
	}
	if q.RingHash != "" {
		rst["ring_hash"] = q.RingHash
	}

	return rst, pi, ps
}

func buildOrderResult(src dao.PageResult) PageResult {

	rst := PageResult{Total: src.Total, PageIndex: src.PageIndex, PageSize: src.PageSize, Data: make([]interface{}, 0)}

	for _, d := range src.Data {
		o := d.(types.OrderState)
		rst.Data = append(rst.Data, orderStateToJson(o))
	}
	return rst
}

func orderStateToJson(src types.OrderState) OrderJsonResult {

	rst := OrderJsonResult{}
	rst.DealtAmountB = types.BigintToHex(src.DealtAmountB)
	rst.DealtAmountS = types.BigintToHex(src.DealtAmountS)
	rst.CancelledAmountB = types.BigintToHex(src.CancelledAmountB)
	rst.CancelledAmountS = types.BigintToHex(src.CancelledAmountS)
	rst.Status = getStringStatus(src.Status)
	rawOrder := RawOrderJsonResult{}
	rawOrder.Protocol = src.RawOrder.Protocol.String()
	rawOrder.Owner = src.RawOrder.Owner.String()
	rawOrder.Hash = src.RawOrder.Hash.String()
	rawOrder.TokenS = util.AddressToAlias(src.RawOrder.TokenS.String())
	rawOrder.TokenB = util.AddressToAlias(src.RawOrder.TokenB.String())
	rawOrder.AmountS = types.BigintToHex(src.RawOrder.AmountS)
	rawOrder.AmountB = types.BigintToHex(src.RawOrder.AmountB)
	rawOrder.Timestamp = src.RawOrder.Timestamp.Int64()
	rawOrder.Ttl = types.BigintToHex(src.RawOrder.Ttl)
	rawOrder.Salt = types.BigintToHex(src.RawOrder.Salt)
	rawOrder.LrcFee = types.BigintToHex(src.RawOrder.LrcFee)
	rawOrder.BuyNoMoreThanAmountB = src.RawOrder.BuyNoMoreThanAmountB
	rawOrder.MarginSplitPercentage = types.BigintToHex(big.NewInt(int64(src.RawOrder.MarginSplitPercentage)))
	rawOrder.V = types.BigintToHex(big.NewInt(int64(src.RawOrder.V)))
	rawOrder.R = src.RawOrder.R.Hex()
	rawOrder.S = src.RawOrder.S.Hex()
	rst.RawOrder = rawOrder
	return rst
}

func (j *JsonrpcServiceImpl) fillBuyAndSell(ticker *market.Ticker, contractVersion string) {
	queryDepth := DepthQuery{1, contractVersion, ticker.Market}

	depth, err := j.GetDepth(queryDepth)
	if err != nil {
		log.Error("fill depth info failed")
	} else {
		if len(depth.Depth.Buy) > 0 {
			ticker.Buy = depth.Depth.Buy[0][0]
		}
		if len(depth.Depth.Sell) > 0 {
			ticker.Sell = depth.Depth.Sell[0][0]
		}
	}
}
