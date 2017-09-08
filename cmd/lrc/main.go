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

import(
	"github.com/Loopring/ringminer/cmd/utils"
	"sort"
	"gopkg.in/urfave/cli.v1"
	"runtime"
	"os"
	"fmt"
	"github.com/Loopring/ringminer/log"
	"github.com/Loopring/ringminer/node"
)

var (
	app = utils.NewApp()
	logger = log.NewLogger()
)

func init() {
	app.Action = miner
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2017 The Looprint Authors"
	app.Commands = []cli.Command{

	}
	sort.Sort(cli.CommandsByName(app.Commands))

	//app.Flags = append(app.Flags, nodeFlags...)

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer logger.Sync()
}

func miner(c *cli.Context) error {
	n := node.NewNode(logger)
	n.Start()
	n.Wait()
	return nil
}
