package crypto_test

import (
	"github.com/Loopring/relay/crypto"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

//func TestEthCrypto_SigToAddress(t *testing.T) {
//	////hash := common.Hex2Bytes("bbceb08054186109713ae09b4d2d89ff2d663d46603a5179c46d829b6ae58752")
//	//sig := common.Hex2Bytes("46a2c4894856d9d264ccdb8968b2ae858f6ea9015dc78f6d08f4bb08997227bc")
//	//ethCrypto := &eth.EthCrypto{}
//	//v := common.Hex2Bytes("fc")
//	//println(big.NewInt(1).SetBytes(v).Int64())
//	//v = crypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(v), v)))
//	////r := common.Hex2Bytes("46a2c4894856d9d264ccdb8968b2ae858f6ea9015dc78f6d08f4bb08997227bc")
//	////s := common.Hex2Bytes("41f7c634bfae52d55adc83a66ee22f8d5e6ee44e97d9625c7b676747f0d6bded")
//	////println(ethCrypto.ValidateSignatureValues(v, big.NewInt(0).SetBytes(r), big.NewInt(0).SetBytes(s), false))
//	//address,err := ethCrypto.SigToAddress(v, sig)
//	//if err != nil {
//	//	t.Log(err.Error())
//	//}
//	//t.Log("address", common.Bytes2Hex(address))
//	config := &config.AccessorOptions{}
//	config.RawUrl = "http://127.0.0.1:8545"
//	ethClient := EthClient.NewChainClient(*config, "sa")
//	tx := &ethTypes.Transaction{}
//	if err := ethClient.GetTransactionByHash(tx, "0xb0fae8141315ea396b5b6c5576e59b15c2ee156ef0d8c06f32753575f4616557"); err != nil {
//		t.Error(err.Error())
//	} else {
//		v, r, s := tx.RawSignatureValues()
//		println(v.Int64(), r.Int64(), s.Int64())
//		//vb,rb,sb := v.Bytes(),r.Bytes(), s.Bytes()
//		//a := new(big.Int).SetBytes([]byte{[]byte("1")[0] + 27})
//		//chainId := tx.ChainId()
//		vbint := int64(0)
//		println(vbint, "#", r.BitLen(), "#", s.BitLen())
//		valid := ethCrypto.ValidateSignatureValues(byte(0), r.Bytes(), s.Bytes())
//		println(valid)
//		sig := make([]byte, 65)
//		copy(sig[32-len(r.Bytes()):32], r.Bytes())
//		copy(sig[64-len(s.Bytes()):64], s.Bytes())
//		sig[64] = byte(0)
//		signer := &ethTypes.HomesteadSigner{}
//		tx.Hash()
//
//		if pubkey, err := ethCrypto.SigToAddress(signer.Hash(tx).Bytes(), sig); err != nil {
//			println(err.Error())
//		} else {
//			println(common.HexToAddress("d86ee51b02c5ac295e59711f4335fed9805c0148").Hex())
//			println(common.BytesToAddress(pubkey).String())
//		}
//
//		//println(string([]byte{byte(15612-7788-27)}))
//		//tx.WithSignature()
//	}
//
//	//0x8e02dc1aa9d9294a2259e79a5a5a8fb0048286c33489b1e81cd37755c37ea8fb, sig:1c5e95ba38e2d9de7003f5de4eed8ee428ac41234f947e6a95315eba077dc86a1857f9841798fd241bd8ce979b56095703acbbd706976a4c0b8fdcfc7ce5168900
//	//address,err := ethCrypto.SigToAddress(common.Hex2Bytes("8e02dc1aa9d9294a2259e79a5a5a8fb0048286c33489b1e81cd37755c37ea8fb"), common.Hex2Bytes("1c5e95ba38e2d9de7003f5de4eed8ee428ac41234f947e6a95315eba077dc86a1857f9841798fd241bd8ce979b56095703acbbd706976a4c0b8fdcfc7ce5168900"))
//	//if err != nil {
//	//	println(err.Error())
//	//}
//	//println(address)
//
//}

//func TestWithContract(t *testing.T) {
//	config := &config.AccessorOptions{}
//	config.RawUrl = "http://127.0.0.1:8545"
//	ethClient := EthClient.NewChainClient(*config, "sa")
//	type SigTest struct {
//		chainclient.Contract
//		CalculateHash          chainclient.AbiMethod
//		CalculateSignerAddress chainclient.AbiMethod
//	}
//	contractAddress := "0xc184dd351f215f689f481c329916bb33d8df8ced"
//	abiStr := `[{"constant":true,"inputs":[{"name":"hash","type":"bytes32"},{"name":"v","type":"uint8"},{"name":"r","type":"bytes32"},{"name":"s","type":"bytes32"}],"name":"calculateSignerAddress","outputs":[{"name":"","type":"address"}],"payable":false,"stateMutability":"view","type":"function"},{"constant":true,"inputs":[{"name":"s","type":"bytes32[]"}],"name":"calculateHash","outputs":[{"name":"","type":"bytes32"}],"payable":false,"stateMutability":"view","type":"function"}]`
//	sigTest := &SigTest{}
//
//	if err := ethClient.NewContract(sigTest, contractAddress, abiStr); err != nil {
//		t.Error(err.Error())
//	}
//
//	bs := common.FromHex("0x093e56de3901764da17fef7e89f016cfdd1a88b98b1f8e3d2ebda4aff2343380")
//	bytes1 := [][]byte{bs} //[][]byte([]byte("a"))
//	res := ""
//	if err := sigTest.CalculateHash.Call(&res, "pending", bytes1); err != nil {
//		t.Error(err.Error())
//	} else {
//		t.Log(res)
//	}
//}

func TestEthCrypto_GenerateHash(t *testing.T) {
	bs := common.FromHex("0x093e56de3901764da17fef7e89f016cfdd1a88b98b1f8e3d2ebda4aff2343380")
	bytes1 := [][]byte{bs}
	res := crypto.GenerateHash(bytes1...)
	t.Log(common.Bytes2Hex(res))
}

func TestWallet(t *testing.T) {
	s := "81181790552cbbff19077f2289e29992bdb5d0eee12ca1a7ce35ac2508406c3c"
	pkstr := "d1d194d90e52aeae4cd3a727b1dbb6ea5f1de8d5379827acc5f358bf1b0acba9"
	if sig, err := crypto.Sign(common.FromHex(s), common.FromHex(pkstr)); err != nil {
		println(err.Error())
	} else {
		println(types.Bytes2Hex(sig))
		addrBytes, _ := ethCrypto.SigToAddress(types.FromHex(s), sig)
		println(types.Bytes2Hex(addrBytes))
	}
}

func init() {
	datadir := "ks_dir"
	ks := keystore.NewKeyStore(datadir, keystore.StandardScryptN, keystore.StandardScryptP)
	c := crypto.NewCrypto(true, ks)
	crypto.Initialize(c)
}
