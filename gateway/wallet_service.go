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
	"encoding/json"
	"fmt"
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/txmanager"
	txtyp "github.com/Loopring/relay/txmanager/types"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"qiniupkg.com/x/errors.v7"
	"sort"
	"strconv"
	"strings"
	"time"
)

const DefaultCapCurrency = "CNY"
const PendingTxPreKey = "PENDING_TX_"

const SYS_10001 = "10001"
const P2P_50001 = "50001"
const P2P_50002 = "50002"
const P2P_50003 = "50003"
const P2P_50004 = "50004"
const P2P_50005 = "50005"
const P2P_50006 = "50006"
const P2P_50008 = "50008"

type Portfolio struct {
	Token      string `json:"token"`
	Amount     string `json:"amount"`
	Percentage string `json:"percentage"`
}

type PageResult struct {
	Data      []interface{} `json:"data"`
	PageIndex int           `json:"pageIndex"`
	PageSize  int           `json:"pageSize"`
	Total     int           `json:"total"`
}

type Depth struct {
	DelegateAddress string `json:"delegateAddress"`
	Market          string `json:"market"`
	Depth           AskBid `json:"depth"`
}

type AskBid struct {
	Buy  [][]string `json:"buy"`
	Sell [][]string `json:"sell"`
}

type DepthElement struct {
	Price  string   `json:"price"`
	Size   *big.Rat `json:"size"`
	Amount *big.Rat `json:"amount"`
}

type CommonTokenRequest struct {
	DelegateAddress string `json:"delegateAddress"`
	Owner           string `json:"owner"`
}

type SingleDelegateAddress struct {
	DelegateAddress string `json:"delegateAddress"`
}

type SingleMarket struct {
	Market string `json:"market"`
}

type TrendQuery struct {
	Market   string `json:"market"`
	Interval string `json:"interval"`
}

type SingleOwner struct {
	Owner string `json:"owner"`
}

type TxNotify struct {
	Hash     string `json:"hash"`
	Nonce    string `json:"nonce"`
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	GasPrice string `json:"gasPrice"`
	Gas      string `json:"gas"`
	Input    string `json:"input"`
	R        string `json:"r"`
	S        string `json:"s"`
	V        string `json:"v"`
}

type PriceQuoteQuery struct {
	Currency string `json:"currency"`
}

type CutoffRequest struct {
	Address         string `json:"address"`
	DelegateAddress string `json:"delegateAddress"`
	BlockNumber     string `json:"blockNumber"`
}

type EstimatedAllocatedAllowanceQuery struct {
	DelegateAddress string `json:"delegateAddress"`
	Owner           string `json: "owner"`
	Token           string `json: "token"`
}

type TransactionQuery struct {
	ThxHash   string   `json:"thxHash"`
	Owner     string   `json:"owner"`
	Symbol    string   `json: "symbol"`
	Status    string   `json: "status"`
	TxType    string   `json:"txType"`
	TrxHashes []string `json:"trxHashes"`
	PageIndex int      `json:"pageIndex"`
	PageSize  int      `json:"pageSize"`
}

type OrderQuery struct {
	Status          string `json:"status"`
	PageIndex       int    `json:"pageIndex"`
	PageSize        int    `json:"pageSize"`
	DelegateAddress string `json:"delegateAddress"`
	Owner           string `json:"owner"`
	Market          string `json:"market"`
	OrderHash       string `json:"orderHash"`
	Side            string `json:"side"`
	OrderType       string `json:"orderType"`
}

type DepthQuery struct {
	DelegateAddress string `json:"delegateAddress"`
	Market          string `json:"market"`
}

type FillQuery struct {
	DelegateAddress string `json:"delegateAddress"`
	Market          string `json:"market"`
	Owner           string `json:"owner"`
	OrderHash       string `json:"orderHash"`
	RingHash        string `json:"ringHash"`
	PageIndex       int    `json:"pageIndex"`
	PageSize        int    `json:"pageSize"`
	Side            string `json:"side"`
	OrderType       string `json:"orderType"`
}

type RingMinedQuery struct {
	DelegateAddress string `json:"delegateAddress"`
	ProtocolAddress string `json:"protocolAddress"`
	RingIndex       string `json:"ringIndex"`
	PageIndex       int    `json:"pageIndex"`
	PageSize        int    `json:"pageSize"`
}

type RawOrderJsonResult struct {
	Protocol        string `json:"protocol"`        // 智能合约地址
	DelegateAddress string `json:"delegateAddress"` // 智能合约地址
	Owner           string `json:"address"`
	Hash            string `json:"hash"`
	TokenS          string `json:"tokenS"`  // 卖出erc20代币智能合约地址
	TokenB          string `json:"tokenB"`  // 买入erc20代币智能合约地址
	AmountS         string `json:"amountS"` // 卖出erc20代币数量上限
	AmountB         string `json:"amountB"` // 买入erc20代币数量上限
	ValidSince      string `json:"validSince"`
	ValidUntil      string `json:"validUntil"` // 订单过期时间
	//Salt                  string `json:"salt"`
	LrcFee                string `json:"lrcFee"` // 交易总费用,部分成交的费用按该次撮合实际卖出代币额与比例计算
	BuyNoMoreThanAmountB  bool   `json:"buyNoMoreThanAmountB"`
	MarginSplitPercentage string `json:"marginSplitPercentage"` // 不为0时支付给交易所的分润比例，否则视为100%
	V                     string `json:"v"`
	R                     string `json:"r"`
	S                     string `json:"s"`
	WalletAddress         string `json:"walletAddress" gencodec:"required"`
	AuthAddr              string `json:"authAddr" gencodec:"required"`       //
	AuthPrivateKey        string `json:"authPrivateKey" gencodec:"required"` //
	Market                string `json:"market"`
	Side                  string `json:"side"`
	CreateTime            int64  `json:"createTime"`
	OrderType             string `json:"orderType"`
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
	Token string  `json:"symbol"`
	Price float64 `json:"price"`
}

