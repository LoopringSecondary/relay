package gateway

import (
	"fmt"
	"github.com/Loopring/ringminer/gateway"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"net/http"
	"testing"
	"time"
)

var (
	impl       *gateway.JsonrpcServiceImpl
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

func prepare() {

	impl = gateway.NewJsonrpcService("8080")
	clientHTTP = jsonrpc2.NewCustomHTTPClient(
		"http://127.0.0.1:8080/rpc",
		jsonrpc2.DoerFunc(func(req *http.Request) (*http.Response, error) {
			// Setup custom HTTP client.
			fmt.Println("fuck here ........................")
			client := &http.Client{}
			fmt.Println(client)
			// Modify request as needed.
			req.Header.Set("Content-Type", "application/json-rpc")
			fmt.Println(req.Method)
			fmt.Println(req.Header)
			resp, err := client.Do(req)
			fmt.Println(resp.StatusCode)
			fmt.Println(resp.Request)
			fmt.Println(err)
			fmt.Println("fuck here ........................")
			return resp, err
		}),
	)
	impl.Start()
}

func TestJsonrpcServiceImpl_SubmitOrder(t *testing.T) {
	prepare()

	time.Sleep(1 * time.Second)

	var relay string

	err := clientHTTP.Call("JsonrpcServiceImpl.SubmitOrder", map[string]int{"a": 10, "b": 20, "c": 30}, &relay)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("result from submit order %s\n", relay)

	time.Sleep(1 * time.Second)
}
