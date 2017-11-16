package gateway

import (
	"fmt"
	"github.com/Loopring/relay/gateway"
	"github.com/powerman/rpc-codec/jsonrpc2"
	"net/http"
	"testing"
	"time"
	//"net/http"
	"encoding/json"
	"github.com/Loopring/relay/types"
	"math/big"
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

	impl.Start()
	//gateway.Example()
}

func TestJsonrpcServiceImpl_SubmitOrder(t *testing.T) {
	prepare()

	var relay string

	clientHTTP = jsonrpc2.NewHTTPClient("http://127.0.0.1:8080/rpc")
	defer clientHTTP.Close()

	var req types.Order
	req.Protocol = types.StringToAddress("testProtocol")
	req.AmountB = new(big.Int)
	req.AmountB.UnmarshalText([]byte("123"))
	req.AmountS = new(big.Int)
	req.AmountS.UnmarshalText([]byte("222"))
	req.Timestamp = new(big.Int)
	req.Timestamp.UnmarshalText([]byte("222"))
	req.Ttl = new(big.Int)
	req.Ttl.UnmarshalText([]byte("222"))
	req.Salt = new(big.Int)
	req.Salt.UnmarshalText([]byte("222"))
	req.LrcFee = new(big.Int)
	req.LrcFee.UnmarshalText([]byte("222"))
	req.BuyNoMoreThanAmountB = true
	req.MarginSplitPercentage = uint8(10)
	req.V = uint8(11)
	req.R = types.StringToSign("ssss")
	req.S = types.StringToSign("ssss")

	fmt.Println(req)
	fmt.Println(json.Marshal(req))

	err := clientHTTP.Call("JsonrpcServiceImpl.SubmitOrder", req, &relay)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("result from submit order %s\n", relay)

	time.Sleep(1 * time.Second)
}
