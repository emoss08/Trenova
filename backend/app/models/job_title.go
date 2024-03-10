package models

import "github.com/google/uuid"

type JobTitle struct {
	TimeStampedModel
	OrganizationID uuid.UUID    `gorm:"type:uuid;not null;index" json:"organizationId" validate:"required"`
	Organization   Organization `json:"-" validate:"omitempty"`
	BusinessUnitID uuid.UUID    `json:"businessUnitId" gorm:"type:uuid;not null" validate:"required"`
	BusinessUnit   BusinessUnit `json:"-" validate:"omitempty"`

	Status      StatusType      `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"required,len=1,oneof=A I"`
	Name        string          `json:"name" gorm:"type:varchar(100);not null;" validate:"required,max=100"`
	Description *string         `json:"description" gorm:"type:varchar(100);" validate:"required,max=100"`
	JobFunction JobFunctionType `json:"jobFunction" gorm:"type:job_function_type;not null;" validate:"required,len=1,oneof=MGR MT SP D B F S SA A"`
}
