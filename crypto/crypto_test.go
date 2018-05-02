package crypto_test

import (
	"github.com/Loopring/relay/crypto"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

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
		t.Error(err.Error())
	} else {
		t.Log(common.ToHex(sig))
		addrBytes, _ := crypto.SigToAddress(common.FromHex(s), sig)
		t.Log(common.ToHex(addrBytes))
	}
}

func init() {
	datadir := "ks_dir"
	ks := keystore.NewKeyStore(datadir, keystore.StandardScryptN, keystore.StandardScryptP)
	c := crypto.NewKSCrypto(true, ks)
	crypto.Initialize(c)
}
