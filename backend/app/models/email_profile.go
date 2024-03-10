package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailProfile struct {
	TimeStampedModel
	OrganizationID uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:idx_email_profile_name_organization_id" json:"organizationId" validate:"required"`
	BusinessUnitID uuid.UUID     `gorm:"type:uuid;not null;index"                                              json:"businessUnitId" validate:"required"`
	Organization   Organization  `json:"-" validate:"omitempty"`
	BusinessUnit   BusinessUnit  `json:"-" validate:"omitempty"`
	Protocol       EmailProtocol `gorm:"type:email_protocol_type;not null"                                                                    json:"protocol" validate:"omitempty"`
	Email          string        `gorm:"type:varchar(255);not null"                                                                           json:"email" validate:"required,email"`
	Name           string        `gorm:"type:varchar(255);not null;uniqueIndex:idx_email_profile_name_organization_id,expression:lower(name)" json:"name" validate:"required"`
	Host           string        `gorm:"type:varchar(255);not null"                                                                           json:"host" validate:"required"`
	Username       string        `gorm:"type:varchar(255);not null"                                                                           json:"username" validate:"required"`
	Password       string        `gorm:"type:varchar(255);not null"                                                                           json:"password" validate:"required"`
	DefaultProfile bool          `gorm:"type:boolean;not null;default:false"                                                                  json:"defaultProfile"`
	Port           int           `gorm:"type:integer;not null"                                                                                json:"port" validate:"required"`
}

var ErrDefaultEmailProfileExists = errors.New("default email profile already exists for the organization")

func (e *EmailProfile) BeforeCreate(tx *gorm.DB) error {
	if e.Protocol == "" {
		e.Protocol = EmailProtocolUnencrypted
	}

	if e.DefaultProfile {
		var count int64
		if err := tx.Model(&EmailProfile{}).Where("organization_id = ? AND default_profile = ?", e.OrganizationID, true).Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return ErrDefaultEmailProfileExists
		}
	}

	return nil
}

func (e *EmailProfile) BeforeUpdate(tx *gorm.DB) error {
	if e.DefaultProfile {
		var count int64
		// Exclude the current record from the count when checking for existing default profiles.
		if err := tx.Model(&EmailProfile{}).Where("organization_id = ? AND default_profile = ? AND id <> ?", e.OrganizationID, true, e.ID).Count(&count).Error; err != nil {
			return err
		}

		// If there's another default profile (excluding this one), return an error.
		if count > 0 {
			return ErrDefaultEmailProfileExists
		}
	}

	return nil
}
