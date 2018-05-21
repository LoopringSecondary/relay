package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/eventemiter"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	txtyp "github.com/Loopring/relay/txmanager/types"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/googollee/go-socket.io"
	"github.com/robfig/cron"
	"gopkg.in/googollee/go-engine.io.v1"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"
)

type BusinessType int

const (
	EventPostfixReq         = "_req"
	EventPostfixRes         = "_res"
	EventPostfixEnd         = "_end"
	DefaultCronSpec3Second  = "0/3 * * * * *"
	DefaultCronSpec5Second  = "0/5 * * * * *"
	DefaultCronSpec10Second = "0/10 * * * * *"
	DefaultCronSpec5Minute  = "0 */5 * * * *"
)

const (
	emitTypeByEvent = 1
	emitTypeByCron  = 2
)

type Server struct {
	socketio.Server
}

type SocketIOJsonResp struct {
	Error string      `json:"error"`
	Code  string      `json:"code"`
	Data  interface{} `json:"data"`
}

func NewServer(s socketio.Server) Server {
	return Server{s}
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	OriginList := r.Header["Origin"]
	Origin := ""
	if len(OriginList) > 0 {
		Origin = OriginList[0]
	}
	w.Header().Add("Access-Control-Allow-Origin", Origin)
	//w.Header().Add("Access-Control-Allow-Origin", "http://localhost:8000")
	w.Header().Add("Access-Control-Allow-Credentials", "true")
	//w.Header().Add("Access-Control-Allow-Origin", "*")
	w.Header().Add("Access-Control-Allow-Headers", "accept, origin, content-type")
	w.Header().Add("Access-Control-Allow-Methods", "PUT,POST,GET,DELETE,OPTIONS")
	//w.Header().Add("Content-Type", "application/json;charset=utf-8")
	s.Server.ServeHTTP(w, r)
}

type InvokeInfo struct {
	MethodName  string
	Query       interface{}
	isBroadcast bool
	emitType    int
	spec        string
}

const (
	eventKeyTickers         = "tickers"
	eventKeyLoopringTickers = "loopringTickers"
	eventKeyTrends          = "trends"
	eventKeyPortfolio       = "portfolio"
	eventKeyMarketCap       = "marketcap"
	eventKeyBalance         = "balance"
	eventKeyTransaction     = "transaction"
	eventKeyPendingTx       = "pendingTx"
	eventKeyDepth           = "depth"
	eventKeyTrades          = "trades"
)

