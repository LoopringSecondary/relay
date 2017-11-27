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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/ssh/terminal"
	"gopkg.in/urfave/cli.v1"
	"io"
	"os"
	"syscall"
)

func accountCommands() cli.Command {
	c := cli.Command{
		Name:     "account",
		Usage:    "manage accounts",
		Category: "account commands:",
		Subcommands: []cli.Command{
			cli.Command{
				Name:   "create",
				Usage:  "create a new account",
				Action: createAccount,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "datadir",
						Usage: "keystore",
					},
					cli.StringFlag{
						Name:  "passphrase,p",
						Usage: "passphrase for lock account ",
					},
				},
			},
			cli.Command{
				Name:   "import",
				Usage:  "import a private key",
				Action: importAccount,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "passphrase,p",
						Usage: "passphrase for lock account",
					},
					cli.StringFlag{
						Name:  "datadir",
						Usage: "keystore",
					},
					cli.StringFlag{
						Name:  "private-key,pk",
						Usage: "the private key to be encrypted",
					},
				},
			},
			cli.Command{
				Name:   "list",
				Usage:  "list all the accounts",
				Action: listAccounts,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "datadir",
						Usage: "keystore",
					},
				},
			},
		},
	}
	return c
}

func createAccount(ctx *cli.Context) {
	dir := ctx.String("datadir")
	if "" == dir {
		exitWithErr(errors.New("keystore file can't empty"), ctx.App.Writer)
	}

	var passphrase string
	if passphrase = ctx.String("passphrase"); "" == passphrase {
		var err error
		if passphrase, err = getPassphraseFromTeminal(true, ctx.App.Writer); nil != err {
			exitWithErr(err, ctx.App.Writer)
		}
	}
	ks := keystore.NewKeyStore(dir, keystore.StandardScryptN, keystore.StandardScryptP)
	if account, err := ks.NewAccount(passphrase); nil != err {
		exitWithErr(err, ctx.App.Writer)
	} else {
		fmt.Fprintf(ctx.App.Writer, "create address:%x \n", account.Address)
	}
}

func importAccount(ctx *cli.Context) {
	dir := ctx.String("datadir")
	if "" == dir {
		exitWithErr(errors.New("keystore file can't empty"), ctx.App.Writer)
	}

	pk := ctx.String("private-key")
	if !common.IsHex(pk) {
		exitWithErr(errors.New("the private-key must be hex"), ctx.App.Writer)
	}
	if privateKey, err := crypto.ToECDSA(common.Hex2Bytes(pk)); nil != err {
		exitWithErr(err, ctx.App.Writer)
	} else {
		var passphrase string
		if passphrase = ctx.String("passphrase"); "" == passphrase {
			var err error
			if passphrase, err = getPassphraseFromTeminal(true, ctx.App.Writer); nil != err {
				exitWithErr(err, ctx.App.Writer)
			}
		}

		ks := keystore.NewKeyStore(dir, keystore.StandardScryptN, keystore.StandardScryptP)
		if account, err := ks.ImportECDSA(privateKey, passphrase); nil != err {
			exitWithErr(err, ctx.App.Writer)
		} else {
			fmt.Fprintf(ctx.App.Writer, "create address:%x \n", account.Address)
		}
	}
}

func listAccounts(ctx *cli.Context) {
	dir := ctx.String("datadir")
	if "" == dir {
		exitWithErr(errors.New("keystore file can't empty"), ctx.App.Writer)
	}
	ks := keystore.NewKeyStore(dir, keystore.StandardScryptN, keystore.StandardScryptP)

	accs := []common.Address{}
	for _, account := range ks.Accounts() {
		accs = append(accs, account.Address)
	}
	bs, _ := json.Marshal(accs)
	fmt.Fprintf(ctx.App.Writer, "%s \n", string(bs))
}

func getPassphraseFromTeminal(confirm bool, writer io.Writer) (string, error) {
	var passphrase []byte
	var err error
	fmt.Fprint(writer, "enter passphrase：")

	if passphrase, err = terminal.ReadPassword(int(syscall.Stdin)); nil != err {
		return "", err
	}
	fmt.Fprint(writer, "\n")

	if confirm {
		fmt.Fprint(writer, "confirm passphrase: ")
		if passphraseRepeat, err := terminal.ReadPassword(int(syscall.Stdin)); nil != err {
			return "", err
		} else {
			fmt.Fprint(writer, "\n")
			if string(passphrase) != string(passphraseRepeat) {
				return "", errors.New("not match")
			}
		}
	}
	return string(passphrase), nil
}

func exitWithErr(err error, writer io.Writer) {
	fmt.Fprintf(writer, "error:%s \n", err.Error())
	os.Exit(1)
}
