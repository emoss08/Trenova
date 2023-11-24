package models

import (
	"github.com/google/uuid"
)

type EmailControl struct {
	TimeStampedModel
	ID uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid();"`
	// Foreign Keys
	BillingEmailProfile          *EmailProfile
	BillingEmailProfileID        uuid.UUID `gorm:"type:uuid;" json:"billingEmailProfileId"`
	RateExpirationEmailProfileID uuid.UUID `gorm:"type:uuid;" json:"rateExpirationEmailProfileId"`
	RateExpirationEmailProfile   *EmailProfile
	OrganizationID               uuid.UUID `gorm:"type:uuid;" json:"organizationId"`
}
