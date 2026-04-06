package model

import (
	"time"
)

type Notification struct {
	ID               int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Message          string     `gorm:"type:text;not null" json:"message"`
	NotificationType string     `gorm:"type:varchar(50);not null" json:"notification_type"`
	Status           *string    `gorm:"type:varchar(50)" json:"status"`
	ReceiverID       *int64     `json:"receiver_id"`
	Subject          *string    `gorm:"type:varchar(255)" json:"subject"`
	ReceiverEmail    *string    `gorm:"type:varchar(255)" json:"receiver_email"`
	SendAt           *time.Time `gorm:"column:send_at;type:timestamp" json:"send_at"`
	ReadAt           *time.Time `gorm:"column:read_at;type:timestamp" json:"read_at"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt        time.Time  `gorm:"index" json:"-"`
}

func (Notification) TableName() string {
	return "notifications"
}
