package models

type Department struct {
	BaseModel
	Name        string `gorm:"size:255;"`
	Description string `gorm:"type:text;"`
}
