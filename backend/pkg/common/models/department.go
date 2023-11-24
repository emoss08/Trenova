package models

import "gorm.io/gorm"

type Department struct {
	gorm.Model
	BaseModel
	Name        string `gorm:"size:255;"`
	Description string `gorm:"type:text;"`
}
