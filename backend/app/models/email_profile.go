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
	Email          string        `gorm:"type:varchar(255);not null"                                                                           json:"email"    validate:"required,email"`
	Name           string        `gorm:"type:varchar(255);not null;uniqueIndex:idx_email_profile_name_organization_id,expression:lower(name)" json:"name"     validate:"required"`
	DefaultProfile bool          `gorm:"type:boolean;not null;default:false"                                                                  json:"defaultProfile"`
	Host           *string       `gorm:"type:varchar(255);"                                                                                   json:"host"     validate:"omitempty"`
	Username       *string       `gorm:"type:varchar(255);"                                                                                   json:"username" validate:"omitempty"`
	Password       *string       `gorm:"type:varchar(255);"                                                                                   json:"password" validate:"omitempty"`
	Port           *int          `gorm:"type:integer;"                                                                                        json:"port"     validate:"omitempty"`
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

func (e *EmailProfile) FetchEmailProfilesForOrg(db *gorm.DB, orgID, buID uuid.UUID, offset, limit int) ([]EmailProfile, int64, error) {
	var emailProfiles []EmailProfile

	var totalRows int64

	if err := db.Model(&EmailProfile{}).Where("organization_id = ? AND business_unit_id = ?", orgID, buID).Count(&totalRows).Error; err != nil {
		return emailProfiles, 0, err
	}

	if err := db.Model(&EmailProfile{}).Where("organization_id = ? AND business_unit_id = ?", orgID, buID).Offset(offset).Limit(limit).Order("created_at desc").Find(&emailProfiles).Error; err != nil {
		return emailProfiles, 0, err
	}

	return emailProfiles, totalRows, nil
}

func (e *EmailProfile) FetchEmailProfileDetails(db *gorm.DB, orgID, buID uuid.UUID, id string) (EmailProfile, error) {
	var emailProfile EmailProfile

	if err := db.Model(&EmailProfile{}).Where("organization_id = ? AND id = ? AND business_unit_id = ?", orgID, id, buID).First(&emailProfile).Error; err != nil {
		return emailProfile, err
	}

	return emailProfile, nil
}