var EventTypeRoute = map[string]InvokeInfo{
	//eventKeyTickers:         {"GetTickers", SingleMarket{}, true, emitTypeByCron, DefaultCronSpec3Second},
	//eventKeyLoopringTickers: {"GetTicker", nil, true, emitTypeByEvent, DefaultCronSpec3Second},
	//eventKeyTrends:          {"GetTrend", TrendQuery{}, true, emitTypeByEvent, DefaultCronSpec3Second},
	//// portfolio has been remove from loopr2
	//// eventKeyPortfolio:       {"GetPortfolio", SingleOwner{}, false, emitTypeByEvent, DefaultCronSpec3Second},
	//eventKeyPortfolio:       {"GetPortfolio", SingleOwner{}, false, emitTypeByCron, DefaultCronSpec3Second},
	//eventKeyMarketCap:       {"GetPriceQuote", PriceQuoteQuery{}, true, emitTypeByCron, DefaultCronSpec5Minute},
	//eventKeyBalance:         {"GetBalance", CommonTokenRequest{}, false, emitTypeByEvent, DefaultCronSpec3Second},
	//eventKeyTransaction:     {"GetTransactions", TransactionQuery{}, false, emitTypeByEvent, DefaultCronSpec3Second},
	//eventKeyPendingTx:       {"GetPendingTransactions", SingleOwner{}, false, emitTypeByEvent, DefaultCronSpec10Second},
	//eventKeyDepth:           {"GetDepth", DepthQuery{}, true, emitTypeByEvent, DefaultCronSpec3Second},
	//eventKeyTrades:          {"GetTrades", FillQuery{}, true, emitTypeByEvent, DefaultCronSpec3Second},
	eventKeyTickers:         {"GetTickers", SingleMarket{}, true, emitTypeByCron, DefaultCronSpec5Second},
	eventKeyLoopringTickers: {"GetTicker", nil, true, emitTypeByEvent, DefaultCronSpec5Second},
	eventKeyTrends:          {"GetTrend", TrendQuery{}, true, emitTypeByEvent, DefaultCronSpec10Second},
	// portfolio has been remove from loopr2
	// eventKeyPortfolio:       {"GetPortfolio", SingleOwner{}, false, emitTypeByEvent, DefaultCronSpec3Second},
	eventKeyPortfolio:   {"GetPortfolio", SingleOwner{}, false, emitTypeByCron, DefaultCronSpec3Second},
	eventKeyMarketCap:   {"GetPriceQuote", PriceQuoteQuery{}, true, emitTypeByCron, DefaultCronSpec5Minute},
	eventKeyBalance:     {"GetBalance", CommonTokenRequest{}, false, emitTypeByEvent, DefaultCronSpec10Second},
	eventKeyTransaction: {"GetTransactions", TransactionQuery{}, false, emitTypeByEvent, DefaultCronSpec10Second},
	eventKeyPendingTx:   {"GetPendingTransactions", SingleOwner{}, false, emitTypeByEvent, DefaultCronSpec10Second},
	eventKeyDepth:       {"GetDepth", DepthQuery{}, true, emitTypeByEvent, DefaultCronSpec10Second},
	eventKeyTrades:      {"GetLatestFills", FillQuery{}, true, emitTypeByEvent, DefaultCronSpec10Second},
}

type SocketIOService interface {
	Start(port string)
	Stop()
}

type SocketIOServiceImpl struct {
	port               string
	walletService      WalletServiceImpl
	connIdMap          *sync.Map
	connBusinessKeyMap map[string]socketio.Conn
	cron               *cron.Cron
}

func NewSocketIOService(port string, walletService WalletServiceImpl) *SocketIOServiceImpl {
	so := &SocketIOServiceImpl{}
	so.port = port
	so.walletService = walletService
	so.connBusinessKeyMap = make(map[string]socketio.Conn)
	so.connIdMap = &sync.Map{}
	so.cron = cron.New()

	// init event watcher
	//loopringTickerWatcher := &eventemitter.Watcher{Concurrent: false, Handle: so.broadcastLoopringTicker}
	//eventemitter.On(eventemitter.LoopringTickerUpdated, loopringTickerWatcher)
	//trendsWatcher := &eventemitter.Watcher{Concurrent: false, Handle: so.broadcastTrends}
	//eventemitter.On(eventemitter.TrendUpdated, trendsWatcher)
	//portfolioWatcher := &eventemitter.Watcher{Concurrent: false, Handle: so.handlePortfolioUpdate}
	//eventemitter.On(eventemitter.PortfolioUpdated, portfolioWatcher)
	//balanceWatcher := &eventemitter.Watcher{Concurrent: false, Handle: so.handleBalanceUpdate}
	//eventemitter.On(eventemitter.BalanceUpdated, balanceWatcher)
	//depthWatcher := &eventemitter.Watcher{Concurrent: false, Handle: so.broadcastDepth}
	//eventemitter.On(eventemitter.DepthUpdated, depthWatcher)
	//transactionWatcher := &eventemitter.Watcher{Concurrent: false, Handle: so.handleTransactionUpdate}
	//eventemitter.On(eventemitter.TransactionEvent, transactionWatcher)
	//pendingTxWatcher := &eventemitter.Watcher{Concurrent: false, Handle: so.handlePendingTransaction}
	//eventemitter.On(eventemitter.TransactionEvent, pendingTxWatcher)
	return so
}

