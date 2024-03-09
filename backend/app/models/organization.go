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
	Name              string            `gorm:"type:varchar(255);not null;unique" json:"name" validate:"required,max=255"`
	ScacCode          string            `gorm:"type:varchar(4);not null;unique" json:"scacCode" validate:"required,len=4"`
	OrgType           OrgType           `gorm:"type:org_type;not null" json:"orgType" validate:"required,oneof=A B X,len=1"`
	DOTNumber         string            `gorm:"type:varchar(12);not null;unique" json:"dotNumber" validate:"required,len=12"`
	LogoURL           *string           `gorm:"type:varchar(255);" json:"logoUrl" validate:"omitempty,url"`
	BusinessUnitID    uuid.UUID         `gorm:"type:uuid;not null;" json:"businessUnitId"`
	BusinessUnit      BusinessUnit      `json:"-"`
	AccountingControl AccountingControl `json:"-"`
}

func (o *Organization) BeforeCreate(tx *gorm.DB) (err error) {
	// Uppercase SCAC and DOT numbers, ensure the name is properly formatted
	o.ScacCode = strings.ToUpper(o.ScacCode)
	o.DOTNumber = strings.ToUpper(o.DOTNumber)

	// TODO(WOLFRED): Add validations for SCAC and DOT numbers as per their standards

	return
}

func (org *Organization) GetOrganizationByID(db *gorm.DB, id uuid.UUID) (Organization, error) {
	var organization Organization

	if err := db.Model(&Organization{}).Where("id = ?", id).First(&organization).Error; err != nil {
		return organization, err
	}

	return organization, nil
}
