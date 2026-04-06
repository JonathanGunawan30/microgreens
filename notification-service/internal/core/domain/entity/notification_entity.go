package entity

import "time"

type NotificationEntity struct {
	ID               int64      `json:"id"`
	Message          string     `json:"message"`
	NotificationType string     `json:"notification_type"`
	Status           *string    `json:"status"`
	ReceiverID       *int64     `json:"receiver_id"`
	Subject          *string    `json:"subject"`
	ReceiverEmail    *string    `json:"receiver_email"`
	SendAt           *time.Time `json:"send_at"`
	ReadAt           *time.Time `json:"read_at"`
}

type NotifQueryString struct {
	Page      int64  `query:"page"`
	Search    string `query:"search"`
	Limit     int64  `query:"limit"`
	Status    string `query:"status"`
	OrderType string `query:"order_type"`
	OrderBy   string `query:"order_by"`
	IsRead    bool   `query:"is_read"`
}
