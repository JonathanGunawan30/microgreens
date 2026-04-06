package request

type CreateCategoryRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Icon        string `json:"icon" validate:"required,max=255"`
	Description string `json:"description" validate:"max=500"`
	Status      bool   `json:"status" validate:"required"`
	ParentID    *int64 `json:"parent_id"`
}

type UpdateCategoryRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Icon        string `json:"icon" validate:"required,max=255"`
	Description string `json:"description" validate:"max=500"`
	Status      bool   `json:"status" validate:"required"`
	ParentID    *int64 `json:"parent_id"`
}