func (so *SocketIOServiceImpl) Start() {
	server, err := socketio.NewServer(&engineio.Options{
		PingInterval: time.Second * 60 * 60,
		PingTimeout:  time.Second * 60 * 60,
	})
	if err != nil {
		log.Fatalf(err.Error())
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		so.connIdMap.Store(s.ID(), s)
		return nil
	})
	server.OnEvent("/", "test", func(s socketio.Conn, msg string) {
		fmt.Println("test:", msg)
		s.Emit("reply", "pong relay msg : "+msg)
		fmt.Println("emit message finished...")
		fmt.Println(s.RemoteAddr())
	})

	for v := range EventTypeRoute {
		aliasOfV := v

		server.OnEvent("/", aliasOfV+EventPostfixReq, func(s socketio.Conn, msg string) {
			fmt.Println("input emit msg is ....." + msg)
			context := make(map[string]string)
			if s != nil && s.Context() != nil {
				context = s.Context().(map[string]string)
			}
			context[aliasOfV] = msg
			s.SetContext(context)
			so.connIdMap.Store(s.ID(), s)
			//log.Infof("[SOCKETIO-EMIT]response emit by key : %s, connId : %s", aliasOfV, s.ID())
			fmt.Println("out emit msg is ....." + aliasOfV)
			so.EmitNowByEventType(aliasOfV, s, msg)
		})

		server.OnEvent("/", aliasOfV+EventPostfixEnd, func(s socketio.Conn, msg string) {
			if s != nil && s.Context() != nil {
				businesses := s.Context().(map[string]string)
				delete(businesses, aliasOfV)
				s.SetContext(businesses)
			}
		})
	}

	for k, events := range EventTypeRoute {
		copyOfK := k
		spec := events.spec

		//if events.emitType != emitTypeByCron {
		//	log.Infof("no cron emit type %d ", events.emitType)
		//	continue
		//}

		switch k {
		case eventKeyTickers:
			so.cron.AddFunc(spec, func() {
				so.broadcastTpTickers(nil)
			})
		case eventKeyLoopringTickers:
			so.cron.AddFunc(spec, func() {
				so.broadcastLoopringTicker(nil)
			})
		case eventKeyDepth:
			so.cron.AddFunc(spec, func() {
				//log.Info("start depth broadcast")
				so.broadcastDepth(nil)
			})
		case eventKeyTrades:
			so.cron.AddFunc(spec, func() {
				//log.Info("start trades broadcast")
				so.broadcastTrades(nil)
			})
		default:
			log.Infof("add cron emit %d ", events.emitType)
			so.cron.AddFunc(spec, func() {
				so.connIdMap.Range(func(key, value interface{}) bool {
					v := value.(socketio.Conn)
					if v.Context() != nil {
						businesses := v.Context().(map[string]string)
						eventContext, ok := businesses[copyOfK]
						if ok {
							//log.Infof("[SOCKETIO-EMIT]cron emit by key : %s, connId : %s", copyOfK, v.ID())
							so.EmitNowByEventType(copyOfK, v, eventContext)
						}
					}
					return true
				})
			})

		}
	}

	//so.cron.AddFunc("0/10 * * * * *", func() {
	//
	//	for _, v := range so.connIdMap {
	//		if v.Context() == nil {
	//			continue
	//		} else {
	//			businesses := v.Context().(map[string]string)
	//			if businesses != nil {
	//				for bk, bv := range businesses {
	//					so.EmitNowByEventType(bk, v, bv)
	//				}
	//			}
	//		}
	//	}
	//})
	so.cron.Start()

	server.OnError("/", func(e error) {
		fmt.Println("meet error:", e)
		infos := strings.Split(e.Error(), "SOCKETFORLOOPRING")
		if len(infos) == 2 {
			so.connIdMap.Delete(infos[0])
		}

	})

	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		s.Close()
		so.connIdMap.Delete(s.ID())
		fmt.Println("closed", msg)
	})
	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", NewServer(*server))
	log.Info("Serving at localhost: " + so.port)
	log.Fatal(http.ListenAndServe(":"+so.port, nil).Error())
	log.Info("finished listen socket io....")

}

