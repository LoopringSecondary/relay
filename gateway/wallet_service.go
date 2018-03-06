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
	"fmt"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"qiniupkg.com/x/errors.v7"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultContractVersion = "v1.1"
const DefaultCapCurrency = "CNY"

type Portfolio struct {
	Token      string
	Amount     string
	Percentage string
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

type DepthElement struct {
	Price  string
	Size   *big.Rat
	Amount *big.Rat
}

type CommonTokenRequest struct {
	ContractVersion string `json:"contractVersion"`
	Owner           string `json:"owner"`
}

type SingleContractVersion struct {
	ContractVersion string `json:"contractVersion"`
}

type SingleMarket struct {
	Market string `json:"market"`
}

type SingleOwner struct {
	owner string `json:"owner"`
}

type PriceQuoteQuery struct {
	currency string `json:"currency"`
}

type CutoffRequest struct {
	Address         string `json:"address"`
	ContractVersion string `json:"contractVersion"`
	BlockNumber     string `json:"blockNumber"`
}

type EstimatedAllocatedAllowanceQuery struct {
	Owner string `json: "owner"`
	Token string `json: "token"`
}

type TransactionQuery struct {
	ThxHash   string `json:"thxHash"`
	Owner     string `json:"owner"`
	PageIndex int    `json:"pageIndex"`
	PageSize  int    `json:"pageSize"`
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
	Currency string       `json:"currency"`
	Tokens   []TokenPrice `json:"tokens"`
}

type TokenPrice struct {
	Token string  `json:"token"`
	Price float64 `json:"price"`
}

type WalletServiceImpl struct {
	trendManager    market.TrendManager
	orderManager    ordermanager.OrderManager
	accountManager  market.AccountManager
	marketCap       marketcap.MarketCapProvider
	ethForwarder    *EthForwarder
	tickerCollector market.CollectorImpl
}

func NewWalletService(trendManager market.TrendManager, orderManager ordermanager.OrderManager, accountManager market.AccountManager,
	capProvider marketcap.MarketCapProvider, ethForwarder *EthForwarder, collector market.CollectorImpl) *WalletServiceImpl {
	w := &WalletServiceImpl{}
	w.trendManager = trendManager
	w.orderManager = orderManager
	w.accountManager = accountManager
	w.marketCap = capProvider
	w.ethForwarder = ethForwarder
	w.tickerCollector = collector
	return w
}
func (w *WalletServiceImpl) TestPing(input int) (resp []byte, err error) {

	var res string
	if input > 0 {
		res = "input is bigger than zero " + time.Now().String()
	} else if input == 0 {
		res = "input is equal zero " + time.Now().String()
	} else if input < 0 {
		res = "input is smaller than zero " + time.Now().String()
	}
	resp = []byte("{'abc' : '" + res + "'}")
	return
}

func (w *WalletServiceImpl) GetPortfolio(query SingleOwner) (res []Portfolio, err error) {
	if len(query.owner) == 0 {
		return nil, errors.New("owner can't be nil")
	}

	account := w.accountManager.GetBalance(DefaultContractVersion, query.owner)
	balances := account.Balances
	if len(balances) == 0 {
		return
	}

	priceQuote, err := w.GetPriceQuote(PriceQuoteQuery{DefaultCapCurrency})
	if err != nil {
		return
	}

	priceQuoteMap := make(map[string]*big.Rat)
	for _, pq := range priceQuote.Tokens {
		priceQuoteMap[pq.Token] = new(big.Rat).SetFloat64(pq.Price)
	}

	var totalAsset *big.Rat
	for k, v := range balances {
		asset := priceQuoteMap[k]
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		totalAsset = totalAsset.Add(totalAsset, asset)
	}

	res = make([]Portfolio, 0)

	for k, v := range balances {
		portfolio := Portfolio{Token: k, Amount: types.BigintToHex(v.Balance)}
		asset := priceQuoteMap[k]
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v.Balance, big.NewInt(1)))
		percentage, _ := asset.Quo(asset, totalAsset).Float64()
		portfolio.Percentage = strconv.FormatFloat(percentage, 'f', 2, 64)
		res = append(res, portfolio)
	}

	return
}

func (w *WalletServiceImpl) GetPriceQuote(query PriceQuoteQuery) (result PriceQuote, err error) {

	rst := PriceQuote{query.currency, make([]TokenPrice, 0)}
	for k, v := range util.AllTokens {
		price, _ := w.marketCap.GetMarketCapByCurrency(v.Protocol, query.currency)
		floatPrice, _ := price.Float64()
		rst.Tokens = append(rst.Tokens, TokenPrice{k, floatPrice})
	}

	return rst, nil
}