type RingMinedDetail struct {
	RingInfo RingMinedInfo   `json:"ringInfo"`
	Fills    []dao.FillEvent `json:"fills"`
}

type RingMinedInfo struct {
	ID                 int                 `json:"id"`
	Protocol           string              `json:"protocol"`
	DelegateAddress    string              `json:"delegateAddress"`
	RingIndex          string              `json:"ringIndex"`
	RingHash           string              `json:"ringHash"`
	TxHash             string              `json:"txHash"`
	Miner              string              `json:"miner"`
	FeeRecipient       string              `json:"feeRecipient"`
	IsRinghashReserved bool                `json:"isRinghashReserved"`
	BlockNumber        int64               `json:"blockNumber"`
	TotalLrcFee        string              `json:"totalLrcFee"`
	TotalSplitFee      map[string]*big.Int `json:"totalSplitFee"`
	TradeAmount        int                 `json:"tradeAmount"`
	Time               int64               `json:"timestamp"`
}

type Token struct {
	Token     string `json:"symbol"`
	Balance   string `json:"balance"`
	Allowance string `json:"allowance"`
}

type AccountJson struct {
	DelegateAddress string  `json:"delegateAddress"`
	Address         string  `json:"owner"`
	Tokens          []Token `json:"tokens"`
}

type LatestFill struct {
	CreateTime int64   `json:"createTime"`
	Price      float64 `json:"price"`
	Amount     float64 `json:"amount"`
	Side       string  `json:"side"`
	RingHash   string  `json:"ringHash"`
	LrcFee     string  `json:"lrcFee"`
	SplitS     string  `json:"splitS"`
	SplitB     string  `json:"splitB"`
}

type P2PRingRequest struct {
	RawTx string `json:"rawTx"`
	//Taker          *types.OrderJsonRequest `json:"taker"`
	TakerOrderHash string `json:"takerOrderHash"`
	MakerOrderHash string `json:"makerOrderHash"`
}

type WalletServiceImpl struct {
	trendManager    market.TrendManager
	orderManager    ordermanager.OrderManager
	accountManager  market.AccountManager
	marketCap       marketcap.MarketCapProvider
	tickerCollector market.CollectorImpl
	rds             dao.RdsService
	oldWethAddress  string
}

func NewWalletService(trendManager market.TrendManager, orderManager ordermanager.OrderManager, accountManager market.AccountManager,
	capProvider marketcap.MarketCapProvider, collector market.CollectorImpl, rds dao.RdsService, oldWethAddress string) *WalletServiceImpl {
	w := &WalletServiceImpl{}
	w.trendManager = trendManager
	w.orderManager = orderManager
	w.accountManager = accountManager
	w.marketCap = capProvider
	w.tickerCollector = collector
	w.rds = rds
	w.oldWethAddress = oldWethAddress
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
	res = make([]Portfolio, 0)
	if !common.IsHexAddress(query.Owner) {
		return nil, errors.New("owner can't be nil")
	}

	balances, _ := w.accountManager.GetBalanceWithSymbolResult(common.HexToAddress(query.Owner))
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

	totalAsset := big.NewRat(0, 1)
	for k, v := range balances {
		asset := new(big.Rat).Set(priceQuoteMap[k])
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v, big.NewInt(1)))
		totalAsset = totalAsset.Add(totalAsset, asset)
	}

	for k, v := range balances {
		portfolio := Portfolio{Token: k, Amount: v.String()}
		asset := new(big.Rat).Set(priceQuoteMap[k])
		asset = asset.Mul(asset, new(big.Rat).SetFrac(v, big.NewInt(1)))
		totalAssetFloat, _ := totalAsset.Float64()
		var percentage float64
		if totalAssetFloat == 0 {
			percentage = 0
		} else {
			percentage, _ = asset.Quo(asset, totalAsset).Float64()
		}
		portfolio.Percentage = fmt.Sprintf("%.4f%%", 100*percentage)
		res = append(res, portfolio)
	}

	sort.Slice(res, func(i, j int) bool {
		percentStrLeft := strings.Replace(res[i].Percentage, "%", "", 1)
		percentStrRight := strings.Replace(res[j].Percentage, "%", "", 1)
		left, _ := strconv.ParseFloat(percentStrLeft, 64)
		right, _ := strconv.ParseFloat(percentStrRight, 64)
		return left > right
	})

	return
}

func (w *WalletServiceImpl) GetPriceQuote(query PriceQuoteQuery) (result PriceQuote, err error) {

	rst := PriceQuote{query.Currency, make([]TokenPrice, 0)}
	for k, v := range util.AllTokens {
		price, err := w.marketCap.GetMarketCapByCurrency(v.Protocol, query.Currency)
		if err != nil {
			log.Debug(">>>>>>>> get market cap error " + err.Error())
			rst.Tokens = append(rst.Tokens, TokenPrice{k, 0.0})
		} else {
			floatPrice, _ := price.Float64()
			rst.Tokens = append(rst.Tokens, TokenPrice{k, floatPrice})
			if k == "WETH" {
				rst.Tokens = append(rst.Tokens, TokenPrice{"ETH", floatPrice})
			}
		}
	}
	return rst, nil
}

