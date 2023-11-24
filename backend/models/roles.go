package models

import "github.com/google/uuid"

type Permission struct {
	TimeStampedModel
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name     string    `gorm:"size:255;not null; unique" json:"name"`
	HelpText string    `gorm:"size:255;not null" json:"help_text"`
}

type Role struct {
	BaseModel
	Name        string       `gorm:"size:255;not null; unique" json:"name"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions"`
}
