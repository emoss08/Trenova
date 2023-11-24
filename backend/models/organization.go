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
	EN Language = "en-US" // American English
	ES Language = "es-US" // American Spanish
)

type DateFormat string

const (
	MMDDYYYY DateFormat = "MM/DD/YYYY"
	DDMMYYYY DateFormat = "DD/MM/YYYY"
	YYYYMMDD DateFormat = "YYYY/MM/DD"
)

type TimeFormat string

const (
	HHmm    TimeFormat = "HH:mm"
	HHmmss  TimeFormat = "HH:mm:ss"
	HHmmssZ TimeFormat = "HH:mm:ssZ"
)

type Organization struct {
	TimeStampedModel
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name                string    `gorm:"size:255;" json:"name" validate:"required"`
	ScacCode            string    `gorm:"size:4;" json:"scacCode" validate:"required,max=4"`
	DOTNumber           *uint     `gorm:"size:12" json:"dotNumber" validate:"omitempty,startswith=USDOT,max=12"` // Example: USDOT1523020
	AddressLine1        *string   `gorm:"size:255;" json:"addressLine1" validate:"omitempty"`
	AddressLine2        *string   `gorm:"size:255;" json:"addressLine2" validate:"omitempty"`
	City                *string   `gorm:"size:255;" json:"city" validate:"omitempty"`
	State               *string   `gorm:"size:2;" json:"state" validate:"omitempty,max=2"`
	ZipCode             *string   `gorm:"size:10;" json:"zipCode" validate:"omitempty,usazipcode,max=10"`
	PhoneNumber         *string   `gorm:"size:12;" json:"phoneNumber" validate:"omitempty,e164,max=10"` // Example: +15555555555
	Website             *string   `gorm:"size:255;" json:"website" validate:"omitempty,url"`
	OrgType             OrgType   `gorm:"size:10;type:org_type" json:"orgType" validate:"required,oneof=ASSET BROKERAGE BOTH,max=10"`
	Timezone            string    `gorm:"size:255;default:'America/New_York';" json:"timezone" validate:"required,timezone"`
	Language            Language  `gorm:"size:5;type:lang_type;default:'en-US';" json:"language" validate:"required,bcp47_language_tag,max=5,oneof=en-US es-US"`
	Currency            string    `gorm:"size:255;default:'USD';" json:"currency" validate:"required,iso4217"`
	DateFormat          string    `gorm:"size:10;default:'MM/DD/YYYY';" json:"dateFormat" validate:"required,oneof=MM/DD/YYYY DD/MM/YYYY YYYY/MM/DD,max=10"`
	TimeFormat          string    `gorm:"size:9;default:'HH:mm';" json:"timeFormat" validate:"required,oneof=HH:mm HH:mm:ss HH:mm:ssZ,max=9"`
	Logo                string    `gorm:"size:255;" json:"logo" validate:"image"`
	TokenExpirationDays uint      `gorm:"default:30;" json:"tokenExpirationDays" validate:"required,number"`
	BusinessUnitID      uuid.UUID `gorm:"type:uuid;" json:"businessUnitId" validate:"required"`
	BusinessUnit        BusinessUnit
	EmailControl        EmailControl
}

func (o *Organization) BeforeCreate(tx *gorm.DB) (err error) {
	caser := cases.Title(language.AmericanEnglish)
	o.Name = caser.String(o.Name)

	o.ScacCode = strings.ToUpper(o.ScacCode)
	return nil
}
