package model

type ProductSnapshot struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	ProductID int64  `gorm:"column:product_id;uniqueIndex"`
	Name      string `gorm:"column:name"`
	Image     string `gorm:"column:image"`
	SalePrice int64  `gorm:"column:sale_price"`
	Unit      string `gorm:"column:unit"`
	Weight    int64  `gorm:"column:weight"`
	IsActive  bool   `gorm:"column:is_active;default:true"`
}

func (ProductSnapshot) TableName() string {
	return "products_snapshot"
}
