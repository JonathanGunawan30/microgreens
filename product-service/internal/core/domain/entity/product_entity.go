package entity

import "time"

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
)

type ProductEntity struct {
	ID           int64           `json:"id"`
	CategorySlug string          `json:"category_slug"`
	CategoryName string          `json:"category_name"`
	ParentID     *int64          `json:"parent_id"`
	Name         string          `json:"name"`
	Image        string          `json:"image"`
	Description  string          `json:"description"`
	RegulerPrice int64           `json:"reguler_price"`
	SalePrice    int64           `json:"sale_price"`
	Unit         string          `json:"unit"`
	Weight       int64           `json:"weight"`
	Stock        int64           `json:"stock"`
	Variant      int             `json:"variant"`
	Status       ProductStatus   `json:"status"`
	Child        []ProductEntity `json:"child"`
	CreatedAt    time.Time       `json:"created_at"`
}
type QueryStringProduct struct {
	Search       string
	Page         int64
	Limit        int64
	OrderBy      string
	OrderType    string
	CategorySlug string
	StartPrice   int64
	EndPrice     int64
	Status       string
}
type StockUpdateMessage struct {
	ProductID int64 `json:"product_id"`
	Quantity  int64 `json:"quantity"`
}
