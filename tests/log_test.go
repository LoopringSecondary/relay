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
package tests

import (
	"encoding/json"
	"go.uber.org/zap"
	"testing"
)

func Test_logger(t *testing.T) {
	rawJSON := []byte(`{
	  "level": "debug",
	  "development": false,
	  "encoding": "json",
	  "outputPaths": ["zap.log"],
	  "errorOutputPaths": ["err.log"],
	  "initialFields": {"foo": "bar"},
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`)

	var cfg zap.Config
	if err := json.Unmarshal(rawJSON, &cfg); err != nil {
		t.Fatal(err.Error())
	}
	logger, err := cfg.Build()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer logger.Sync()

	logger.Info("logger construction succeeded")

	url := "loopring.org"
	for i := 1; i < 100000; i++ {
		logger.Info("saving number", zap.String("url", url), zap.Int("attempt", i))
	}
}
