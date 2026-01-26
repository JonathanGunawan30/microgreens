package entity

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
)

type ProductEntity struct {
	ID           int64
	CategorySlug string
	CategoryName string
	ParentID     *int64
	Name         string
	Image        string
	Description  string
	RegulerPrice int64
	SalePrice    int64
	Unit         string
	Weight       int64
	Stock        int64
	Variant      int
	Status       ProductStatus
}
