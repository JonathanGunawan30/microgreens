package entity

type PaymentEntity struct {
	ID                int64
	OrderID           int64
	OrderCode         string
	UserID            int64
	PaymentMethod     string
	PaymentStatus     string
	OrderShippingType string
	PaymentGatewayID  *string
	GrossAmount       float64
	PaymentURL        *string
	PaymentAt         string
	Remarks           string
	CustomerName      string
	CustomerEmail     string
	CustomerAddress   string
	OrderAt           string
	OrderRemarks      string
	OrderStatus       string
	PaymentLogs       []PaymentLogEntity
}

type PaymentQueryStringRequest struct {
	Page      int64  `query:"page"`
	Search    string `query:"search"`
	Limit     int64  `query:"limit"`
	Status    string `query:"status"`
	UserID    int64  `query:"user_id"`
	OrderType string `query:"order_type"`
	OrderBy   string `query:"order_by"`
}
