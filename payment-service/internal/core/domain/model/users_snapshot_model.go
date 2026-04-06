package model

import "time"

type UserSnapshot struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"column:user_id;uniqueIndex"`
	Name      string    `gorm:"column:name"`
	Email     string    `gorm:"column:email"`
	Address   string    `gorm:"column:address"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (UserSnapshot) TableName() string {
	return "users_snapshot"
}
