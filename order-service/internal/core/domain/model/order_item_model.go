package model

import "time"

type OrderItem struct {
	ID        int64      `gorm:"primaryKey;column:id"`
	ProductID int64      `gorm:"column:product_id;not null"`
	OrderID   int64      `gorm:"column:order_id;not null"`
	Quantity  int        `gorm:"column:quantity;default:1"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
	Order     Order      `gorm:"foreignKey:OrderID;references:ID"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
