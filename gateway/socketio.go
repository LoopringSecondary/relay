package gateway

import (
	"encoding/json"
	"fmt"
	"github.com/googollee/go-socket.io"
	"github.com/robfig/cron"
	"log"
	"net/http"
	"reflect"
)

type BusinessType int

const (
	EventPostfixReq = "_req"
	EventPostfixRes = "_res"
	EventPostfixEnd = "_end"
)

var EventPostfixs = []string{EventPostfixReq, EventPostfixRes, EventPostfixEnd}

type Server struct {
	socketio.Server
}

type SocketIOJsonResp struct {
	Error string `json:"error"`
	Code  string `json:"code"`
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
	MethodName string
	Query interface{}
}

var EventTypeRoute = map[string] InvokeInfo {
	"tickers" : {"GetTickers", SingleMarket{}},
	"loopringTickers" : {"GetAllMarketTickers", nil},
	"trends" : {"GetTrend", TrendQuery{}},
	"portfolio" : {"GetPortfolio", SingleOwner{}},
	"marketcap" : {"GetPriceQuote", PriceQuoteQuery{}},
	"balance" : {"GetBalance", CommonTokenRequest{}},
	"transaction" : {"GetTransactions", TransactionQuery{}},
	"trxByHashes" : {"GetTransactionsByHash", TransactionQuery{}},
	"depth" : {"GetDepth", DepthQuery{}},
}

type SocketIOService interface {
	Start(port string)
	Stop()
}

type SocketIOServiceImpl struct {
	port               string
	walletService      WalletServiceImpl
	connIdMap          map[string]socketio.Conn
	connBusinessKeyMap map[string]socketio.Conn
	cron               *cron.Cron
}

func NewSocketIOService(port string, walletService WalletServiceImpl) *SocketIOServiceImpl {
	so := &SocketIOServiceImpl{}
	so.port = port
	so.walletService = walletService
	so.connBusinessKeyMap = make(map[string]socketio.Conn)
	so.connIdMap = make(map[string]socketio.Conn)
	so.cron = cron.New()
	return so
}

func (so *SocketIOServiceImpl) Start() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		fmt.Println("connected:", s.ID())
		so.connIdMap[s.ID()] = s
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
			context := make(map[string]string)
			if s != nil && s.Context() != nil {
				context = s.Context().(map[string]string)
			}
			context[aliasOfV] = msg
			s.SetContext(context)
			so.connIdMap[s.ID()] = s
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

	so.cron.AddFunc("0/10 * * * * *", func() {

		for _, v := range so.connIdMap {
			if v.Context() == nil {
				continue
			} else {
				fmt.Println("......start cron emit on id " + v.ID())
				businesses := v.Context().(map[string]string)
				fmt.Println(businesses)
				if businesses != nil {
					for bk, bv := range businesses {
						so.EmitNowByEventType(bk, v, bv)
					}
				}
			}
		}
	})
	so.cron.Start()

	server.OnError("/", func(e error) {
		fmt.Println("meet error:", e)
	})

	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})
	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", NewServer(*server))
	log.Println("Serving at localhost: " + so.port)
	log.Fatal(http.ListenAndServe(":"+so.port, nil))
	log.Println("finished listen socket io....")

}

func (so *SocketIOServiceImpl) EmitNowByEventType(bk string, v socketio.Conn, bv string) {
	if invokeInfo, ok := EventTypeRoute[bk]; ok {
		so.handleWith(bk, invokeInfo.Query, invokeInfo.MethodName, v, bv)
	}
}

func (so *SocketIOServiceImpl) handleWith(eventType string, query interface{}, methodName string, conn socketio.Conn, ctx string) {

	results := make([]reflect.Value, 0)
	var err error

	if query == nil {
		results = reflect.ValueOf(&so.walletService).MethodByName(methodName).Call(nil)
	} else {
		queryType := reflect.TypeOf(query)
		queryClone := reflect.New(queryType)
		err = json.Unmarshal([]byte(ctx), queryClone.Interface())
		if err != nil {
			log.Println("unmarshal error " + err.Error())
			errJson, _ := json.Marshal(SocketIOJsonResp{Error: err.Error()})
			conn.Emit(eventType + EventPostfixRes, string(errJson[:]))
			if conn != nil && conn.Context() != nil {
				context := conn.Context().(map[string]string)
				delete(context, eventType)
				conn.SetContext(context)
				so.connIdMap[conn.ID()] = conn
			}
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
		conn.Emit(eventType + EventPostfixRes, string(errJson[:]))
	} else {
		rst := SocketIOJsonResp{Data:res.Interface()}
		b, _ := json.Marshal(rst)
		conn.Emit(eventType + EventPostfixRes, string(b[:]))
	}
}