package gateway

import (
	"github.com/powerman/rpc-codec/jsonrpc2"
	"github.com/Loopring/ringminer/gateway"
	"fmt"
	"testing"
	"time"
	//"net/http"
)

var (
	impl    *gateway.JsonrpcServiceImpl
	clientHTTP *jsonrpc2.Client
)

//func main() {
//	lnHTTP, err := net.Listen("tcp", "127.0.0.1:8080")
//	if err != nil {
//		panic(err)
//	}
//	defer lnHTTP.Close()
//
//	clientHTTP := jsonrpc2.NewHTTPClient("http://" + lnHTTP.Addr().String() + "/rpc")
//
//	var relay int
//
//	clientHTTP.Call("DefaultHandleSvc.SubmitOrder", []int{3, 5, -2}, &relay)
//	fmt.Printf("SumAll(3,5,-2)=%d\n", relay)
//}

func prepare()  {

	impl = gateway.NewJsonrpcService("8080")

	impl.Start()
	//gateway.Example()
}

func TestJsonrpcServiceImpl_SubmitOrder(t *testing.T) {
	prepare()

	var relay string

	clientHTTP = jsonrpc2.NewHTTPClient("http://127.0.0.1:8080/rpc")
	defer clientHTTP.Close()

	err := clientHTTP.Call("JsonrpcServiceImpl.SubmitOrder", map[string]int{"a": 10, "b": 20, "c": 30}, &relay)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("result from submit order %s\n", relay)

	time.Sleep(1 * time.Second)
}