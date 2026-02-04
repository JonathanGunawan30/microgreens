package request

type ProductRequest struct {
	Name          string                 `json:"name" validate:"required"`
	CategorySlug  string                 `json:"category_slug" validate:"required"`
	Unit          string                 `json:"unit" validate:"required"`
	Variant       int                    `json:"variant" validate:"required"`
	Description   string                 `json:"description" validate:"required"`
	Status        string                 `json:"status" validate:"required"`
	VariantDetail []ProductDetailRequest `json:"variant_detail" validate:"required"`
}

type ProductDetailRequest struct {
	Stock        int64  `json:"stock" validate:"required,number"`
	Image        string `json:"image" validate:"required,url"`
	Weight       int64  `json:"weight" validate:"required,number"`
	SalePrice    int64  `json:"sale_price" validate:"required,number"`
	RegulerPrice int64  `json:"reguler_price" validate:"required,number"`
}
