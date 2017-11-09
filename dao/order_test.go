package dao_test

import (
	"github.com/Loopring/ringminer/dao"
	"github.com/Loopring/ringminer/test"
	"testing"
)

func TestRdsServiceImpl_Prepare(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()
	s.Prepare()
}

func TestRdsServiceImpl_Add(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()
	s.Prepare()

	ord := dao.Order{OrderHash: []byte("113222")}
	err := s.Add(&ord)
	t.Log(err)
}

func TestRdsServiceImpl_First(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()
	ord := &dao.Order{}

	if err := s.First(ord); err != nil {
		t.Fatal(err)
	}

	t.Log(ord.ID)
}

func TestRdsServiceImpl_Update(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()

	model := &dao.Order{}
	if err := s.First(model); err != nil {
		t.Fatal(err)
	}

	model.OrderHash = []byte("hahahhahahah")
	if err := s.Update(model); err != nil {
		t.Fatal(err)
	}
}

func TestRdsServiceImpl_FindAll(t *testing.T) {
	s := test.LoadConfigAndGenerateDaoService()
	var orders []dao.Order

	if err := s.FindAll(&orders); err != nil {
		t.Fatal(err)
	}

	for _, v := range orders {
		t.Log(v.ID)
	}
}
