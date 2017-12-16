/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package gateway_test

import (
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ipfs/go-ipfs-api"
	"math/big"
	"testing"
	"time"
	"math"
)

const (
	suffix       = "0000000000000000" //0.01
	TOKEN_SYMBOL = "LRC"
	WETH         = "WETH"
)

func TestSingleOrder(t *testing.T) {
	c := test.Cfg()
	entity := test.Entity()

	// get keystore and unlock account
	tokenAddressA := util.AllTokens[TOKEN_SYMBOL].Protocol
	tokenAddressB := util.AllTokens[WETH].Protocol
	testAcc := entity.Accounts[0]

	ks := keystore.NewKeyStore(c.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	account := accounts.Account{Address: testAcc.Address}
	ks.Unlock(account, testAcc.Passphrase)
	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	order := test.CreateOrder(
		tokenAddressA,
		tokenAddressB,
		protocol,
		account.Address,
		amountS1,
		amountB1,
		lrcFee1,
	)
	bs, _ := order.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewLocalShell()
	pubMessage(sh, string(bs))
}

func TestRing(t *testing.T) {
	c := test.Cfg()
	entity := test.Entity()

	lrc := util.SupportTokens[TOKEN_SYMBOL].Protocol
	eth := util.SupportMarkets[WETH].Protocol

	account1 := entity.Accounts[0]
	account2 := entity.Accounts[1]

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	// 卖出0.1个eth， 买入300个lrc,lrcFee为20个lrc
	amountS1, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("30000"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(123)) // 20个lrc
	order1 := test.CreateOrder(
		eth,
		lrc,
		protocol,
		account1.Address,
		amountS1,
		amountB1,
		lrcFee1,
	)
	bs1, _ := order1.MarshalJSON()

	// 卖出1000个lrc,买入0.1个eth,lrcFee为20个lrc
	amountS2, _ := new(big.Int).SetString("100000"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(120))
	order2 := test.CreateOrder(
		lrc,
		eth,
		protocol,
		account2.Address,
		amountS2,
		amountB2,
		lrcFee2,
	)
	bs2, _ := order2.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewLocalShell()
	pubMessage(sh, string(bs1))
	pubMessage(sh, string(bs2))
}

func TestPrepareProtocol(t *testing.T) {
	test.PrepareTestData()
}

func TestPrepareAccount(t *testing.T) {
	test.SetTokenBalances()
}

func TestAllowance(t *testing.T) {
	test.AllowanceToLoopring(nil, nil)
}

func pubMessage(sh *shell.Shell, data string) {
	c := test.Cfg()
	topic := c.Ipfs.BroadcastTopics[0]
	err := sh.PubSubPublish(topic, data)
	if err != nil {
		panic(err.Error())
	}
}

func MatchTestPrepare() (*config.GlobalConfig, *test.TestEntity, *ethaccessor.EthNodeAccessor) {
	c := test.Cfg()
	entity := test.Entity()
	accessor, _ := test.GenerateAccessor()

	testAcc1 := entity.Accounts[0]
	testAcc2 := entity.Accounts[1]
	password1 := entity.Accounts[0].Passphrase
	password2 := entity.Accounts[1].Passphrase

	// get keystore and unlock account
	ks := keystore.NewKeyStore(entity.KeystoreDir, keystore.StandardScryptN, keystore.StandardScryptP)
	acc1 := accounts.Account{Address: testAcc1.Address}
	acc2 := accounts.Account{Address: testAcc2.Address}

	ks.Unlock(acc1, password1)
	ks.Unlock(acc2, password2)

	cyp := crypto.NewCrypto(true, ks)
	crypto.Initialize(cyp)
	return c, entity, accessor
}

//test the amount and discount
func TestMatcher_Case1(t *testing.T) {
	c, entity, accessor := MatchTestPrepare()

	tokenAddressA := util.SupportTokens["LRC"].Protocol
	tokenAddressB := util.SupportMarkets["WETH"].Protocol

	tokenCallMethodA := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddressA)
	tokenCallMethodB := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddressB)
	var tokenAmountA types.Big
	var tokenAmountB types.Big
	tokenCallMethodA(&tokenAmountA, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountB, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountA.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountB.BigInt().String())

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	order1 := test.CreateOrder(
		tokenAddressA,
		tokenAddressB,
		protocol,
		entity.Accounts[0].Address,
		amountS1,
		amountB1,
		lrcFee1,
	)
	bs1, _ := order1.MarshalJSON()

	amountS2, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("1"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	order2 := test.CreateOrder(
		tokenAddressB,
		tokenAddressA,
		protocol,
		entity.Accounts[1].Address,
		amountS2,
		amountB2,
		lrcFee2,
	)
	bs2, _ := order2.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewLocalShell()
	pubMessage(sh, string(bs1))
	pubMessage(sh, string(bs2))

	//waiting for the result of match and submit ring,

	time.Sleep(time.Minute)
	var tokenAmountAAfterMatch types.Big
	var tokenAmountBAfterMatch types.Big
	tokenCallMethodA(&tokenAmountAAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountBAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountAAfterMatch.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountBAfterMatch.BigInt().String())

}

//test account lrcFee insufficient
func TestMatcher_Case2(t *testing.T) {
	c, entity, accessor := MatchTestPrepare()

	tokenAddressA := util.SupportTokens["EOS"].Protocol
	tokenAddressB := util.SupportMarkets["WETH"].Protocol

	tokenCallMethodA := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddressA)
	tokenCallMethodB := accessor.ContractCallMethod(accessor.Erc20Abi, tokenAddressB)
	var tokenAmountA types.Big
	var tokenAmountB types.Big
	tokenCallMethodA(&tokenAmountA, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountB, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountA.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountB.BigInt().String())

	// set order and marshal to json
	protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])

	amountS1, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("100"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	order1 := test.CreateOrder(
		tokenAddressA,
		tokenAddressB,
		protocol,
		entity.Accounts[0].Address,
		amountS1,
		amountB1,
		lrcFee1,
	)
	bs1, _ := order1.MarshalJSON()
	println(string(bs1))

	amountS2, _ := new(big.Int).SetString("50"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("5"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	order2 := test.CreateOrder(
		tokenAddressB,
		tokenAddressA,
		protocol,
		entity.Accounts[1].Address,
		amountS2,
		amountB2,
		lrcFee2,
	)
	bs2, _ := order2.MarshalJSON()
	//bs3, _ := order2.MarshalJSON()

	// get ipfs shell and sub order
	sh := shell.NewLocalShell()
	//pubMessage(sh, string(bs1))
	pubMessage(sh, string(bs2))

	//waiting for the result of match and submit ring,

	var tokenAmountAAfterMatch types.Big
	var tokenAmountBAfterMatch types.Big
	tokenCallMethodA(&tokenAmountAAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountBAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountAAfterMatch.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountBAfterMatch.BigInt().String())
}

//test account balance insufficient
func TestMatcher_Case3(t *testing.T) {

}

//test multi orders
func TestMatcher_Case4(t *testing.T) {

}

//test multi round
func TestMatcher_Case5(t *testing.T) {
	num := int64(math.Pow(10.0, 18.0))
	ret := new(big.Rat).SetInt64(num).FloatString(0)
	t.Log(ret)

	//
	v := new(big.Rat).SetFrac(big.NewInt(1), big.NewInt(1))
	v.Quo(big.NewRat(1,1), v)
	t.Log(v.FloatString(18))
}
