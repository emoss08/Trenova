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
	TimeStampedModel
	ID               uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid();"`
	Status           string          `gorm:"size:10;default:'A'" json:"status" validate:"required,max=10"`
	Name             string          `gorm:"size:255;" json:"name" validate:"required"`
	EntityKey        string          `gorm:"size:10;" json:"entityKey" validate:"required"`
	AddressLine1     *string         `gorm:"size:255;" json:"addressLine1" validate:"required"`
	AddressLine2     *string         `gorm:"size:255;" json:"addressLine2" validate:"omitempty"`
	City             *string         `gorm:"size:100;" json:"city" validate:"omitempty,max=100"`
	State            *string         `gorm:"size:2;" json:"state" validate:"omitempty,max=2"`
	ZipCode          *string         `gorm:"size:5;" json:"zipCode" validate:"omitempty,max=5"`
	ContactEmail     *string         `gorm:"size:255;" json:"contactEmail" validate:"omitempty,email"`
	ContactPhone     *string         `gorm:"size:10;" json:"contactPhone" validate:"omitempty,e164,max=10"`
	Description      *string         `gorm:"type:text;" json:"description" validate:"omitempty"`
	PaidUntil        *time.Time      `gorm:"type:timestamp with time zone;" json:"paidUntil" validate:"omitempty"`
	FreeTrial        bool            `gorm:"default:false;" json:"freeTrial" validate:"omitempty"`
	BillingInfo      *datatypes.JSON `gorm:"type:jsonb;" json:"billingInfo" validate:"omitempty,json"`
	TaxID            string          `gorm:"size:255;" json:"taxId" validate:"required"`
	LegalName        string          `gorm:"size:255;" json:"legalName" validate:"required"`
	Metadata         *datatypes.JSON `gorm:"type:jsonb;" json:"metadata" validate:"omitempty,json"`
	Notes            *string         `gorm:"type:text;" json:"notes" validate:"omitempty"`
	IsSuspended      bool            `gorm:"default:false;" json:"isSuspended" validate:"omitempty"`
	SuspensionReason *string         `gorm:"type:text;" json:"suspensionReason" validate:"omitempty"`
	Contract         *string         `gorm:"type:text;" json:"contract" validate:"omitempty,filepath"`
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
