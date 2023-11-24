package models

import (
	"time"

	"github.com/google/uuid"
)

type TimeStampedModel struct {
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP"`
}

type BaseModel struct {
	TimeStampedModel
	BusinessUnitID uuid.UUID `gorm:"type:uuid;" json:"businessUnitId"`
	BusinessUnit   BusinessUnit
	OrganizationID uuid.UUID `gorm:"type:uuid;" json:"organizationId"`
	Organization   Organization
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
}
