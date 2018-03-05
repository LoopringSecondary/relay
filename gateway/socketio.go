package gateway

import (
	"encoding/json"
	"fmt"
	"github.com/googollee/go-socket.io"
	"github.com/robfig/cron"
	"log"
	"net/http"
)

type BusinessType int

const (
	EventPostfixReq = "_req"
	EventPostfixRes = "_res"
	EventPostfixEnd = "_end"
)

var EventPostfixs = []string{EventPostfixReq, EventPostfixRes, EventPostfixEnd}

const (
	TICKER BusinessType = iota
	PORTFOLIO
	MARKETCAP
	BALANCE
	TRANSACTION
	DEPTH
	TRENDS
	TEST
)

type Server struct {
	socketio.Server
}

func NewServer(s socketio.Server) Server {
	return Server{s}
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Header)
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
	fmt.Println(w.Header())
	s.Server.ServeHTTP(w, r)
}

var MsgTypeRoute = map[BusinessType]string{
	TICKER:      "tickers",
	TRENDS:      "trends",
	PORTFOLIO:   "portfolio",
	MARKETCAP:   "marketcap",
	BALANCE:     "balance",
	TRANSACTION: "transaction",
	DEPTH:       "depth",
	TEST:        "test",
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

	for _, v := range MsgTypeRoute {

		server.OnEvent("/", v+EventPostfixReq, func(s socketio.Conn, msg string) {
			fmt.Println("input msg is : " + msg)
			context := make(map[string]string)
			if s != nil && s.Context() != nil {
				context = s.Context().(map[string]string)
			}
			context[v] = msg
			s.SetContext(context)
			fmt.Println("current context is")
			fmt.Println(s.Context())
			so.connIdMap[s.ID()] = s
		})

		server.OnEvent("/", v+EventPostfixEnd, func(s socketio.Conn, msg string) {
			if s != nil && s.Context() != nil {
				businesses := s.Context().(map[string]string)
				delete(businesses, v)
				s.SetContext(businesses)
			}
		})
	}

	so.cron.AddFunc("0/10 * * * * *", func() {

		for id, v := range so.connIdMap {
			fmt.Println("start for loopring id " + id)
			fmt.Println(v)
			fmt.Println(v.Context())
			if v.Context() == nil {
				continue
			} else {
				businesses := v.Context().(map[string]string)
				if businesses != nil {
					for bk, bv := range businesses {
						if bk == "balance" {
							var query CommonTokenRequest
							err := json.Unmarshal([]byte(bv), &query)
							if err != nil {
								fmt.Println("unmarshal error " + bv)
							}
							res, err := so.walletService.GetBalance(query)
							if err != nil {
								v.Emit("balance_res", "get balance error")
							} else {

								b, _ := json.Marshal(res)
								v.Emit("balance_res", string(b[:]))
							}
						}
						if bk == "portfolio" {
							var query SingleOwner
							err := json.Unmarshal([]byte(bv), &query)
							if err != nil {
								fmt.Println("unmarshal error " + bv)
							}
							res, err := so.walletService.GetPortfolio(query)
							if err != nil {
								v.Emit("portfolio_res", "get portfolio error")
							} else {
								b, _ := json.Marshal(res)
								v.Emit("portfolio_res", string(b[:]))
							}
						}
						if bk == "tickers" {
							var query SingleMarket
							err := json.Unmarshal([]byte(bv), &query)
							if err != nil {
								fmt.Println("unmarshal error " + bv)
							}
							res, err := so.walletService.GetTickers(query)
							if err != nil {
								v.Emit("tickers_res", "get portfolio error")
							} else {
								b, _ := json.Marshal(res)
								v.Emit("tickers_res", string(b[:]))
							}
						}
						if bk == "trends" {
							var query SingleMarket
							err := json.Unmarshal([]byte(bv), &query)
							if err != nil {
								fmt.Println("unmarshal error " + bv)
							}
							res, err := so.walletService.GetTrend(query)
							if err != nil {
								v.Emit("trends_res", "get trends error")
							} else {
								b, _ := json.Marshal(res)
								v.Emit("trends_res", string(b[:]))
							}
						}
						if bk == "marketcap" {
							var query PriceQuoteQuery
							err := json.Unmarshal([]byte(bv), &query)
							if err != nil {
								fmt.Println("unmarshal error " + bv)
							}
							res, err := so.walletService.GetPriceQuote(query)
							if err != nil {
								v.Emit("marketcap_res", "get marketcap error")
							} else {
								b, _ := json.Marshal(res)
								v.Emit("marketcap_res", string(b[:]))
							}
						}
						if bk == "transaction" {
							var query TransactionQuery
							err := json.Unmarshal([]byte(bv), &query)
							if err != nil {
								fmt.Println("unmarshal error " + bv)
							}
							res, err := so.walletService.GetTransactions(query)
							if err != nil {
								v.Emit("transaction_res", "get transaction error")
							} else {
								b, _ := json.Marshal(res)
								v.Emit("transaction_res", string(b[:]))
							}
						}
						if bk == "depth" {
							var query DepthQuery
							err := json.Unmarshal([]byte(bv), &query)
							if err != nil {
								fmt.Println("unmarshal error " + bv)
							}
							res, err := so.walletService.GetDepth(query)
							if err != nil {
								v.Emit("depth_res", "get depth error")
							} else {
								b, _ := json.Marshal(res)
								v.Emit("depth_res", string(b[:]))
							}
						}

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
