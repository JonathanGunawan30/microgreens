package entity

type CategoryEntity struct {
	ID           int64  `json:"id"`
	ParentID     *int64 `json:"parent_id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Status       bool   `json:"status"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	ProductCount int64  `json:"product_count"`
}

type QueryStringCategory struct {
	Search    string
	Page      int64
	Limit     int64
	OrderBy   string
	OrderType string
}
