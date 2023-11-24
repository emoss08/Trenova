package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	BaseModel
	IsActive   bool   `gorm:"default:true;"`
	Username   string `gorm:"size:255;"`
	IsStaff    bool   `gorm:"default:false;"`
	DateJoined string `gorm:"type:date;"`
	SessionKey string `gorm:"size:255;"`
}
