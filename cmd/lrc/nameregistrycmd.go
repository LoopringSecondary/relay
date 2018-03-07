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
	"gopkg.in/urfave/cli.v1"
	"github.com/Loopring/relay/cmd/utils"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"errors"
	"fmt"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/accounts"
)

//name registry
func nameRegistryCommands() cli.Command {
	c := cli.Command{
		Name:     "nameRegistry",
		Usage:    "Registry name and feeRecipient address by miner.",
		Category: "nameRegistry commands:",
		Subcommands: []cli.Command{
			cli.Command{
				Name:   "registerName",
				Usage:  "register a name in contract",
				Action: registerName,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "name",
						Usage: "miner name",
					},
					cli.StringFlag{
						Name:  "sender",
						Usage: "used to send transaction",
					},
				},
			},
			cli.Command{
				Name:   "transferOwnership",
				Usage:  "import a private key",
				Action: importAccount,
				Flags: []cli.Flag{

				},
			},
			cli.Command{
				Name:   "addParticipant",
				Usage:  "add participant to sender",
				Action: importAccount,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "feeRecipient",
						Usage: "feeRecipient",
					},
					cli.StringFlag{
						Name:  "signer",
						Usage: "used to sign the ring",
					},
					cli.StringFlag{
						Name:  "sender",
						Usage: "used to send transaction",
					},
				},
			},

		},
	}
	return c
}

func registerName(ctx *cli.Context) {
	dir := ctx.String("name")
	if "" == dir {
		utils.ExitWithErr(ctx.App.Writer, errors.New("keystore file can't empty"))
	}

	//cansubmit()
	//registry
	sender := common.HexToAddress(ctx.String("sender"))

	if passphrase, err := getPassphraseFromTeminal(false, ctx.App.Writer); nil != err {
		utils.ExitWithErr(ctx.App.Writer, err)
	} else {
		ks := keystore.NewKeyStore(dir, keystore.StandardScryptN, keystore.StandardScryptP)
		if err := ks.Unlock(accounts.Account{Address:sender}, passphrase);nil != err {
			fmt.Fprintf(ctx.App.Writer, "can't unlock account:%x \n", err.Error())
			utils.ExitWithErr(ctx.App.Writer, err)
		} else {
			ethaccessor.ContractSendTransactionMethod()
		}
	}
}

func Init(ctx *cli.Context) {
	globalConfig := utils.SetGlobalConfig(ctx)
	err := ethaccessor.Initialize(globalConfig.Accessor, globalConfig.Common, common.Address{})
	if nil != err {
		fmt.Fprintf(ctx.App.Writer, "err:%s", err.Error())
	}

}