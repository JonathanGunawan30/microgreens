package response

type PaymentListResponse struct {
	ID            int64   `json:"id"`
	OrderCode     string  `json:"order_code"`
	PaymentStatus string  `json:"payment_status"`
	PaymentMethod string  `json:"payment_method"`
	GrossAmount   float64 `json:"gross_amount"`
	ShippingType  string  `json:"shipping_type"`
}

type PaymentDetailResponse struct {
	ID              int64   `json:"id"`
	OrderCode       string  `json:"order_code"`
	PaymentStatus   string  `json:"payment_status"`
	PaymentMethod   string  `json:"payment_method"`
	GrossAmount     float64 `json:"gross_amount"`
	ShippingType    string  `json:"shipping_type"`
	PaymentAt       string  `json:"payment_at"`
	OrderDate       string  `json:"order_date"`
	OrderTime       string  `json:"order_time"`
	OrderRemarks    string  `json:"order_remarks"`
	CustomerName    string  `json:"customer_name"`
	CustomerAddress string  `json:"customer_address"`
}
