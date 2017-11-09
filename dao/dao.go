package dao

import (
	"github.com/Loopring/ringminer/config"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"log"
)

type RdsService interface {
	Prepare()
	Add(item interface{}) error
	First(item interface{}) error
	Last(item interface{}) error
	Update(item interface{}) error
	FindAll(item interface{}) error
}

type RdsServiceImpl struct {
	options config.MysqlOptions
	db      *gorm.DB
}

func NewRdsService(options config.MysqlOptions) *RdsServiceImpl {
	impl := &RdsServiceImpl{}
	impl.options = options

	gorm.DefaultTableNameHandler = func(db *gorm.DB, defaultTableName string) string {
		return options.TablePrefix + defaultTableName
	}

	url := options.User + ":" + options.Password + "@/" + options.DbName + "?charset=utf8&parseTime=True&loc=" + options.Loc
	db, err := gorm.Open("mysql", url)
	if err != nil {
		log.Fatalf("mysql connection error:%s", err.Error())
	}

	impl.db = db

	return impl
}

// create tables if not exists
func (s *RdsServiceImpl) Prepare() {
	var tables []interface{}
	tables = append(tables, &Order{})

	for _, t := range tables {
		if ok := s.db.HasTable(t); !ok {
			s.db.CreateTable(t)
		}
	}
}

////////////////////////////////////////////////////
//
// base functions
//
////////////////////////////////////////////////////

// add single item
func (s *RdsServiceImpl) Add(item interface{}) error {
	return s.db.Create(item).Error
}

// del single item
func (s *RdsServiceImpl) Del(item interface{}) error {
	return s.db.Delete(item).Error
}

// select first item order by primary key asc
func (s *RdsServiceImpl) First(item interface{}) error {
	return s.db.First(item).Error
}

// select the last item order by primary key asc
func (s *RdsServiceImpl) Last(item interface{}) error {
	return s.db.Last(item).Error
}

// update single item
func (s *RdsServiceImpl) Update(item interface{}) error {
	return s.db.Save(item).Error
}

// find all items in table where primary key > 0
func (s *RdsServiceImpl) FindAll(item interface{}) error {
	return s.db.Table("lpr_orders").Find(item, s.db.Where("id > ", 0)).Error
}
