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

package utils

import (
	"github.com/Loopring/ringminer/params"
	"gopkg.in/urfave/cli.v1"
	"os"
	"path/filepath"
)

func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Version = params.Version
	app.Usage = "the Loopring/ringminer command line interface"
	app.Author = ""
	app.Email = ""
	return app
}

func matchengineCommands() cli.Command {
	matchengineCommand := cli.Command{
		Name:     "matchengine",
		Usage:    "matchengine ",
		Category: "Matchengine Commands",
		Action:   nil,
	}
	return matchengineCommand
}

func chainclientCommands() cli.Command {
	c := cli.Command{
		Name:        "chainclient",
		Usage:       "chainclient ",
		Category:    "Chainclient Commands",
		Subcommands: []cli.Command{
		//秘钥以及地址，生成时的密码
		},
	}
	return c
}

//todo:imp it
func accountCommands() cli.Command {
	c := cli.Command{
		Name:     "account",
		Usage:    "account",
		Category: "account Commands",
		Subcommands: []cli.Command{
			cli.Command{
				Name:  "generate",
				Usage: "generate a new account",
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "display",
						Usage: "display the privatekey",
					},
					cli.StringFlag{
						Name:  "pass",
						Usage: "passphrase for encrypted the private",
					},
				},
			},
		},
	}
	return c
}
