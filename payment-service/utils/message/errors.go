package message

import "errors"

var (
	ErrInvalidPaymentMethod = errors.New("invalid payment method")
	ErrPaymentNotFound      = errors.New("payment id not found")
	ErrOrderNotFound        = errors.New("order not found")
	ErrUserNotFound         = errors.New("user not found")
)