func (so *SocketIOServiceImpl) EmitNowByEventType(bk string, v socketio.Conn, bv string) {
	if invokeInfo, ok := EventTypeRoute[bk]; ok {
		so.handleAfterEmit(bk, invokeInfo.Query, invokeInfo.MethodName, v, bv)
	}
}

func (so *SocketIOServiceImpl) handleWith(eventType string, query interface{}, methodName string, ctx string) string {

	results := make([]reflect.Value, 0)
	var err error

	if query == nil {
		results = reflect.ValueOf(&so.walletService).MethodByName(methodName).Call(nil)
	} else {
		queryType := reflect.TypeOf(query)
		queryClone := reflect.New(queryType)
		err = json.Unmarshal([]byte(ctx), queryClone.Interface())
		if err != nil {
			log.Info("unmarshal error " + err.Error())
			errJson, _ := json.Marshal(SocketIOJsonResp{Error: err.Error()})
			return string(errJson[:])

		}
		params := make([]reflect.Value, 1)
		params[0] = queryClone.Elem()
		results = reflect.ValueOf(&so.walletService).MethodByName(methodName).Call(params)
	}

	res := results[0]
	if results[1].Interface() == nil {
		err = nil
	} else {
		err = results[1].Interface().(error)
	}
	if err != nil {
		errJson, _ := json.Marshal(SocketIOJsonResp{Error: err.Error()})
		return string(errJson[:])
	} else {
		rst := SocketIOJsonResp{Data: res.Interface()}
		b, _ := json.Marshal(rst)
		return string(b[:])
	}
}

func (so *SocketIOServiceImpl) handleAfterEmit(eventType string, query interface{}, methodName string, conn socketio.Conn, ctx string) {
	result := so.handleWith(eventType, query, methodName, ctx)
	conn.Emit(eventType+EventPostfixRes, result)
}

func (so *SocketIOServiceImpl) broadcastTpTickers(input eventemitter.EventData) (err error) {

	//log.Infof("[SOCKETIO-RECEIVE-EVENT] tickers input. %s", input)
	mkts, _ := so.walletService.GetSupportedMarket()

	tickerMap := make(map[string]SocketIOJsonResp)

	for _, mkt := range mkts {
		ticker, err := so.walletService.GetTickers(SingleMarket{mkt})
		resp := SocketIOJsonResp{}

		if err != nil {
			resp = SocketIOJsonResp{Error: err.Error()}
		} else {
			resp.Data = ticker
		}
		tickerMap[mkt] = resp
	}

	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			ctx, ok := businesses[eventKeyLoopringTickers]
			if ok {
				var singleMarket SingleMarket
				err = json.Unmarshal([]byte(ctx), &singleMarket)
				if err != nil {
					return true
				}
				tks, ok := tickerMap[strings.ToUpper(singleMarket.Market)]
				if ok {
					v.Emit(eventKeyTickers+EventPostfixRes, tks)
				}
			}
		}
		return true
	})
	return nil
}

func (so *SocketIOServiceImpl) broadcastLoopringTicker(input eventemitter.EventData) (err error) {

	//log.Infof("[SOCKETIO-RECEIVE-EVENT] loopring ticker input. %s", input)

	resp := SocketIOJsonResp{}
	tickers, err := so.walletService.GetTicker()

	if err != nil {
		resp = SocketIOJsonResp{Error: err.Error()}
	} else {
		resp.Data = tickers
	}

	respJson, _ := json.Marshal(resp)

	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			_, ok := businesses[eventKeyLoopringTickers]
			if ok {
				//log.Info("emit loopring ticker info")
				v.Emit(eventKeyLoopringTickers+EventPostfixRes, string(respJson[:]))
			}
		}
		return true
	})
	return nil
}

