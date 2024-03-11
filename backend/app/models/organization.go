package models

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrgType string

const (
	OrgTypeAsset OrgType = "A"
	OrgBrokerage OrgType = "B"
	OrgBoth      OrgType = "X"
)

type Organization struct {
	TimeStampedModel
	Name                   string                 `gorm:"type:varchar(100);not null;uniqueIndex:idx_organization_business_unit_name,expression:lower(name)" json:"name"              validate:"required,max=100"`
	ScacCode               string                 `gorm:"type:varchar(4);not null;unique"                                                                   json:"scacCode"          validate:"required,max=4"`
	DOTNumber              string                 `gorm:"type:varchar(12);not null;unique"                                                                  json:"dotNumber"         validate:"required,max=12"`
	LogoURL                *string                `gorm:"type:varchar(255);"                                                                                json:"logoUrl"           validate:"omitempty,url"`
	OrgType                OrgType                `gorm:"type:org_type;not null"                                                                            json:"orgType"           validate:"required,oneof=A B X,max=1"`
	Timezone               TimezoneType           `gorm:"type:timezone_type;not null;default:'America/Los_Angeles'"                                         json:"timezone"          validate:"omitempty,oneof=America/Los_Angeles America/Denver"`
	BusinessUnitID         uuid.UUID              `gorm:"type:uuid;not null;uniqueIndex:idx_organization_business_unit_name"                                json:"businessUnitId"    validate:"required"`
	BusinessUnit           BusinessUnit           `json:"-" validate:"omitempty"`
	AccountingControl      AccountingControl      `json:"-" validate:"omitempty"`
	BillingControl         BillingControl         `json:"-" validate:"omitempty"`
	InvoiceControl         InvoiceControl         `json:"-" validate:"omitempty"`
	DispatchControl        DispatchControl        `json:"-" validate:"omitempty"`
	ShipmentControl        ShipmentControl        `json:"-" validate:"omitempty"`
	RouteControl           RouteControl           `json:"-" validate:"omitempty"`
	FeasibilityToolControl FeasibilityToolControl `json:"-" validate:"omitempty"`
}

func (org *Organization) BeforeCreate(_ *gorm.DB) (err error) {
	// Uppercase SCAC and DOT numbers, ensure the name is properly formatted
	org.ScacCode = strings.ToUpper(org.ScacCode)
	org.DOTNumber = strings.ToUpper(org.DOTNumber)

	// TODO(WOLFRED): Add validations for SCAC and DOT numbers as per their standards

	return nil
}

// AfterCreate is a hook that is called after a new organization is created.
func (org *Organization) AfterCreate(tx *gorm.DB) error {
	controls := []interface{}{
		&AccountingControl{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID},
		&InvoiceControl{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID},
		&BillingControl{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID},
		&DispatchControl{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID},
		&ShipmentControl{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID},
		&RouteControl{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID},
		&FeasibilityToolControl{OrganizationID: org.ID, BusinessUnitID: org.BusinessUnitID},
	}

	for _, control := range controls {
		if err := tx.Create(control).Error; err != nil {
			return fmt.Errorf("error creating control (%T) for Organization: %w", control, err)
		}
	}

	return nil
}
