package models

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrgType string

const (
	Asset     OrgType = "A"
	Brokerage OrgType = "B"
	Both      OrgType = "X"
)

type Organization struct {
	TimeStampedModel
	Name              string            `json:"name" gorm:"type:varchar(100);not null;uniqueIndex:idx_organization_business_unit_name,expression:lower(name)" validate:"required,max=100"`
	ScacCode          string            `json:"scacCode" gorm:"type:varchar(4);not null;unique" validate:"required,max=4"`
	OrgType           OrgType           `json:"orgType" gorm:"type:org_type;not null" validate:"required,oneof=A B X,max=1"`
	DOTNumber         string            `json:"dotNumber" gorm:"type:varchar(12);not null;unique" validate:"required,max=12"`
	LogoURL           *string           `json:"logoUrl" gorm:"type:varchar(255);" validate:"omitempty,url"`
	BusinessUnitID    uuid.UUID         `json:"businessUnitId" gorm:"type:uuid;not null;uniqueIndex:idx_organization_business_unit_name" validate:"required"`
	Timezone          TimezoneType      `json:"timezone" gorm:"type:timezone_type;not null;default:'America/Los_Angeles'" validate:"omitempty,oneof=America/Los_Angeles America/Denver"`
	BusinessUnit      BusinessUnit      `json:"-" validate:"omitempty"`
	AccountingControl AccountingControl `json:"-" validate:"omitempty"`
}

func (o *Organization) BeforeCreate(tx *gorm.DB) (err error) {
	// Uppercase SCAC and DOT numbers, ensure the name is properly formatted
	o.ScacCode = strings.ToUpper(o.ScacCode)
	o.DOTNumber = strings.ToUpper(o.DOTNumber)

	// TODO(WOLFRED): Add validations for SCAC and DOT numbers as per their standards

	return
}

func (org *Organization) GetByID(db *gorm.DB, orgId, buId uuid.UUID) (Organization, error) {
	var organization Organization

	if err := db.Model(&Organization{}).Where("id = ?", orgId).First(&organization).Error; err != nil {
		return organization, err
	}

	return organization, nil
}
