package message

import "errors"

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrUserNotFound      = errors.New("user not found")
	ErrProductNotFound   = errors.New("product not found")
	ErrPhoneIsRequired   = errors.New("phone number is required to place an order")
	ErrAddressIsRequired = errors.New("address is required to place an order")
)
