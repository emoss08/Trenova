package models

import (
	"time"

	"github.com/google/uuid"
)

type TimeStampedModel struct {
	CreatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

type BaseModel struct {
	TimeStampedModel
	BusinessUnitID uuid.UUID `gorm:"type:uuid;" json:"businessUnitId" validate:"required,uuid"`
	BusinessUnit   BusinessUnit
	OrganizationID uuid.UUID `gorm:"type:uuid;" json:"organizationId" validate:"required,uuid"`
	Organization   Organization
	ID             uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
}
