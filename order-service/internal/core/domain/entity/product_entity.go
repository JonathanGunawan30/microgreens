package entity

type ActionType string

const (
	ActionInsert ActionType = "INSERT"
	ActionDelete ActionType = "DELETE"
)

type ProductEventData struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Image     string `json:"image"`
	SalePrice int64  `json:"sale_price"`
	Unit      string `json:"unit"`
	Weight    int64  `json:"weight"`
	Status    string `json:"status"`
}

type ProductEvent struct {
	Action ActionType        `json:"action"`
	Data   *ProductEventData `json:"data,omitempty"`
	ID     int64             `json:"id,omitempty"`
}
