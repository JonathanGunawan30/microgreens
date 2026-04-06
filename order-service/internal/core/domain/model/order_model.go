package model

import "time"

type Order struct {
	ID            int64       `gorm:"primaryKey;column:id"`
	OrderCode     string      `gorm:"column:order_code;type:varchar(64);unique;not null"`
	BuyerID       int64       `gorm:"column:buyer_id;not null"`
	OrderDate     time.Time   `gorm:"column:order_date;type:date;not null"`
	Status        string      `gorm:"column:status;type:varchar(20);index;default:pending"`
	TotalAmount   float64     `gorm:"column:total_amount;type:decimal(10,2);default:0"`
	ShippingType  string      `gorm:"column:shipping_type;type:varchar(20);default:pickup"`
	ShippingFee   float64     `gorm:"column:shipping_fee;type:decimal(10,2);default:0"`
	OrderTime     string      `gorm:"column:order_time;type:time"`
	Remarks       string      `gorm:"column:remarks;type:text"`
	PaymentMethod string      `gorm:"column:payment_method;type:varchar(50)"`
	BuyerName     string      `gorm:"column:buyer_name;type:varchar(255)"`
	BuyerEmail    string      `gorm:"column:buyer_email;type:varchar(255)"`
	BuyerAddress  string      `gorm:"column:buyer_address;type:text"`
	BuyerPhone    string      `gorm:"column:buyer_phone;type:varchar(17)"`
	CreatedAt     time.Time   `gorm:"column:created_at"`
	UpdatedAt     time.Time   `gorm:"column:updated_at"`
	DeletedAt     *time.Time  `gorm:"column:deleted_at"`
	OrderItems    []OrderItem `gorm:"foreignKey:OrderID;references:ID"`
}

func (Order) TableName() string {
	return "orders"
}
