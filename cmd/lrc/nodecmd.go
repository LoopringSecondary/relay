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
	"github.com/Loopring/relay/log"
	"github.com/Loopring/relay/node"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
)

func minerNode(ctx *cli.Context) error {

	globalConfig := setGlobalConfig(ctx)

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

	n = node.NewEthNode(logger, globalConfig)
	n.Start()

	log.Info("started")

	n.Wait()
	return nil
}