func (w *WalletServiceImpl) GetTickers(mkt SingleMarket) (result map[string]market.Ticker, err error) {
	result = make(map[string]market.Ticker)
	loopringTicker, err := w.trendManager.GetTickerByMarket(mkt.Market)
	if err != nil {
		result["loopring"] = loopringTicker
	}
	outTickers, err := w.tickerCollector.GetTickers(mkt.Market)
	if err != nil {
		for _, v := range outTickers {
			result[v.Exchange] = v
		}
	}
	return result, nil
}

func (w *WalletServiceImpl) UnlockWallet(owner SingleOwner) (err error) {
	if len(owner.owner) == 0 {
		return errors.New("owner can't be null string")
	}
	return w.accountManager.UnlockedWallet(owner.owner)
}

func (w *WalletServiceImpl) SubmitOrder(order *types.OrderJsonRequest) (res string, err error) {
	err = HandleOrder(types.ToOrder(order))
	if err != nil {
		fmt.Println(err)
	}
	res = "SUBMIT_SUCCESS"
	return res, err
}

func (w *WalletServiceImpl) GetOrders(query *OrderQuery) (res PageResult, err error) {
	orderQuery, pi, ps := convertFromQuery(query)
	queryRst, err := w.orderManager.GetOrders(orderQuery, pi, ps)
	if err != nil {
		fmt.Println(err)
	}
	return buildOrderResult(queryRst), err
}

