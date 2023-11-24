package models

import (
	"github.com/google/uuid"
)

type EmailControl struct {
	TimeStampedModel
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid();" json:"id"`
	// Foreign Keys
	BillingEmailProfile          *EmailProfile
	BillingEmailProfileID        *uuid.UUID `gorm:"type:uuid;" json:"billingEmailProfileId" validate:"omitempty,uuid"`
	RateExpirationEmailProfile   *EmailProfile
	RateExpirationEmailProfileID *uuid.UUID `gorm:"type:uuid;" json:"rateExpirationEmailProfileId" validate:"omitempty,uuid"`
	OrganizationID               uuid.UUID  `gorm:"type:uuid;" json:"organizationId" validate:"required,uuid"`
}
