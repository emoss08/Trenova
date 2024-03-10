package models

import "github.com/google/uuid"

type Tag struct {
	TimeStampedModel
	OrganizationID uuid.UUID    `json:"organizationId" gorm:"type:uuid;not null;uniqueIndex:idx_tag_name_organization_id" validate:"required"`
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnitID uuid.UUID    `json:"businessUnitId" gorm:"type:uuid;not null;index;" validate:"required"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
	Name           string       `json:"name" gorm:"type:varchar(50);not null;uniqueIndex:idx_tag_name_organization_id,expression:lower(name)" validate:"required,max=50"`
	Description    string       `json:"description" gorm:"type:text;not null;" validate:"omitempty,max=255"`
}
