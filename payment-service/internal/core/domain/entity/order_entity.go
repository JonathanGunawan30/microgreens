package entity

type OrderHttpClientResponse struct {
	Message string                  `json:"message"`
	Data    OrderDetailHttpResponse `json:"data"`
}

type OrderDetailHttpResponse struct {
	ID            int64         `json:"id"`
	OrderCode     string        `json:"order_code"`
	ProductImage  string        `json:"product_image"`
	OrderDateTime string        `json:"order_date_time"`
	Status        string        `json:"status"`
	PaymentMethod string        `json:"payment_method"`
	ShippingFee   float64       `json:"shipping_fee"`
	ShippingType  string        `json:"shipping_type"`
	Remarks       string        `json:"remarks"`
	TotalAmount   float64       `json:"total_amount"`
	Customer      CustomerOrder `json:"customer"`
	OrderDetail   []OrderDetail `json:"order_detail"`
}

type CustomerOrder struct {
	CustomerID      int64  `json:"customer_id"`
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerAddress string `json:"customer_address"`
	CustomerEmail   string `json:"customer_email"`
}

type OrderDetail struct {
	ProductName  string `json:"product_name"`
	ProductImage string `json:"product_image"`
	ProductPrice int64  `json:"product_price"`
	Quantity     int64  `json:"quantity"`
}
