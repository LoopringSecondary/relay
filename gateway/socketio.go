package gateway

import (
	"gopkg.in/googollee/go-socket.io.v1"
	"net/http"
	"log"
	"fmt"
	"encoding/json"
)

type SocketIOService interface {
	Start(port string)
	Stop()
}

type SocketIOServiceImpl struct {
	port           string
	walletService  WalletServiceImpl
}

func NewSocketIOService(port string, walletService WalletServiceImpl) *SocketIOServiceImpl {
	so := &SocketIOServiceImpl{}
	so.port = port
	so.walletService  = walletService
	return so
}

func (so *SocketIOServiceImpl) Start() {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.OnConnect("/", func(s socketio.Conn) error {
		s.SetContext("")
		fmt.Println("connected:", s.ID())
		return nil
	})
	server.OnEvent("/", "test", func(s socketio.Conn, msg string) {
		fmt.Println("test:", msg)
		s.Emit("reply", "pong relay msg : "+msg)
	})
	//server.OnEvent("/chat", "msg", func(s socketio.Conn, msg string) string {
	//	s.SetContext(msg)
	//	return "recv " + msg
	//})
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
	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		s.Emit("bye", last)
		s.Close()
		return last
	})
	server.OnError("/", func(e error) {
		fmt.Println("meet error:", e)
	})
	server.OnDisconnect("/", func(s socketio.Conn, msg string) {
		fmt.Println("closed", msg)
	})
	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("./asset")))
	log.Println("Serving at localhost: " + so.port)
	log.Fatal(http.ListenAndServe(":" + so.port, nil))
	log.Println("finished listen socket io....")

}

