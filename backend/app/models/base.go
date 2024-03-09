package models

import (
	"time"

	"github.com/google/uuid"
)

type TimeStampedModel struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid();"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type BaseModel struct {
	TimeStampedModel
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null" json:"organizationId" validate:"required"`
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnitID uuid.UUID    `gorm:"type:uuid;not null" json:"businessUnitId" validate:"required"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
}