func (so *SocketIOServiceImpl) broadcastDepth(input eventemitter.EventData) (err error) {

	//log.Infof("[SOCKETIO-RECEIVE-EVENT] loopring depth input. %s", input)

	markets := so.getConnectedMarketForDepth()

	respMap := make(map[string]string, 0)
	for mk := range markets {
		mktAndDelegate := strings.Split(mk, "_")
		delegate := mktAndDelegate[0]
		mkt := mktAndDelegate[1]
		resp := SocketIOJsonResp{}
		depth, err := so.walletService.GetDepth(DepthQuery{delegate, mkt})
		if err == nil {
			resp.Data = depth
		} else {
			resp = SocketIOJsonResp{Error: err.Error()}
		}
		respJson, _ := json.Marshal(resp)
		respMap[mk] = string(respJson[:])
	}

	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			ctx, ok := businesses[eventKeyDepth]
			if ok {
				dQuery := &DepthQuery{}
				err := json.Unmarshal([]byte(ctx), dQuery)
				if err == nil && len(dQuery.DelegateAddress) > 0 && len(dQuery.Market) > 0 {
					depthKey := strings.ToLower(dQuery.DelegateAddress) + "_" + strings.ToLower(dQuery.Market)
					v.Emit(eventKeyDepth+EventPostfixRes, respMap[depthKey])
				}
			}
		}
		return true
	})
	return nil
}

func (so *SocketIOServiceImpl) broadcastTrades(input eventemitter.EventData) (err error) {

	//log.Infof("[SOCKETIO-RECEIVE-EVENT] loopring depth input. %s", input)

	markets := so.getConnectedMarketForFill()

	respMap := make(map[string]string, 0)
	for mk := range markets {
		mktAndDelegate := strings.Split(mk, "_")
		delegate := mktAndDelegate[0]
		mkt := mktAndDelegate[1]
		resp := SocketIOJsonResp{}
		fills, err := so.walletService.GetLatestFills(FillQuery{DelegateAddress: delegate, Market: mkt, Side: util.SideSell})
		if err == nil {
			//log.Infof("fetch fill from wallet %d, %s", len(fills), mkt)
			resp.Data = fills
		} else {
			resp = SocketIOJsonResp{Error: err.Error()}
		}
		respJson, _ := json.Marshal(resp)
		respMap[mk] = string(respJson[:])
	}

	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			ctx, ok := businesses[eventKeyTrades]
			if ok {
				fQuery := &FillQuery{}
				err := json.Unmarshal([]byte(ctx), fQuery)
				if err == nil && len(fQuery.DelegateAddress) > 0 && len(fQuery.Market) > 0 {
					fillKey := strings.ToLower(fQuery.DelegateAddress) + "_" + strings.ToLower(fQuery.Market)
					v.Emit(eventKeyTrades+EventPostfixRes, respMap[fillKey])
				}
			}
		}
		return true
	})
	return nil
}

func (so *SocketIOServiceImpl) getConnectedMarketForDepth() map[string]bool {
	markets := make(map[string]bool, 0)
	count := 0
	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		count++
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			DCtx, ok := businesses[eventKeyDepth]
			if ok {
				dQuery := &DepthQuery{}
				err := json.Unmarshal([]byte(DCtx), dQuery)
				if err == nil && len(dQuery.DelegateAddress) > 0 && len(dQuery.Market) > 0 {
					markets[strings.ToLower(dQuery.DelegateAddress)+"_"+strings.ToLower(dQuery.Market)] = true
				}
			}
		}
		return true
	})
	log.Infof("SOCKETIO current conn number is %d", count)
	return markets
}

