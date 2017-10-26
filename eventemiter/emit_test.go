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

package eventemitter_test

import (
	"fmt"
	"github.com/Loopring/ringminer/eventemiter"
	"reflect"
	"testing"
	"time"
)

type ForkEvent struct {
	Name string
}

func TestEmit(t *testing.T) {
	watcher := &eventemitter.Watcher{Concurrent: true, Handle: func(event eventemitter.EventData) error {
		fmt.Printf("==========%#v\n", event)
		e := event.(ForkEvent)
		println(e.Name)

		return nil
	}}
	watcher1 := &eventemitter.Watcher{Concurrent: false, Handle: func(event eventemitter.EventData) error {
		fmt.Printf("%#v\n", event)
		e := event.(ForkEvent)
		println("dsfioewuoriuowieuoirw" + e.Name)
		return nil
	}}
	eventemitter.On(eventemitter.Fork.Name(), watcher)
	eventemitter.On(eventemitter.Fork.Name(), watcher1)
	//eventData := ForkEvent{Name:"nnnnnn"}
	eventemitter.Emit("Fork", nil)

	t1 := eventemitter.Fork
	println(reflect.TypeOf(t1).String())
	fmt.Printf("\n%#v\n", t1)
	time.Sleep(time.Duration(100000000))
}