func (w *WalletServiceImpl) GetTickers(mkt SingleMarket) (result map[string]market.Ticker, err error) {
	result = make(map[string]market.Ticker)
	loopringTicker, err := w.trendManager.GetTickerByMarket(mkt.Market)
	if err == nil {
		result["loopr"] = loopringTicker
	} else {
		log.Info("get ticker from loopring error" + err.Error())
		return result, err
	}
	outTickers, err := w.tickerCollector.GetTickers(mkt.Market)
	if err == nil {
		for _, v := range outTickers {
			result[v.Exchange] = v
		}
	} else {
		log.Info("get other exchanges ticker error" + err.Error())
	}
	return result, nil
}

func (w *WalletServiceImpl) UnlockWallet(owner SingleOwner) (result string, err error) {
	if len(owner.Owner) == 0 {
		return "", errors.New("owner can't be null string")
	}

	unlockRst := w.accountManager.UnlockedWallet(owner.Owner)
	if unlockRst != nil {
		return "", unlockRst
	} else {
		return "unlock_notice_success", nil
	}
}

func (w *WalletServiceImpl) NotifyTransactionSubmitted(txNotify TxNotify) (result string, err error) {

	log.Info("input transaciton found > >>>>>>>>" + txNotify.Hash)

	if len(txNotify.Hash) == 0 {
		return "", errors.New("raw tx can't be null string")
	}
	if !common.IsHexAddress(txNotify.From) || !common.IsHexAddress(txNotify.To) {
		return "", errors.New("from or to address is illegal")
	}

	tx := &ethaccessor.Transaction{}
	tx.Hash = txNotify.Hash
	tx.Input = txNotify.Input
	tx.From = txNotify.From
	tx.To = txNotify.To
	tx.Gas = *types.NewBigPtr(types.HexToBigint(txNotify.Gas))
	tx.GasPrice = *types.NewBigPtr(types.HexToBigint(txNotify.GasPrice))
	tx.Nonce = *types.NewBigPtr(types.HexToBigint(txNotify.Nonce))
	tx.Value = *types.NewBigPtr(types.HexToBigint(txNotify.Value))
	if len(txNotify.V) > 0 {
		tx.V = txNotify.V
	}
	if len(txNotify.R) > 0 {
		tx.R = txNotify.R
	}
	if len(txNotify.S) > 0 {
		tx.S = txNotify.S
	}
	tx.BlockNumber = *types.NewBigWithInt(0)
	tx.BlockHash = ""
	tx.TransactionIndex = *types.NewBigWithInt(0)

	log.Debug("emit Pending tx >>>>>>>>>>>>>>>> " + tx.Hash)
	eventemitter.Emit(eventemitter.PendingTransaction, tx)
	txByte, err := json.Marshal(txNotify)
	if err == nil {
		err = cache.Set(PendingTxPreKey+strings.ToUpper(txNotify.Hash), txByte, 3600*24*7)
		if err != nil {
			return "", err
		}
	}
	log.Info("emit transaction info " + tx.Hash)
	return tx.Hash, nil
}

func (w *WalletServiceImpl) GetOldVersionWethBalance(owner SingleOwner) (res string, err error) {
	b, err := ethaccessor.Erc20Balance(common.HexToAddress(w.oldWethAddress), common.HexToAddress(owner.Owner), "latest")
	if err != nil {
		return
	} else {
		return types.BigintToHex(b), nil
	}
}

func (w *WalletServiceImpl) SubmitOrder(order *types.OrderJsonRequest) (res string, err error) {

	if order.OrderType != types.ORDER_TYPE_MARKET && order.OrderType != types.ORDER_TYPE_P2P {
		order.OrderType = types.ORDER_TYPE_MARKET
	}

	return HandleInputOrder(types.ToOrder(order))
}

func (w *WalletServiceImpl) GetOrders(query *OrderQuery) (res PageResult, err error) {
	orderQuery, statusList, pi, ps := convertFromQuery(query)
	queryRst, err := w.orderManager.GetOrders(orderQuery, statusList, pi, ps)
	if err != nil {
		log.Info("query order error : " + err.Error())
	}
	return buildOrderResult(queryRst), err
}

func (w *WalletServiceImpl) GetOrderByHash(query OrderQuery) (order OrderJsonResult, err error) {
	if len(query.OrderHash) == 0 {
		return order, errors.New("order hash can't be null")
	} else {
		state, err := w.orderManager.GetOrderByHash(common.HexToHash(query.OrderHash))
		if err != nil {
			return order, err
		} else {
			return orderStateToJson(*state), err
		}
	}
}

func (w *WalletServiceImpl) SubmitRingForP2P(p2pRing P2PRingRequest) (res string, err error) {

	maker, err := w.orderManager.GetOrderByHash(common.HexToHash(p2pRing.MakerOrderHash))
	if err != nil {
		return res, errors.New(P2P_50001)
	}

	taker, err := w.orderManager.GetOrderByHash(common.HexToHash(p2pRing.TakerOrderHash))
	if err != nil {
		return res, errors.New(P2P_50008)
	}

	if taker.RawOrder.OrderType != types.ORDER_TYPE_P2P || maker.RawOrder.OrderType != types.ORDER_TYPE_P2P {
		//return res, errors.New("only p2p order can be submitted")
		return res, errors.New(P2P_50002)
	}

	if !maker.IsEffective() {
		//return res, errors.New("maker order has been finished, can't be match ring again")
		return res, errors.New(P2P_50003)
	}

	if taker.RawOrder.AmountS.Cmp(maker.RawOrder.AmountB) != 0 || taker.RawOrder.AmountB.Cmp(maker.RawOrder.AmountS) != 0 {
		//return res, errors.New("the amount of maker and taker are not matched")
		return res, errors.New(P2P_50004)
	}

	if taker.RawOrder.Owner.Hex() == maker.RawOrder.Owner.Hex() {
		//return res, errors.New("taker and maker's address can't be same")
		return res, errors.New(P2P_50005)
	}

	if ordermanager.IsP2PMakerLocked(maker.RawOrder.Hash.Hex()) {
		//return res, errors.New("maker order has been locked by other taker or expired")
		return res, errors.New(P2P_50006)
	}

	var txHashRst string
	err = ethaccessor.SendRawTransaction(&txHashRst, p2pRing.RawTx)
	if err != nil {
		return res, err
	}

	err = ordermanager.SaveP2POrderRelation(taker.RawOrder.Owner.Hex(), taker.RawOrder.Hash.Hex(), maker.RawOrder.Owner.Hex(), maker.RawOrder.Hash.Hex(), txHashRst)
	if err != nil {
		return res, errors.New(SYS_10001)
	}

	return txHashRst, nil
}

