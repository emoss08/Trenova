package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BusinessUnit struct {
	gorm.Model
	ID               uuid.UUID `gorm:"type:uuid;primary_key;"`
	Status           string    `gorm:"size:10;default:'A'"`
	Name             string    `gorm:"size:255;"`
	EntityKey        string    `gorm:"size:10;"`
	AddressLine1     *string   `gorm:"size:255;"`
	AddressLine2     *string   `gorm:"size:255;"`
	City             *string   `gorm:"size:100;"`
	State            *string   `gorm:"size:2;"` // Assuming US state codes
	ZipCode          *string   `gorm:"size:5;"` // Assuming US zip codes
	ContactEmail     *string   `gorm:"size:255;"`
	ContactPhone     *string   `gorm:"size:15;"`
	Description      *string   `gorm:"type:text;"`
	PaidUntil        *time.Time
	FreeTrial        bool `gorm:"default:false;"`
	BillingInfo      *datatypes.JSON
	TaxID            string `gorm:"size:255;"`
	LegalName        string `gorm:"size:255;"`
	Metadata         *datatypes.JSON
	Notes            *string `gorm:"type:text;"`
	IsSuspended      bool
	SuspensionReason *string `gorm:"type:text;"`
	Contract         string  `gorm:"type:text;"` // File paths or URLs to the contract
}

const (
	Active    = "A"
	Inactive  = "I"
	Suspended = "S"
)

func (b *BusinessUnit) BeforeCreate(tx *gorm.DB) (err error) {

	if b.EntityKey == "" {
		baseKey := strings.ToUpper(strings.ReplaceAll(b.Name, " ", "")[:8])

		var counter int64 = 1
		var entityKey string

		// Loop to find a unique entity key
		for {
			entityKey = fmt.Sprintf("%s%02d", baseKey, counter) // Start with 01
			var count int64
			tx.Model(&BusinessUnit{}).Where("entity_key = ?", entityKey).Count(&count)

			if count == 0 {
				break
			}
			counter++
		}

		b.EntityKey = entityKey
	}

	b.ID = uuid.New()

	return
}

func (b *BusinessUnit) Paid() bool {
	return b.PaidUntil != nil && b.PaidUntil.After(time.Now())
}
