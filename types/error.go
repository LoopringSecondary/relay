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

package types

type (
	ErrorLevel int
	ErrorCode  int
)

const (
	ErrorLevelUnknown ErrorLevel = 0
	ErrorLevelDebug   ErrorLevel = 1
	ErrorLevelWarning ErrorLevel = 2
	ErrorLevelWaiting ErrorLevel = 3
	ErrorLevelFatal   ErrorLevel = 4
)

const (
	ErrIteratorValidate  ErrorCode = 1
	ErrIteratorRetryCall ErrorCode = 2
	ErrIteractorGetBlock ErrorCode = 3
)

type ErrorMsg struct {
	Msg   string
	Code  ErrorCode
	Level ErrorLevel
}

func NewError(msg string, code ErrorCode, level ErrorLevel) ErrorMsg {
	var e ErrorMsg
	e.Msg = msg
	e.Code = code
	e.Level = level

	return e
}
