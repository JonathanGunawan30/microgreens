package request

type ProductRequest struct {
	Name          string                 `json:"name" validate:"required,max=100"`
	CategorySlug  string                 `json:"category_slug" validate:"required,max=100"`
	Unit          string                 `json:"unit" validate:"required,max=100"`
	Variant       int                    `json:"variant" validate:"required"`
	Description   string                 `json:"description" validate:"required,max=500"`
	Status        string                 `json:"status" validate:"required"`
	VariantDetail []ProductDetailRequest `json:"variant_detail" validate:"required"`
}

type ProductDetailRequest struct {
	Stock        int64  `json:"stock" validate:"required,number"`
	Image        string `json:"image" validate:"required,url,max=255"`
	Weight       int64  `json:"weight" validate:"required,number"`
	SalePrice    int64  `json:"sale_price" validate:"required,number"`
	RegulerPrice int64  `json:"reguler_price" validate:"required,number"`
}
