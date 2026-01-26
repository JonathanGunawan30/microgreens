package message

import "errors"

var (
	ErrCategoryNotFound      = errors.New("category not found")
	ErrCategoryHasProducts   = errors.New("cannot delete category, products are still associated with this category")
	ErrCategoryAlreadyExists = errors.New("category name already exists")
)
