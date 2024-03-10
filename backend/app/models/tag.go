package models

import "github.com/google/uuid"

type Tag struct {
	TimeStampedModel
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_tag_name_organization_id"                                json:"organizationId" validate:"required"`
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnitID uuid.UUID    `gorm:"type:uuid;not null;index;"                                                                  json:"businessUnitId" validate:"required"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`
	Name           string       ` gorm:"type:varchar(50);not null;uniqueIndex:idx_tag_name_organization_id,expression:lower(name)" json:"name"           validate:"required,max=50"`
	Description    string       ` gorm:"type:text;not null;"                                                                       json:"description"    validate:"omitempty,max=255"`
}
