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

package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/Loopring/relay/cmd/utils"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/node"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/urfave/cli.v1"
)

func startNode(ctx *cli.Context) error {

	globalConfig := utils.SetGlobalConfig(ctx)

	logger := log.Initialize(globalConfig.Log)
	defer func() {
		if nil != logger {
			logger.Sync()
		}
	}()

	var n *node.Node
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	signal.Notify(signalChan, os.Kill)
	go func() {
		for {
			select {
			case sig := <-signalChan:
				log.Infof("captured %s, exiting...\n", sig.String())
				if nil != n {
					n.Stop()
				}
				os.Exit(1)
			}
		}
	}()

	n = node.NewNode(logger, globalConfig)

	unlockAccount(ctx, globalConfig)

	n.Start()

	log.Info("started")

	n.Wait()
	return nil
}

func unlockAccount(ctx *cli.Context, globalConfig *config.GlobalConfig) {
	if "full" == globalConfig.Mode || "miner" == globalConfig.Mode {
		minerUnlocked := false
		unlockAccs := []accounts.Account{}
		if ctx.IsSet(utils.UnlockFlag.Name) {
			unlocks := strings.Split(ctx.String(utils.UnlockFlag.Name), ",")
			for _, acc := range unlocks {
				if common.IsHexAddress(acc) {
					if globalConfig.Miner.Miner == acc {
						minerUnlocked = true
					}
					unlockAccs = append(unlockAccs, accounts.Account{Address: common.HexToAddress(acc)})
				} else {
					utils.ExitWithErr(ctx.App.Writer, errors.New(acc+" is not a HexAddress"))
				}
			}
		}
		if !minerUnlocked {
			utils.ExitWithErr(ctx.App.Writer, errors.New("the address used to mine ring must be unlocked "))
		}
		for _, acc := range unlockAccs {
			unlocked := false
			for trials := 1; trials < 4; trials++ {
				fmt.Fprintf(ctx.App.Writer, "Unlocking account %s | Attempt %d/%d \n", acc.Address.Hex(), trials, 3)
				passphrase, _ := getPassphraseFromTeminal(false, ctx.App.Writer)
				if err := crypto.UnlockAccount(acc, passphrase); nil != err {
					if keystore.ErrNoMatch == err {
						log.Fatalf("err:", err.Error())
					} else {
						log.Infof("failed to unlock, try again")
					}
				} else {
					unlocked = true
					log.Infof("Unlocked address:%s", acc.Address.Hex())
					break
				}
			}
			if !unlocked {
				utils.ExitWithErr(ctx.App.Writer, errors.New("3 incorrect passphrase attempts when unlocking address:"+acc.Address.Hex()))
			}
		}
	}
}
