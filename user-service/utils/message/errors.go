package message

import "errors"

var (
	ErrInvalidCredential  = errors.New("invalid credential")
	ErrUserNotFound       = errors.New("user not found")
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenExpired       = errors.New("token expired")
	ErrSessionFailed      = errors.New("session failed")
	ErrFileTooLarge       = errors.New("file size exceeds the maximum limit of 1 MB")
	ErrInvalidMime        = errors.New("invalid file type, only PNG, JPG, and WEBP are allowed")
	ErrRoleNotFound       = errors.New("role not found")
	ErrRoleAssociated     = errors.New("role cannot be deleted because it is still assigned to one or more users")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrWrongPassword      = errors.New("password is wrong")
)
