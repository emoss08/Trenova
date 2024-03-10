package models

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EmailProfile struct {
	TimeStampedModel
	OrganizationID uuid.UUID     `json:"organizationId" gorm:"type:uuid;not null;uniqueIndex:idx_email_profile_name_organization_id" validate:"required"`
	Organization   Organization  `json:"-" validate:"omitempty"`
	BusinessUnitID uuid.UUID     `json:"businessUnitId" gorm:"type:uuid;not null;index" validate:"required"`
	BusinessUnit   BusinessUnit  `json:"-" validate:"omitempty"`
	Name           string        `json:"name" gorm:"type:varchar(255);not null;uniqueIndex:idx_email_profile_name_organization_id,expression:lower(name)" validate:"required"`
	Email          string        `json:"email" gorm:"type:varchar(255);not null" validate:"required,email"`
	Protocol       EmailProtocol `json:"protocol" gorm:"type:email_protocol_type;not null" validate:"omitempty"`
	Host           string        `json:"host" gorm:"type:varchar(255);not null" validate:"required"`
	Port           int           `json:"port" gorm:"type:integer;not null" validate:"required"`
	Username       string        `json:"username" gorm:"type:varchar(255);not null" validate:"required"`
	Password       string        `json:"password" gorm:"type:varchar(255);not null" validate:"required"`
	DefaultProfile bool          `json:"defaultProfile" gorm:"type:boolean;not null;default:false"`
}

var ErrDefaultEmailProfileExists = errors.New("default email profile already exists for the organization")

func (e *EmailProfile) BeforeCreate(tx *gorm.DB) error {
	if e.Protocol == "" {
		e.Protocol = Unencrypted
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
