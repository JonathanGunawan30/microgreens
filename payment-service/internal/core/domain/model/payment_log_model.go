package model

import "time"

type PaymentLog struct {
	ID        int64     `gorm:"primaryKey;column:id"`
	PaymentID int64     `gorm:"column:payment_id;not null;index"`
	Status    string    `gorm:"column:status;type:varchar(50);not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (PaymentLog) TableName() string {
	return "payment_logs"
}
