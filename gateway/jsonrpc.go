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
	"context"
	"errors"
	"fmt"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/types"
	"github.com/gorilla/mux"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"net"
	"net/http"
	"net/rpc"
	"github.com/Loopring/relay/market"
	"github.com/Loopring/relay/ordermanager"
	"math/big"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/rpc/v2/json2"
	rpc2 "github.com/gorilla/rpc/v2"
)

func (*JsonrpcServiceImpl) Ping(val [1]string, res *string) error {
	*res = "pong for first connect, meaning server is OK"
	return nil
}

type PageResult struct {
	Data      []interface{}
	PageIndex int
	PageSize  int
	Total     int
}

type Depth struct {
	contractVersion string
	market string
	Depth AskBid
}

type AskBid struct {
	Buy [][]string
	Sell [][]string
}

type CommonTokenRequest struct {
	contractVersion string
	owner string
}

type OrderQuery struct {
	Status int
	PageIndex int
	PageSize  int
	ContractVersion string
	Owner string

}

var RemoteAddrContextKey = "RemoteAddr"

type JsonrpcService interface {
	Start(port string)
	Stop()
}

type JsonrpcServiceImpl struct {
	port         string
	trendManager market.TrendManager
	orderManager ordermanager.OrderManager
	accountManager market.AccountManager
}

func NewJsonrpcService(port string, trendManager market.TrendManager, orderManager ordermanager.OrderManager, accountManager market.AccountManager) *JsonrpcServiceImpl {
	l := &JsonrpcServiceImpl{}
	l.port = port
	l.trendManager = trendManager
	l.orderManager = orderManager
	l.accountManager = accountManager
	return l
}

func (j *JsonrpcServiceImpl) Start2() {
	// Server export an object of type JsonrpcServiceImpl.
	rpc.Register(&JsonrpcServiceImpl{})

	// Server provide a TCP transport.
	lnTCP, err := net.Listen("tcp", "127.0.0.1:8886")
	if err != nil {
		panic(err)
	}
	defer lnTCP.Close()
	go func() {
		for {
			conn, err := lnTCP.Accept()
			if err != nil {
				return
			}
			ctx := context.WithValue(context.Background(), RemoteAddrContextKey, conn.RemoteAddr())
			go jsonrpc2.ServeConnContext(ctx, conn)
		}
	}()

	// Server provide a HTTP transport on /rpc endpoint.
	http.Handle("/rpc", jsonrpc2.HTTPHandler(nil))
	lnHTTP, err := net.Listen("tcp", ":"+j.port)
	if err != nil {
		panic(err)
	}
	defer lnHTTP.Close()
	go http.Serve(lnHTTP, nil)

	// Client use HTTP transport.
	fmt.Println(lnHTTP.Addr())
	clientHTTP := jsonrpc2.NewHTTPClient("http://" + lnHTTP.Addr().String() + "/rpc")
	defer clientHTTP.Close()

	var pong string
	err = clientHTTP.Call("JsonrpcServiceImpl.Ping", []string{"ping"}, &pong)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("ping result is %s\n", pong)
	}

}

func (j *JsonrpcServiceImpl) Start() {

	s := rpc2.NewServer()
	s.RegisterCodec(json2.NewCodec(), "application/json")
	s.RegisterCodec(json2.NewCodec(), "application/json;charset=UTF-8")
	jsonrpc := new(JsonrpcServiceImpl)
	s.RegisterService(jsonrpc, "jsonrpc")
	r := mux.NewRouter()
	r.Handle("/rpc", s)
	http.ListenAndServe(":"+j.port, r)
}

func (j *JsonrpcServiceImpl) SubmitOrder(r *http.Request, order *types.Order, res *string) error {
	HandleOrder(order)
	*res = "SUBMIT_SUCCESS"
	return nil
}

func (j *JsonrpcServiceImpl) getOrders(r *http.Request, query map[string]interface{}, res *dao.PageResult) error {

	orderQuery, pi, ps, err := convertFromMap(query)
	if err != nil {
		return err
	}

	result, queryErr := j.orderManager.GetOrders(&orderQuery, pi, ps)
	res = &result
	return queryErr
}

