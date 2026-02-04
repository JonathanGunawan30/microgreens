package message

import "errors"

var (
	ErrCategoryNotFound      = errors.New("category not found")
	ErrCategoryHasProducts   = errors.New("cannot delete category, products are still associated with this category")
	ErrCategoryAlreadyExists = errors.New("category name already exists")
	ErrProductNotFound       = errors.New("product not found")
	ErrFileTooLarge          = errors.New("file size exceeds the maximum limit of 1 MB")
	ErrInvalidMime           = errors.New("invalid file type, only PNG, JPG, and WEBP are allowed")
)
