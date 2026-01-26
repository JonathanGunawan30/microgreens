package model

import "time"

type ProductStatus string

const (
	ProductStatusDraft    ProductStatus = "draft"
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
)

type Product struct {
	ID           int64  `gorm:"primaryKey"`
	CategorySlug string `gorm:"index"`
	ParentID     *int64
	Name         string
	Image        string
	Description  string
	RegulerPrice int64
	SalePrice    int64 `gorm:"index"`
	Unit         string
	Weight       int64
	Stock        int64
	Variant      int
	Status       ProductStatus `gorm:"type:product_status_enum;default:'draft';index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
