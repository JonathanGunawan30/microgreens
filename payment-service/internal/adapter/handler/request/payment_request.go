package request

type PaymentRequest struct {
	OrderID       int64   `json:"order_id" validate:"required"`
	PaymentMethod string  `json:"payment_method" validate:"required"`
	GrossAmount   float64 `json:"gross_amount" validate:"required"`
	UserID        int64   `json:"user_id" validate:"required"`
	Remarks       string  `json:"remarks"`
}

type MidtransWebhookPayload struct {
	TransactionStatus string `json:"transaction_status"`
	OrderID           string `json:"order_id"`
	SignatureKey      string `json:"signature_key"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
}