func (w *WalletServiceImpl) GetDepth(query DepthQuery) (res Depth, err error) {

	defaultDepthLength := 50

	mkt := strings.ToUpper(query.Market)
	delegateAddress := query.DelegateAddress

	if mkt == "" || !common.IsHexAddress(delegateAddress) {
		err = errors.New("market and correct contract address must be applied")
		return
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
	depth := Depth{DelegateAddress: delegateAddress, Market: mkt, Depth: askBid}

	//(TODO) 考虑到需要聚合的情况，所以每次取2倍的数据，先聚合完了再cut, 不是完美方案，后续再优化
	asks, askErr := w.orderManager.GetOrderBook(
		common.HexToAddress(delegateAddress),
		util.AllTokens[a].Protocol,
		util.AllTokens[b].Protocol, defaultDepthLength*2)

	if askErr != nil {
		err = errors.New("get depth error , please refresh again")
		return
	}

	depth.Depth.Sell = w.calculateDepth(asks, defaultDepthLength, true, util.AllTokens[a].Decimals, util.AllTokens[b].Decimals)

	bids, bidErr := w.orderManager.GetOrderBook(
		common.HexToAddress(delegateAddress),
		util.AllTokens[b].Protocol,
		util.AllTokens[a].Protocol, defaultDepthLength*2)

	if bidErr != nil {
		err = errors.New("get depth error , please refresh again")
		return
	}

	depth.Depth.Buy = w.calculateDepth(bids, defaultDepthLength, false, util.AllTokens[b].Decimals, util.AllTokens[a].Decimals)

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
		//if util.IsBuy(fill.TokenB) {
		//	fill.Side = "buy"
		//} else {
		//	fill.Side = "sell"
		//}
		fill.TokenS = util.AddressToAlias(fill.TokenS)
		fill.TokenB = util.AddressToAlias(fill.TokenB)

		result.Data = append(result.Data, fill)
	}
	return result, nil
}

func (w *WalletServiceImpl) GetLatestFills(query FillQuery) ([]LatestFill, error) {

	rst := make([]LatestFill, 0)
	fillQuery, _, _ := fillQueryToMap(query)
	res, err := w.orderManager.GetLatestFills(fillQuery, 40)

	if err != nil {
		return rst, err
	}

	for _, f := range res {
		lf, err := toLatestFill(f)
		if err == nil && lf.Price > 0 && lf.Amount > 0 {
			rst = append(rst, lf)
		}
	}
	return rst, nil
}

func (w *WalletServiceImpl) GetTicker() (res []market.Ticker, err error) {
	return w.trendManager.GetTicker()
}

func (w *WalletServiceImpl) GetTrend(query TrendQuery) (res []market.Trend, err error) {
	res, err = w.trendManager.GetTrends(query.Market, query.Interval)
	sort.Slice(res, func(i, j int) bool {
		return res[i].Start > res[j].Start
	})
	return
}

func (w *WalletServiceImpl) GetRingMined(query RingMinedQuery) (res dao.PageResult, err error) {
	return w.orderManager.RingMinedPageQuery(ringMinedQueryToMap(query))
}

func (w *WalletServiceImpl) GetRingMinedDetail(query RingMinedQuery) (res RingMinedDetail, err error) {

	if query.RingIndex == "" {
		return res, errors.New("ringIndex must be supplied")
	}

	rings, err := w.orderManager.RingMinedPageQuery(ringMinedQueryToMap(query))

	// todo:如果ringhash重复暂时先取第一条
	if err != nil || rings.Total > 1 {
		log.Errorf("query ring error, %s, %d", err.Error(), rings.Total)
		return res, errors.New("query ring error occurs")
	}

	if rings.Total == 0 {
		return res, errors.New("no ring found by hash")
	}

	ring := rings.Data[0].(dao.RingMinedEvent)
	fills, err := w.orderManager.FindFillsByRingHash(common.HexToHash(ring.RingHash))
	if err != nil {
		return res, err
	}
	return fillDetail(ring, fills)
}

func (w *WalletServiceImpl) GetBalance(balanceQuery CommonTokenRequest) (res AccountJson, err error) {
	if !common.IsHexAddress(balanceQuery.Owner) {
		return res, errors.New("owner can't be null")
	}
	if !common.IsHexAddress(balanceQuery.DelegateAddress) {
		return res, errors.New("delegate must be address")
	}
	owner := common.HexToAddress(balanceQuery.Owner)
	balances, _ := w.accountManager.GetBalanceWithSymbolResult(owner)
	allowances, _ := w.accountManager.GetAllowanceWithSymbolResult(owner, common.HexToAddress(balanceQuery.DelegateAddress))

	res = AccountJson{}
	res.DelegateAddress = balanceQuery.DelegateAddress
	res.Address = balanceQuery.Owner
	res.Tokens = []Token{}
	for symbol, balance := range balances {
		token := Token{}
		token.Token = symbol

		if allowance, exists := allowances[symbol]; exists {
			token.Allowance = allowance.String()
		} else {
			token.Allowance = "0"
		}
		token.Balance = balance.String()
		res.Tokens = append(res.Tokens, token)
	}

	return
}

