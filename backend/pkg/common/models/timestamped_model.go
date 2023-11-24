package models

import (
	"time"

	"github.com/google/uuid"
)

type TimeStampedModel struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

type BaseModel struct {
	TimeStampedModel
	BusinessUnitID uuid.UUID `gorm:"type:uuid;"`
	BusinessUnit   BusinessUnit
	OrganizationID uuid.UUID `gorm:"type:uuid;"`
	Organization   Organization
	ID             uint `gorm:"primary_key"`
}
