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
	"github.com/Loopring/relay/cache"
	"github.com/Loopring/relay/cmd/utils"
	"github.com/Loopring/relay/config"
	"github.com/Loopring/relay/crypto"
	"github.com/Loopring/relay/ethaccessor"
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/urfave/cli.v1"
	"math/big"
	"strings"
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
					cli.StringFlag{
						Name:  "config",
						Usage: "config file",
					},
					cli.StringFlag{
						Name:  "gasPrice",
						Usage: "gasPrice",
					},
					cli.StringFlag{
						Name:  "protocolAddress",
						Usage: "protocolAddress",
					},
				},
			},
			cli.Command{
				Name:   "transferOwnership",
				Usage:  "import a private key",
				Action: transferOwnership,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "config",
						Usage: "config file",
					},
					cli.StringFlag{
						Name:  "protocolAddress",
						Usage: "protocolAddress",
					},
				},
			},
			cli.Command{
				Name:   "addParticipant",
				Usage:  "add participant to sender",
				Action: addParticipant,
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
					cli.StringFlag{
						Name:  "config",
						Usage: "config file",
					},
					cli.StringFlag{
						Name:  "gasPrice",
						Usage: "gasPrice",
					},
					cli.StringFlag{
						Name:  "protocolAddress",
						Usage: "protocolAddress",
					},
				},
			},
		},
	}
	return c
}

func addParticipant(ctx *cli.Context) {
	globalConfig := utils.SetGlobalConfig(ctx)
	logger := log.Initialize(globalConfig.Log)
	defer func() {
		if nil != logger {
			logger.Sync()
		}
	}()
	cache.NewCache(globalConfig.Redis)
	initEthaccessor(globalConfig)

	gasPrice := new(big.Int)
	gasPrice.SetString(ctx.String("gasPrice"), 10)
	protocolAddress := common.HexToAddress(ctx.String("protocolAddress"))
	registerAddress := ethaccessor.ProtocolAddresses()[protocolAddress].NameRegistryAddress

	sender := unlockSender(ctx, globalConfig.Keystore)

	sendMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.NameRegistryAbi(), registerAddress)

	feeRecipient := common.HexToAddress(ctx.String("feeRecipient"))
	signer := common.HexToAddress(ctx.String("signer"))
	if txHash, err := sendMethod(sender, "addParticipant", big.NewInt(int64(1000000)), gasPrice, big.NewInt(int64(0)), feeRecipient, signer); nil != err {
		fmt.Fprintf(ctx.App.Writer, "addParticipant err:%s sender:%s feeRecipient:%s signer:%s \n", err.Error(), sender.Hex(), feeRecipient, signer)
		utils.ExitWithErr(ctx.App.Writer, err)
	} else {
		fmt.Fprintf(ctx.App.Writer, "have send addParticipant transaction with hash:%s, you can see this in etherscan.io.\n", txHash)
	}
}

func registerName(ctx *cli.Context) {

	globalConfig := utils.SetGlobalConfig(ctx)
	logger := log.Initialize(globalConfig.Log)
	defer func() {
		if nil != logger {
			logger.Sync()
		}
	}()
	cache.NewCache(globalConfig.Redis)
	initEthaccessor(globalConfig)

	gasPrice := new(big.Int)
	gasPrice.SetString(ctx.String("gasPrice"), 10)
	protocolAddress := common.HexToAddress(ctx.String("protocolAddress"))
	registerAddress := ethaccessor.ProtocolAddresses()[protocolAddress].NameRegistryAddress

	//registry
	name := ctx.String("name")
	if "" == name {
		utils.ExitWithErr(ctx.App.Writer, errors.New("the name to register can't be empty"))
	}
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.NameRegistryAbi(), registerAddress)
	var ownerAddressHex string
	err := callMethod(&ownerAddressHex, "getOwner", "latest", name)
	if nil != err {
		utils.ExitWithErr(ctx.App.Writer, err)
	} else {
		if !types.IsZeroAddress(common.HexToAddress(ownerAddressHex)) {
			utils.ExitWithErr(ctx.App.Writer, errors.New(" the name: \""+name+"\" has been registered."))
		}
	}
	sender := unlockSender(ctx, globalConfig.Keystore)

	sendMethod := ethaccessor.ContractSendTransactionMethod("latest", ethaccessor.NameRegistryAbi(), registerAddress)

	if txHash, err := sendMethod(sender, "registerName", big.NewInt(int64(100000)), gasPrice, big.NewInt(int64(0)), name); nil != err {
		fmt.Fprintf(ctx.App.Writer, "can't register name:%s err:%s \n", name, err.Error())
		utils.ExitWithErr(ctx.App.Writer, err)
	} else {
		fmt.Fprintf(ctx.App.Writer, "have send registerName transaction with hash:%s, you can see this in etherscan.io.\n", txHash)
	}
}

func transferOwnership(ctx *cli.Context) {
	globalConfig := utils.SetGlobalConfig(ctx)
	logger := log.Initialize(globalConfig.Log)
	defer func() {
		if nil != logger {
			logger.Sync()
		}
	}()
	cache.NewCache(globalConfig.Redis)
	initEthaccessor(globalConfig)

	protocolAddress := common.HexToAddress(ctx.String("protocolAddress"))
	registerAddress := ethaccessor.ProtocolAddresses()[protocolAddress].NameRegistryAddress
	callMethod := ethaccessor.ContractCallMethod(ethaccessor.NameRegistryAbi(), registerAddress)
	var ownerAddressHex string
	err := callMethod(&ownerAddressHex, "getParticipantIds", "latest", "miner1", big.NewInt(int64(0)), big.NewInt(int64(10)))
	if nil != err {
		utils.ExitWithErr(ctx.App.Writer, err)
	} else {
		ownerAddressHex = strings.TrimPrefix(ownerAddressHex, "0x")
		b := common.Hex2Bytes(ownerAddressHex)
		res := []*big.Int{}
		err1 := ethaccessor.NameRegistryAbi().Unpack(&res, "getParticipantIds", b, 1)
		if nil != err1 {
			println(err1.Error())
		}
		println(res[0].String())
		println(res[1].String())
		println(len(res))
		//type A struct {
		//	FeeRecipient common.Address
		//	Signer common.Address
		//}
		//res := &A{}
		//err1 := ethaccessor.NameRegistryAbi().Unpack(res, "getParticipantId", b, 1)
		//if nil != err1 {
		//	println(err1.Error())
		//}
		//println("ownerAddressHex", ownerAddressHex)
		//println("res[0]", res.FeeRecipient.Hex())
		//println("res[1]", res.Signer.Hex())
	}
}

func unlockSender(ctx *cli.Context, ksOptions config.KeyStoreOptions) common.Address {
	ks := keystore.NewKeyStore(ksOptions.Keydir, ksOptions.ScryptN, ksOptions.ScryptP)
	c := crypto.NewKSCrypto(true, ks)
	crypto.Initialize(c)

	sender := accounts.Account{Address: common.HexToAddress(ctx.String("sender"))}
	unlockAccountFromTerminal(sender, ctx)
	return sender.Address
}

func initEthaccessor(globalConfig *config.GlobalConfig) error {
	return ethaccessor.Initialize(globalConfig.Accessor, globalConfig.Common, common.Address{})
}
