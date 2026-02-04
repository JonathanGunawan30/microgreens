package model

import "time"

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
)

type Product struct {
	ID           int64         `gorm:"primaryKey" json:"id"`
	CategorySlug string        `gorm:"index" json:"category_slug"`
	ParentID     *int64        `json:"parent_id"`
	Name         string        `json:"name"`
	Image        string        `json:"image"`
	Description  string        `json:"description"`
	RegulerPrice int64         `json:"reguler_price"`
	SalePrice    int64         `gorm:"index" json:"sale_price"`
	Unit         string        `json:"unit"`
	Weight       int64         `json:"weight"`
	Stock        int64         `json:"stock"`
	Variant      int           `json:"variant"`
	Status       ProductStatus `gorm:"type:product_status_enum;default:'draft';index" json:"status"`
	Child        []Product     `json:"child" gorm:"-"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	DeletedAt    *time.Time    `json:"deleted_at"`
}
