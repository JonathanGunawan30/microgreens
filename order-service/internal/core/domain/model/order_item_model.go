package model

import "time"

type OrderItem struct {
	ID            int64      `gorm:"primaryKey;column:id"`
	ProductID     int64      `gorm:"column:product_id;not null"`
	OrderID       int64      `gorm:"column:order_id;not null"`
	Quantity      int        `gorm:"column:quantity;default:1"`
	ProductName   string     `gorm:"column:product_name"`
	ProductImage  string     `gorm:"column:product_image"`
	Price         int64      `gorm:"column:price"`
	ProductUnit   string     `gorm:"column:product_unit"`
	ProductWeight int64      `gorm:"column:product_weight"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
	DeletedAt     *time.Time `gorm:"column:deleted_at"`
	Order         Order      `gorm:"foreignKey:OrderID;references:ID"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
