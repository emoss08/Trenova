package models

// import (
// 	"errors"
// 	"fmt"
// 	"strings"
// 	"time"

// 	"github.com/google/uuid"
// 	"gorm.io/datatypes"
// 	"gorm.io/gorm"
// )

// type StatusType string

// const (
// 	Active   StatusType = "A"
// 	Inactive StatusType = "I"
// )

// type BusinessUnit struct {
// 	ID               uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid();" json:"id"`
// 	Status           StatusType `gorm:"type:status_type;not null;default:'A'" json:"status" validate:"required,len=1,oneof=A I"`
// 	Name             string     `gorm:"type:varchar(255);not null;uniqueIndex:idx_business_unit_name,expression:lower(name)" json:"name" validate:"required,max=255"`
// 	EntityKey        string     `gorm:"type:varchar(10);not null;uniqueIndex:idx_entity_key,expression:lower(entity_key)" json:"entityKey" validate:"omitempty,len=10"`
// 	PhoneNumber      string     `gorm:"type:varchar(15)" json:"phoneNumber" validate:"omitempty,len=15"`
// 	Address          string     `gorm:"type:text" json:"address" validate:"omitempty,max=255"`
// 	City             string     `gorm:"type:varchar(255)" json:"city" validate:"omitempty,max=255"`
// 	State            string     `gorm:"type:varchar(2)" json:"state" validate:"len=2"`
// 	Country          string     `gorm:"type:varchar(2)" json:"country" validate:"len=2"`
// 	PostalCode       string     `gorm:"type:varchar(10)" json:"postalCode" validate:"omitempty,len=10"`
// 	TaxID            string     `gorm:"type:varchar(20)" json:"taxId" validate:"omitempty,len=20"`
// 	SubscriptionPlan string     `gorm:"type:string;not null" json:"subscriptionPlan" validate:"required"`
// 	Description      string     `gorm:"type:text" json:"description" validate:"omitempty,max=255"`
// 	LegalName        string     `gorm:"type:string;not null" json:"legalName" validate:"required"`
// 	ContactName      *string    `gorm:"type:varchar(255)" json:"contactName" validate:"omitempty,max=255"`
// 	ContactEmail     *string    `gorm:"type:string" json:"contactEmail" validate:"omitempty"`
// 	PaidUntil        *time.Time `json:"-" validate:"omitempty"`
// 	ParentID         *uuid.UUID `gorm:"type:uuid;index" json:"parentId" validate:"omitempty"`
// 	Parent           *BusinessUnit
// 	Settings         *datatypes.JSON `json:"settings" validate:"omitempty"`
// 	FreeTrial        bool            `gorm:"type:boolean;not null;default:false" json:"freeTrial" `
// 	CreatedAt        time.Time       `json:"createdAt"`
// 	UpdatedAt        time.Time       `json:"updatedAt"`
// }

// var (
// 	errBusUnitNameEmpty = errors.New("the name of the business unit cannot be empty")
// 	errGenEntityKey     = errors.New("unable to generate a unique entity key after 1000 attempts")
// 	errInvalidStatus    = errors.New("status must be either 'A' or 'I'")
// )

// const maxEntityKeyValue = 7

// func (b *BusinessUnit) BeforeCreate(tx *gorm.DB) error {
// 	if err := b.generateEntityKey(tx); err != nil {
// 		return err
// 	}

// 	return b.validateBusinessUnit()
// }

// func (b *BusinessUnit) BeforeUpdate(_ *gorm.DB) error {
// 	return b.validateBusinessUnit()
// }

// func (b *BusinessUnit) validateBusinessUnit() error {
// 	if b.Status != Active && b.Status != Inactive {
// 		return errInvalidStatus
// 	}

// 	return nil
// }

// func (b *BusinessUnit) generateEntityKey(tx *gorm.DB) error {
// 	if b.EntityKey != "" {
// 		return nil
// 	}

// 	if b.Name == "" {
// 		return errBusUnitNameEmpty
// 	}

// 	cleanedName := strings.ToUpper(strings.ReplaceAll(b.Name, " ", ""))
// 	baseKey := cleanedName

// 	if len(cleanedName) > maxEntityKeyValue {
// 		baseKey = cleanedName[:7]
// 	}

// 	for counter := 1; counter <= 1000; counter++ {
// 		entityKey := fmt.Sprintf("%s%03d", baseKey, counter)

// 		var count int64

// 		if err := tx.Model(&BusinessUnit{}).Where("entity_key = ?", entityKey).Count(&count).Error; err != nil {
// 			return fmt.Errorf("failed to check entity key uniqueness: %w", err)
// 		}

// 		if count == 0 {
// 			b.EntityKey = entityKey
// 			return nil
// 		}
// 	}

// 	return errGenEntityKey
// }
