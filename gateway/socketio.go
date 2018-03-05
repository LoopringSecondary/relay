package gateway

import (
	"net/http"
	"log"
	"fmt"
	"encoding/json"
	"github.com/robfig/cron"
	"github.com/googollee/go-socket.io"
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
	w.Header().Add("Access-Control-Allow-Methods","PUT,POST,GET,DELETE,OPTIONS")
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
	DEPTH: "depth",
	TEST:        "test",
}

type SocketIOService interface {
	Start(port string)
	Stop()
}

type SocketIOServiceImpl struct {
	port           string
	walletService  WalletServiceImpl
	connIdMap map[string]socketio.Conn
	connBusinessKeyMap map[string]socketio.Conn
	cron *cron.Cron
}

func NewSocketIOService(port string, walletService WalletServiceImpl) *SocketIOServiceImpl {
	so := &SocketIOServiceImpl{}
	so.port = port
	so.walletService  = walletService
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
	server.OnEvent("/", "balance", func(s socketio.Conn, msg string) {
		query := CommonTokenRequest{ContractVersion: "v1.0", Owner:msg}
		res, err := so.walletService.GetBalance(query)
		if err != nil {
			s.Emit("balance", "get balance error")
		} else {

			b, _ := json.Marshal(res)
			s.Emit("balance", string(b[:]))
		}
	})

	server.OnEvent("/", "balance_req", func(s socketio.Conn, msg string) {
		fmt.Println("input msg is : " + msg)
		context := make(map[string]string)
		if s.Context() != nil {
			context = s.Context().(map[string]string)
		}
		context["balance"] = msg
		s.SetContext(context)
		fmt.Println("current context is")
		fmt.Println(s.Context())
		so.connIdMap[s.ID()] = s
	})

	server.OnEvent("/", "balance_end", func(s socketio.Conn, msg string) {
		delete(so.connIdMap, s.ID())
		s.Close()
	})

	so.cron.AddFunc("0/10 * * * * *", func() {

		for id, v := range so.connIdMap {
			fmt.Println("start for loopring id " + id)
			fmt.Println(v)
			fmt.Println(v.Context())
			if v.Context() ==  nil  {
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
					}
				}
			}
		}
	})
	so.cron.Start()




	//for k, v := range MsgTypeRoute {
	//	server.OnEvent("/", v + EventPostfixRes, func(s socketio.Conn, msg string) {
	//		query := CommonTokenRequest{ContractVersion: "v1.0", Owner:msg}
	//		res, err := so.walletService.GetBalance(query)
	//		if err != nil {
	//			s.Emit("balance", "get balance error")
	//		} else {
	//			b, _ := json.Marshal(res)
	//			s.Emit("balance", string(b[:]))
	//		}
	//	})
	//
	//}

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
	log.Fatal(http.ListenAndServe(":" + so.port, nil))
	log.Println("finished listen socket io....")

}

func buildContext(msgType, msg string) map[string]string {


	return make(map[string]string)
}

