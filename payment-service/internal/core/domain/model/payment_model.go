package model

import "time"

type Payment struct {
	ID               int64        `gorm:"primaryKey;column:id"`
	OrderID          int64        `gorm:"column:order_id;not null"`
	UserID           int64        `gorm:"column:user_id;not null"`
	PaymentMethod    string       `gorm:"column:payment_method;type:varchar(50);not null"`
	PaymentStatus    string       `gorm:"column:payment_status;type:varchar(50);not null"`
	PaymentGatewayID *string      `gorm:"column:payment_gateway_id;type:varchar(50);null"`
	GrossAmount      float64      `gorm:"column:gross_amount;type:decimal(10,2);not null"`
	PaymentURL       *string      `gorm:"column:payment_url;type:text;null"`
	OrderCode        string       `gorm:"column:order_code"`
	ShippingType     string       `gorm:"column:shipping_type"`
	OrderDate        string       `gorm:"column:order_date"`
	OrderTime        string       `gorm:"column:order_time"`
	OrderRemarks     string       `gorm:"column:order_remarks"`
	CustomerName     string       `gorm:"column:customer_name"`
	CustomerEmail    string       `gorm:"column:customer_email"`
	CustomerAddress  string       `gorm:"column:customer_address"`
	CreatedAt        time.Time    `gorm:"column:created_at"`
	UpdatedAt        time.Time    `gorm:"column:updated_at"`
	DeletedAt        *time.Time   `gorm:"column:deleted_at"`
	PaymentLogs      []PaymentLog `gorm:"foreignKey:PaymentID;references:ID"`
}

func (Payment) TableName() string {
	return "payments"
}
