package entity

import "time"

type OrderEntity struct {
	ID            int64             `json:"id"`
	OrderCode     string            `json:"order_code"`
	BuyerID       int64             `json:"buyer_id"`
	ProductImage  string            `json:"product_image"`
	OrderDate     string            `json:"order_date"`
	Status        string            `json:"status"`
	TotalAmount   float64           `json:"total_amount"`
	ShippingType  string            `json:"shipping_type"`
	ShippingFee   float64           `json:"shipping_fee"`
	OrderTime     string            `json:"order_time"`
	Remarks       string            `json:"remarks"`
	BuyerName     string            `json:"customer_name"`
	BuyerEmail    string            `json:"buyer_email"`
	BuyerPhone    string            `json:"buyer_phone"`
	BuyerAddress  string            `json:"buyer_address"`
	PaymentMethod string            `json:"payment_method"`
	CreatedAt     time.Time         `json:"created_at"`
	OrderItems    []OrderItemEntity `json:"order_items"`
}
type QueryStringEntity struct {
	Page    int64  `query:"page"`
	Search  string `query:"search"`
	Limit   int64  `query:"limit"`
	Status  string `query:"status"`
	BuyerID int64  `query:"buyer_id"`
}
