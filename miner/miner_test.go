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

package miner_test

import (
	"fmt"
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/market/util"
	"github.com/Loopring/relay/marketcap"
	"github.com/Loopring/relay/miner"
	"github.com/Loopring/relay/miner/timing_matcher"
	"github.com/Loopring/relay/ordermanager"
	"github.com/Loopring/relay/test"
	"github.com/Loopring/relay/types"
	"github.com/Loopring/relay/usermanager"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

func loadConfig() *config.GlobalConfig {
	path := strings.TrimSuffix(os.Getenv("GOPATH"), "/") + "/src/github.com/Loopring/relay/config/relay.toml"
	c := config.LoadConfig(path)
	log.Initialize(c.Log)

	return c
}

func TestApprove(t *testing.T) {
	cfg := loadConfig()
	cache.NewCache(cfg.Redis)
	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	//accountManager := test.GenerateAccountManager()
	acc1 := accounts.Account{Address: common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")}
	acc2 := accounts.Account{Address: common.HexToAddress("0x48ff2269e58a373120ffdbbdee3fbcea854ac30a")}
	acc3 := accounts.Account{Address: common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259")}
	ks.Unlock(acc1, "1")
	ks.Unlock(acc2, "1")
	ks.Unlock(acc3, "1")
	sender := accounts.Account{Address: common.HexToAddress("0x3acdf3e3d8ec52a768083f718e763727b0210650")}
	ks.Unlock(sender, "loopring")
	c := crypto.NewKSCrypto(false, ks)
	crypto.Initialize(c)

	sendMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.Erc20Abi(), common.HexToAddress("0xef68e7c694f40c8202821edf525de3782458639f"))

	lrcAmount := new(big.Int)
	lrcAmount.SetString("100000000000000000000", 10)
	if txHash, err := sendMethod(sender.Address, "approve", big.NewInt(int64(1000000)), big.NewInt(int64(15000000000)), big.NewInt(int64(0)), common.HexToAddress("0x17233e07c67d086464fD408148c3ABB56245FA64"), lrcAmount); nil != err {
	} else {
		t.Logf("have send addParticipant transaction with hash:%s, you can see this in etherscan.io.\n", txHash)
	}
}

func TestMatch(t *testing.T) {
	cfg := loadConfig()
	cache.NewCache(cfg.Redis)
	ks := keystore.NewKeyStore(cfg.Keystore.Keydir, keystore.StandardScryptN, keystore.StandardScryptP)
	accountManager := test.GenerateAccountManager()
	acc1 := accounts.Account{Address: common.HexToAddress("0xb5fab0b11776aad5ce60588c16bd59dcfd61a1c2")}
	acc2 := accounts.Account{Address: common.HexToAddress("0x48ff2269e58a373120ffdbbdee3fbcea854ac30a")}
	acc3 := accounts.Account{Address: common.HexToAddress("0x750ad4351bb728cec7d639a9511f9d6488f1e259")}
	ks.Unlock(acc1, "1")
	ks.Unlock(acc2, "1")
	ks.Unlock(acc3, "1")
	rdsService := dao.NewRdsService(cfg.Mysql)
	userManager := usermanager.NewUserManager(&cfg.UserManager, rdsService)
	ethaccessor.Initialize(cfg.Accessor, cfg.Common, util.WethTokenAddress())
	ethaccessor.IncludeGasPriceEvaluator()

	marketCapProvider := marketcap.NewMarketCapProvider(cfg.MarketCap)
	om := ordermanager.NewOrderManager(&cfg.OrderManager, rdsService, userManager, marketCapProvider)
	submitter, _ := miner.NewSubmitter(cfg.Miner, rdsService, marketCapProvider)
	evaluator := miner.NewEvaluator(marketCapProvider, cfg.Miner)
	rds := test.GenerateDaoService()
	matcher := timing_matcher.NewTimingMatcher(cfg.Miner.TimingMatcher, submitter, evaluator, om, &accountManager, rds)
	evaluator.SetMatcher(matcher)

	m := miner.NewMiner(submitter, matcher, evaluator, marketCapProvider)
	m.Start()
	time.Sleep(1 * time.Minute)
}

func TestPrepareTestData(t *testing.T) {
	test.PrepareTestData()
}

type OrderMatchedState struct {
	//ringHash      common.Hash `json:"ringhash"`
	FilledAmountS *types.Rat `json:"filled_amount_s"`
	FilledAmountB types.Big  `json:"filled_amount_b"`
}
type A struct {
	a OrderMatchedState
}

func TestSetTokenBalances(t *testing.T) {
	test.SetTokenBalances()
	//states := []*OrderMatchedState{}
	//matchedState := &OrderMatchedState{}
	//a := big.NewRat(int64(1), int64(10))
	//matchedState.FilledAmountS = types.NewBigRat(a)
	//matchedState.FilledAmountB = *types.NewBigPtr(big.NewInt(int64(100)))
	//b := &A{}
	//b.a = *matchedState
	//println(matchedState.FilledAmountS.BigRat().RatString())
	//states = append(states, matchedState)
	//if matchedData,err := json.Marshal(matchedState); nil == err {
	//	println(string(matchedData))
	//}

}

func TestMiner_PrepareOrders(t *testing.T) {
	suffix := "0000000000000000" //0.01

	//c := test.Cfg()
	entity := test.Entity()

	lrc := util.SupportTokens["LRC"].Protocol

	eth := util.SupportMarkets["WETH"].Protocol

	account1 := entity.Accounts[0]
	account2 := entity.Accounts[1]

	//privkey := entity.PrivateKey

	db := test.GenerateDaoService()
	db.Prepare()
	// set order and marshal to json
	//protocol := common.HexToAddress(c.Common.ProtocolImpl.Address[test.Version])
	lrcFee1 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(5)) // 20个lrc
	for i := 0; i < 20; i++ {

		// 卖出0.1个eth， 买入300个lrc,lrcFee为20个lrc
		amountS1, _ := new(big.Int).SetString("10"+suffix, 0)
		amountB1, _ := new(big.Int).SetString("1000"+suffix, 0)
		lrcFee1.Add(lrcFee1, big.NewInt(int64(1)))
		order1 := test.CreateOrder(
			eth,
			lrc,
			account1.Address,
			amountS1,
			amountB1,
			lrcFee1,
		)
		order1.Price = new(big.Rat).SetFrac(order1.AmountS, order1.AmountB)
		//bs1, _ := order1.MarshalJSON()
		state1 := &types.OrderState{}
		state1.RawOrder = *order1
		if model, err := newOrderEntity(state1); nil == err {
			db.Add(model)
		} else {
			t.Fatalf("err:%s", err.Error())
		}

		// 卖出1000个lrc,买入0.1个eth,lrcFee为20个lrc
		amountS2, _ := new(big.Int).SetString("1000"+suffix, 0)
		amountB2, _ := new(big.Int).SetString("10"+suffix, 0)
		//lrcFee2 := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(3))
		lrcFee1.Add(lrcFee1, big.NewInt(int64(1)))
		order2 := test.CreateOrder(
			lrc,
			eth,
			account2.Address,
			amountS2,
			amountB2,
			lrcFee1,
			//lrcFee2,
		)
		order2.Price = new(big.Rat).SetFrac(order2.AmountS, order2.AmountB)
		//bs2, _ := order2.MarshalJSON()
		state2 := &types.OrderState{}
		state2.RawOrder = *order2
		if model, err := newOrderEntity(state2); nil == err {
			db.Add(model)
		} else {
			t.Fatalf("err:%s", err.Error())
		}
		//pubMessage(sh, string(bs1))
		//pubMessage(sh, string(bs2))
	}
}
func newOrderEntity(state *types.OrderState) (*dao.Order, error) {

	state.DealtAmountS = big.NewInt(0)
	state.DealtAmountB = big.NewInt(0)
	state.SplitAmountS = big.NewInt(0)
	state.SplitAmountB = big.NewInt(0)
	state.CancelledAmountB = big.NewInt(0)
	state.CancelledAmountS = big.NewInt(0)
	state.UpdatedBlock = big.NewInt(0)
	state.Status = types.ORDER_NEW

	model := &dao.Order{}
	var err error
	model.Market, err = util.WrapMarketByAddress(state.RawOrder.TokenB.Hex(), state.RawOrder.TokenS.Hex())
	if err != nil {
		return nil, fmt.Errorf("order manager,newOrderEntity error:%s", err.Error())
	}
	model.ConvertDown(state)

	return model, nil
}

//18428729675200069633
//9223372036854775807
