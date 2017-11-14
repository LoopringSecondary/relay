package gateway

import (
	"fmt"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"net"
	"net/http"
	"net/rpc"
)

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

	fmt.Println("start jsonrpc at port" + j.port)

	rpc.Register(&JsonrpcServiceImpl{})

	// Server provide a TCP transport.
	lnTCP, err := net.Listen("tcp", ":8888")
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
			go jsonrpc2.ServeConn(conn)
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

}

func (*JsonrpcServiceImpl) SubmitOrder(order map[string]int, res *string) error {

	fmt.Printf("request is %s", order)

	var orderHash = "orderHash"
	res = &orderHash
	fmt.Println(res)

	return nil
}