func (w *WalletServiceImpl) GetCutoff(query CutoffRequest) (result int64, err error) {
	cutoff, err := ethaccessor.GetCutoff(common.HexToAddress(query.DelegateAddress), common.HexToAddress(query.Address), query.BlockNumber)
	if err != nil {
		return 0, err
	}
	return cutoff.Int64(), nil
}

func (w *WalletServiceImpl) GetEstimatedAllocatedAllowance(query EstimatedAllocatedAllowanceQuery) (frozenAmount string, err error) {
	statusSet := make([]types.OrderStatus, 0)
	statusSet = append(statusSet, types.ORDER_NEW)
	statusSet = append(statusSet, types.ORDER_PARTIAL)
	statusSet = append(statusSet, types.ORDER_PENDING_FOR_P2P)

	token := query.Token
	owner := query.Owner

	tokenAddress := util.AliasToAddress(token)
	if tokenAddress.Hex() == "" {
		return "", errors.New("unsupported token alias " + token)
	}
	amount, err := w.orderManager.GetFrozenAmount(common.HexToAddress(owner), tokenAddress, statusSet, common.HexToAddress(query.DelegateAddress))
	if err != nil {
		return "", err
	}

	return types.BigintToHex(amount), err
}

func (w *WalletServiceImpl) GetFrozenLRCFee(query SingleOwner) (frozenAmount string, err error) {
	statusSet := make([]types.OrderStatus, 0)
	statusSet = append(statusSet, types.ORDER_NEW)
	statusSet = append(statusSet, types.ORDER_PARTIAL)
	statusSet = append(statusSet, types.ORDER_PENDING_FOR_P2P)

	owner := query.Owner

	allLrcFee, err := w.orderManager.GetFrozenLRCFee(common.HexToAddress(owner), statusSet)
	if err != nil {
		return "", err
	}

	return types.BigintToHex(allLrcFee), err
}

func (w *WalletServiceImpl) GetLooprSupportedMarket() (markets []string, err error) {
	return w.GetSupportedMarket()
}

func (w *WalletServiceImpl) GetLooprSupportedTokens() (markets []types.Token, err error) {
	return w.GetSupportedTokens()
}

func (w *WalletServiceImpl) GetContracts() (contracts map[string][]string, err error) {
	rst := make(map[string][]string)
	for k, protocol := range ethaccessor.ProtocolAddresses() {
		lprP := k.Hex()
		lprDP := protocol.DelegateAddress.Hex()

		v, ok := rst[lprDP]
		if ok {
			v = append(v, lprP)
			rst[lprDP] = v
		} else {
			lprPS := make([]string, 0)
			lprPS = append(lprPS, lprP)
			rst[lprDP] = lprPS
		}
	}
	return rst, nil
}

func (w *WalletServiceImpl) GetSupportedMarket() (markets []string, err error) {
	return util.AllMarkets, err
}

func (w *WalletServiceImpl) GetSupportedTokens() (markets []types.Token, err error) {
	markets = make([]types.Token, 0)
	for _, v := range util.AllTokens {
		markets = append(markets, v)
	}
	return markets, err
}

func (w *WalletServiceImpl) GetTransactions(query TransactionQuery) (PageResult, error) {
	var (
		rst PageResult
		// should be make
		txs           = make([]txtyp.TransactionJsonResult, 0)
		limit, offset int
		err           error
	)

	rst.Data = make([]interface{}, 0)
	rst.PageIndex, rst.PageSize, limit, offset = pagination(query.PageIndex, query.PageSize)
	rst.Total, err = txmanager.GetAllTransactionCount(query.Owner, query.Symbol, query.Status, query.TxType)
	if err != nil {
		return rst, err
	}
	txs, err = txmanager.GetAllTransactions(query.Owner, query.Symbol, query.Status, query.TxType, limit, offset)
	for _, v := range txs {
		rst.Data = append(rst.Data, v)
	}

	if err != nil {
		return rst, err
	}

	return rst, nil
}

func pagination(pageIndex, pageSize int) (int, int, int, int) {
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	limit := pageSize
	offset := (pageIndex - 1) * pageSize

	return pageIndex, pageSize, limit, offset
}

func (w *WalletServiceImpl) GetTransactionsByHash(query TransactionQuery) (result []txtyp.TransactionJsonResult, err error) {
	return txmanager.GetTransactionsByHash(query.Owner, query.TrxHashes)
}

func (w *WalletServiceImpl) GetPendingTransactions(query SingleOwner) (result []txtyp.TransactionJsonResult, err error) {
	return txmanager.GetPendingTransactions(query.Owner)
}

func (w *WalletServiceImpl) GetPendingRawTxByHash(query TransactionQuery) (result TxNotify, err error) {
	if len(query.ThxHash) == 0 {
		return result, errors.New("tx hash can't be nil")
	}

	txBytes, err := cache.Get(PendingTxPreKey + strings.ToUpper(query.ThxHash))
	if err != nil {
		return result, err
	}

	var tx TxNotify

	err = json.Unmarshal(txBytes, &tx)
	if err != nil {
		return result, err
	}

	return tx, nil

}

