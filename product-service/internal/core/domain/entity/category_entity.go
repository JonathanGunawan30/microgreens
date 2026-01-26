package entity

type CategoryEntity struct {
	ID           int64
	ParentID     *int64
	Name         string
	Icon         string
	Status       bool
	Slug         string
	Description  string
	ProductCount int64
}

type QueryStringCategory struct {
	Search    string
	Page      int64
	Limit     int64
	OrderBy   string
	OrderType string
}
