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
	"github.com/Loopring/relay/cache"
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
	cyp := crypto.NewKSCrypto(true, ks)
	crypto.Initialize(cyp)

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	test.CreateOrder(tokenAddressA, tokenAddressB, account.Address, amountS1, amountB1, lrcFee1)
}

func TestRing(t *testing.T) {
	entity := test.Entity()

	// get ipfs shell and sub order
	lrc := util.SupportTokens[TOKEN_SYMBOL].Protocol

	eth := util.SupportMarkets[WETH].Protocol

	account1 := entity.Accounts[0]
	account2 := entity.Accounts[1]

	// 卖出0.1个eth， 买入300个lrc,lrcFee为20个lrc
	amountS1, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("30000"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5)) // 20个lrc
	test.CreateOrder(eth, lrc, account1.Address, amountS1, amountB1, lrcFee1)

	// 卖出1000个lrc,买入0.1个eth,lrcFee为20个lrc
	amountS2, _ := new(big.Int).SetString("30000"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(3))
	test.CreateOrder(lrc, eth, account2.Address, amountS2, amountB2, lrcFee2)
}

func TestBatchRing(t *testing.T) {
	entity := test.Entity()

	lrc := util.SupportTokens[TOKEN_SYMBOL].Protocol
	eth := util.SupportMarkets[WETH].Protocol

	account1 := entity.Accounts[0]
	account2 := entity.Accounts[1]

	// 卖出0.1个eth， 买入300个lrc,lrcFee为20个lrc
	amountS1, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("30000"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5)) // 20个lrc
	test.CreateOrder(eth, lrc, account1.Address, amountS1, amountB1, lrcFee1)

	// 卖出0.1个eth， 买入200个lrc,lrcFee为20个lrc
	amountS3, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB3, _ := new(big.Int).SetString("20000"+suffix, 0)
	lrcFee3 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5)) // 20个lrc
	test.CreateOrder(eth, lrc, account1.Address, amountS3, amountB3, lrcFee3)

	// 卖出1000个lrc,买入0.1个eth,lrcFee为20个lrc
	amountS2, _ := new(big.Int).SetString("100000"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5))
	test.CreateOrder(lrc, eth, account2.Address, amountS2, amountB2, lrcFee2)
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

func MatchTestPrepare() (*config.GlobalConfig, *test.TestEntity) {
	c := test.Cfg()
	entity := test.Entity()
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

	cyp := crypto.NewKSCrypto(true, ks)
	crypto.Initialize(cyp)
	return c, entity
}