func (w *WalletServiceImpl) GetEstimateGasPrice() (result string, err error) {
	return types.BigintToHex(ethaccessor.EstimateGasPrice(nil, nil)), nil
}

func convertFromQuery(orderQuery *OrderQuery) (query map[string]interface{}, statusList []types.OrderStatus, pageIndex int, pageSize int) {

	query = make(map[string]interface{})
	statusList = convertStatus(orderQuery.Status)
	if orderQuery.Owner != "" {
		query["owner"] = orderQuery.Owner
	}
	if common.IsHexAddress(orderQuery.DelegateAddress) {
		query["delegate_address"] = orderQuery.DelegateAddress
	}

	if orderQuery.Market != "" {
		query["market"] = orderQuery.Market
	}

	if orderQuery.Side != "" {
		query["side"] = orderQuery.Side
	}

	if orderQuery.OrderHash != "" {
		query["order_hash"] = orderQuery.OrderHash
	}

	if orderQuery.OrderType == types.ORDER_TYPE_MARKET || orderQuery.OrderType == types.ORDER_TYPE_P2P {
		query["order_type"] = orderQuery.OrderType
	} else {
		query["order_type"] = types.ORDER_TYPE_MARKET
	}

	pageIndex = orderQuery.PageIndex
	pageSize = orderQuery.PageSize
	return

}

func convertStatus(s string) []types.OrderStatus {
	switch s {
	case "ORDER_OPENED":
		return []types.OrderStatus{types.ORDER_NEW, types.ORDER_PARTIAL}
	case "ORDER_NEW":
		return []types.OrderStatus{types.ORDER_NEW}
	case "ORDER_PARTIAL":
		return []types.OrderStatus{types.ORDER_PARTIAL}
	case "ORDER_FINISHED":
		return []types.OrderStatus{types.ORDER_FINISHED}
	case "ORDER_CANCELLED":
		return []types.OrderStatus{types.ORDER_CANCEL, types.ORDER_CUTOFF}
	case "ORDER_CUTOFF":
		return []types.OrderStatus{types.ORDER_CUTOFF}
	case "ORDER_EXPIRE":
		return []types.OrderStatus{types.ORDER_EXPIRE}
	}
	return []types.OrderStatus{}
}

func getStringStatus(order types.OrderState) string {
	s := order.Status

	if order.IsExpired() {
		return "ORDER_EXPIRE"
	}

	if order.RawOrder.OrderType == types.ORDER_TYPE_P2P && ordermanager.IsP2PMakerLocked(order.RawOrder.Hash.Hex()) {
		return "ORDER_PENDING"
	}

	switch s {
	case types.ORDER_NEW:
		return "ORDER_OPENED"
	case types.ORDER_PARTIAL:
		return "ORDER_OPENED"
	case types.ORDER_FINISHED:
		return "ORDER_FINISHED"
	case types.ORDER_CANCEL:
		return "ORDER_CANCELLED"
	case types.ORDER_CUTOFF:
		return "ORDER_CUTOFF"
	case types.ORDER_PENDING:
		return "ORDER_PENDING"
	case types.ORDER_EXPIRE:
		return "ORDER_EXPIRE"
	}
	return "ORDER_UNKNOWN"
}

