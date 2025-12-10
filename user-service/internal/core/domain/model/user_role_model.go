package model

import "time"

type UserRole struct {
	ID        int64 `gorm:"primaryKey"`
	RoleID    int64 `gorm:"index"`
	UserID    int64 `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func (UserRole) TableName() string {
	return "user_role"
}
