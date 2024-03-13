package models

import "github.com/google/uuid"

// HazardousMaterial represents a hazardous material.
type HazardousMaterial struct {
	BaseModel
	Organization       Organization `json:"-" validate:"omitempty"`
	BusinessUnit       BusinessUnit `json:"-" validate:"omitempty"`
	OrganizationID     uuid.UUID    `gorm:"type:uuid;not null;index"           json:"organizationId"     validate:"required"`
	BusinessUnitID     uuid.UUID    `gorm:"type:uuid;not null;index"           json:"businessUnitId"     validate:"required"`
	Name               string       `gorm:"type:varchar(100);not null;index"   json:"name"               validate:"required"`
	HazardClass        string       `gorm:"type:hazardous_class_type;not null" json:"hazardClass"        validate:"required"`
	ERGNumber          *string      `gorm:"type:varchar(255);"                 json:"ergNumber"          validate:"omitempty"`
	Description        *string      `gorm:"type:text;"                         json:"description"        validate:"omitempty"`
	PackingGroup       *string      `gorm:"type:packing_group_type;"           json:"packingGroup"       validate:"omitempty"`
	ProperShippingName *string      `gorm:"type:text;"                         json:"properShippingName" validate:"omitempty"`
}
