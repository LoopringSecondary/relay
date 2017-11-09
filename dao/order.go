package dao

type Order struct {
	ID        int    `gorm:"column:id;primary_key;"`
	OrderHash []byte `gorm:"column:order_hash;type:varchar(64);unique_index"`
}
