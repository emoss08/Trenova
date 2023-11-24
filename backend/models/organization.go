package models

import (
	"strings"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

type OrgType string

const (
	ASSET     OrgType = "ASSET"
	BROKERAGE OrgType = "BROKERAGE"
	BOTH      OrgType = "BOTH"
)

type Language string

const (
	EN Language = "en"
	ES Language = "es"
)

type Organization struct {
	TimeStampedModel
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name                string    `gorm:"size:255;"`
	ScacCode            string    `gorm:"size:4;" json:"scacCode"`
	DOTNumber           *uint     `json:"dotNumber"`
	AddressLine1        *string   `gorm:"size:255;" json:"addressLine1"`
	AddressLine2        *string   `gorm:"size:255;" json:"addressLine2"`
	City                *string   `gorm:"size:255;"`
	State               *string   `gorm:"size:2;"`
	ZipCode             *string   `gorm:"size:5;" json:"zipCode"`
	PhoneNumber         *string   `gorm:"size:20;" json:"phoneNumber"`
	Website             *string   `gorm:"size:255;"`
	OrgType             OrgType   `gorm:"size:10;" json:"orgType"`
	Timezone            string    `gorm:"size:255;default:'America/New_York';"`
	Language            Language  `gorm:"size:2;default:'en';"`
	Currency            string    `gorm:"size:255;default:'USD';"`
	DateFormat          string    `gorm:"size:255;default:'MM/DD/YYYY';" json:"dateFormat"`
	TimeFormat          string    `gorm:"size:255;default:'HH:mm';" json:"timeFormat"`
	Logo                string
	TokenExpirationDays uint      `gorm:"default:30;" json:"tokenExpirationDays"`
	BusinessUnitID      uuid.UUID `gorm:"type:uuid;" json:"businessUnitId"`
	BusinessUnit        BusinessUnit
	EmailControl        EmailControl
}

func (o *Organization) BeforeCreate(tx *gorm.DB) (err error) {
	caser := cases.Title(language.AmericanEnglish)
	o.Name = caser.String(o.Name)

	o.ScacCode = strings.ToUpper(o.ScacCode)
	return nil
}
