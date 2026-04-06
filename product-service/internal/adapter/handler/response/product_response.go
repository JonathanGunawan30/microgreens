package response

import "time"

type ProductStatus string

type ProductListResponse struct {
	ID           int64         `json:"id"`
	Name         string        `json:"name"`
	Image        string        `json:"image"`
	CategoryName string        `json:"category_name"`
	Status       ProductStatus `json:"status"`
	SalePrice    int64         `json:"sale_price"`
	RegulerPrice int64         `json:"reguler_price"`
	CreatedAt    time.Time     `json:"created_at"`
}

type ProductDetailResponse struct {
	ID           int64                  `json:"id"`
	Name         string                 `json:"name"`
	ParentID     *int64                 `json:"parent_id"`
	Image        string                 `json:"image"`
	CategoryName string                 `json:"category_name"`
	CategorySlug string                 `json:"category_slug"`
	Status       ProductStatus          `json:"status"`
	Description  string                 `json:"description"`
	SalePrice    int64                  `json:"sale_price"`
	RegulerPrice int64                  `json:"reguler_price"`
	CreatedAt    time.Time              `json:"created_at"`
	Unit         string                 `json:"unit"`
	Weight       int64                  `json:"weight"`
	Stock        int64                  `json:"stock"`
	Child        []ProductChildResponse `json:"child"`
}

type ProductChildResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	SalePrice    int64  `json:"sale_price"`
	RegulerPrice int64  `json:"reguler_price"`
	Weight       int64  `json:"weight"`
	Stock        int64  `json:"stock"`
	Unit         string `json:"unit"`
	Image        string `json:"image"`
}

type ProductHomeListResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Image        string `json:"image"`
	CategoryName string `json:"category_name"`
	SalePrice    int64  `json:"sale_price"`
	RegulerPrice int64  `json:"reguler_price"`
}

type ProductChildHomeResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	SalePrice    int64  `json:"sale_price"`
	RegulerPrice int64  `json:"reguler_price"`
	Weight       int64  `json:"weight"`
	Stock        int64  `json:"stock"`
	Image        string `json:"image"`
}

type ProductHomeDetailResponse struct {
	ID           int64                      `json:"id"`
	Name         string                     `json:"name"`
	CategoryName string                     `json:"category_name"`
	CategorySlug string                     `json:"category_slug"`
	Description  string                     `json:"description"`
	Unit         string                     `json:"unit"`
	Image        string                     `json:"image"`
	SalePrice    int64                      `json:"sale_price"`
	RegulerPrice int64                      `json:"reguler_price"`
	Stock        int64                      `json:"stock"`
	Weight       int64                      `json:"weight"`
	Child        []ProductChildHomeResponse `json:"child"`
}
