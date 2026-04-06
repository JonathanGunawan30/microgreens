package request

type CreateOrderRequest struct {
	BuyerID      int64                `json:"buyer_id" validate:"required"`
	OrderDate    string               `json:"order_date" validate:"required"`
	ShippingType string               `json:"shipping_type" validate:"required,max=20"`
	PaymentType  string               `json:"payment_type" validate:"required,max=50"`
	Remarks      string               `json:"remarks" validate:"max=500"`
	OrderTime    string               `json:"order_time"  validate:"required"`
	OrderDetails []OrderDetailRequest `json:"order_details"  validate:"required"`
}

type OrderDetailRequest struct {
	ProductID int64 `json:"product_id" validate:"required"`
	Quantity  int   `json:"quantity" validate:"required"`
}

type OrderUpdateStatusRequest struct {
	Status  string `json:"status" validate:"required,max=20"`
	Remarks string `json:"remarks" validate:"max=500"`
}
