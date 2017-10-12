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
	"fmt"
	"github.com/Loopring/ringminer/cmd/utils"
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/node"
	"go.uber.org/zap"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"sort"
)

var (
	app          *cli.App
	configFile   string
	globalConfig *config.GlobalConfig
	logger       *zap.Logger
)

func main() {
	app = utils.NewApp()
	app.Action = minerNode
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2017 The Loopring Authors"
	app.Flags = utils.GlobalFlags()

	app.Commands = []cli.Command{
		accountCommands(),
	}

	sort.Sort(cli.CommandsByName(app.Commands))

	app.Before = func(ctx *cli.Context) error {
		//runtime.GOMAXPROCS(runtime.NumCPU())
		file := ""
		if ctx.IsSet(configFile) {
			file = ctx.String("conf")
		}
		globalConfig = config.LoadConfig(file)

		//todo:merge flags to config, 区分node

		//if _, err := config.Validator(reflect.ValueOf(globalConfig).Elem()); nil != err {
		//	panic(err)
		//}

		logger = log.Initialize(globalConfig.Log)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer func() {
		if nil != logger {
			logger.Sync()
		}
	}()
}

func minerNode(c *cli.Context) error {
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

	//todo：设置flag到config中
	n = node.NewEthNode(logger, globalConfig)
	n.Start()

	log.Info("started")
	//captiure stop signal

	n.Wait()
	return nil
}
