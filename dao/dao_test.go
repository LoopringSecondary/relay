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

package dao_test

import (
	"github.com/Loopring/relay/dao"
	"github.com/Loopring/relay/test"
	"github.com/ethereum/go-ethereum/common"
	"testing"
)

func TestRdsServiceImpl_Prepare(t *testing.T) {
	s := test.GenerateDaoService()
	s.Prepare()
}

func TestRdsServiceImpl_Add(t *testing.T) {
	s := test.GenerateDaoService()
	s.Prepare()

	ord := dao.Order{OrderHash: "111222"}
	err := s.Add(&ord)
	t.Log(err)
}

func TestRdsServiceImpl_First(t *testing.T) {
	s := test.GenerateDaoService()
	ord := &dao.Order{}

	if err := s.First(ord); err != nil {
		t.Fatal(err)
	}

	t.Log(ord.ID)
}

func TestRdsServiceImpl_Update(t *testing.T) {
	s := test.GenerateDaoService()

	model := &dao.Order{}
	if err := s.First(model); err != nil {
		t.Fatal(err)
	}

	model.OrderHash = "hahahahah"
	if err := s.Save(model); err != nil {
		t.Fatal(err)
	}
}

func TestRdsServiceImpl_FindAll(t *testing.T) {
	s := test.GenerateDaoService()
	var orders []dao.Order

	if err := s.FindAll(&orders); err != nil {
		t.Fatal(err)
	}

	for _, v := range orders {
		t.Log(v.ID)
	}
}

func TestRdsServiceImpl_Hash(t *testing.T) {
	orderhash := common.HexToHash("")
	t.Log(orderhash.Hex())
}
