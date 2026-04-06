package entity

type OrderEntity struct {
	ID        int64  `json:"id"`
	OrderCode string `json:"order_code"`
	BuyerID   int64  `json:"buyer_id"`
}