func (j *JsonrpcServiceImpl) getDepth(r *http.Request, query map[string]interface{}, res *Depth) error {

	mkt := query["market"].(string)
	protocol := query["contractVersion"].(string)
	length := query["length"].(int)

	if mkt == "" || protocol == "" || market.ContractVersionConfig[protocol] == "" {
		return errors.New("market and correct contract version must be applied")
	}

	if length <= 0 || length > 20 {
		length = 20
	}

	a, b := market.UnWrap(mkt)
	if market.SupportTokens[a] == "" || market.SupportMarket[b] == "" {
		return errors.New("unsupported market type")
	}

	empty := make([][]string, 0)
	for i := range empty {
		empty[i] = make([]string, 0)
	}
	askBid := AskBid{Buy:empty, Sell:empty}
	depth := Depth{contractVersion:market.ContractVersionConfig[protocol], market:mkt, Depth:askBid}

	//(TODO) 考虑到需要聚合的情况，所以每次取2倍的数据，先聚合完了再cut, 不是完美方案，后续再优化
	asks, askErr := j.orderManager.GetOrderBook(
		common.StringToAddress(market.ContractVersionConfig[protocol]),
		common.StringToAddress(a),
		common.StringToAddress(b), length * 2)

	if askErr != nil {
		return errors.New("get depth error , please refresh again")
	}

	depth.Depth.Sell = calculateDepth(asks, length)

	bids, bidErr := j.orderManager.GetOrderBook(
		common.StringToAddress(market.ContractVersionConfig[protocol]),
		common.StringToAddress(b),
		common.StringToAddress(a), length * 2)

	if bidErr != nil {
		return errors.New("get depth error , please refresh again")
	}

	depth.Depth.Buy = calculateDepth(bids, length)

	return nil
}

func (j *JsonrpcServiceImpl) getFills(r *http.Request, market string, res *map[string]int) error {
	return nil
}

func (j *JsonrpcServiceImpl) getTicker(r *http.Request, market string, res *[]market.Ticker) error {
	tickers, err := j.trendManager.GetTicker()
	res = &tickers
	return err
}

func (j *JsonrpcServiceImpl) getTrend(r *http.Request, market string, res *[]market.Trend) error {
	trends, err := j.trendManager.GetTrends(market)
	res = &trends
	return err
}

func (*JsonrpcServiceImpl) getRingMined(r *http.Request, market string, res *map[string]int) error {
	// not support now
	return nil
}

func (j *JsonrpcServiceImpl) getBalance(r *http.Request, balanceQuery CommonTokenRequest, res *market.AccountJson) error {
	account := j.accountManager.GetBalance(balanceQuery.contractVersion, balanceQuery.owner)
	accountJson := account.ToJsonObject(balanceQuery.contractVersion)
	res = &accountJson
	return nil
}

func convertFromMap(src map[string]interface{}) (query dao.Order, pageIndex int, pageSize int, err error) {

	for k, v := range src {
		switch k {
		//TODO(xiaolu) change status to string not uint8
		case "status":
			query.Status = v.(uint8)
		case "pageIndex":
			pageIndex = v.(int)
		case "pageSize":
			pageSize = v.(int)
		case "owner":
			query.Owner = v.(string)
		case "contractVersion":
			query.Protocol = v.(string)
		default:
			err = errors.New("unsupported query found " + k)
			return
		}
	}

	return

}

func FromMap(src map[string]interface{}) (query dao.Order, pageIndex int, pageSize int, err error) {

	for k, v := range src {
		switch k {
		//TODO(xiaolu) change status to string not uint8
		case "status":
			query.Status = v.(uint8)
		case "pageIndex":
			pageIndex = v.(int)
		case "pageSize":
			pageSize = v.(int)
		case "owner":
			query.Owner = v.(string)
		case "contractVersion":
			query.Protocol = v.(string)
		default:
			err = errors.New("unsupported query found " + k)
			return
		}
	}

	return

}

type Args struct {
	A, B int
}

type Arith int

type Result int

func (j *JsonrpcServiceImpl) Multiply(r *http.Request, args *Args, result *int) error {
	fmt.Printf("Multiplying %d with %d\n", args.A, args.B)

	*result = args.A * args.B
	return nil
}

func calculateDepth(states []types.OrderState, length int) [][]string {

	if len(states) == 0 {
		return [][]string{}
	}

	depth := make([][]string, 0)
	for i := range depth {
		depth[i] = make([]string, 0)
	}

	var tempSumAmountS, tempSumAmountB big.Int
	var lastPrice big.Rat

	for i,s := range states {

		if i == 0 {
			lastPrice = *s.RawOrder.Price
			tempSumAmountS = *s.RawOrder.AmountS
			tempSumAmountB = *s.RawOrder.AmountB
		} else {
			if lastPrice.Cmp(s.RawOrder.Price) != 0 {
				depth = append(depth, []string{tempSumAmountS.String(), tempSumAmountB.String()})
				tempSumAmountS.Set(big.NewInt(0))
				tempSumAmountB.Set(big.NewInt(0))
				lastPrice = *s.RawOrder.Price
			} else {
				tempSumAmountS.Add(&tempSumAmountS, s.RawOrder.AmountS)
				tempSumAmountB.Add(&tempSumAmountB, s.RawOrder.AmountB)
			}
		}
	}

	return depth[:length]
}
