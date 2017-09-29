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

package log

import (
	"encoding/json"
	"go.uber.org/zap"
)

//todo: I'm not sure whether zap support Rotating
var logger *zap.Logger
var sugaredLogger *zap.SugaredLogger

const logConfig = `{
	  "level": "debug",
	  "development": true,
	  "disableStacktrace": false,
	  "encoding": "console",
	  "outputPaths": ["zap.log","stderr"],
	  "errorOutputPaths": ["err.log"],
	  "encoderConfig": {
	    "messageKey": "msg",
	    "levelKey": "level",
	    "stacktraceKey":"trace",
	    "timeKey":"ts",
	    "levelEncoder": "lowercase",
	    "timeEncoder":"iso8601"
	  }
	}`

func Initialize(cfg zap.Config) *zap.Logger {
	rawJSON := []byte(logConfig)

	var err error

	if err = json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}

	//cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	//"callerKey":"C"
	//cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	//cfg.EncoderConfig.LineEnding = zapcore.DefaultLineEnding
	//opts := zap.AddStacktrace(zap.DebugLevel)

	logger, err = cfg.Build()
	if err != nil {
		panic(err)
	}
	sugaredLogger = logger.Sugar()

	return logger
}