func (so *SocketIOServiceImpl) getConnectedMarketForFill() map[string]bool {
	markets := make(map[string]bool, 0)
	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			fCtx, ok := businesses[eventKeyTrades]
			if ok {
				fQuery := &FillQuery{}
				err := json.Unmarshal([]byte(fCtx), fQuery)
				if err == nil && len(fQuery.DelegateAddress) > 0 && len(fQuery.Market) > 0 {
					markets[strings.ToLower(fQuery.DelegateAddress)+"_"+strings.ToLower(fQuery.Market)] = true
				}
			}
		}
		return true
	})
	return markets
}

func (so *SocketIOServiceImpl) broadcastTrends(input eventemitter.EventData) (err error) {

	//log.Infof("[SOCKETIO-RECEIVE-EVENT] trend input. %s", input)

	req := input.(TrendQuery)
	resp := SocketIOJsonResp{}
	trends, err := so.walletService.GetTrend(req)

	if err != nil {
		resp = SocketIOJsonResp{Error: err.Error()}
	} else {
		resp.Data = trends
	}

	respJson, _ := json.Marshal(resp)

	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			ctx, ok := businesses[eventKeyTrends]

			if ok {
				trendQuery := &TrendQuery{}
				err = json.Unmarshal([]byte(ctx), trendQuery)
				if err != nil {
					log.Error("trend query unmarshal error, " + err.Error())
				} else if strings.ToUpper(req.Market) == strings.ToUpper(trendQuery.Market) &&
					strings.ToUpper(req.Interval) == strings.ToUpper(trendQuery.Interval) {
					log.Info("emit trend " + ctx)
					v.Emit(eventKeyTrends+EventPostfixRes, string(respJson[:]))
				}
			}
		}
		return true
	})
	return nil
}

// portfolio has removed from loopr2
func (so *SocketIOServiceImpl) handlePortfolioUpdate(input eventemitter.EventData) (err error) {
	return nil
}

func (so *SocketIOServiceImpl) handleBalanceUpdate(input eventemitter.EventData) (err error) {

	log.Infof("[SOCKETIO-RECEIVE-EVENT] balance input. %s", input)

	req := input.(types.BalanceUpdateEvent)
	if len(req.Owner) == 0 {
		return errors.New("owner can't be nil")
	}

	if common.IsHexAddress(req.DelegateAddress) {
		so.notifyBalanceUpdateByDelegateAddress(req.Owner, req.DelegateAddress)
	} else {
		for k := range ethaccessor.DelegateAddresses() {
			so.notifyBalanceUpdateByDelegateAddress(req.Owner, k.Hex())
		}
	}
	return nil
}

func (so *SocketIOServiceImpl) notifyBalanceUpdateByDelegateAddress(owner, delegateAddress string) (err error) {
	req := CommonTokenRequest{owner, delegateAddress}
	resp := SocketIOJsonResp{}
	balance, err := so.walletService.GetBalance(req)

	if err != nil {
		resp = SocketIOJsonResp{Error: err.Error()}
	} else {
		resp.Data = balance
	}

	respJson, _ := json.Marshal(resp)

	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			_, ok := businesses[eventKeyBalance]
			if ok {
				//log.Info("emit balance info")
				v.Emit(eventKeyBalance+EventPostfixRes, string(respJson[:]))
			}
		}
		return true
	})
	return nil
}

