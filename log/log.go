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
	"go.uber.org/zap"
	"time"
	"go.uber.org/zap/zapcore"
)

const key = "content"

// TODO(fk): logger should be used more convenient

func Info(level, value string) {
	logger.Info(level, setTime(), zap.String(key, value))
}

func Error(level, value string) {
	logger.Error(level, zap.String(key, value))
}

func Warn(level, value string) {
	logger.Warn(level, zap.String(key, value))
}

func Crit(level, value string) {
	logger.Fatal(level, zap.String(key, value))
}

func setTime() zapcore.Field {
	return zap.Int64("timestamp", time.Now().Unix())
}
