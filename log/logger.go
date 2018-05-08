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
	"github.com/Loopring/relay/config"
	"go.uber.org/zap"
	"strings"
	"time"
)

//todo: I'm not sure whether zap support Rotating
var logger *zap.Logger
var sugaredLogger *zap.SugaredLogger

func Initialize(logOpts config.LogOptions) *zap.Logger {
	var err error

	cfg := logOpts.ZapOpts
	outputPaths := []string{}
	for idx, path := range cfg.OutputPaths {
		outputPaths = append(outputPaths, path)
		if !strings.HasPrefix(path, "std") {
			cfg.OutputPaths[idx] = path + time.Now().Format("")
		}
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

func logRotate(logOpts config.LogOptions, outputPaths []string) {
	now := time.Now()
	// 计算下一个零点
	next := now.Add(time.Hour * 24)
	next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
	for {
		select {
		case <-time.After(time.Until(next)):

			logger.WithOptions()
		}

	}
}
