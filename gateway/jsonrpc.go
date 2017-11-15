package gateway

import (
	"fmt"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"net"
	"net/http"
	"net/rpc"
)

func (*JsonrpcServiceImpl) Ping(val [1]string, res *string) error {
	*res = "pong for first connect, meaning server is OK"
	return nil
}


var RemoteAddrContextKey = "RemoteAddr"

type JsonrpcService interface {
	Start(port string)
	Stop()
}

type JsonrpcServiceImpl struct {
	port string
}

func NewJsonrpcService(port string) *JsonrpcServiceImpl {
	l := &JsonrpcServiceImpl{}
	l.port = port
	return l
}

func (j *JsonrpcServiceImpl) Start() {
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

func (*JsonrpcServiceImpl) SubmitOrder(order types.Order, res *string) error {
	HandleOrder(&order)
	*res = "SUBMIT_SUCCESS"
	return nil
}