//test the amount and discount
func TestMatcher_Case1(t *testing.T) {
	var (
		tokenAmountA, tokenAmountB                     types.Big
		tokenAmountAAfterMatch, tokenAmountBAfterMatch types.Big
	)

	_, entity := MatchTestPrepare()

	tokenAddressA := util.SupportTokens["LRC"].Protocol
	tokenAddressB := util.SupportMarkets["WETH"].Protocol

	tokenCallMethodA := ethaccessor.ContractCallMethod(ethaccessor.Erc20Abi(), tokenAddressA)
	tokenCallMethodB := ethaccessor.ContractCallMethod(ethaccessor.Erc20Abi(), tokenAddressB)
	tokenCallMethodA(&tokenAmountA, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountB, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountA.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountB.BigInt().String())

	amountS1, _ := new(big.Int).SetString("1"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("10"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	test.CreateOrder(tokenAddressA, tokenAddressB, entity.Accounts[0].Address, amountS1, amountB1, lrcFee1)

	amountS2, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("1"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	test.CreateOrder(tokenAddressB, tokenAddressA, entity.Accounts[1].Address, amountS2, amountB2, lrcFee2)

	//waiting for the result of match and submit ring,
	time.Sleep(time.Minute)
	tokenCallMethodA(&tokenAmountAAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountBAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountAAfterMatch.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountBAfterMatch.BigInt().String())
}

//test account lrcFee insufficient
func TestMatcher_Case2(t *testing.T) {
	var (
		tokenAmountA, tokenAmountB                     types.Big
		tokenAmountAAfterMatch, tokenAmountBAfterMatch types.Big
	)
	_, entity := MatchTestPrepare()

	tokenAddressA := util.SupportTokens["EOS"].Protocol
	tokenAddressB := util.SupportMarkets["WETH"].Protocol

	tokenCallMethodA := ethaccessor.ContractCallMethod(ethaccessor.Erc20Abi(), tokenAddressA)
	tokenCallMethodB := ethaccessor.ContractCallMethod(ethaccessor.Erc20Abi(), tokenAddressB)
	tokenCallMethodA(&tokenAmountA, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountB, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountA.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountB.BigInt().String())

	amountS1, _ := new(big.Int).SetString("10"+suffix, 0)
	amountB1, _ := new(big.Int).SetString("100"+suffix, 0)
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	test.CreateOrder(tokenAddressA, tokenAddressB, entity.Accounts[0].Address, amountS1, amountB1, lrcFee1)

	amountS2, _ := new(big.Int).SetString("50"+suffix, 0)
	amountB2, _ := new(big.Int).SetString("5"+suffix, 0)
	lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(20))
	test.CreateOrder(tokenAddressB, tokenAddressA, entity.Accounts[1].Address, amountS2, amountB2, lrcFee2)

	//waiting for the result of match and submit ring,
	tokenCallMethodA(&tokenAmountAAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	tokenCallMethodB(&tokenAmountBAfterMatch, "balanceOf", "latest", entity.Accounts[0].Address)
	t.Logf("before match, addressA:%s -> tokenA:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressA.Hex(), tokenAmountAAfterMatch.BigInt().String())
	t.Logf("before match, addressA:%s -> tokenB:%s, amount:%s", entity.Accounts[0].Address.Hex(), tokenAddressB.Hex(), tokenAmountBAfterMatch.BigInt().String())
}

func TestBalanceSub(t *testing.T) {
	account1 := test.Entity().Accounts[0].Address
	account2 := test.Entity().Accounts[1].Address
	miner := test.Entity().Creator.Address

	lrcTokenAddress := util.AllTokens["LRC"].Protocol
	wethTokenAddress := util.AllTokens["WETH"].Protocol

	accounts := []common.Address{account1, account2, miner}
	tokens := []common.Address{lrcTokenAddress, wethTokenAddress}

	redisprefix := "testmatch_"
	for _, tokenAddress := range tokens {
		for _, account := range accounts {
			key := redisprefix + tokenAddress.Hex() + "_" + account.Hex()
			bs, err := cache.Get(key)
			if err != nil {
				balanceAfterSave, _ := ethaccessor.Erc20Balance(tokenAddress, account, "latest")
				cache.Set(key, []byte(balanceAfterSave.String()), 0)
			} else {
				balanceBeforeSave, _ := new(big.Int).SetString(string(bs), 0)
				balanceAfterSave, _ := ethaccessor.Erc20Balance(tokenAddress, account, "latest")
				cache.Set(key, []byte(balanceAfterSave.String()), 0)
				balance := new(big.Int).Sub(balanceAfterSave, balanceBeforeSave)

				symbol, _ := util.GetSymbolWithAddress(tokenAddress)
				t.Logf("symbol:%s account:%s amount:%s", symbol, account.Hex(), balance.String())
			}
		}
	}
}

func TestOrderFilled(t *testing.T) {
	order1 := "0x2b4be18b97b734f9c619367d7b422086f4476a78d2f946edf66f39ad0604cc20"
	order2 := "0xae99509109129fc957410242c56a576bf7600d173d90ed04f3bfd91e9d0ea268"
	hashlist := []string{order1, order2}
	orders, _ := test.Rds().GetOrdersByHash(hashlist)

	for _, v := range orders {
		if common.HexToAddress(v.Owner) == common.HexToAddress("0x1B978a1D302335a6F2Ebe4B8823B5E17c3C84135") {
			symbolS, _ := util.GetSymbolWithAddress(common.HexToAddress(v.TokenS))
			symbolB, _ := util.GetSymbolWithAddress(common.HexToAddress(v.TokenB))

			t.Logf("acc1 order,sell %s:%s, buy %s:%s, dealtAmount %s:%s dealtAmount %s:%s, split %s:%s, split %s:%s", symbolS, v.AmountS, symbolB, v.AmountB, symbolS, v.DealtAmountS, symbolB, v.DealtAmountB, symbolS, v.SplitAmountS, symbolB, v.SplitAmountB)
		} else {
			symbolS, _ := util.GetSymbolWithAddress(common.HexToAddress(v.TokenS))
			symbolB, _ := util.GetSymbolWithAddress(common.HexToAddress(v.TokenB))

			t.Logf("acc2 order,sell %s:%s, buy %s:%s, dealtAmount %s:%s dealtAmount %s:%s, split %s:%s, split %s:%s", symbolS, v.AmountS, symbolB, v.AmountB, symbolS, v.DealtAmountS, symbolB, v.DealtAmountB, symbolS, v.SplitAmountS, symbolB, v.SplitAmountB)
		}
	}
}
