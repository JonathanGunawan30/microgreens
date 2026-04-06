package request

type PaymentRequest struct {
	OrderID       int64  `json:"order_id" validate:"required"`
	PaymentMethod string `json:"payment_method" validate:"required,max=50"`
	UserID        int64  `json:"user_id" validate:"required"`
	Remarks       string `json:"remarks" validate:"max=500"`
}

type MidtransWebhookPayload struct {
	TransactionStatus string `json:"transaction_status" validate:"max=50"`
	OrderID           string `json:"order_id" validate:"max=64"`
	SignatureKey      string `json:"signature_key"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
}
