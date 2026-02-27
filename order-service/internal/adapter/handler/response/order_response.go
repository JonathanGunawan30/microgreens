package response

type OrderAdminList struct {
	ID            int64  `json:"id"`
	OrderCode     string `json:"order_code"`
	ProductImage  string `json:"product_image"`
	CustomerName  string `json:"customer_name"`
	Status        string `json:"status"`
	PaymentMethod string `json:"payment_method"`
	TotalAmount   int64  `json:"total_amount"`
}

type OrderAdminDetail struct {
	ID            int64         `json:"id"`
	OrderCode     string        `json:"order_code"`
	ProductImage  string        `json:"product_image"`
	OrderDateTime string        `json:"order_date_time"`
	Status        string        `json:"status"`
	PaymentMethod string        `json:"payment_method"`
	ShippingFee   int64         `json:"shipping_fee"`
	ShippingType  string        `json:"shipping_type"`
	Remarks       string        `json:"remarks"`
	TotalAmount   int64         `json:"total_amount"`
	Customer      CustomerOrder `json:"customer"`
	OrderDetail   []OrderDetail `json:"order_detail"`
}

type CustomerOrder struct {
	CustomerID      int64  `json:"customer_id"`
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerEmail   string `json:"customer_email"`
	CustomerAddress string `json:"customer_address"`
}

type OrderDetail struct {
	ProductName  string `json:"product_name"`
	ProductImage string `json:"product_image"`
	ProductPrice int64  `json:"product_price"`
	Quantity     int64  `json:"quantity"`
}

type OrderCustomerList struct {
	ID            int64  `json:"id"`
	OrderCode     string `json:"order_code"`
	ProductImage  string `json:"product_image"`
	Status        string `json:"status"`
	PaymentMethod string `json:"payment_method"`
	TotalAmount   int64  `json:"total_amount"`
	Weight        int64  `json:"weight"`
	Unit          string `json:"unit"`
	Quantity      int64  `json:"quantity"`
	OrderDateTime string `json:"order_datetime"`
}
type UserHttpClientResponse struct {
	Message string `json:"message"`
	Data    struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
		Address string `json:"address"`
	} `json:"data"`
}

type ProductHttpClientResponse struct {
	Message string `json:"message"`
	Data    struct {
		ID        int64   `json:"id"`
		Name      string  `json:"name"`
		Image     string  `json:"image"`
		SalePrice float64 `json:"sale_price"`
		Unit      string  `json:"unit"`
		Weight    float64 `json:"weight"`
	} `json:"data"`
}

type PaymentMessage struct {
	OrderID       int64  `json:"order_id"`
	PaymentMethod string `json:"payment_method"`
}

type UpdateStatusMessage struct {
	OrderID int64  `json:"order_id"`
	Status  string `json:"status"`
}
