package model

import "time"

type Category struct {
	ID           int64 `gorm:"primaryKey"`
	ParentID     *int64
	Name         string
	Icon         string
	Status       bool   `gorm:"default:true"`
	Slug         string `gorm:"uniqueIndex"`
	Description  string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
	Products     []Product `gorm:"foreignKey:CategorySlug;references:Slug"`
	ProductCount int64     `gorm:"->;dataType:int"`
}
