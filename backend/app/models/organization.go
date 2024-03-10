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
	Name              string            `gorm:"type:varchar(100);not null;uniqueIndex:idx_organization_business_unit_name,expression:lower(name)" json:"name"              validate:"required,max=100"`
	ScacCode          string            `gorm:"type:varchar(4);not null;unique"                                                                   json:"scacCode"          validate:"required,max=4"`
	DOTNumber         string            `gorm:"type:varchar(12);not null;unique"                                                                  json:"dotNumber"         validate:"required,max=12"`
	LogoURL           string            `gorm:"type:varchar(255);"                                                                                json:"logoUrl"           validate:"omitempty,url"`
	OrgType           OrgType           `gorm:"type:org_type;not null"                                                                            json:"orgType"           validate:"required,oneof=A B X,max=1"`
	Timezone          TimezoneType      `gorm:"type:timezone_type;not null;default:'America/Los_Angeles'"                                         json:"timezone"          validate:"omitempty,oneof=America/Los_Angeles America/Denver"`
	BusinessUnitID    uuid.UUID         `gorm:"type:uuid;not null;uniqueIndex:idx_organization_business_unit_name"                                json:"businessUnitId"    validate:"required"`
	BusinessUnit      BusinessUnit      `json:"-" validate:"omitempty"`
	AccountingControl AccountingControl `json:"-" validate:"omitempty"`
}

func (org *Organization) BeforeCreate(_ *gorm.DB) (err error) {
	// Uppercase SCAC and DOT numbers, ensure the name is properly formatted
	org.ScacCode = strings.ToUpper(org.ScacCode)
	org.DOTNumber = strings.ToUpper(org.DOTNumber)

	// TODO(WOLFRED): Add validations for SCAC and DOT numbers as per their standards

	return
}

func (org *Organization) AfterCreate(tx *gorm.DB) (err error) {
	// Create the accounting control record in a transaction
	ac := &AccountingControl{
		OrganizationID: org.ID,
		BusinessUnitID: org.BusinessUnitID,
	}
	if err = tx.Create(ac).Error; err != nil {
		return
	}

	return
}
