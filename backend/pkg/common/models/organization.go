package models

import (
	"strings"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

type Organization struct {
	TimeStampedModel
	gorm.Model
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;"`
	Name                string    `gorm:"size:255;"`
	ScacCode            string    `gorm:"size:4;"`
	DOTNumber           *uint
	AddressLine1        *string `gorm:"size:255;"`
	AddressLine2        *string `gorm:"size:255;"`
	City                *string `gorm:"size:255;"`
	State               *string `gorm:"size:2;"`
	ZipCode             *string `gorm:"size:5;"`
	PhoneNumber         *string `gorm:"size:20;"`
	Website             *string `gorm:"size:255;"`
	OrgType             string  `gorm:"size:10;"`
	Timezone            string  `gorm:"size:255;default:'America/New_York';"`
	Language            string  `gorm:"size:2;default:'en';"`
	Currency            string  `gorm:"size:255;default:'USD';"`
	DateFormat          string  `gorm:"size:255;default:'MM/DD/YYYY';"`
	TimeFormat          string  `gorm:"size:255;default:'HH:mm';"`
	Logo                string
	TokenExpirationDays uint      `gorm:"default:30;"`
	BusinessUnitID      uuid.UUID `gorm:"type:uuid;"`
	BusinessUnit        BusinessUnit
}

func (o *Organization) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}

	caser := cases.Title(language.AmericanEnglish)
	o.Name = caser.String(o.Name)

	o.ScacCode = strings.ToUpper(o.ScacCode)
	return nil
}
