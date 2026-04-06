package model

import "time"

type OrdersSnapshot struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	OrderID      int64     `gorm:"column:order_id;uniqueIndex"`
	OrderCode    string    `gorm:"column:order_code"`
	TotalAmount  float64   `gorm:"column:total_amount"`
	ShippingType string    `gorm:"column:shipping_type"`
	Remarks      string    `gorm:"column:remarks"`
	OrderDate    string    `gorm:"column:order_date"`
	OrderTime    string    `gorm:"column:order_time"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (OrdersSnapshot) TableName() string {
	return "orders_snapshot"
}
