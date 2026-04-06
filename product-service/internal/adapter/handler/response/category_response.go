package response

type CategoryListAdminResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	Slug         string `json:"slug"`
	Status       bool   `json:"status"`
	TotalProduct int64  `json:"total_product"`
}

type CategoryResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
	Status      bool   `json:"status"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

type CategoryListHomeResponse struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
	Slug string `json:"slug"`
}

type CategoryListShopChildResponse struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type CategoryListShopResponse struct {
	Name  string                           `json:"name"`
	Slug  string                           `json:"slug"`
	Child []*CategoryListShopChildResponse `json:"child,omitempty"`
}
