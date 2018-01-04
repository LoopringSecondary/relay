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

package config_test

import (
	"encoding/json"
	"github.com/Loopring/relay/test"
	"testing"
)

func TestCreateTokenJsonFile(t *testing.T) {
	type Token struct {
		Protocol string
		Symbol   string
		Source   string
		Deny     bool
		Decimals int
		IsMarket bool
	}

	var list []Token
	rds := test.Rds()
	tokens, err := rds.FindUnDeniedTokens()
	if err != nil {
		panic(err)
	}
	markets, err := rds.FindUnDeniedMarkets()
	if err != nil {
		panic(err)
	}

	tokens = append(tokens, markets...)
	for _, v := range tokens {
		var t Token

		t.Protocol = v.Protocol
		t.Symbol = v.Symbol
		t.Source = v.Source
		t.Deny = v.Deny
		t.Decimals = v.Decimals
		t.IsMarket = v.IsMarket

		list = append(list, t)
	}

	bs, err := json.Marshal(&list)
	if err != nil {
		panic(err)
	}

	t.Log(string(bs))
}
