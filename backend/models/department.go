package models

type Department struct {
	BaseModel
	Name        string `gorm:"size:255;" json:"name" validate:"required"`
	Description string `gorm:"type:text;" json:"description" validate:"required"`
}