func (w *WalletServiceImpl) GetDepth(query DepthQuery) (res Depth, err error) {

	mkt := strings.ToUpper(query.Market)
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

	_, err = util.WrapMarket(a, b)
	if err != nil {
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
	asks, askErr := w.orderManager.GetOrderBook(
		common.HexToAddress(util.ContractVersionConfig[protocol]),
		util.AllTokens[a].Protocol,
		util.AllTokens[b].Protocol, length*2)

	if askErr != nil {
		err = errors.New("get depth error , please refresh again")
		return
	}

	depth.Depth.Sell = calculateDepth(asks, length, true, util.AllTokens[a].Decimals, util.AllTokens[b].Decimals)

	bids, bidErr := w.orderManager.GetOrderBook(
		common.HexToAddress(util.ContractVersionConfig[protocol]),
		util.AllTokens[b].Protocol,
		util.AllTokens[a].Protocol, length*2)

	if bidErr != nil {
		err = errors.New("get depth error , please refresh again")
		return
	}

	depth.Depth.Buy = calculateDepth(bids, length, false, util.AllTokens[b].Decimals, util.AllTokens[a].Decimals)

	return depth, err
}

func (w *WalletServiceImpl) GetFills(query FillQuery) (dao.PageResult, error) {
	res, err := w.orderManager.FillsPageQuery(fillQueryToMap(query))

	if err != nil {
		return dao.PageResult{}, nil
	}

	result := dao.PageResult{PageIndex: res.PageIndex, PageSize: res.PageSize, Total: res.Total, Data: make([]interface{}, 0)}

	for _, f := range res.Data {
		fill := f.(dao.FillEvent)
		fill.TokenS = util.AddressToAlias(fill.TokenS)
		fill.TokenB = util.AddressToAlias(fill.TokenB)
		result.Data = append(result.Data, fill)
	}
	return result, nil
}

func (w *WalletServiceImpl) GetTicker(query SingleContractVersion) (res []market.Ticker, err error) {
	res, err = w.trendManager.GetTicker()

	for i, t := range res {
		w.fillBuyAndSell(&t, query.ContractVersion)
		res[i] = t
	}
	return
}

func (w *WalletServiceImpl) GetTrend(query SingleMarket) (res []market.Trend, err error) {
	res, err = w.trendManager.GetTrends(query.Market)
	sort.Slice(res, func(i, j int) bool {
		return res[i].Start < res[j].Start
	})
	return
}

func (w *WalletServiceImpl) GetRingMined(query RingMinedQuery) (res dao.PageResult, err error) {
	return w.orderManager.RingMinedPageQuery(ringMinedQueryToMap(query))
}

func (w *WalletServiceImpl) GetBalance(balanceQuery CommonTokenRequest) (res market.AccountJson, err error) {
	account := w.accountManager.GetBalance(balanceQuery.ContractVersion, balanceQuery.Owner)
	ethBalance := market.Balance{Token: "ETH", Balance: big.NewInt(0)}
	b, bErr := w.ethForwarder.GetBalance(balanceQuery.Owner, "latest")
	if bErr == nil {
		ethBalance.Balance = types.HexToBigint(b)
		newBalances := make(map[string]market.Balance)
		for k, v := range account.Balances {
			newBalances[k] = v
		}
		newBalances["ETH"] = ethBalance
		account.Balances = newBalances
	}
	res = account.ToJsonObject(balanceQuery.ContractVersion)
	return
}

func (w *WalletServiceImpl) GetCutoff(query CutoffRequest) (result string, err error) {
	cutoff, err := ethaccessor.GetCutoff(common.HexToAddress(util.ContractVersionConfig[query.ContractVersion]), common.HexToAddress(query.Address), query.BlockNumber)
	if err != nil {
		return "", err
	}
	return cutoff.String(), nil
}

func (w *WalletServiceImpl) GetEstimatedAllocatedAllowance(query EstimatedAllocatedAllowanceQuery) (frozenAmount string, err error) {
	statusSet := make([]types.OrderStatus, 0)
	statusSet = append(statusSet, types.ORDER_NEW)
	statusSet = append(statusSet, types.ORDER_PARTIAL)

	token := query.Token
	owner := query.Owner

	tokenAddress := util.AliasToAddress(token)
	if tokenAddress.Hex() == "" {
		return "", errors.New("unsupported token alias " + token)
	}
	amount, err := w.orderManager.GetFrozenAmount(common.HexToAddress(owner), tokenAddress, statusSet)
	if err != nil {
		return "", err
	}

	if token == "LRC" {
		allLrcFee, err := w.orderManager.GetFrozenLRCFee(common.HexToAddress(owner), statusSet)
		if err != nil {
			return "", err
		}
		amount.Add(amount, allLrcFee)
	}

	return types.BigintToHex(amount), err
}

func (w *WalletServiceImpl) GetSupportedMarket() (markets []string, err error) {
	return util.AllMarkets, err
}

func (w *WalletServiceImpl) GetTransactions(query TransactionQuery) (transactions []types.Transaction, err error) {
	transactions = make([]types.Transaction, 0)
	mockTxn1 := types.Transaction{
		Protocol:    common.StringToAddress("0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC9"),
		Owner:       common.StringToAddress("0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC2"),
		From:        common.StringToAddress("0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC3"),
		To:          common.StringToAddress("0x66727f5DE8Fbd651Dc375BB926B16545DeD71EC4"),
		CreateTime:  150134131,
		UpdateTime:  150101931,
		TxHash:      common.StringToHash(""),
		BlockNumber: big.NewInt(5029675),
		Status:      1,
		Value:       big.NewInt(0x0000000a7640001),
	}

	transactions = append(transactions, mockTxn1)
	return transactions, nil
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

func calculateDepth(states []types.OrderState, length int, isAsk bool, tokenSDecimal, tokenBDecimal *big.Int) [][]string {

	if len(states) == 0 {
		return [][]string{}
	}

	depth := make([][]string, 0)
	for i := range depth {
		depth[i] = make([]string, 0)
	}

	depthMap := make(map[string]DepthElement)

	for _, s := range states {

		price := *s.RawOrder.Price
		amountS, amountB := s.RemainedAmount()
		amountS = amountS.Quo(amountS, new(big.Rat).SetFrac(tokenSDecimal, big.NewInt(1)))
		amountB = amountB.Quo(amountB, new(big.Rat).SetFrac(tokenBDecimal, big.NewInt(1)))

		if amountS.Cmp(new(big.Rat).SetFloat64(0)) == 0 {
			log.Debug("amount s is zero, skipped")
			continue
		}

		if amountB.Cmp(new(big.Rat).SetFloat64(0)) == 0 {
			log.Debug("amount b is zero, skipped")
			continue
		}

		if isAsk {
			price = *price.Inv(&price)
			priceFloatStr := price.FloatString(10)
			if v, ok := depthMap[priceFloatStr]; ok {
				amount := v.Amount
				size := v.Size
				amount = amount.Add(amount, amountS)
				size = size.Add(size, amountB)
				depthMap[priceFloatStr] = DepthElement{Price: v.Price, Amount: amount, Size: size}
			} else {
				depthMap[priceFloatStr] = DepthElement{Price: priceFloatStr, Amount: amountS, Size: amountB}
			}
		} else {
			priceFloatStr := price.FloatString(10)
			if v, ok := depthMap[priceFloatStr]; ok {
				amount := v.Amount
				size := v.Size
				amount = amount.Add(amount, amountB)
				size = size.Add(size, amountS)
				depthMap[priceFloatStr] = DepthElement{Price: v.Price, Amount: amount, Size: size}
			} else {
				depthMap[priceFloatStr] = DepthElement{Price: priceFloatStr, Amount: amountB, Size: amountS}
			}
		}
	}

	for k, v := range depthMap {
		amount, _ := v.Amount.Float64()
		size, _ := v.Size.Float64()
		depth = append(depth, []string{k, strconv.FormatFloat(amount, 'f', 10, 64), strconv.FormatFloat(size, 'f', 10, 64)})
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
		ps = q.PageSize
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
	rawOrder.Timestamp = src.RawOrder.ValidSince.Int64()
	rawOrder.Ttl = types.BigintToHex(src.RawOrder.ValidUntil)
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

func (w *WalletServiceImpl) fillBuyAndSell(ticker *market.Ticker, contractVersion string) {
	queryDepth := DepthQuery{1, contractVersion, ticker.Market}

	depth, err := w.GetDepth(queryDepth)
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