//func (so *SocketIOServiceImpl) broadcastDepth(input eventemitter.EventData) (err error) {
//
//	log.Infof("[SOCKETIO-RECEIVE-EVENT] depth input. %s", input)
//
//	req := input.(types.DepthUpdateEvent)
//	resp := SocketIOJsonResp{}
//	depths, err := so.walletService.GetDepth(DepthQuery{req.DelegateAddress, req.DelegateAddress})
//
//	if err != nil {
//		resp = SocketIOJsonResp{Error: err.Error()}
//	} else {
//		resp.Data = depths
//	}
//
//	respJson, _ := json.Marshal(resp)
//
//	so.connIdMap.Range(func(key, value interface{}) bool {
//		v := value.(socketio.Conn)
//		if v.Context() != nil {
//			businesses := v.Context().(map[string]string)
//			ctx, ok := businesses[eventKeyDepth]
//
//			if ok {
//				depthQuery := &DepthQuery{}
//				err = json.Unmarshal([]byte(ctx), depthQuery)
//				if err != nil {
//					log.Error("depth query unmarshal error, " + err.Error())
//				} else if strings.ToUpper(req.DelegateAddress) == strings.ToUpper(depthQuery.DelegateAddress) &&
//					strings.ToUpper(req.Market) == strings.ToUpper(depthQuery.Market) {
//					log.Info("emit trend " + ctx)
//					v.Emit(eventKeyDepth+EventPostfixRes, string(respJson[:]))
//				}
//			}
//		}
//		return true
//	})
//	return nil
//}

func (so *SocketIOServiceImpl) handleTransactionUpdate(input eventemitter.EventData) (err error) {

	log.Infof("[SOCKETIO-RECEIVE-EVENT] transaction input. %s", input)

	req := input.(*txtyp.TransactionView)
	owner := req.Owner.Hex()
	log.Infof("received owner is %s ", owner)
	fmt.Println(so.connIdMap)
	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			ctx, ok := businesses[eventKeyTransaction]
			log.Infof("cxt contains event key %b", ok)

			if ok {
				txQuery := &TransactionQuery{}
				log.Info("txQuery owner is " + txQuery.Owner)
				err = json.Unmarshal([]byte(ctx), txQuery)
				if err != nil {
					log.Error("tx query unmarshal error, " + err.Error())
				} else if strings.ToUpper(owner) == strings.ToUpper(txQuery.Owner) {
					log.Info("emit trend " + ctx)

					txs, err := so.walletService.GetTransactions(*txQuery)
					resp := SocketIOJsonResp{}

					if err != nil {
						resp = SocketIOJsonResp{Error: err.Error()}
					} else {
						resp.Data = txs
					}
					respJson, _ := json.Marshal(resp)
					v.Emit(eventKeyTransaction+EventPostfixRes, string(respJson[:]))
				}
			}
		}
		return true
	})

	return nil
}

func (so *SocketIOServiceImpl) handlePendingTransaction(input eventemitter.EventData) (err error) {

	log.Infof("[SOCKETIO-RECEIVE-EVENT] transaction input (for pending). %s", input)

	req := input.(*txtyp.TransactionView)
	owner := req.Owner.Hex()
	log.Infof("received owner is %s ", owner)
	fmt.Println(so.connIdMap)
	so.connIdMap.Range(func(key, value interface{}) bool {
		v := value.(socketio.Conn)
		fmt.Println(key)
		fmt.Println(value)
		if v.Context() != nil {
			businesses := v.Context().(map[string]string)
			ctx, ok := businesses[eventKeyPendingTx]
			log.Infof("cxt contains event key %b", ok)

			if ok {
				txQuery := &SingleOwner{}
				err = json.Unmarshal([]byte(ctx), txQuery)
				log.Info("single owner is: " + txQuery.Owner)
				if err != nil {
					log.Error("tx query unmarshal error, " + err.Error())
				} else if strings.ToUpper(owner) == strings.ToUpper(txQuery.Owner) {
					log.Info("emit tx pending " + ctx)
					txs, err := so.walletService.GetPendingTransactions(SingleOwner{owner})
					resp := SocketIOJsonResp{}

					if err != nil {
						resp = SocketIOJsonResp{Error: err.Error()}
					} else {
						resp.Data = txs
					}
					respJson, _ := json.Marshal(resp)
					v.Emit(eventKeyPendingTx+EventPostfixRes, string(respJson[:]))
				}
			}
		}
		return true
	})

	return nil
}
