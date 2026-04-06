package entity

type OrdersSnapshotEntity struct {
	OrderID      int64
	OrderCode    string
	TotalAmount  float64
	ShippingType string
	Remarks      string
	OrderDate    string
	OrderTime    string
}
