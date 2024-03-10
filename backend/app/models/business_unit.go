package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BusinessUnit struct {
	ID               uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid();"`
	Status           StatusType `json:"status" gorm:"type:status_type;not null;default:'A'" validate:"required,len=1,oneof=A I"`
	Name             string     `json:"name" gorm:"type:varchar(255);not null;uniqueIndex:idx_business_unit_name,expression:lower(name)" validate:"required,max=255"`
	EntityKey        string     `json:"entityKey" gorm:"type:varchar(10);not null;uniqueIndex:idx_entity_key,expression:lower(entity_key)" validate:"omitempty,len=10"`
	ContactName      *string    `json:"contactName" gorm:"type:varchar(255)" validate:"omitempty,max=255"`
	ContactEmail     *string    `json:"contactEmail" gorm:"type:string;" validate:"omitempty"`
	PaidUntil        *time.Time `json:"-" validate:"omitempty"`
	PhoneNumber      string     `json:"phoneNumber" gorm:"type:varchar(15)" validate:"omitempty,len=15"`
	Address          string     `json:"address" gorm:"type:text" validate:"omitempty,max=255"`
	City             string     `json:"city" gorm:"type:varchar(255)" validate:"omitempty,max=255"`
	State            string     `json:"state" gorm:"type:varchar(2)" validate:"len=2"`
	Country          string     `json:"country" gorm:"type:varchar(2)" validate:"len=2"`
	PostalCode       string     `json:"postalCode" gorm:"type:varchar(10)" validate:"omitempty,len=10"`
	ParentID         *uuid.UUID `json:"parentId" gorm:"type:uuid;index" validate:"omitempty"`
	Parent           *BusinessUnit
	Settings         *datatypes.JSON `json:"settings" validate:"omitempty"`
	TaxID            string          `json:"taxId" gorm:"type:varchar(20)" validate:"omitempty,len=20"`
	SubscriptionPlan string          `json:"subscriptionPlan" gorm:"type:string;not null" validate:"required"`
	Description      string          `json:"description" gorm:"type:text" validate:"omitempty,max=255"`
	FreeTrial        bool            `json:"freeTrial" gorm:"type:boolean;not null;default:false"`
	LegalName        string          `json:"legalName" gorm:"type:string;not null" validate:"required"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

func (b *BusinessUnit) BeforeCreate(tx *gorm.DB) error {
	if err := b.generateEntityKey(tx); err != nil {
		return err
	}

	return b.validateBusinessUnit()
}

func (b *BusinessUnit) BeforeUpdate(tx *gorm.DB) error {
	return b.validateBusinessUnit()
}

func (b *BusinessUnit) validateBusinessUnit() error {
	if b.Status != Active && b.Status != Inactive {
		return errors.New("status must be either 'A' or 'I'")
	}
	return nil
}

func (b *BusinessUnit) generateEntityKey(tx *gorm.DB) error {
	if b.EntityKey != "" {
		return nil
	}

	if b.Name == "" {
		return errors.New("the name of the business unit cannot be empty")
	}

	cleanedName := strings.ToUpper(strings.ReplaceAll(b.Name, " ", ""))
	baseKey := cleanedName
	if len(cleanedName) > 7 {
		baseKey = cleanedName[:7]
	}

	for counter := 1; counter <= 1000; counter++ {
		entityKey := fmt.Sprintf("%s%03d", baseKey, counter)
		var count int64
		if err := tx.Model(&BusinessUnit{}).Where("entity_key = ?", entityKey).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to check entity key uniqueness: %v", err)
		}

		if count == 0 {
			b.EntityKey = entityKey
			return nil
		}
	}

	return errors.New("unable to generate a unique entity key after 1000 attempts")
}