func (w *WalletServiceImpl) calculateDepth(states []types.OrderState, length int, isAsk bool, tokenSDecimal, tokenBDecimal *big.Int) [][]string {

	if len(states) == 0 {
		return [][]string{}
	}

	depth := make([][]string, 0)
	for i := range depth {
		depth[i] = make([]string, 0)
	}

	depthMap := make(map[string]DepthElement)

	for _, s := range states {

		//log.Infof("handle order ....... %s", s.RawOrder.Hash.Hex())

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

		minAmountB := amountB
		minAmountS := amountS
		var err error

		minAmountS, err = w.getAvailableMinAmount(amountS, s.RawOrder.Owner, s.RawOrder.TokenS, s.RawOrder.DelegateAddress, tokenSDecimal)
		if err != nil {
			//log.Debug(err.Error())
			continue
		}

		sellPrice := new(big.Rat).SetFrac(s.RawOrder.AmountS, s.RawOrder.AmountB)
		buyPrice := new(big.Rat).SetFrac(s.RawOrder.AmountB, s.RawOrder.AmountS)
		if s.RawOrder.BuyNoMoreThanAmountB {
			//log.Info("order BuyNoMoreThanAmountB is true")
			//log.Infof("amount s is %s", minAmountS)
			//log.Infof("amount b is %s", minAmountB)
			//log.Infof("sellprice is %s", sellPrice.String())
			limitedAmountS := new(big.Rat).Mul(minAmountB, sellPrice)
			//log.Infof("limit amount s is %s", limitedAmountS)
			if limitedAmountS.Cmp(minAmountS) < 0 {
				minAmountS = limitedAmountS
			}

			minAmountB = minAmountB.Mul(minAmountS, buyPrice)
		} else {
			//log.Infof("amount s is %s", minAmountS)
			//log.Infof("amount b is %s", minAmountB)
			//log.Infof("buyprice is %s", buyPrice.String())
			limitedAmountB := new(big.Rat).Mul(minAmountS, buyPrice)
			//log.Infof("limit amount b is %s", limitedAmountB)
			if limitedAmountB.Cmp(minAmountB) < 0 {
				minAmountB = limitedAmountB
			}
			minAmountS = minAmountS.Mul(minAmountB, sellPrice)
		}

		if isAsk {
			price = *price.Inv(&price)
			priceFloatStr := price.FloatString(10)
			if v, ok := depthMap[priceFloatStr]; ok {
				amount := v.Amount
				size := v.Size
				amount = amount.Add(amount, minAmountS)
				size = size.Add(size, minAmountB)
				depthMap[priceFloatStr] = DepthElement{Price: v.Price, Amount: amount, Size: size}
			} else {
				depthMap[priceFloatStr] = DepthElement{Price: priceFloatStr, Amount: minAmountS, Size: minAmountB}
			}
		} else {
			priceFloatStr := price.FloatString(10)
			if v, ok := depthMap[priceFloatStr]; ok {
				amount := v.Amount
				size := v.Size
				amount = amount.Add(amount, minAmountB)
				size = size.Add(size, minAmountS)
				depthMap[priceFloatStr] = DepthElement{Price: v.Price, Amount: amount, Size: size}
			} else {
				depthMap[priceFloatStr] = DepthElement{Price: priceFloatStr, Amount: minAmountB, Size: minAmountS}
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
		//if isAsk {
		//	return cmpA < cmpB
		//} else {
		//	return cmpA > cmpB
		//}
		return cmpA > cmpB

	})

	if length < len(depth) {
		if isAsk {
			return depth[len(depth)-length-1:]
		} else {
			return depth[:length]
		}
	}
	return depth
}

func (w *WalletServiceImpl) getAvailableMinAmount(depthAmount *big.Rat, owner, token, spender common.Address, decimal *big.Int) (amount *big.Rat, err error) {

	amount = depthAmount

	balance, allowance, err := w.accountManager.GetBalanceAndAllowance(owner, token, spender)
	if err != nil {
		return
	}

	balanceRat := new(big.Rat).SetFrac(balance, decimal)
	allowanceRat := new(big.Rat).SetFrac(allowance, decimal)

	//log.Info(amount.String())
	//log.Info(balanceRat.String())
	//log.Info(allowanceRat.String())

	if amount.Cmp(balanceRat) > 0 {
		amount = balanceRat
	}

	if amount.Cmp(allowanceRat) > 0 {
		amount = allowanceRat
	}

	if amount.Cmp(new(big.Rat).SetFloat64(1e-8)) < 0 {
		return nil, errors.New("amount is zero, skipped")
	}

	//log.Infof("get reuslt amount is  %s", amount)

	return
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
	if common.IsHexAddress(q.DelegateAddress) {
		rst["delegate_address"] = q.DelegateAddress
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

	if q.Side != "" {
		rst["side"] = q.Side
	}

	if q.OrderType == types.ORDER_TYPE_MARKET || q.OrderType == types.ORDER_TYPE_P2P {
		rst["order_type"] = q.OrderType
	} else {
		rst["order_type"] = types.ORDER_TYPE_MARKET
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
	if common.IsHexAddress(q.DelegateAddress) {
		rst["delegate_address"] = q.DelegateAddress
	}
	if common.IsHexAddress(q.ProtocolAddress) {
		rst["contract_address"] = q.ProtocolAddress
	}
	if q.RingIndex != "" {
		rst["ring_index"] = types.HexToBigint(q.RingIndex).String()
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
	rst.Status = getStringStatus(src)
	rawOrder := RawOrderJsonResult{}
	rawOrder.Protocol = src.RawOrder.Protocol.Hex()
	rawOrder.DelegateAddress = src.RawOrder.DelegateAddress.Hex()
	rawOrder.Owner = src.RawOrder.Owner.Hex()
	rawOrder.Hash = src.RawOrder.Hash.Hex()
	rawOrder.TokenS = util.AddressToAlias(src.RawOrder.TokenS.String())
	rawOrder.TokenB = util.AddressToAlias(src.RawOrder.TokenB.String())
	rawOrder.AmountS = types.BigintToHex(src.RawOrder.AmountS)
	rawOrder.AmountB = types.BigintToHex(src.RawOrder.AmountB)
	rawOrder.ValidSince = types.BigintToHex(src.RawOrder.ValidSince)
	rawOrder.ValidUntil = types.BigintToHex(src.RawOrder.ValidUntil)
	rawOrder.LrcFee = types.BigintToHex(src.RawOrder.LrcFee)
	rawOrder.BuyNoMoreThanAmountB = src.RawOrder.BuyNoMoreThanAmountB
	rawOrder.MarginSplitPercentage = types.BigintToHex(big.NewInt(int64(src.RawOrder.MarginSplitPercentage)))
	rawOrder.V = types.BigintToHex(big.NewInt(int64(src.RawOrder.V)))
	rawOrder.R = src.RawOrder.R.Hex()
	rawOrder.S = src.RawOrder.S.Hex()
	rawOrder.WalletAddress = src.RawOrder.WalletAddress.Hex()
	rawOrder.AuthAddr = src.RawOrder.AuthAddr.Hex()
	rawOrder.Market = src.RawOrder.Market
	auth, _ := src.RawOrder.AuthPrivateKey.MarshalText()
	rawOrder.AuthPrivateKey = string(auth)
	rawOrder.CreateTime = src.RawOrder.CreateTime
	rawOrder.Side = src.RawOrder.Side
	rawOrder.OrderType = src.RawOrder.OrderType
	rst.RawOrder = rawOrder
	return rst
}

func txStatusToUint8(txType string) int {
	switch txType {
	case "pending":
		return 1
	case "success":
		return 2
	case "failed":
		return 3
	default:
		return -1
	}
}

func txTypeToUint8(status string) int {
	switch status {
	case "approve":
		return 1
	case "send":
		return 2
	case "receive":
		return 3
	case "sell":
		return 4
	case "buy":
		return 5
	case "convert":
		return 7
	case "cancel_order":
		return 8
	case "cutoff":
		return 9
	case "cutoff_trading_pair":
		return 10
	default:
		return -1
	}
}

//func toTxJsonResult(tx types.Transaction) txmanager.TransactionJsonResult {
//	dst := txmanager.TransactionJsonResult{}
//	dst.Protocol = tx.Protocol
//	dst.Owner = tx.Owner
//	dst.From = tx.From
//	dst.To = tx.To
//	dst.TxHash = tx.TxHash
//
//	if tx.Type == types.TX_TYPE_CUTOFF_PAIR {
//		ctx, err := tx.GetCutoffPairContent()
//		if err == nil && ctx != nil {
//			mkt, err := util.WrapMarketByAddress(ctx.Token1.Hex(), ctx.Token2.Hex())
//			if err == nil {
//				dst.Content = txmanager.TransactionContent{Market: mkt}
//			}
//		}
//	} else if tx.Type == types.TX_TYPE_CANCEL_ORDER {
//		ctx, err := tx.GetCancelOrderHash()
//		if err == nil && ctx != "" {
//			dst.Content = txmanager.TransactionContent{OrderHash: ctx}
//		}
//	}
//
//	dst.BlockNumber = tx.BlockNumber.Int64()
//	dst.LogIndex = tx.TxLogIndex
//	if tx.Value == nil {
//		dst.Value = "0"
//	} else {
//		dst.Value = tx.Value.String()
//	}
//	dst.Type = tx.TypeStr()
//	dst.Status = tx.StatusStr()
//	dst.CreateTime = tx.CreateTime
//	dst.UpdateTime = tx.UpdateTime
//	dst.Symbol = tx.Symbol
//	dst.Nonce = tx.TxInfo.Nonce.String()
//	return dst
//}

func fillDetail(ring dao.RingMinedEvent, fills []dao.FillEvent) (rst RingMinedDetail, err error) {
	rst = RingMinedDetail{Fills: fills}
	ringInfo := RingMinedInfo{}
	ringInfo.ID = ring.ID
	ringInfo.RingHash = ring.RingHash
	ringInfo.BlockNumber = ring.BlockNumber
	ringInfo.Protocol = ring.Protocol
	ringInfo.DelegateAddress = ring.DelegateAddress
	ringInfo.TxHash = ring.TxHash
	ringInfo.Time = ring.Time
	ringInfo.RingIndex = ring.RingIndex
	ringInfo.Miner = ring.Miner
	ringInfo.FeeRecipient = ring.FeeRecipient
	ringInfo.IsRinghashReserved = ring.IsRinghashReserved
	ringInfo.TradeAmount = ring.TradeAmount
	ringInfo.TotalLrcFee = ring.TotalLrcFee
	ringInfo.TotalSplitFee = make(map[string]*big.Int)

	for _, f := range fills {
		if len(f.SplitS) > 0 && f.SplitS != "0" {
			symbol := util.AddressToAlias(f.TokenS)
			if len(symbol) > 0 {
				splitS, _ := new(big.Int).SetString(f.SplitS, 0)
				totalSplitS, ok := ringInfo.TotalSplitFee[symbol]
				if ok {
					ringInfo.TotalSplitFee[symbol] = totalSplitS.Add(splitS, totalSplitS)
				} else {
					ringInfo.TotalSplitFee[symbol] = splitS
				}
			}
		}
		if len(f.SplitB) > 0 && f.SplitB != "0" {
			symbol := util.AddressToAlias(f.TokenB)
			if len(symbol) > 0 {
				splitB, _ := new(big.Int).SetString(f.SplitB, 0)
				totalSplitB, ok := ringInfo.TotalSplitFee[symbol]
				if ok {
					ringInfo.TotalSplitFee[symbol] = totalSplitB.Add(splitB, totalSplitB)
				} else {
					ringInfo.TotalSplitFee[symbol] = splitB
				}
			}
		}
	}

	rst.RingInfo = ringInfo
	return rst, nil
}

func toLatestFill(f dao.FillEvent) (latestFill LatestFill, err error) {
	rst := LatestFill{CreateTime: f.CreateTime}
	price := util.CalculatePrice(f.AmountS, f.AmountB, f.TokenS, f.TokenB)
	rst.Price, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", price), 64)
	rst.Side = f.Side
	rst.RingHash = f.RingHash
	rst.LrcFee = f.LrcFee
	rst.SplitS = f.SplitS
	rst.SplitB = f.SplitB
	var amount float64
	if util.GetSide(f.TokenS, f.TokenB) == util.SideBuy {
		amountB, _ := new(big.Int).SetString(f.AmountB, 0)
		tokenB, ok := util.AllTokens[util.AddressToAlias(f.TokenB)]
		if !ok {
			return latestFill, err
		}
		ratAmount := new(big.Rat).SetFrac(amountB, tokenB.Decimals)
		amount, _ = ratAmount.Float64()
		rst.Amount, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", amount), 64)
	} else {
		amountS, _ := new(big.Int).SetString(f.AmountS, 0)
		tokenS, ok := util.AllTokens[util.AddressToAlias(f.TokenS)]
		if !ok {
			return latestFill, err
		}
		ratAmount := new(big.Rat).SetFrac(amountS, tokenS.Decimals)
		amount, _ = ratAmount.Float64()
		rst.Amount, _ = strconv.ParseFloat(fmt.Sprintf("%0.8f", amount), 64)
	}
	return rst, nil
}

func saveMatchedRelation(takerOrderHash, makerOrderHash, ringTxHash string) (err error) {
	return nil
}
